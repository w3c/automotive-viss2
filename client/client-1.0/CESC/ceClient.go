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
//	"encoding/json"
	"io/ioutil"
	"os"

	"crypto/tls"
	"crypto/x509"
	"net/http"

//	"strconv"
//	"strings"
	"time"

	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/akamensky/argparse"
	"github.com/w3c/automotive-viss2/utils"
)

var muxServer *http.ServeMux
var clientCert tls.Certificate
var caCertPool x509.CertPool

var dbHandle *sql.DB
var dbErr error
var vinId string

const MAX_ACCUMULATED_DPS = 25  // 500?
const MAX_ACCUMULATED_TIME = 5000  // 10000?, in msec
const ovdsDbFileName = "ovdsCESC.db"

var latestTsIngested string = "1957-04-15T13:37:00Z" // start with a medieval ts"

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

func initCloudApi() {
}

func writeDpsToCloud(accumulatedDps int) {
	utils.Info.Printf("Number of new data points written to cloud API = %d", accumulatedDps)
}

func main() {
	// Create new parser object
	parser := argparse.NewParser("print", "Cloud Edge Server Client")
	// Create string flag
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

	dpCountChan := make(chan int)
	go InitCecServer(dpCountChan)  // handles CEC-to-VEC comm, writes to OVDS

	initCloudApi()  //get cloud API credentials, ...
	accumulatedDps := 0
	accumulatedTimeTicker := time.NewTicker(MAX_ACCUMULATED_TIME * time.Millisecond)

	utils.Info.Println("**** Cloud Edge Client started... ****")
	for {
		select {
		case numOfDps := <- dpCountChan:
			accumulatedDps += numOfDps
			utils.Info.Printf("Number of new data points written to OVDS = %d, accumulated = %d", numOfDps, accumulatedDps)
			if (accumulatedDps >= MAX_ACCUMULATED_DPS) {
				writeDpsToCloud(accumulatedDps)
				accumulatedDps = 0
				accumulatedTimeTicker = time.NewTicker(MAX_ACCUMULATED_TIME * time.Millisecond)
			}
		case <-accumulatedTimeTicker.C:
			writeDpsToCloud(accumulatedDps)
			accumulatedDps = 0
		}
	}
}
