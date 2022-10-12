/**
* (C) 2022 Geotab
*
* All files and artifacts in the repository at https://github.com/w3c/automotive-viss2
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package main

import (
	"fmt"
	"encoding/json"
	"io/ioutil"
//	"bytes"
	"os"
	"os/exec"

	"crypto/tls"
	"crypto/x509"
	"net/http"

	"strconv"
	"strings"
	"time"

	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/akamensky/argparse"
	"github.com/w3c/automotive-viss2/utils"
)

const IGNITION_ON_PATH = "Vehicle.LowVoltageSystemState"

var muxServer *http.ServeMux
var clientCert tls.Certificate
var caCertPool x509.CertPool

var dbHandle *sql.DB
var dbErr error
var vinId string

const MAX_ACCUMULATED_TIME = 30  // in seconds
const ovdsDbFileName = "ovdsCESC.db"

type DataMapItem struct {
	VssPath string `json:"VssPath"`
	CloudType string `json:"CloudType"`
	CloudKey string `json:"CloudKey"`
	CloudValue string `json:"CloudValue"`
}
type DataMapList struct {
	DataMap []DataMapItem
}
var dataMapList DataMapList

type GpsData struct {
	Value string
	Ts string
}
var lat GpsData  //TODO This set of variables should be per vehicle Id
var long GpsData
var ignitionOn string
var speed string

func initCeClient() {
	lat.Ts = ""
	long.Ts = ""
	ignitionOn = "OFF"
	speed = "0"
}

func writeDpsToCloud(tokenChan chan string, vinId string) {
	checkToken(tokenChan) //make sure fresh token to be used
	latestTsIngested := ingestDps(vinId)
	updateAccumulatedData(vinId, latestTsIngested)
}

func ingestDps(vinId string) string {  // retrieve all new data points for vinId, issue ingest request to DIG
	latestTsIngested := getAccumulatedDataTs(vinId)
	payload, newLatestTs := retrieveDigPayload(vinId, latestTsIngested)
	url := "https://dig.geotab.com:443/records"

	curl := exec.Command("curl", "--header", "Authorization: Bearer " + digData.Token, "--header", "content-type:application/json", "--data", payload, url)
	resp, err := curl.Output()
	    if err != nil {
	    utils.Error.Printf("curl error=%s", err)
	    utils.Error.Printf("ingestDps:Payload related to the error = %s ", payload)
	    return latestTsIngested
	}
	utils.Info.Printf("ingestDps: Ingest response = %s ", string(resp))
	utils.Info.Printf("ingestDps: Successful ingest of payload = %s", payload)
	return newLatestTs

/*
	url := "https://dig.geotab.com:443/records"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		utils.Error.Printf("ingestDps: Error creating request=%s.", err)
		return latestTsIngested
	}

	// Set headers
	req.Header.Set("Access-Control-Allow-Origin", "*")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", "dig.geotab.com:443")
	req.Header.Set("Authorization", tokenData.Token)

	// Configure client
	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{clientCert},
			RootCAs:      &caCertPool,
		},
	}
	client := &http.Client{Transport: t, Timeout: 10 * time.Second}

	// Send request
	resp, err := client.Do(req)
//	if err != nil {
//		utils.Error.Printf("ingestDps: Error in issuing request= %s ", err)
//		return latestTsIngested
//	}
	defer resp.Body.Close()

	switch resp.StatusCode {
		case 202:
			return newLatestTs
		case 400:
			utils.Error.Printf("Bad request error = %s ", err)
		case 401:
			utils.Error.Printf("Payload too large error = %s ", err)
		case 413:
			utils.Error.Printf("Unauthorized error = %s ", err)
		default:
			utils.Error.Printf("Unknown error = %s ", err)
	}
	utils.Error.Printf("Payload related to the error = %s ", payload)
	return latestTsIngested
*/
/*	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		utils.Error.Printf("sendCecRequest: Error in reading response= %s ", err)
		return latestTsIngested
	}
	return newLatestTs*/
}

