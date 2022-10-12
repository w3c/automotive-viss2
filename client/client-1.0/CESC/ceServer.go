/**
* (C) 2022 Geotab
*
* All files and artifacts in the repository at https://github.com/w3c/automotive-viss2
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package main

import (
	"io/ioutil"
	"os"

	"crypto/tls"
	"crypto/x509"
	"net/http"

	"strconv"
//	"strings"
//	"time"

	"database/sql"
	_ "github.com/mattn/go-sqlite3"
//	"github.com/akamensky/argparse"
	"github.com/w3c/automotive-viss2/utils"
)

const MAX_ACCUMULATED_DPS = 25  // 500?
const MAX_VINS = 100   // max no of different VINs on list / in OVDS
type AccumulatedData struct { 
	VinId string
	LatestIngestedTs string
	DpCount int
}
var accumulatedData []AccumulatedData

func InitCecServer(vinIdChan chan string) {
	accumulatedData = make([]AccumulatedData, MAX_VINS)
	initAccumulatedData()
	cecHandler := makeCecHandler(vinIdChan)
	muxServer.HandleFunc("/", cecHandler)
	utils.Info.Printf("CecServer:IP address=%s", utils.GetModelIP(3))
	utils.Info.Printf("initCecServer():secConfig.TransportSec=%s", secConfig.TransportSec)
	if secConfig.TransportSec == "yes" {
		secPortNum, _ := strconv.Atoi(secConfig.HttpSecPort)
		server := http.Server{
			Addr: ":" + strconv.Itoa(secPortNum),
			TLSConfig: getTLSConfig("localhost", trSecConfigPath+secConfig.CaSecPath+"Root.CA.crt",
				tls.ClientAuthType(certOptToInt(secConfig.ServerCertOpt))),
			Handler: muxServer,
		}
		utils.Info.Printf("HTTPS:CerOpt=%s", secConfig.ServerCertOpt)
		utils.Error.Fatal(server.ListenAndServeTLS(trSecConfigPath+secConfig.ServerSecPath+"server.crt", trSecConfigPath+secConfig.ServerSecPath+"server.key"))
	} else {
		utils.Error.Fatal(http.ListenAndServe(":8000", muxServer))
	}
}

func getTLSConfig(host string, caCertFile string, certOpt tls.ClientAuthType) *tls.Config {
	var caCert []byte
	var err error
	var caCertPool *x509.CertPool
	if certOpt > tls.RequestClientCert {
		caCert, err = ioutil.ReadFile(caCertFile)
		if err != nil {
			utils.Error.Printf("Error opening cert file", caCertFile, ", error ", err)
			return nil
		}
		caCertPool = x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
	}

	return &tls.Config{
		ServerName: host,
		ClientAuth: certOpt,
		ClientCAs:  caCertPool,
		MinVersion: tls.VersionTLS12, // TLS versions below 1.2 are considered insecure - see https://www.rfc-editor.org/rfc/rfc7525.txt for details
	}
}

func certOptToInt(serverCertOpt string) int {
	if serverCertOpt == "NoClientCert" {
		return 0
	}
	if serverCertOpt == "ClientCertNoVerification" {
		return 2
	}
	if serverCertOpt == "ClientCertVerification" {
		return 4
	}
	return 4 // if unclear, apply max security
}

func makeCecHandler(vinIdChan chan string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Upgrade") == "websocket" {
			http.Error(w, "400 Incorrect protocol", http.StatusBadRequest)
			utils.Warning.Printf("Client call to incorrect protocol.")
			return
		}
		cecRequestHandler(w, req, vinIdChan)
	}
}

func cecRequestHandler(w http.ResponseWriter, req *http.Request, vinIdChan chan string) {
	body, _ := ioutil.ReadAll(req.Body)
	writeToDb(string(body))
	for i := range accumulatedData {
		if accumulatedData[i].DpCount > MAX_ACCUMULATED_DPS {
			utils.Info.Printf("Max no of data points written is reached")
			vinIdChan <- accumulatedData[i].VinId
		}
	}
}

func writeToDb(dataResponse string) { // {"vin":"xxx","data": <see the four possible formats for "data" in the VISSv2 spec>}
    var responseMap = make(map[string]interface{})
    utils.MapRequest(dataResponse, &responseMap)
    processDataLevel1(responseMap["vin"].(string), responseMap["data"])
}

func processDataLevel1(vinId string, dataObject interface{}) { // data or []data level
	switch vv := dataObject.(type) {
	case []interface{}: // []data
//		utils.Info.Println(dataObject, "is an array:, len=", strconv.Itoa(len(vv)))
		processDataLevel2(vinId, vv)
	case map[string]interface{}:
//		utils.Info.Println(dataObject, "is a map:")
		processDataLevel3(vinId, vv)
	default:
		utils.Info.Println(dataObject, "is of an unknown type")
	}
	return
}

func processDataLevel2(vinId string, dataArray []interface{}) { // []data level
	for k, v := range dataArray {
		switch vv := v.(type) {
		case map[string]interface{}:
//			utils.Info.Println(k, "is a map:")
			processDataLevel3(vinId, vv)
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
	}
	return
}

func processDataLevel3(vinId string, data map[string]interface{}) { // inside data, dp or []dp level
	path := ""
	callLevel4 := false
	callLevel5 := false
	var dpArray[]interface{}
	var dp map[string]interface{}
	for k, v := range data {
		switch vv := v.(type) {
		case []interface{}: // []dp
//			utils.Info.Println(vv, "is an array:, len=", strconv.Itoa(len(vv)))
			dpArray = vv
			callLevel4 = true
		case map[string]interface{}:
//			utils.Info.Println(k, "is a map:")
			dp = vv
			callLevel5 = true
		case string: // path
//			utils.Info.Println(k, "is string", vv)
			path = vv
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
	}
	if (callLevel4 == true) {
		processDataLevel4(vinId, dpArray, path)
	} else if (callLevel5 == true) {
		processDataLevel5(vinId, dp, path)
	}
	return
}


func processDataLevel4(vinId string, dpArray []interface{}, path string) { // []dp level
	for k, v := range dpArray {
		switch vv := v.(type) {
		case map[string]interface{}:
//			utils.Info.Println(k, "is a map:")
			processDataLevel5(vinId, vv, path)
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
	}
	return
}

func processDataLevel5(vinId string, dp map[string]interface{}, path string) { // inside dp level
	var value, ts string
	for k, v := range dp {
		switch vv := v.(type) {
		case string:
//			utils.Info.Println(k, "is string", vv)
			if k == "value" {
				value = vv
			} else if k == "ts" {
				ts = vv
			}
		default:
			utils.Info.Println(k, "is of an unknown type")
			return
		}
	}
	if incAccumulatedDataDpCount(vinId) {
		writeDpToDB(vinId, path, value, ts)
	}
}

func incAccumulatedDataDpCount(vinId string) bool {  // if on list, inc DpCount for that entry, else create a new entry and inc
	for i := range accumulatedData {
		if accumulatedData[i].VinId == vinId {
			accumulatedData[i].DpCount++
			return true
		}
	}
	for j := range accumulatedData {
		if len(accumulatedData[j].VinId) == 0 {
			accumulatedData[j].VinId = vinId
			accumulatedData[j].DpCount++
			return true
		}
	}
	utils.Error.Printf("accumulatedData list is full, vin id=%s missed", vinId)
	return false
}

func initAccumulatedData() {
	for i := range accumulatedData {
		accumulatedData[i].VinId = ""
		accumulatedData[i].DpCount = 0
		accumulatedData[i].LatestIngestedTs = "1957-04-15T13:37:00Z" // start with a very old ts"
	}
}

func updateAccumulatedData(vinId string, latestTsIngested string) {
	for i := range accumulatedData {
		if accumulatedData[i].VinId == vinId {
			accumulatedData[i].LatestIngestedTs = latestTsIngested
			accumulatedData[i].DpCount = 0
			return
		}
	}
	for j := range accumulatedData {
		if len(accumulatedData[j].VinId) == 0 {
			accumulatedData[j].VinId = vinId
			accumulatedData[j].DpCount = 0
			accumulatedData[j].LatestIngestedTs = latestTsIngested
			return
		}
	}
	utils.Error.Printf("accumulatedData list is full, vin id=%s missed", vinId)
}

func getAccumulatedDataTs(vinId string) string {
	for i := range accumulatedData {
		if accumulatedData[i].VinId == vinId {
			return accumulatedData[i].LatestIngestedTs
		}
	}
	utils.Error.Printf("getAccumulatedDataTs: vin id=%s not found", vinId)
	return ""
}

func getAccumulatedDataDpCount(vinId string) int {
	for i := range accumulatedData {
		if accumulatedData[i].VinId == vinId {
			return accumulatedData[i].DpCount
		}
	}
	utils.Error.Printf("getAccumulatedDataDpCount: vin id=%s not found", vinId)
	return 0
}

func writeDpToDB(vinId string, path string, value string, ts string) {
utils.Info.Printf("Entering writeDpToDB")
	vinIndex := readVinIndex(vinId)
	if vinIndex == -1 {
		err := writeVIN(vinId)
		if err != 0 {
			utils.Error.Printf("writeDpToDB: vin id=%s failed to be added to DB", vinId)
			return
		}
		vinIndex = readVinIndex(vinId)
		if vinIndex == -1 {
			utils.Error.Printf("writeDpToDB: vin id=%s could not be found in DB", vinId)
			return
		}
		createTvVin(vinIndex)
	}
	tableName := "TV_" + strconv.Itoa(vinIndex)
	sqlString := "INSERT INTO " + tableName + "(value, timestamp, path) values(?, ?, ?)"
	stmt, err := dbHandle.Prepare(sqlString)
	if err != nil {
		utils.Error.Printf("OVDS DB prepare() failed, err = %s", err)
		return
	}

	_, err = stmt.Exec(value, ts, path)
	if err != nil {
		utils.Error.Printf("OVDS DB exec() failed, err = %s", err)
		return
	}
}

func readVinIndex(vin string) int {
	rows, err := dbHandle.Query("SELECT `vin_id` FROM VIN_TIV WHERE `vin`=?", vin)
	if err != nil {
		utils.Error.Printf("OVDS DB query() failed, err = %s", err)
		return -1
	}
	defer rows.Close()

	var vinId int
	if rows.Next() {
		err = rows.Scan(&vinId)
		if err != nil {
			utils.Error.Printf("OVDS DB scan() failed, err = %s", err)
			return -1
		}
		return vinId
	}
	return -1
}

func writeVIN(vin string) int {
	stmt, err := dbHandle.Prepare("INSERT INTO VIN_TIV(vin) values(?)")
	if err != nil {
		utils.Error.Printf("OVDS DB prepare() failed, err = %s", err)
		return -1
	}

	_, err = stmt.Exec(vin)
	if err != nil {
		utils.Error.Printf("OVDS DB exec() failed, err = %s", err)
		return -1
	}
	return 0
}

func createTvVin(vinId int) {
	tableName := "TV_" + strconv.Itoa(vinId)
	sqlString := "CREATE TABLE " + tableName + " (`value` TEXT NOT NULL, `timestamp` TEXT NOT NULL, `path` TEXT, UNIQUE(`path`, `timestamp`) ON CONFLICT IGNORE)"
	stmt, err := dbHandle.Prepare(sqlString)
	if err != nil {
		utils.Error.Printf("OVDS DB prepare() for VIN index = %d failed, err = %s", vinId, err)
		return
	}
	_, err = stmt.Exec()
	if err != nil {
		utils.Error.Printf("OVDS DB exec() failed, err = %s", err)
		return
	}
}

func createStaticTables() int {
	stmt1, err := dbHandle.Prepare(`CREATE TABLE "VIN_TIV" ( "vin_id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, "vin" TEXT NOT NULL )`)
	if err != nil {
		utils.Error.Printf("OVDS DB prepare() failed, err = %s", err)
		return -1
	}

	_, err = stmt1.Exec()
	if err != nil {
		utils.Error.Printf("OVDS DB prepare() failed, err = %s", err)
		return -1
	}

        stmt2, err2 := dbHandle.Prepare(`CREATE TABLE "TIV" ( "vin_id" INTEGER NOT NULL, "path" TEXT NOT NULL, "value" TEXT, UNIQUE("vin_id", "path") ON CONFLICT IGNORE, FOREIGN KEY("vin_id") REFERENCES "VIN_TIV"("vin_id") )`)
	if err2 != nil {
		utils.Error.Printf("OVDS DB prepare() failed, err = %s", err)
		return -1
	}

	_, err2 = stmt2.Exec()
	if err2 != nil {
		utils.Error.Printf("OVDS DB prepare() failed, err = %s", err)
		return -1
	}

	if err != nil || err2 != nil {
		return -1
	}
	return 0
}

func InitDb(dbFile string) {
        doCreate := true
	if utils.FileExists(dbFile) {
	    doCreate = false
	}
	dbHandle, dbErr = sql.Open("sqlite3", dbFile)
	if (doCreate) {
		err := createStaticTables()
		if err != 0 {
			utils.Error.Printf("\novdsServer: Unable to make static tables : %s\n", err)
			os.Exit(1)
		}
	}
}

