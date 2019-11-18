package manager

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/ioutil"
//	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/utils"

	"github.com/gorilla/websocket"
)

func backendHttpAppSession(message string, w *http.ResponseWriter) {
	utils.Info.Printf("backendWSAppSession(): Message received=%s\n", message)

	var responseMap = make(map[string]interface{})
	utils.ExtractPayload(message, &responseMap)
	var response string
	if responseMap["error"] != nil {
		http.Error(*w, "400 Error", http.StatusBadRequest) // TODO select error code from responseMap-error:number
		return
	}
	switch responseMap["action"] {
	case "get":
		response = responseMap["value"].(string)
	case "getmetadata":
		response = responseMap["metadata"].(string)
	case "set":
		response = "200 OK" //??
	default:
		http.Error(*w, "500 Internal error", http.StatusInternalServerError) // TODO select error code from responseMap-error:number
		return

	}
	resp := []byte(response)
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Content-Length", strconv.Itoa(len(resp)))
	written, err := (*w).Write(resp)
	if err != nil {
		utils.Error.Printf("HTTP manager error on response write.Written bytes=%d. Error=%s\n", written, err.Error())
	}
}

func FrontendWSdataSession(conn *websocket.Conn, clientChannel chan string, backendChannel chan string) {
	defer conn.Close()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			utils.Error.Printf("Service data read error:", err)
			break
		}
		utils.Info.Printf("%s request: %s \n", conn.RemoteAddr(), string(msg))

		clientChannel <- string(msg) // forward to mgr hub,
		message := <-clientChannel   //  and wait for response

		backendChannel <- message
	}
}

func BackendWSdataSession(conn *websocket.Conn, backendChannel chan string) {
	defer conn.Close()
	for {
		message := <-backendChannel

		utils.Info.Printf("Service:BackendWSdataSession(): message received=%s\n", message)
		// Write message back to server core
		response := []byte(message)

		err := conn.WriteMessage(websocket.TextMessage, response)
		if err != nil {
			utils.Error.Printf("Service data write error:", err)
			break
		}
	}
}

func frontendHttpAppSession(w http.ResponseWriter, req *http.Request, clientChannel chan string) {
	path := utils.UrlToPath(req.RequestURI)
	utils.Info.Printf("HTTP method:%s, path: %s\n", req.Method, path)
	var requestMap = make(map[string]interface{})
	switch req.Method {
	case "GET": // get/getmetadata
		if strings.Contains(path, "$spec") {
			requestMap["action"] = "getmetadata"
		} else {
			requestMap["action"] = "get"
		}
		requestMap["path"] = path
		requestMap["requestId"] = strconv.Itoa(requestTag)
		requestTag++
	case "POST": // set
		requestMap["action"] = "set"
		requestMap["path"] = path
		body, _ := ioutil.ReadAll(req.Body)
		requestMap["value"] = string(body)
		requestMap["requestId"] = strconv.Itoa(requestTag)
		requestTag++
	default:
		http.Error(w, "400 Unsupported method", http.StatusBadRequest)
		utils.Warning.Printf("Only GET and POST methods are supported.")
		return
	}
	clientChannel <- finalizeResponse(requestMap) // forward to mgr hub,
	response := <-clientChannel                   //  and wait for response

	backendHttpAppSession(response, &w)
}

func InitDataSession(muxServer *http.ServeMux, regData RegData) (dataConn *websocket.Conn) {
	var addr = flag.String("addr", "localhost:"+strconv.Itoa(regData.Portnum), "http service address")
	dataSessionUrl := url.URL{Scheme: "ws", Host: *addr, Path: regData.Urlpath}
	dataConn, _, err := websocket.DefaultDialer.Dial(dataSessionUrl.String(), nil)
	if err != nil {
		utils.Error.Fatal("Data session dial error:" + err.Error())
	}
	return dataConn
}

/**
* registerAsTransportMgr:
* Registers with servercore as WebSocket protocol manager, and stores response in regData
**/
func RegisterAsTransportMgr(regData *RegData, protocol string) {
	url := "http://localhost:8081/transport/reg"

//	data := []byte(`{"protocol": "WebSocket"}`)
	data := []byte(`{"protocol": "` + protocol + `"}`)
	//    data := []byte(`{"protocol": "HTTP"}`)  // use in HTTP manager

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		utils.Error.Fatal("registerAsTransportMgr: Error reading request. ", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", "localhost:8081")

	// Set client timeout
	client := &http.Client{Timeout: time.Second * 10}

	// Validate headers are attached
	utils.Info.Println(req.Header)

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		utils.Error.Fatal("registerAsTransportMgr: Error reading response. ", err)
	}
	defer resp.Body.Close()

	utils.Info.Println("response Status:", resp.Status)
	utils.Info.Println("response Headers:", resp.Header)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		utils.Error.Fatal("Error reading response. ", err)
	}
	utils.Info.Printf("%s\n", body)

	err = json.Unmarshal(body, regData)
	if err != nil {
		utils.Error.Fatal("Error JSON decoding of response. ", err)
	}
}

