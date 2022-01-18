/**
* (C) 2019 Geotab Inc
* (C) 2019 Volvo Cars
*
* All files and artifacts in the repository at https://github.com/MEAE-GOT/WAII
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package utils

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/ioutil"

	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/url"

	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func ReadTransportSecConfig() {
	data, err := ioutil.ReadFile(trSecConfigPath + "transportSec.json")
	if err != nil {
		Info.Printf("ReadTransportSecConfig():%stransportSec.json error=%s", trSecConfigPath, err)
		secConfig.TransportSec = "no"
		return
	}
	err = json.Unmarshal(data, &secConfig)
	if err != nil {
		Error.Printf("ReadTransportSecConfig():Error unmarshal transportSec.json=%s", err)
		secConfig.TransportSec = "no"
		return
	}
	Info.Printf("ReadTransportSecConfig():secConfig.TransportSec=%s", secConfig.TransportSec)
}

func backendHttpAppSession(message string, w *http.ResponseWriter) {
	Info.Printf("backendHttpAppSession(): Message received=%s", message)

	var responseMap = make(map[string]interface{})
	MapRequest(message, &responseMap)
	if responseMap["action"] != nil {
		delete(responseMap, "action")
	}
	if responseMap["requestId"] != nil {
		delete(responseMap, "requestId")
	}
	response := FinalizeMessage(responseMap)

	resp := []byte(response)
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Headers", "*")
	(*w).Header().Set("Content-Length", strconv.Itoa(len(resp)))
	written, err := (*w).Write(resp)
	if err != nil {
		Error.Printf("HTTP manager error on response write.Written bytes=%d. Error=%s", written, err.Error())
	}
}

func FrontendWSdataSession(conn *websocket.Conn, clientChannel chan string, backendChannel chan string) {
	defer conn.Close()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			Error.Printf("Service data read error: %s", err)
			break
		}
		Info.Printf("%s request: %s", conn.RemoteAddr(), string(msg))

		clientChannel <- string(msg) // forward to mgr hub,
		message := <-clientChannel   //  and wait for response

		backendChannel <- message
	}
}

func BackendWSdataSession(conn *websocket.Conn, backendChannel chan string) {
	defer conn.Close()
	for {
		message := <-backendChannel

		Info.Printf("Service:BackendWSdataSession(): message received=%s", message)
		// Write message back to server core
		response := []byte(message)

		//		err := conn.WriteMessage(websocket.TextMessage, response)
		err := conn.WriteMessage(websocket.BinaryMessage, response)
		if err != nil {
			Error.Printf("Service data write error: %s", err)
			break
		}
	}
}

func splitToPathQueryKeyValue(path string) (string, string, string) {
	delim := strings.Index(path, "?")
	if delim != -1 {
		if path[delim+1] == 'f' {
			return path[:delim], "filter", path[delim+8:] // path?filter=json-exp
		} else if path[delim+1] == 'm' {
			return path[:delim], "metadata", path[delim+10:] // path?metadata=static (or dynamic)
		}
	}
	return path, "", ""
}

func frontendHttpAppSession(w http.ResponseWriter, req *http.Request, clientChannel chan string) {
	path := req.RequestURI
	if len(path) == 0 {
		path = "empty-path" // will generate error as not found in VSS tree
	}
	path = strings.ReplaceAll(path, "%22", "\"")
	path = strings.ReplaceAll(path, "%20", "")
	var requestMap = make(map[string]interface{})
	queryKey := ""
	queryValue := ""
	if strings.Contains(path, "?") == true {
		requestMap["path"], queryKey, queryValue = splitToPathQueryKeyValue(path)
	} else {
		requestMap["path"] = path
	}
	Info.Printf("HTTP method:%s, path: %s", req.Method, path)
	token := req.Header.Get("Authorization")
	Info.Printf("HTTP token:%s", token)
	if len(token) > 0 {
		requestMap["token"] = token
	}
	requestMap["requestId"] = strconv.Itoa(requestTag)
	requestTag++
	switch req.Method {
	case "OPTIONS":
		fallthrough // should work for POST also...
	case "GET":
		requestMap["action"] = "get"
	case "POST": // set
		requestMap["action"] = "set"
		body, _ := ioutil.ReadAll(req.Body)
		requestMap["value"] = string(body)
	default:
		//		http.Error(w, "400 Unsupported method", http.StatusBadRequest)
		Warning.Printf("Only GET and POST methods are supported.")
		backendHttpAppSession(`{"error": "400", "reason": "Bad request", "message":"Unsupported HTTP method"}`, &w)
		return
	}
	clientChannel <- AddKeyValue(FinalizeMessage(requestMap), queryKey, queryValue) // forward to mgr hub,
	response := <-clientChannel                                                     //  and wait for response

	backendHttpAppSession(response, &w)
}

func InitDataSession(muxServer *http.ServeMux, regData RegData) (dataConn *websocket.Conn) {
	var addr = flag.String("addr", GetServerIP()+":"+strconv.Itoa(regData.Portnum), "http service address")
	dataSessionUrl := url.URL{Scheme: "ws", Host: *addr, Path: regData.Urlpath}
	dataConn, _, err := websocket.DefaultDialer.Dial(dataSessionUrl.String(), nil)
	if err != nil {
		Error.Fatal("Data session dial error:" + err.Error())
	}
	return dataConn
}

/**
* registerAsTransportMgr:
* Registers with servercore as WebSocket protocol manager, and stores response in regData
**/
func RegisterAsTransportMgr(regData *RegData, protocol string) {
	url := "http://" + GetServerIP() + ":8081/transport/reg"

	data := []byte(`{"protocol": "` + protocol + `"}`)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		Error.Fatal("registerAsTransportMgr: Error reading request. ", err)
	}

	// Set headers
	req.Header.Set("Access-Control-Allow-Origin", "*")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", GetServerIP()+":8081")

	// Set client timeout
	client := &http.Client{Timeout: time.Second * 10}

	// Validate headers are attached
	Info.Println(req.Header)

	// Send request loop until connection succeeds
	var resp *http.Response
	for {
		resp, err = client.Do(req)
		if err != nil {
			Error.Error("registerAsTransportMgr: Error reading response. ", err)
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}
	defer resp.Body.Close()

	Info.Println("response Status:", resp.Status)
	Info.Println("response Headers:", resp.Header)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Error.Fatal("Error reading response. ", err)
	}
	Info.Printf("%s", body)

	err = json.Unmarshal(body, regData)
	if err != nil {
		Error.Fatal("Error JSON decoding of response. ", err)
	}
}

