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
//	"crypto/x509"
	"net/http"

	"strconv"
//	"strings"
//	"time"

	"database/sql"
	_ "github.com/mattn/go-sqlite3"
//	"github.com/akamensky/argparse"
	"github.com/w3c/automotive-viss2/utils"
)

func InitCecServer(dpCountChan chan int) {
	cecHandler := makeCecHandler(dpCountChan)
	muxServer.HandleFunc("/", cecHandler)
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

func makeCecHandler(dpCountChan chan int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Upgrade") == "websocket" {
			http.Error(w, "400 Incorrect protocol", http.StatusBadRequest)
			utils.Warning.Printf("Client call to incorrect protocol.")
			return
		}
		cecRequestHandler(w, req, dpCountChan)
	}
}

func cecRequestHandler(w http.ResponseWriter, req *http.Request, dpCountChan chan int) {
	body, _ := ioutil.ReadAll(req.Body)
	numOfDpsWritten := writeToDb(string(body))
//	utils.Info.Printf("Data points written to DB=%d", numOfDpsWritten)
	dpCountChan <- numOfDpsWritten
}

func writeToDb(dataResponse string) int { // {"vin":"xxx","data": <see the four possible formats for "data" in the VISSv2 spec>}
    var responseMap = make(map[string]interface{})
    utils.MapRequest(dataResponse, &responseMap)
    return processDataLevel1(responseMap["vin"].(string), responseMap["data"])
}

func processDataLevel1(vinId string, dataObject interface{}) int { // data or []data level
	numOfDpsWritten := 0
	switch vv := dataObject.(type) {
	case []interface{}: // []data
//		utils.Info.Println(dataObject, "is an array:, len=", strconv.Itoa(len(vv)))
		numOfDpsWritten = processDataLevel2(vinId, vv)
	case map[string]interface{}:
//		utils.Info.Println(dataObject, "is a map:")
		numOfDpsWritten = processDataLevel3(vinId, vv)
	default:
		utils.Info.Println(dataObject, "is of an unknown type")
	}
	return numOfDpsWritten
}

func processDataLevel2(vinId string, dataArray []interface{}) int { // []data level
	numOfDpsWritten := 0
	for k, v := range dataArray {
		switch vv := v.(type) {
		case map[string]interface{}:
//			utils.Info.Println(k, "is a map:")
			numOfDpsWritten = processDataLevel3(vinId, vv)
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
	}
	return numOfDpsWritten
}

func processDataLevel3(vinId string, data map[string]interface{}) int { // inside data, dp or []dp level
	numOfDpsWritten := 0
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
		numOfDpsWritten = processDataLevel4(vinId, dpArray, path)
	} else if (callLevel5 == true) {
		numOfDpsWritten = processDataLevel5(vinId, dp, path)
	}
	return numOfDpsWritten
}


func processDataLevel4(vinId string, dpArray []interface{}, path string) int { // []dp level
	var numOfDpsWritten int
	for k, v := range dpArray {
		switch vv := v.(type) {
		case map[string]interface{}:
//			utils.Info.Println(k, "is a map:")
			numOfDpsWritten = processDataLevel5(vinId, vv, path)
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
	}
	return numOfDpsWritten
}

func processDataLevel5(vinId string, dp map[string]interface{}, path string) int { // inside dp level
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
			return 0
		}
	}
	writeDpToDB(vinId, path, value, ts)
	return 1
}

func writeDpToDB(vinId string, path string, value string, ts string) {
	vinIndex := readVinIndex(vinId)
	if vinIndex == -1 {
		err := writeVIN(vinId)
		if err != 0 {
			return
		}
		vinIndex = readVinIndex(vinId)
		if vinIndex == -1 {
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

	rows.Next()
	err = rows.Scan(&vinId)
	if err != nil {
		utils.Error.Printf("OVDS DB scan() failed, err = %s", err)
		return -1
	}
	return vinId
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
		utils.Error.Printf("OVDS DB prepare() failed, err = %s", err)
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

