/**
* (C) 2022 Geotab Inc
*
* All files and artifacts in the repository at https://github.com/MEAE-GOT/WAII
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"fmt"
	"time"

	"github.com/akamensky/argparse"
	"github.com/gorilla/websocket"
	"github.com/MEAE-GOT/WAII/utils"
)

var commandNumber string

var clientCert tls.Certificate
var caCertPool x509.CertPool

var vissv2Url string
var protocol string
var compression string

type RequestList struct {
	Request []string
}

var requestList RequestList

func pathToUrl(path string) string {
	var url string = strings.Replace(path, ".", "/", -1)
	return "/" + url
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
        utils.Info.Println(jsonList, "is an array:, len=",strconv.Itoa(len(vv)))
        requestList.Request = make([]string, len(vv))
        for i := 0 ; i < len(vv) ; i++ {
  	    requestList.Request[i] = retrieveRequest(vv[i].(map[string]interface{}))
  	}
      case map[string]interface{}:
        utils.Info.Println(jsonList, "is a map:")
        requestList.Request = make([]string, 1)
  	requestList.Request[0] = retrieveRequest(vv)
      default:
        utils.Info.Println(vv, "is of an unknown type")
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

func createListFromFile(fname string) int {
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		fmt.Printf("Error reading file=%s", fname)
		return 0
	}
	return jsonToStructList(string(data))
}

func saveListAsFile(fname string) {
	buf, err := json.Marshal(requestList)
	if err != nil {
		fmt.Printf("Error marshalling from file %s: %s\n", fname, err)
		return
	}

	err = ioutil.WriteFile(fname, buf, 0644)
	if err != nil {
		fmt.Printf("Error writing file %s: %s\n", fname, err)
		return
	}
}

func getGen2Response(path string) string {
	secPort := "8888"
	scheme := "http"
	if secConfig.TransportSec == "yes" {
		scheme = "https"
		secPortNum, _ := strconv.Atoi(secConfig.SecPort)
		secPort = strconv.Itoa(secPortNum + 1) // to diff from WSS portno
	}
	url := scheme + "://" + vissv2Url + ":" + secPort + pathToUrl(path)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("getGen2Response: Error creating request=%s.", err)
		return ""
	}

	// Set headers
	req.Header.Set("Access-Control-Allow-Origin", "*")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", vissv2Url+":"+secPort)

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
		fmt.Printf("getGen2Response: Error in issuing request/response= %s ", err)
		return ""
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("getGen2Response: Error in reading response= %s ", err)
		return ""
	}

	return string(body)
}

func displayMessage(message string) {
    fmt.Printf("displayMessage: message = %s\n", message)
}

func iterateGetAndWrite(elements int, sleepTime int) {
	for i := 0; i < elements; i++ {
		response := getGen2Response(requestList.Request[i])
		if len(response) == 0 {
			fmt.Printf("iterateGetAndWrite: Cannot connect to server.\n")
			os.Exit(-1)
		}
		displayMessage(response)
		time.Sleep((time.Duration)(sleepTime) * time.Millisecond)
	}
	fmt.Printf("\n\n****************** Iteration cycle over all paths completed ************************************\n\n")
}

func initVissV2WebSocket(compression string) *websocket.Conn {
	scheme := "ws"
	portNum := "8080"
	if secConfig.TransportSec == "yes" {
		scheme = "wss"
		portNum = secConfig.SecPort
		websocket.DefaultDialer.TLSClientConfig = &tls.Config{
			Certificates: []tls.Certificate{clientCert},
			RootCAs:      &caCertPool,
		}
	}
	var addr = flag.String("addr", vissv2Url+":"+portNum, "http service address")
	dataSessionUrl := url.URL{Scheme: scheme, Host: *addr, Path: ""}
//	h := http.Header{}
//	h.Set("Sec-Websocket-Protocol", "VISSv2" + compression)
//	conn, _, err := websocket.DefaultDialer.Dial(dataSessionUrl.String(), nil)
	subProtocol := make([]string, 1)
	subProtocol[0] = "VISSv2" + compression
	dialer := websocket.Dialer{
		HandshakeTimeout: time.Second,
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		Subprotocols:     subProtocol,
	}
	conn, _, err := dialer.Dial(dataSessionUrl.String(), nil)
	if err != nil {
		fmt.Printf("Data session dial error:%s\n", err)
		os.Exit(-1)
	}
	return conn
}

func iterateNotificationAndWrite(conn *websocket.Conn) {
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("Subscription response error: %s\n", err)
			return
		}
		message := string(msg)
		if strings.Contains(message, "subscribe") {
			fmt.Printf("Subscription response:%s\n", message)
		} else {
			//	    var msgMap = make(map[string]interface{})
			//	    jsonToMap(message, &msgMap)
			//	    data, _ := json.Marshal(msgMap["data"])
			//	    displayMessage(`{"data":` + string(data) + "}")
			displayMessage(message)
		}
	}
}

func extractMessage(message string) (string, string, string) { // message is expected to contain the key-value: “data”:{“path”:”B”, “dp”:{“value”:”C”, “ts”:”D”}}
	var msgMap = make(map[string]interface{})
	jsonToMap(message, &msgMap)
	if msgMap["data"] == nil {
		fmt.Printf("Error: Message does not contain vehicle data.\n")
		return "", "", ""
	}
	data, _ := json.Marshal(msgMap["data"])

	jsonToMap(string(data), &msgMap)
	path := msgMap["path"].(string)
	dp, _ := json.Marshal(msgMap["dp"])

	jsonToMap(string(dp), &msgMap)
	value := msgMap["value"].(string)
	ts := msgMap["ts"].(string)
	fmt.Printf("path=%s, value=%s, ts=%s\n", path, value, ts)
	return path, value, ts
}

func jsonToMap(request string, rMap *map[string]interface{}) {
	decoder := json.NewDecoder(strings.NewReader(request))
	err := decoder.Decode(rMap)
	if err != nil {
		fmt.Printf("jsonToMap: JSON decode failed for request:%s, err=%s\n", request, err)
	}
}

func subscribeToPaths(conn *websocket.Conn, elements int, sleepTime int) {
	for i := 0; i < elements; i++ {
		subscribeToPath(conn, requestList.Request[i])
		time.Sleep((time.Duration)(sleepTime) * time.Millisecond)
	}
}

func subscribeToPath(conn *websocket.Conn, path string) {
	request := `{"action":"subscribe", "path":"` + path + `", "filter":{"op-type":"capture", "op-value":"time-based", "op-extra":{"period":"3"}}, "requestId": "6578"}`

	err := conn.WriteMessage(websocket.TextMessage, []byte(request))
	if err != nil {
		fmt.Printf("Subscribe request error:%s\n", err)
	}

}

func getResponse(conn *websocket.Conn, request []byte) []byte {
	err := conn.WriteMessage(websocket.BinaryMessage, request)
	if err != nil {
		fmt.Printf("Request error:%s\n", err)
		return nil
	}
	_, msg, err := conn.ReadMessage()
	if err != nil {
		fmt.Printf("Response error: %s\n", err)
		return nil
	}
	return msg
}

/*func transferData(elements int, sleepTime int, protocol string) {
	if protocol == "http" {
		for {
			iterateGetAndWrite(elements, sleepTime)
		}
	} else {
		conn := initVissV2WebSocket(compression)
		go iterateNotificationAndWrite(conn)
		subscribeToPaths(conn, elements, sleepTime)
		for {
			time.Sleep(1000 * time.Millisecond) // just to keep alive...
		}
	}
}*/