func frontendWSAppSession(conn *websocket.Conn, clientChannel chan string, clientBackendChannel chan string, compression Compression) {
	defer conn.Close()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			Error.Printf("App client read error: %s", err)
			break
		}

		var payload string
		if (compression == PROPRIETARY) {
		    payload = string(DecompressMessage(msg))
		} else if (compression == PB_LEVEL1 || compression == PB_LEVEL2) {
		    payload = ProtobufToJson(msg, compression)
		} else {
		    payload = string(msg)
		}
		Info.Printf("%s request: %s, len=%d", conn.RemoteAddr(), payload, len(payload))
		Info.Printf("Compression variant=%d", compression)

		clientChannel <- payload    // forward to mgr hub,
		response := <-clientChannel //  and wait for response

		clientBackendChannel <- response
	}
}

func backendWSAppSession(conn *websocket.Conn, clientBackendChannel chan string, compression Compression) {
	defer conn.Close()
	for {
		message := <-clientBackendChannel

		Info.Printf("backendWSAppSession(): Message received=%s", message)
		// Write message back to app client
		var response []byte
	        var messageType int

		if (compression == PROPRIETARY) {
		    response = CompressMessage([]byte(message))
		    messageType = websocket.BinaryMessage
		} else if (compression == PB_LEVEL1 || compression == PB_LEVEL2) {
		    response = []byte(JsonToProtobuf(message, compression))
		    messageType = websocket.BinaryMessage
		} else {
		    response = []byte(message)
		    messageType = websocket.TextMessage
               }
	        err := conn.WriteMessage(messageType, response)
		if err != nil {
			Error.Print("App client write error:", err)
			break
		}
	}
}