func retrieveDigPayload(vinId string, from string) (string, string) {
	serialNo := getSerialNo(vinId)
	vinIndex := getVinIndex(vinId)
	tableName := "TV_" + strconv.Itoa(vinIndex)
	sqlString := "SELECT `path`, `value`, `timestamp` FROM " + tableName + " WHERE `timestamp` > ?"
	rows, err := dbHandle.Query(sqlString, from)
	defer rows.Close()
	if err != nil {
		utils.Error.Printf("retrieveDigPayload: Error in querying DB = %s ", err)
		return "", from
	}

	var path string
	var value string
	var timestamp string
	var mapIndex int
	payload := "["
	numOfDatapoints := 0
	for rows.Next() {
		err = rows.Scan(&path, &value, &timestamp)
		if err != nil {
			utils.Error.Printf("retrieveDigPayload: Error in reading DB result= %s ", err)
			return "", from
		}
		mapIndex = getMapIndex(path)
		if mapIndex == -1 {
			utils.Error.Printf("retrieveDigPayload: Unsupported cloud type= %s ", dataMapList.DataMap[mapIndex].CloudType)
			continue
		}
		switch dataMapList.DataMap[mapIndex].CloudType {
			case "GpsRecord":
				pLoad := getGpsRecord(serialNo, path, value, timestamp, ignitionOn, speed)
				if len(pLoad) > 0 {
					payload += pLoad
					numOfDatapoints++
				}
			case "GenericStatusRecord":
				payload += getGenericStatusRecord(serialNo, path, value, timestamp)
				numOfDatapoints++
			case "VinRecord":
				payload += getVinRecord(serialNo, path, value, timestamp)
				numOfDatapoints++
			default:
				utils.Error.Printf("retrieveDigPayload: Unsupported cloud type= %s ", dataMapList.DataMap[mapIndex].CloudType)
		}
		if path == IGNITION_ON_PATH {
			ignitionOn = value
		} else if path == "Vehicle.Speed" {
			speed = value
		}

	}
	if (numOfDatapoints == 0) {
 	    utils.Warning.Printf("retrieveDigPayload: Data not found.")
	    return "", from
	}
	utils.Info.Printf("retrieveDigPayload:Number of new data points to be written to cloud API = %d", numOfDatapoints)
	payload = payload[:len(payload)-1]
	payload += "]"
	return payload, timestamp
}

func getGpsRecord(serialNo string, path string, value string, timestamp string, ignitionOn string, speed string) string {
	var payload string
	if path == "Vehicle.CurrentLocation.Latitude" {
		if timestamp == long.Ts {
			payload += `{"DateTime":"` + timestamp + `","SerialNo":"`+ serialNo + `","Type":"GpsRecord", "IsGpsValid": true, "IsIgnitionOn":` + 
			 transformIgnitionOn(ignitionOn, false) + `, "Latitude":` + value + `, "Longitude":` + long.Value + `, "Speed":` + speed + "},"
			return payload
		} else {
			lat.Ts = timestamp
			lat.Value = value
			return ""
		}
	} else if path == "Vehicle.CurrentLocation.Longitude" {
		if timestamp == lat.Ts {
			payload += `{"DateTime":"` + timestamp + `","SerialNo":"`+ serialNo + `","Type":"GpsRecord", "IsGpsValid": true, "IsIgnitionOn":` + 
			 transformIgnitionOn(ignitionOn, false) + `, "Latitude":` + lat.Value + `, "Longitude":` + value + `, "Speed":` + speed + "},"
			return payload
		} else {
			long.Ts = timestamp
			long.Value = value
			return ""
		}
	}
	return ""
}

func getGenericStatusRecord(serialNo string, path string, value string, timestamp string) string {
	return `{"DateTime":"` + timestamp + `","SerialNo":"`+ serialNo + `","Type":"` + getCloudType(path) + `","Code":` + 
				    getGeotabCode(path) + `,"Value":` + transformGsrValue(path, value) + "},"
}

func getVinRecord(serialNo string, path string, value string, timestamp string) string {
	return `{"DateTime":"` + timestamp + `","SerialNo":"`+ serialNo + `","Type":"` + getCloudType(path) + `","VehicleIdentificationNumber":` + value + "},"
}

func transformGsrValue(path string, value string) string { // specific for GenericStatusRecord
	if path == IGNITION_ON_PATH {
		if value == "ON" {
			return transformIgnitionOn(value, true)
		}
	}
	if value == "true" {
		return "1"
	} else if value == "false"{
		return "0"
	}
	if strings.Contains(value, ".") {
		dotIndex := strings.Index(value, ".")
		return value[:dotIndex]
	}
	return value
}

func transformIgnitionOn(value string, isGsr bool) string { // allowed = ['UNDEFINED', 'LOCK', 'OFF', 'ACC', 'ON', 'START']
	trueVal := "true"
	falseVal := "false"
	if isGsr {
		trueVal = "1"
		falseVal = "0"
	} 
	if value == "ON" {
		return trueVal
	}
	return falseVal
}

func getVinIndex(vinId string) int {
	rows, err := dbHandle.Query("SELECT `vin_id` FROM VIN_TIV WHERE `vin`=?", vinId)
	if err != nil {
		return -1
	}
	defer rows.Close()
	var vinIndex int

	rows.Next()
	err = rows.Scan(&vinIndex)
	if err != nil {
		return -1
	}
	return vinIndex
}

func getMapIndex(path string) int {
	for i := range dataMapList.DataMap {
		if dataMapList.DataMap[i].VssPath == path {
			return i
		}
	}
	return -1
}

func getCloudType(path string) string {
	for i := range dataMapList.DataMap {
		if dataMapList.DataMap[i].VssPath == path {
			return dataMapList.DataMap[i].CloudType
		}
	}
	return ""
}

func getGeotabCode(path string) string {
	for i := range dataMapList.DataMap {
		if dataMapList.DataMap[i].VssPath == path {
			return dataMapList.DataMap[i].CloudValue
		}
	}
	return ""
}

