/**
* (C) 2019 Geotab Inc
* (C) 2019 Volvo Cars
*
* All files and artifacts in the repository at https://github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package utils

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/ioutil"

	//	"log"
	"net/http"
	"net/url"
	"strconv"

	//	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func backendHttpAppSession(message string, w *http.ResponseWriter) {
	Info.Printf("backendHttpAppSession(): Message received=%s", message)

	var responseMap = make(map[string]interface{})
	ExtractPayload(message, &responseMap)
	if responseMap["action"] != nil {
            delete(responseMap, "action")
        }
	if responseMap["requestId"] != nil {
            delete(responseMap, "requestId")
        }
        response := finalizeResponse(responseMap)

	resp := []byte(response)
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Headers", "*")
	(*w).Header().Set("Content-Length", strconv.Itoa(len(resp)))
	written, err := (*w).Write(resp)
	if err != nil {
		Error.Printf("HTTP manager error on response write.Written bytes=%d. Error=%s\n", written, err.Error())
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
		Info.Printf("%s request: %s \n", conn.RemoteAddr(), string(msg))

		clientChannel <- string(msg) // forward to mgr hub,
		message := <-clientChannel   //  and wait for response

		backendChannel <- message
	}
}

func BackendWSdataSession(conn *websocket.Conn, backendChannel chan string) {
	defer conn.Close()
	for {
		message := <-backendChannel

		Info.Printf("Service:BackendWSdataSession(): message received=%s\n", message)
		// Write message back to server core
		response := []byte(message)

		err := conn.WriteMessage(websocket.TextMessage, response)
		if err != nil {
			Error.Printf("Service data write error: %s", err)
			break
		}
	}
}

func frontendHttpAppSession(w http.ResponseWriter, req *http.Request, clientChannel chan string) {
	path := UrlToPath(req.RequestURI)
	Info.Printf("HTTP method:%s, path: %s", req.Method, path)
	var requestMap = make(map[string]interface{})
	requestMap["path"] = path
        token := req.Header.Get("Authorization")
	Info.Printf("HTTP token:%s", token)
        if (len(token) > 0) {
            requestMap["token"] = token
        }
	requestMap["requestId"] = strconv.Itoa(requestTag)
	requestTag++
	switch req.Method {
	case "OPTIONS":
                fallthrough  // should work for POST also...
	case "GET":
		requestMap["action"] = "get"
	case "POST": // set
		requestMap["action"] = "set"
		body, _ := ioutil.ReadAll(req.Body)
		requestMap["value"] = string(body)
	default:
//		http.Error(w, "400 Unsupported method", http.StatusBadRequest)
		Warning.Printf("Only GET and POST methods are supported.")
 	        backendHttpAppSession(`{"error": "Unrecognized HTTP method."}`, &w) // ???
		return
	}
	clientChannel <- finalizeResponse(requestMap) // forward to mgr hub,
	response := <-clientChannel                   //  and wait for response

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

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		Error.Fatal("registerAsTransportMgr: Error reading response. ", err)
	}
	defer resp.Body.Close()

	Info.Println("response Status:", resp.Status)
	Info.Println("response Headers:", resp.Header)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Error.Fatal("Error reading response. ", err)
	}
	Info.Printf("%s\n", body)

	err = json.Unmarshal(body, regData)
	if err != nil {
		Error.Fatal("Error JSON decoding of response. ", err)
	}
}

func frontendWSAppSession(conn *websocket.Conn, clientChannel chan string, clientBackendChannel chan string) {
	defer conn.Close()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			Error.Printf("App client read error: %s", err)
			break
		}

		payload := UrlToPath(string(msg)) // if path in payload slash delimited, replace with dot delimited
		Info.Printf("%s request: %s, len=%d\n", conn.RemoteAddr(), payload, len(payload))

		clientChannel <- payload    // forward to mgr hub,
		response := <-clientChannel //  and wait for response

		clientBackendChannel <- response
	}
}

func backendWSAppSession(conn *websocket.Conn, clientBackendChannel chan string) {
	defer conn.Close()
	for {
		message := <-clientBackendChannel

		Info.Printf("backendWSAppSession(): Message received=%s\n", message)
		// Write message back to app client
		response := []byte(message)

		err := conn.WriteMessage(websocket.TextMessage, response)
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
			Warning.Printf("Client call to incorrect port number for websocket connection.\n")
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
			conn, err := Upgrader.Upgrade(w, req, nil)
			if err != nil {
				Error.Print("upgrade error:", err)
				return
			}
			Info.Printf("len(appClientChannel)=%d", len(appClientChannel))
			if *wsH.serverIndex < len(appClientChannel) {
				go frontendWSAppSession(conn, appClientChannel[*wsH.serverIndex], wsH.clientBackendChannel[*wsH.serverIndex])
				go backendWSAppSession(conn, wsH.clientBackendChannel[*wsH.serverIndex])
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
	Info.Println(http.ListenAndServe(":8888", muxServer))
}

func (server WsServer) InitClientServer(muxServer *http.ServeMux, serverIndex *int) {
	*serverIndex = 0
	appClientHandler := WsChannel{server.ClientBackendChannel, serverIndex}.makeappClientHandler(AppClientChan)
	muxServer.HandleFunc("/", appClientHandler)
	Error.Fatal(http.ListenAndServe(":8080", muxServer))
}

func finalizeResponse(responseMap map[string]interface{}) string {
	response, err := json.Marshal(responseMap)
	if err != nil {
		Error.Printf(err.Error(), " ", TransportErrorMessage)
		return "JSON marshal error" // what to do here?
	}
	return string(response)
}

func (httpCoreSocketSession HttpWSsession) TransportHubFrontendWSsession(dataConn *websocket.Conn, appClientChannel []chan string) {
	for {
		_, response, err := dataConn.ReadMessage()
		if err != nil {
			Error.Println("Datachannel read error:" + err.Error())
			return // ??
		}
		Info.Printf("Server hub: HTTP response from server core:%s\n", string(response))
		var responseMap = make(map[string]interface{})
		ExtractPayload(string(response), &responseMap)
		clientId := int(responseMap["ClientId"].(float64))
		//              Info.Printf("Client id: %s", clientId)
		delete(responseMap, "MgrId")
		delete(responseMap, "ClientId")
		appClientChannel[clientId] <- finalizeResponse(responseMap) // no need for clientBackendChannel as subscription notifications not supported
		//              Info.Println("HTTP core server hub channel message to transport frontend:" + finalizeResponse(responseMap))
	}
}

func (wsCoreSocketSession WsWSsession) TransportHubFrontendWSsession(dataConn *websocket.Conn, appClientChannel []chan string) {
	for {
		_, response, err := dataConn.ReadMessage()
		if err != nil {
			Error.Println("Datachannel read error:", err)
			return // ??
		}
		Info.Printf("Server hub: WS response from server core:%s\n", string(response))
		var responseMap = make(map[string]interface{})
		ExtractPayload(string(response), &responseMap)
		clientId := int(responseMap["ClientId"].(float64))
		//              Info.Printf("Client id: %s", clientId)
		delete(responseMap, "MgrId")
		delete(responseMap, "ClientId")
		if responseMap["action"] == "subscription" {
			wsCoreSocketSession.ClientBackendChannel[clientId] <- finalizeResponse(responseMap) //subscription notification
		} else {
			appClientChannel[clientId] <- finalizeResponse(responseMap)
			//                      utils.Info.Println("WS core server hub channel message to transport frontend:" + finalizeResponse(responseMap))
		}
	}
}