func performCommand(commandNumber int, conn *websocket.Conn, optionChannel chan string) {
    fmt.Printf("Request: %s\n", requestList.Request[commandNumber])
    pbRequest := utils.JsonToProtobuf(requestList.Request[commandNumber], convertToCompression(compression)) 
    fmt.Printf("JSON request size= %d, Protobuf request size=%d\n", len(requestList.Request[commandNumber]), len(pbRequest))
    fmt.Printf("Compression= %d%\n", (100*len(requestList.Request[commandNumber])) / len(pbRequest) - 100)
    compressedResponse := getResponse(conn, pbRequest)
    jsonResponse := utils.ProtobufToJson(compressedResponse, convertToCompression(compression))
    fmt.Printf("Response: %s\n", jsonResponse)
    fmt.Printf("JSON response size= %d, Protobuf response size=%d\n", len(jsonResponse), len(compressedResponse))
    fmt.Printf("Compression= %d%\n", (100*len(jsonResponse)) / len(compressedResponse) - 100)
    if (strings.Contains(requestList.Request[commandNumber], "subscribe") == true) {
        for {
	    _, msg, err := conn.ReadMessage()
	    if err != nil {
		fmt.Printf("Notification error: %s\n", err)
		return
	    }
	    jsonNotification := utils.ProtobufToJson(msg, convertToCompression(compression))
	    fmt.Printf("Notification: %s\n", jsonNotification)
	    fmt.Printf("JSON notification size= %d, Protobuf notification size=%d\n", len(jsonNotification), len(msg))
	    fmt.Printf("Compression= %d%\n", (100*len(jsonNotification)) / len(msg) - 100)
	    select {
	        case <- optionChannel:
	            // issue unsubscribe request
	            subscriptionId := utils.ExtractSubscriptionId(msg)
	            unsubReq := `{"action":"unsubscribe", "subscriptionId":"` + subscriptionId + `"}`
	            pbUnsubReq := utils.JsonToProtobuf(unsubReq, convertToCompression(compression))
	            getResponse(conn, pbUnsubReq)
	            return
	        default:
	    }
        }
    }
}

