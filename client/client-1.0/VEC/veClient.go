/**
* (C) 2022 Geotab
*
* All files and artifacts in the repository at https://github.com/w3c/automotive-viss2
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package main

import (
//	"fmt"
	"encoding/json"
	"io/ioutil"
	"os"

	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/url"

	"flag"
	"strconv"
	"strings"
	"bytes"
	"time"

	"github.com/akamensky/argparse"
	"github.com/gorilla/websocket"
	"github.com/w3c/automotive-viss2/utils"
)

var muxServer *http.ServeMux
var clientCert tls.Certificate
var caCertPool x509.CertPool
var cecUrl string
var vissv2Url string

const vecRequestListFileName = "vecRequests.json"
type RequestList struct {
	Request []string
}
var requestList RequestList

var vinId string

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

func readRequestList(fname string) int {
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		utils.Error.Printf("Error reading file=%s", fname)
		return 0
	}
	return jsonToStructList(string(data))
}

func jsonToStructList(jsonList string) int {
    var reqList map[string]interface{}
    err := json.Unmarshal([]byte(jsonList), &reqList)
    if err != nil {
	utils.Error.Printf("jsonToStructList:error jsonList=%s", jsonList)
	return 0
    }
    switch vv := reqList["request"].(type) {
      case []interface{}:
//        utils.Info.Println(jsonList, "is an array:, len=",strconv.Itoa(len(vv)))
        requestList.Request = make([]string, len(vv))
        for i := 0 ; i < len(vv) ; i++ {
  	    requestList.Request[i] = retrieveRequest(vv[i].(map[string]interface{}))
  	}
      case map[string]interface{}:
//        utils.Info.Println(jsonList, "is a map:")
        requestList.Request = make([]string, 1)
  	requestList.Request[0] = retrieveRequest(vv)
      default:
//        utils.Info.Println(vv, "is of an unknown type")
    }
    return len(requestList.Request)
}

func retrieveRequest(jsonRequest map[string]interface{}) string {
    request, err := json.Marshal(jsonRequest)
    if err != nil {
	utils.Error.Print("retrieveRequest(): JSON array encode failed. ", err)
	return ""
    }
    return string(request)
}

func sendToCesc(response string) {
	var responseMap = make(map[string]interface{})
	var requestMap = make(map[string]interface{})
	utils.MapRequest(response, &responseMap)
	requestMap["vin"] = vinId
	requestMap["data"] = responseMap["data"]
	request, err := json.Marshal(requestMap)
	if (err != nil) {
		utils.Error.Printf("Error marshalling request map", err)
		return
	}
	sendCescRequest(request)
}

func sendCescRequest(payload []byte) {
	secPort := "8000"
	scheme := "http"
	if secConfig.TransportSec == "yes" {
		scheme = "https"
		secPortNum, _ := strconv.Atoi(secConfig.HttpSecPort)
		secPort = strconv.Itoa(secPortNum)
	}
	url := scheme + "://" + cecUrl + ":" + secPort // + "server path"??

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		utils.Error.Printf("sendCecRequest: Error creating request=%s.", err)
		return
	}

	// Set headers
	req.Header.Set("Access-Control-Allow-Origin", "*")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", cecUrl+":"+secPort)

	// Configure client
	var client *http.Client
	if secConfig.TransportSec == "yes" {
		t := &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{clientCert},
				RootCAs:      &caCertPool,
			},
		}

		client = &http.Client{Transport: t, Timeout: 10 * time.Second}
	} else {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		utils.Error.Printf("sendCecRequest: Error in issuing request= %s ", err)
		return
	}
	defer resp.Body.Close()

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		utils.Error.Printf("sendCecRequest: Error in reading response= %s ", err)
		return
	}
}

func getVinId(conn *websocket.Conn) { // response is received in receiveNotifications()
	request := `{"action":"get", "path":"Vehicle.VehicleIdentification.VIN", "requestId": "666"}`

	err := conn.WriteMessage(websocket.TextMessage, []byte(request))
	if err != nil {
		utils.Error.Printf("Subscribe request error:%s", err)
	}
}

func initDataTransfer(dataChannel chan string, requestList []string) {
	conn := initVissV2WebSocket()
	go receiveNotifications(conn, dataChannel)
	getVinId(conn)
	subscribeToPaths(conn, requestList)
}

func initVissV2WebSocket() *websocket.Conn {
	scheme := "ws"
	portNum := "8080"
	if secConfig.TransportSec == "yes" {
		scheme = "wss"
		portNum = secConfig.WsSecPort
		websocket.DefaultDialer.TLSClientConfig = &tls.Config{
			Certificates: []tls.Certificate{clientCert},
			RootCAs:      &caCertPool,
		}
	}
	var addr = flag.String("addr", vissv2Url+":"+portNum, "http service address")
	dataSessionUrl := url.URL{Scheme: scheme, Host: *addr, Path: ""}
	connectionEstablished := false
	var conn *websocket.Conn
	var err error
	for connectionEstablished == false {
		conn, _, err = websocket.DefaultDialer.Dial(dataSessionUrl.String(), nil)
		if err != nil {
			utils.Warning.Printf("Data session dial error:%s", err)
			time.Sleep(5 * time.Second)
			continue
		}
		connectionEstablished = true
	}
	return conn
}

func receiveNotifications(conn *websocket.Conn, dataChannel chan string) {
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			utils.Error.Printf("Subscription response error: %s", err)
			return
		}
		message := string(msg)
		if strings.Contains(message, "\"subscribe\"") {
//			utils.Info.Printf("Subscription response:%s", message)
		} else if strings.Contains(message, "\"get\"") {
			vinId = extractValue(message)
			utils.Info.Printf("VIN Id=%s", vinId)
		} else {
			dataChannel <- message
		}
	}
}

func extractValue(message string) string { // {....,"value":"xxxx","ts":....}
	startIndex := strings.Index(message, "value\":")
	if (startIndex == -1) {
		utils.Error.Printf("VinId could not be found in message: %s", message)
		return ""
	}
	stopIndex := strings.Index(message[startIndex+8:], "\",")
	return message[startIndex+8:startIndex+8+stopIndex]
}

func subscribeToPaths(conn *websocket.Conn, requestList []string) {
	for i := range requestList {
		subscribeToPath(conn, requestList[i])
		time.Sleep(13 * time.Millisecond)
	}
}

func subscribeToPath(conn *websocket.Conn, request string) {
	err := conn.WriteMessage(websocket.TextMessage, []byte(request))
	if err != nil {
		utils.Error.Printf("Subscribe request error:%s", err)
	}
}

func main() {
	// Create new parser object
	parser := argparse.NewParser("print", "VEC client")
	// Create string flag
	logFile := parser.Flag("", "logfile", &argparse.Options{Required: false, Help: "outputs to logfile in ./logs folder"})
	logLevel := parser.Selector("", "loglevel", []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}, &argparse.Options{
		Required: false,
		Help:     "changes log output level",
		Default:  "info"})
	url_cec := parser.String("u", "cecUrl", &argparse.Options{Required: true, Help: "IP/URL to CEC cloud end point (REQUIRED)"})
	url_viss := parser.String("w", "vissv2Url", &argparse.Options{Required: true, Help: "IP/URL to W3C VISS v2 server (REQUIRED)"})

	// Parse input
	err := parser.Parse(os.Args)
	if err != nil {
		utils.Error.Print(parser.Usage(err))
		os.Exit(1)
	}

	cecUrl = *url_cec
	vissv2Url = *url_viss

	utils.InitLog("VEC-client.txt", "./logs", *logFile, *logLevel)

	dataChannel := make(chan string)
	readTransportSecConfig()
	utils.Info.Printf("InitClientServer():secConfig.TransportSec=%s", secConfig.TransportSec)
	if secConfig.TransportSec == "yes" {
		caCertPool = *prepareTransportSecConfig()
	}

	muxServer = http.NewServeMux()

	if readRequestList(vecRequestListFileName) == 0 {
		utils.Error.Printf("Failed in creating list from %s", vecRequestListFileName)
		os.Exit(1)
	}

	initDataTransfer(dataChannel, requestList.Request)

	utils.Info.Println("**** Vehicle Edge Client started... ****")
	for {
		select {
		case notification := <- dataChannel:
//utils.Info.Println("Received on data channel:%s", notification)
			sendToCesc(notification)
		}
	}
}