func frontendWSAppSession(conn *websocket.Conn, clientChannel chan string, clientBackendChannel chan string) {
	defer conn.Close()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			utils.Error.Printf("App client read error:", err)
			break
		}

		payload := utils.UrlToPath(string(msg)) // if path in payload slash delimited, replace with dot delimited
		utils.Info.Printf("%s request: %s, len=%d\n", conn.RemoteAddr(), payload, len(payload))

		clientChannel <- payload    // forward to mgr hub,
		response := <-clientChannel //  and wait for response

		clientBackendChannel <- response
	}
}

func backendWSAppSession(conn *websocket.Conn, clientBackendChannel chan string) {
	defer conn.Close()
	for {
		message := <-clientBackendChannel

		utils.Info.Printf("backendWSAppSession(): Message received=%s\n", message)
		// Write message back to app client
		response := []byte(message)

		err := conn.WriteMessage(websocket.TextMessage, response)
		if err != nil {
			utils.Error.Print("App client write error:", err)
			break
		}
	}
}

func (httpH HttpChannel) makeappClientHandler(appClientChannel []chan string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Upgrade") == "websocket" {
			http.Error(w, "400 Incorrect port number", http.StatusBadRequest)
			utils.Warning.Printf("Client call to incorrect port number for websocket connection.\n")
			return
		}
		frontendHttpAppSession(w, req, appClientChannel[0])
	}
}

func (wsH WsChannel) makeappClientHandler(appClientChannel []chan string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Upgrade") == "websocket" {
			utils.Info.Printf("we are upgrading to a websocket connection. Server index=%d\n", *wsH.serverIndex)
			Upgrader.CheckOrigin = func(r *http.Request) bool { return true }
			conn, err := Upgrader.Upgrade(w, req, nil)
			if err != nil {
				utils.Error.Print("upgrade error:", err)
				return
			}
			if *wsH.serverIndex < len(appClientChannel) {
				go frontendWSAppSession(conn, appClientChannel[*wsH.serverIndex], wsH.clientBackendChannel[*wsH.serverIndex])
				go backendWSAppSession(conn, wsH.clientBackendChannel[*wsH.serverIndex])
				*wsH.serverIndex += 1
			} else {
				utils.Warning.Printf("not possible to start more app client sessions.\n")
			}
		} else {
			utils.Warning.Printf("Client must set up a Websocket session.\n")
		}
	}
}

func (server HttpServer) InitClientServer(muxServer *http.ServeMux) {

	appClientHandler := HttpChannel{}.makeappClientHandler(AppClientChan)
	muxServer.HandleFunc("/", appClientHandler)
	utils.Info.Println(http.ListenAndServe(HostIP+":8888", muxServer))
}

func (server WsServer) InitClientServer(muxServer *http.ServeMux) {
	serverIndex := 0
	appClientHandler := WsChannel{server.ClientBackendChannel, &serverIndex}.makeappClientHandler(AppClientChan)
	muxServer.HandleFunc("/", appClientHandler)
	utils.Error.Fatal(http.ListenAndServe(HostIP+":8080", muxServer))
}

func finalizeResponse(responseMap map[string]interface{}) string {
	response, err := json.Marshal(responseMap)
	if err != nil {
		utils.Error.Printf(err.Error(), " ", TransportErrorMessage)
		return "JSON marshal error" // what to do here?
	}
	return string(response)
}

func (httpCoreSocketSession HttpWSsession) TransportHubFrontendWSsession(dataConn *websocket.Conn, appClientChannel []chan string) {
	for {
		_, response, err := dataConn.ReadMessage()
		if err != nil {
			utils.Error.Println("Datachannel read error:" + err.Error())
			return // ??
		}
		utils.Info.Printf("Server hub: HTTP response from server core:%s\n", string(response))
		var responseMap = make(map[string]interface{})
		utils.ExtractPayload(string(response), &responseMap)
		clientId := int(responseMap["ClientId"].(float64))
//              utils.Info.Printf("Client id: %s", clientId)
		delete(responseMap, "MgrId")
		delete(responseMap, "ClientId")
		appClientChannel[clientId] <- finalizeResponse(responseMap) // no need for clientBackendChannel as subscription notifications not supported
//              utils.Info.Println("HTTP core server hub channel message to transport frontend:" + finalizeResponse(responseMap))
	}
}

func (wsCoreSocketSession WsWSsession) TransportHubFrontendWSsession(dataConn *websocket.Conn, appClientChannel []chan string) {
	for {
		_, response, err := dataConn.ReadMessage()
		if err != nil {
			utils.Error.Println("Datachannel read error:", err)
			return // ??
		}
		utils.Info.Printf("Server hub: WS response from server core:%s\n", string(response))
		var responseMap = make(map[string]interface{})
		utils.ExtractPayload(string(response), &responseMap)
		clientId := int(responseMap["ClientId"].(float64))
//              utils.Info.Printf("Client id: %s", clientId)
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