func (httpH HttpChannel) makeappClientHandler(appClientChannel []chan string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Upgrade") == "websocket" {
			http.Error(w, "400 Incorrect port number", http.StatusBadRequest)
			Warning.Printf("Client call to incorrect port number for websocket connection.")
			return
		}
		frontendHttpAppSession(w, req, appClientChannel[0])
	}
}

func (wsH WsChannel) makeappClientHandler(appClientChannel []chan string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Upgrade") == "websocket" {
			Info.Printf("we are upgrading to a websocket connection. Server index=%d", *wsH.serverIndex)
			Upgrader.CheckOrigin = func(r *http.Request) bool { return true }
			var compression Compression
			compression = NONE
			h := http.Header{}
			for _, sub := range websocket.Subprotocols(req) {
			   if sub == "VISSv2" {
			      compression = NONE
			      h.Set("Sec-Websocket-Protocol", sub)
			      break
			   }
			   if sub == "VISSv2prop" {
			      if (InitCompression("../vsspathlist.json") == true) {
			          compression = PROPRIETARY
			          h.Set("Sec-Websocket-Protocol", sub)
			      } else {
			          Error.Printf("Cannot find vsspathlist.json.")
			          compression = NONE // revert back to no compression
			          h.Set("Sec-Websocket-Protocol", "VISSv2")
			      }
			      break
			   }
			   if sub == "VISSv2pbl1" {
			      compression = PB_LEVEL1
			      h.Set("Sec-Websocket-Protocol", sub)
			      break
			   }
			   if sub == "VISSv2pbl2" {
			      if (InitCompression("../vsspathlist.json") == true) {
			          compression = PB_LEVEL2
			          h.Set("Sec-Websocket-Protocol", sub)
			      } else {
			          Error.Printf("Cannot find vsspathlist.json.")
			          compression = PB_LEVEL1 // revert back to level 1
			          h.Set("Sec-Websocket-Protocol", "VISSv2pbl1")
			      }
			      break
			   }
			}
			conn, err := Upgrader.Upgrade(w, req, h)
			if err != nil {
				Error.Print("upgrade error:", err)
				return
			}
			Info.Printf("len(appClientChannel)=%d", len(appClientChannel))
			if *wsH.serverIndex < len(appClientChannel) {
				go frontendWSAppSession(conn, appClientChannel[*wsH.serverIndex], wsH.clientBackendChannel[*wsH.serverIndex], compression)
				go backendWSAppSession(conn, wsH.clientBackendChannel[*wsH.serverIndex], compression)
				*wsH.serverIndex += 1
			} else {
				Error.Printf("not possible to start more app client sessions.")
			}
		} else {
			Error.Printf("Client must set up a Websocket session.")
		}
	}
}

func (server HttpServer) InitClientServer(muxServer *http.ServeMux) {

	appClientHandler := HttpChannel{}.makeappClientHandler(AppClientChan)
	muxServer.HandleFunc("/", appClientHandler)
	Info.Printf("InitClientServer():secConfig.TransportSec=%s", secConfig.TransportSec)
	if secConfig.TransportSec == "yes" {
		secPortNum, _ := strconv.Atoi(secConfig.HttpSecPort)
		server := http.Server{
			Addr: ":" + strconv.Itoa(secPortNum),
			TLSConfig: getTLSConfig("localhost", trSecConfigPath+secConfig.CaSecPath+"Root.CA.crt",
				tls.ClientAuthType(certOptToInt(secConfig.ServerCertOpt))),
			Handler: muxServer,
		}
		Info.Printf("HTTPS:CerOpt=%s", secConfig.ServerCertOpt)
		Error.Fatal(server.ListenAndServeTLS(trSecConfigPath+secConfig.ServerSecPath+"server.crt", trSecConfigPath+secConfig.ServerSecPath+"server.key"))
	} else {
		Error.Fatal(http.ListenAndServe(":8888", muxServer))
	}
}