func getSerialNo(vinId string) string {
	return "CX0B21338D8D"  // out of band provisioned by cloud service provider
}

func checkToken(tokenChan chan string) {
	for true {
		select {
		  case digData.Token = <- tokenChan: //update, if new token is available
		  	return
		  default:
		  	if len(digData.Token) > 0 {
		  		return  // if no new token, but token is initialised, no update is needed
		  	}
		}
	  	time.Sleep(5 * time.Second)
	}
}

func getOvdsVins() []string {
	rows, err := dbHandle.Query("SELECT `vin` FROM VIN_TIV")
	defer rows.Close()
	if err != nil {
		utils.Error.Printf("getOvdsVins: Error in querying DB = %s ", err)
		return nil
	}
	var vinId string
	vinList := make([]string, MAX_VINS)
	numOfDatapoints := 0
	for rows.Next() {
		err = rows.Scan(&vinId)
		if err != nil {
			utils.Error.Printf("getOvdsVins: Error in reading DB result= %s ", err)
			return nil
		}
		vinList[numOfDatapoints] = vinId
		numOfDatapoints++
	}
	return vinList[:numOfDatapoints]
}

func initDataMap(fname string) int {
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		utils.Error.Printf("Error reading file=%s", fname)
		return 0
	}
	return jsonToStructList(string(data))
}

func jsonToStructList(jsonList string) int {
    var mapList map[string]interface{}
    err := json.Unmarshal([]byte(jsonList), &mapList)
    if err != nil {
	utils.Error.Printf("jsonToStructList:error jsonList=%s", jsonList)
	return 0
    }
    switch vv := mapList["DataMap"].(type) {
      case []interface{}:
//        utils.Info.Println(jsonList, "is an array:, len=",strconv.Itoa(len(vv)))
        dataMapList.DataMap = make([]DataMapItem, len(vv))
        for i := 0 ; i < len(vv) ; i++ {
  	    dataMapList.DataMap[i] = retrieveMap(vv[i].(map[string]interface{}))
  	}
      case map[string]interface{}:
//        utils.Info.Println(jsonList, "is a map:")
        dataMapList.DataMap = make([]DataMapItem, 1)
  	dataMapList.DataMap[0] = retrieveMap(vv)
      default:
        utils.Info.Println(vv, "is of an unknown type")
    }
    return len(dataMapList.DataMap)
}

func retrieveMap(mapItem map[string]interface{}) DataMapItem {
    var dataMapItem DataMapItem
    for k, v := range mapItem {
		switch vv := v.(type) {
		case string:
//			utils.Info.Println(k, "is string", vv)
			if k == "VssPath" {
				dataMapItem.VssPath = vv
			} else if k == "CloudType" {
				dataMapItem.CloudType = vv
			} else if k == "CloudKey" {
				dataMapItem.CloudKey = vv
			} else {
				dataMapItem.CloudValue = vv
			}
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
    }
    return dataMapItem
}

func main() {
	// Create new parser object
	parser := argparse.NewParser("print", "Cloud Edge Server Client")
	// Create string flag
	mapFileName := parser.String("d", "datamap", &argparse.Options{Required: false, Help: "Path to file containing VSS-to-Cloud data mapping", Default:  "VssGeotabMap.json"})
	logFile := parser.Flag("", "logfile", &argparse.Options{Required: false, Help: "outputs to logfile in ./logs folder"})
	logLevel := parser.Selector("", "loglevel", []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}, &argparse.Options{
		Required: false,
		Help:     "changes log output level",
		Default:  "info"})

	// Parse input
	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}

	utils.InitLog("CEC-client.txt", "./logs", *logFile, *logLevel)

	readTransportSecConfig()
	utils.Info.Printf("InitClientServer():secConfig.TransportSec=%s", secConfig.TransportSec)

	muxServer = http.NewServeMux()

	InitDb(ovdsDbFileName)
	if initDataMap(*mapFileName) == 0 {
		utils.Error.Printf("Failed in creating list from %s", *mapFileName)
		os.Exit(1)
	}

	vinIdChan := make(chan string)
	go InitCecServer(vinIdChan)  // handles CEC-to-VEC comm, writes to OVDS

	tokenChan := make(chan string)
	go initTokenMgr(tokenChan)
	initCeClient()
	accumulatedTimeTicker := time.NewTicker(MAX_ACCUMULATED_TIME * time.Second)

	utils.Info.Println("**** Cloud Edge Server/Client started... ****")
	for {
		select {
		case vinId := <- vinIdChan:
			writeDpsToCloud(tokenChan, vinId)			
		case <-accumulatedTimeTicker.C:
			vinList := getOvdsVins()
			for i := range vinList {
				if getAccumulatedDataDpCount(vinList[i]) > 0 {
					writeDpsToCloud(tokenChan, vinList[i])
				}
			}
			accumulatedTimeTicker = time.NewTicker(MAX_ACCUMULATED_TIME * time.Second)
		}
	}
}