func convertToCompression(compression string) utils.Compression {
    switch compression {
        case "prop": return utils.PROPRIETARY
        case "pbl1": return utils.PB_LEVEL1
        case "pbl2": return utils.PB_LEVEL2
    }
    return utils.NONE
}

func displayOptions() {
    fmt.Printf("\n\nSelect one of the following numbers:\n")
    fmt.Printf("0: Exit program\n")
    for i := 0 ; i < len(requestList.Request) ; i++ {
        fmt.Printf("%d: %s\n", i+1, requestList.Request[i])
    }
    fmt.Printf("In the case of an ongoing subscription session, a RETURN key input will lead to unsubscribe.\n")
    fmt.Printf("\nOption number selected: ")
}

func readOption(optionChannel chan string) {
    for {
	fmt.Scanf("%s", &commandNumber)
	optionChannel <- commandNumber
    }
}

func main() {
	// Create new parser object
	parser := argparse.NewParser("print", "Prints provided string to stdout")

	// Create flags
	url_vissv2 := parser.String("v", "vissv2Url", &argparse.Options{Required: true, Help: "IP/url to VISSv2 server"})
	prot := parser.Selector("p", "protocol", []string{"http", "ws"}, &argparse.Options{Required: false, 
	                        Help: "Protocol must be either http or websocket", Default:"ws"})
	comp := parser.Selector("c", "compression", []string{"prop", "pbl1", "pbl2"}, &argparse.Options{Required: false, 
	                         Help: "Compression must be either proprietary or protobuf level 1 or 2", Default:"pbl1"})
	logFile := parser.Flag("", "logfile", &argparse.Options{Required: false, Help: "outputs to logfile in ./logs folder"})
	logLevel := parser.Selector("", "loglevel", []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}, &argparse.Options{
		Required: false,
		Help:     "changes log output level",
		Default:  "info"})

	// Parse input
	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		//exits due to required info not provided by user
		os.Exit(1)
	}

	//conversion since parsed flags are of *string type and not string
	vissv2Url = *url_vissv2
	protocol = *prot
	compression = *comp

	utils.InitLog("pb_client-log.txt", "./logs", *logFile, *logLevel)

	readTransportSecConfig()
	utils.Info.Printf("secConfig.TransportSec=%s", secConfig.TransportSec)
	if secConfig.TransportSec == "yes" {
		caCertPool = *prepareTransportSecConfig()
	}

	if createListFromFile("requests.json") == 0 {
		fmt.Printf("Failed in creating list from requests.json")
		os.Exit(1)
	}

	conn := initVissV2WebSocket(compression)

	optionChannel := make(chan string)	
	go readOption(optionChannel)

	for {
	    displayOptions()
	    select {
	        case commandNumber = <- optionChannel:
		    if (commandNumber == "0") {
	        	return
	            }
	    }
	    cNo, err := strconv.Atoi(commandNumber)
	    if (err != nil) {
	        fmt.Printf("Selected option not supported\n")
	        continue
	    }
	    performCommand(cNo-1, conn, optionChannel)
	}
}