func (server WsServer) InitClientServer(muxServer *http.ServeMux, serverIndex *int) {
	*serverIndex = 0
	appClientHandler := WsChannel{server.ClientBackendChannel, serverIndex}.makeappClientHandler(AppClientChan)
	muxServer.HandleFunc("/", appClientHandler)
	Info.Printf("InitClientServer():secConfig.TransportSec=%s", secConfig.TransportSec)
	if secConfig.TransportSec == "yes" {
		server := http.Server{
			Addr: ":" + secConfig.WsSecPort,
			TLSConfig: getTLSConfig("localhost", trSecConfigPath+secConfig.CaSecPath+"Root.CA.crt",
				tls.ClientAuthType(certOptToInt(secConfig.ServerCertOpt))),
			Handler: muxServer,
		}
		Info.Printf("HTTPS:CerOpt=%s", secConfig.ServerCertOpt)
		Error.Fatal(server.ListenAndServeTLS(trSecConfigPath+secConfig.ServerSecPath+"server.crt", trSecConfigPath+secConfig.ServerSecPath+"server.key"))
	} else {
		Error.Fatal(http.ListenAndServe(":8080", muxServer))
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

func getTLSConfig(host string, caCertFile string, certOpt tls.ClientAuthType) *tls.Config {
	var caCert []byte
	var err error
	var caCertPool *x509.CertPool
	if certOpt > tls.RequestClientCert {
		caCert, err = ioutil.ReadFile(caCertFile)
		if err != nil {
			Error.Printf("Error opening cert file", caCertFile, ", error ", err)
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

func RemoveInternalData(response string) (string, int) { // "RouterId" : "mgrId?clientId",
	routerIdStart := strings.Index(response, "RouterId") - 1
	clientIdStart := strings.Index(response[routerIdStart:], "?") + 1 + routerIdStart
	clientIdStop := NextQuoteMark([]byte(response), clientIdStart)
	clientId, _ := strconv.Atoi(response[clientIdStart:clientIdStop])
	routerIdStop := strings.Index(response[clientIdStop:], ",") + 1 + clientIdStop
	trimmedResponse := response[:routerIdStart] + response[routerIdStop:]
	return trimmedResponse, clientId
}

func (httpCoreSocketSession HttpWSsession) TransportHubFrontendWSsession(dataConn *websocket.Conn, appClientChannel []chan string) {
	for {
		_, response, err := dataConn.ReadMessage()
		if err != nil {
			Error.Println("Datachannel read error:" + err.Error())
			return // ??
		}
		Info.Printf("Server hub: HTTP response from server core:%s", string(response))
		trimmedResponse, clientId := RemoveInternalData(string(response))
		appClientChannel[clientId] <- trimmedResponse // no need for clientBackendChannel as subscription notifications not supported
	}
}

func (wsCoreSocketSession WsWSsession) TransportHubFrontendWSsession(dataConn *websocket.Conn, appClientChannel []chan string) {
	for {
		_, response, err := dataConn.ReadMessage()
		if err != nil {
			Error.Println("Datachannel read error:", err)
			return // ??
		}
		Info.Printf("Server hub: WS response from server core:%s", string(response))
		trimmedResponse, clientId := RemoveInternalData(string(response))
		if strings.Contains(trimmedResponse, "\"subscription\"") {
			wsCoreSocketSession.ClientBackendChannel[clientId] <- trimmedResponse //subscription notification
		} else {
			appClientChannel[clientId] <- trimmedResponse
		}
	}
}
