/**
* (C) 2019 Volvo Cars
*
* All files and artifacts in the repository at https://github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package main

import (
//    "fmt"
    "flag"
    "github.com/gorilla/websocket"
//    "log"
    "math/rand"
    "time"
    "net/http"
    "net/url"
    "strconv"
    "encoding/json"
    "strings"
    "utils"
)

// #include <stdlib.h>
// #include <stdio.h>
// #include <stdbool.h>
// #include "vssparserutilities.h"
import "C"

import "unsafe"
var rootHandle C.long

type searchData_t struct {  // searchData_t defined in vssparserutilities.h
    responsePath [512]byte  // vssparserutilities.h: #define MAXCHARSPATH 512; typedef char path_t[MAXCHARSPATH];
    foundNodeHandle int64  // defined as long in vssparserutilities.h
}

var transportRegChan chan int
var transportRegPortNum int = 8081
var transportDataPortNum int = 8100  // port number interval [8100-]

// add element to both channels if support for new transport protocol is added
var transportDataChan = []chan string {
    make(chan string),
    make(chan string),
}

var backendChan = []chan string {
    make(chan string),
    make(chan string),
}

/*
* To add support for one more transport manager protocol:
*    - add a map entry to supportedProtocols
*    - add a komponent to the muxServer array
*    - add a component to the transportDataChan array
*    - add a select case in the main loop
*/
var supportedProtocols = map[int]string {
    0: "HTTP",
    1: "WebSocket",
}

var serviceRegChan chan string
var serviceRegPortNum int = 8082
var serviceDataPortNum int = 8200  // port number interval [8200-]

// add element if support for new service manager is added
var serviceDataChan = []chan string {
    make(chan string),
    make(chan string),
}

/** muxServer[0] is assigned to transport registration server, 
*   muxServer[1] is assigned to service registration server, 
*   of the following the first half is assigned for transport data servers,
*   and the second half is assigned for service data clients
**/
var muxServer = []*http.ServeMux {
    http.NewServeMux(),  // 0 = transport reg
    http.NewServeMux(),  // 1 = service reg
    http.NewServeMux(),  // 2 = transport data
    http.NewServeMux(),  // 3 = transport data
    http.NewServeMux(),  // 4 = service data
    http.NewServeMux(),  // 5 = service data
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type RouterTable_t struct {
    mgrId int
    mgrIndex int
}

var routerTable []RouterTable_t

var errorResponseMap = map[string]interface{} {
    "MgrId":0,
    "ClientId":0,
    "action":"unknown",
    "requestId":"XXX",
    "error":"{\"number\":99, \"reason\": \"BBB\", \"message\": \"CCC\"}",
    "timestamp":1234,
}

/*
* Core-server main tasks:
    - server for transportmgr registrations
    - server for servicemgr registrations
    - server in transportmgr data channel requests
    - client in servicemgr data channel requests
    - router hub for request-response messages
    - request message path verification
    - request message access restriction control
    - service discovery response synthesis
*/

func routerTableAdd(mgrId int, mgrIndex int) {
    var tableElement RouterTable_t
    tableElement.mgrId = mgrId
    tableElement.mgrIndex = mgrIndex
    routerTable = append(routerTable, tableElement)
}

func routerTableSearchForMgrIndex(mgrId int) int {
    for _, element := range routerTable {
        if (element.mgrId == mgrId) {
            utils.Info.Printf("routerTableSearchForMgrIndex: Found index=%d\n", element.mgrIndex)
            return element.mgrIndex
        }
    }
    return -1
}

func getPayloadMgrId(request string) int {
    type Payload struct {
        MgrId int
    }
    decoder := json.NewDecoder(strings.NewReader(request))
    var payload Payload
    err := decoder.Decode(&payload)
    if err != nil {
        utils.Error.Printf("Server core-getPayloadMgrId: JSON decode failed for request:%s\n", request)
        return -1
    }
    return payload.MgrId
}

/*
* The transportRegisterServer assigns a requesting transport mgr the data channel port number to use, 
* the data channel URL path, and the transport mgr ID that shall be added to the server internal req/resp messages.
* This is communicated to the coreserver that will save it in its router database. 
* The port number returned is unique per protocol supported.
* If there is a need to support registering of multiple mgrs for the same protocol, 
* then caching assigned mgr data can be used to assign other unique portno + mgr ID. Currently not supported.
*/
func maketransportRegisterHandler(transportRegChannel chan int) func(http.ResponseWriter, *http.Request) {
    return func(w http.ResponseWriter, req *http.Request) {
    utils.Info.Printf("transportRegisterServer():url=%s\n", req.URL.Path)
    mgrIndex := -1
    if (req.URL.Path != "/transport/reg") {
        http.Error(w, "404 url path not found.", 404)
    } else if (req.Method != "POST") {
        http.Error(w, "400 bad request method.", 400)
    } else {
        type Payload struct {
            Protocol string
        }
        decoder := json.NewDecoder(req.Body)
        var payload Payload
        err := decoder.Decode(&payload)
        if err != nil {
            panic(err)
        }
    utils.Info.Printf("transportRegisterServer():POST request=%s\n", payload.Protocol)
        for key, value := range supportedProtocols {
            if (payload.Protocol == value) {
                mgrIndex = key
            }
        }
        if (mgrIndex != -1) {  // communicate: port no + mgr Id to server hub, port no + url path + mgr Id to transport mgr
            transportRegChannel <- transportDataPortNum + mgrIndex  // port no
            mgrId := rand.Intn(65535)  // [0 -65535], 16-bit value
            transportRegChannel <- mgrId  // mgr id
	    w.Header().Set("Content-Type", "application/json")
            response := "{ \"Portnum\" : " + strconv.Itoa(transportDataPortNum + mgrIndex) + " , \"Urlpath\" : \"/transport/data/" + strconv.Itoa(mgrIndex) + "\"" + " , \"Mgrid\" : " + strconv.Itoa(mgrId) + " }"
            
            utils.Info.Printf("transportRegisterServer():POST response=%s\n", response)
            w.Write([]byte(response)) // correct JSON?
            routerTableAdd(mgrId, mgrIndex)
        } else {
            http.Error(w, "404 protocol not supported.", 404)
        }
    }
    }
}

func initTransportRegisterServer(transportRegChannel chan int) {
    utils.Info.Printf("initTransportRegisterServer():localhost:8081/transport/reg\n")
    transportRegisterHandler := maketransportRegisterHandler(transportRegChannel)
    muxServer[0].HandleFunc("/transport/reg", transportRegisterHandler)
    utils.Error.Fatal(http.ListenAndServe("localhost:8081", muxServer[0]))
}

func frontendServiceDataComm(dataConn *websocket.Conn, request string) {
    err := dataConn.WriteMessage(websocket.TextMessage, []byte(request)); 
    if (err != nil) {
        utils.Error.Printf("Service datachannel write error:", err)
    }
}

func backendServiceDataComm(dataConn *websocket.Conn, backendChannel []chan string, serviceIndex int) {
    for {
        _, response, err := dataConn.ReadMessage()
        utils.Info.Printf("Server core: Response from service mgr:%s\n", string(response))
        var responseMap = make(map[string]interface{})
        if err != nil {
            utils.Error.Println("Service datachannel read error:", err)
            response = []byte(finalizeMessage(errorResponseMap))  // needs improvement
        } else {
            utils.ExtractPayload(string(response), &responseMap)
        }
        if responseMap["action"] == "subscription" {
            mgrIndex := routerTableSearchForMgrIndex(int(responseMap["MgrId"].(float64)))
            backendChannel[mgrIndex] <- string(response)
        } else {
            serviceDataChan[serviceIndex] <- string(response)  // response to request
        }
    }
}

/**
* initServiceDataSession:
* sets up the WS based communication (as client) with a service manager
**/
func initServiceDataSession(muxServer *http.ServeMux, serviceIndex int, backendChannel []chan string) (dataConn *websocket.Conn) {
    var addr = flag.String("addr", "localhost:" + strconv.Itoa(serviceDataPortNum+serviceIndex), "http service address")
    dataSessionUrl := url.URL{Scheme: "ws", Host: *addr, Path: "/service/data/"+strconv.Itoa(serviceIndex)}
    utils.Info.Printf("Connecting to:%s\n", dataSessionUrl.String())
    dataConn, _, err := websocket.DefaultDialer.Dial(dataSessionUrl.String(), http.Header{"Access-Control-Allow-Origin":{"*"}})
//    dataConn, _, err := websocket.DefaultDialer.Dial(dataSessionUrl.String(), nil)
    if err != nil {
        utils.Error.Fatal("Service data session dial error:", err)
        return nil
    }
    go backendServiceDataComm(dataConn, backendChannel, serviceIndex)
    return dataConn
}

func initServiceClientSession(serviceDataChannel chan string, serviceIndex int, backendChannel []chan string) {
    time.Sleep(3*time.Second)  //wait for service data server to be initiated (initiate at first app-client request instead...)
    muxIndex := (len(muxServer) -2)/2 + 1 + (serviceIndex +1)  //could be more intuitive...
    utils.Info.Printf("initServiceClientSession: muxIndex=%d\n", muxIndex)
    dataConn := initServiceDataSession(muxServer[muxIndex], serviceIndex, backendChannel)
    for {
        select {
            case request := <- serviceDataChannel:
                frontendServiceDataComm(dataConn, request)
        }
    }
}

func makeServiceRegisterHandler(serviceRegChannel chan string, serviceIndex *int, backendChannel []chan string) func(http.ResponseWriter, *http.Request) {
    return func(w http.ResponseWriter, req *http.Request) {
    utils.Info.Printf("serviceRegisterServer():url=%s\n", req.URL.Path)
    if (req.URL.Path != "/service/reg") {
        http.Error(w, "404 url path not found.", 404)
    } else if (req.Method != "POST") {
        http.Error(w, "400 bad request method.", 400)
    } else {
        type Payload struct {
            Rootnode string
        }
        decoder := json.NewDecoder(req.Body)
        var payload Payload
        err := decoder.Decode(&payload)
        if err != nil {
            panic(err)
        }
        utils.Info.Printf("serviceRegisterServer(index=%d):received POST request=%s\n", *serviceIndex, payload.Rootnode)
        if (*serviceIndex < 2) {  // communicate: port no + root node to server hub, port no + url path to transport mgr, and start a client session
            serviceRegChannel <- strconv.Itoa(serviceDataPortNum + *serviceIndex)
            serviceRegChannel <- payload.Rootnode
            *serviceIndex += 1
	    w.Header().Set("Content-Type", "application/json")
            response := "{ \"Portnum\" : " + strconv.Itoa(serviceDataPortNum + *serviceIndex-1) + " , \"Urlpath\" : \"/service/data/" + strconv.Itoa(*serviceIndex-1) + "\"" + " }"
            
            utils.Info.Printf("serviceRegisterServer():POST response=%s\n", response)
            w.Write([]byte(response))
            go initServiceClientSession(serviceDataChan[*serviceIndex-1], *serviceIndex-1, backendChannel)
        } else {
            utils.Info.Printf("serviceRegisterServer():Max number of services already registered.\n")
        }
    }
    }
}

func initServiceRegisterServer(serviceRegChannel chan string, serviceIndex *int, backendChannel []chan string) {
    utils.Info.Printf("initServiceRegisterServer():localhost:8082/service/reg\n")
    serviceRegisterHandler := makeServiceRegisterHandler(serviceRegChannel, serviceIndex, backendChannel)
    muxServer[1].HandleFunc("/service/reg", serviceRegisterHandler)
    utils.Error.Fatal(http.ListenAndServe("localhost:8082", muxServer[1]))
}

func frontendWSDataSession(conn *websocket.Conn, transportDataChannel chan string, backendChannel chan string){
    defer conn.Close()  // ???
    for {
        _, msg, err := conn.ReadMessage()
        if err != nil {
            utils.Error.Printf("read error data WS protocol.", err)
            break
        }

        utils.Info.Printf("%s request: %s \n", conn.RemoteAddr(), string(msg))
        transportDataChannel <- string(msg) // send request to server hub
        response := <- transportDataChannel    // wait for response from server hub

        backendChannel <- response 
    }
}

func backendWSDataSession(conn *websocket.Conn, backendChannel chan string){
    defer conn.Close()
    for {
        message := <- backendChannel

        utils.Info.Printf("%s Transport mgr server: message= %s \n", conn.RemoteAddr(), message)
        err := conn.WriteMessage(websocket.TextMessage, []byte(message)); 
        if err != nil {
            utils.Error.Printf("write error data WS protocol.", err)
            break
        }
    }
}

func makeTransportDataHandler(transportDataChannel chan string, backendChannel chan string) func(http.ResponseWriter, *http.Request) {
    return func(w http.ResponseWriter, req *http.Request) {
        if  req.Header.Get("Upgrade") == "websocket" {
            utils.Info.Printf("we are upgrading to a websocket connection\n")
            upgrader.CheckOrigin = func(r *http.Request) bool { return true }
            conn, err := upgrader.Upgrade(w, req, nil)
            if err != nil {
                utils.Error.Printf("upgrade:", err)
                return
            }
            utils.Info.Printf("WS data session initiated.\n")
            go frontendWSDataSession(conn, transportDataChannel, backendChannel)
            go backendWSDataSession(conn, backendChannel)
        }else{
            http.Error(w, "400 protocol must be websocket.", 400)
        }
    }
}

/**
*  All transport data servers implement a WS server which communicates with a transport protocol manager.
**/
func initTransportDataServer(mgrIndex int, muxServer *http.ServeMux, transportDataChannel []chan string, backendChannel []chan string) {
    utils.Info.Printf("initTransportDataServer():mgrIndex=%d\n", mgrIndex)
    transportDataHandler := makeTransportDataHandler(transportDataChannel[mgrIndex], backendChannel[mgrIndex])
    muxServer.HandleFunc("/transport/data/" + strconv.Itoa(mgrIndex), transportDataHandler)
    utils.Error.Fatal(http.ListenAndServe("localhost:" + strconv.Itoa(transportDataPortNum+mgrIndex), muxServer))
}

func initTransportDataServers(transportDataChannel []chan string, backendChannel []chan string) {
    for key, _ := range supportedProtocols {
        go initTransportDataServer(key, muxServer[key+2], transportDataChannel, backendChannel)  //muxelements 0 and one assigned to reg servers
    }
}

func updateServiceRouting(portNo string, rootNode string) {
    utils.Info.Printf("updateServiceRouting(): portnum=%s, rootNode=%s\n", portNo, rootNode)
}

func initVssFile() bool{
	filePath := "vss_rel_1.0.cnative"
	cfilePath := C.CString(filePath)
	rootHandle = C.VSSReadTree(cfilePath)
	C.free(unsafe.Pointer(cfilePath))

	if (rootHandle == 0) {
//		utils.Error.Println("Tree file not found")
		return false
	}

	return true
}

func searchTree(path string, searchData *searchData_t) int {
    utils.Info.Printf("getMatches(): path=%s\n", path)
    if (len(path) > 0) {
        // call int VSSSearchNodes(char* searchPath, long rootNode, int maxFound, searchData_t* searchData, bool wildcardAllDepths);
        cpath := C.CString(path)
        var matches C.int = C.VSSSearchNodes(cpath, rootHandle, 150, (*C.struct_searchData_t)(unsafe.Pointer(searchData)), false)
        C.free(unsafe.Pointer(cpath))
        return int(matches)
    } else {
        return 0
    }
}

func getPathLen(path string) int {
    for i := 0 ; i < len(path) ; i++ {
        if (path[i] == 0x00) {   // the path buffer defined in searchData_t is initiated with all zeros
            return i
        }
    }
    return len(path)
}

func aggregateValue(oldValue string, newValue string) string {
    return oldValue + ", " + newValue  // Needs improvement
}

func aggregateResponse(iterator int, response string, aggregatedResponseMap *map[string]interface{}) {
    if (iterator == 0) {
        utils.ExtractPayload(response, aggregatedResponseMap)
    } else {
        var multipleResponseMap map[string]interface{}
        utils.ExtractPayload(response, &multipleResponseMap)
        switch multipleResponseMap["action"] {
        case "get":
            (*aggregatedResponseMap)["value"] = aggregateValue((*aggregatedResponseMap)["value"].(string), multipleResponseMap["value"].(string))
        default: // TODO check if error
        }
    }
}

func retrieveServiceResponse(requestMap map[string]interface{}, tDChanIndex int, sDChanIndex int) {
    searchData := [150]searchData_t {}  // vssparserutilities.h: #define MAXFOUNDNODES 150
    matches := searchTree(requestMap["path"].(string), &searchData[0])
    if (matches == 0) {
        errorResponseMap["MgrId"] = requestMap["MgrId"]
        errorResponseMap["ClientId"] = requestMap["ClientId"]
        errorResponseMap["action"] = requestMap["action"]
        errorResponseMap["requestId"] = requestMap["requestId"]
        transportDataChan[tDChanIndex] <- finalizeMessage(errorResponseMap)
    } else {
        if (matches == 1) {
            pathLen := getPathLen(string(searchData[0].responsePath[:]))
            requestMap["path"] = string(searchData[0].responsePath[:pathLen])
            serviceDataChan[sDChanIndex] <- finalizeMessage(requestMap)
            response := <- serviceDataChan[sDChanIndex]
            transportDataChan[tDChanIndex] <- response
        } else {
            var aggregatedResponseMap map[string]interface{}
            for i := 0; i < matches; i++ {
                pathLen := getPathLen(string(searchData[i].responsePath[:]))
                requestMap["path"] = string(searchData[0].responsePath[:pathLen])
                serviceDataChan[sDChanIndex] <- finalizeMessage(requestMap)
                response := <- serviceDataChan[sDChanIndex]
                aggregateResponse(i, response, &aggregatedResponseMap)
            }
            transportDataChan[tDChanIndex] <- finalizeMessage(aggregatedResponseMap)
        }
    }
}

func finalizeMessage(responseMap map[string]interface{}) string {
    response, err := json.Marshal(responseMap)
    if err != nil {
        utils.Error.Printf("Server core-finalizeMessage: JSON encode failed.\n")
        return "{\"error\":\"JSON marshal error\"}"   // what to do here?
    }
    return string(response)
}


func removeQuery(path string) string {
    pathEnd := strings.Index(path, "?$spec")
    if (pathEnd != -1) {
        return path[:pathEnd]
    }
    return path
}

// vssparserutilities.h: nodeTypes_t; 0-9 -> the data types, 10-16 -> the node types. Should be separated in the C code declarations...
func nodeTypesToString(nodeType int) string {
    switch (nodeType) {
      case 0:  return "int8"
      case 1: return "uint8"
      case 2: return "int16"
      case 3: return "uint16"
      case 4: return "int32"
      case 5:  return "uint32"
      case 6: return "double"
      case 7: return "float"
      case 8: return "boolean"
      case 9: return "string"
      case 10: return "sensor"
      case 11: return "actuator"
      case 12: return "stream"
      case 13: return "attribute"
      case 14:  return "branch"
      case 15: return "rbranch"
      case 16: return "element"
      default:
          return ""
    }
}

func jsonifyTreeNode(nodeHandle C.long, jsonBuffer string) string{
    var newJsonBuffer string
    nodeName := C.GoString(C.getName(nodeHandle))
    newJsonBuffer += `"` + nodeName + `":{`
    nodeType := int(C.getType(nodeHandle))
    newJsonBuffer += `"type":` + `"` + nodeTypesToString(nodeType) + `",`
    nodeDescr := C.GoString(C.getDescr(nodeHandle))
    newJsonBuffer += `"description":` + `"` + nodeDescr + `",`
    nodeNumofChildren := int(C.getNumOfChildren(nodeHandle))
    switch (nodeType) {
      case 14:  // branch
      case 15: // rbranch
      case 16: // element
      case 10: // sensor
          fallthrough
      case 11: // actuator
          fallthrough
      case 13: // attribute
          nodeDatatype := int(C.getDatatype(nodeHandle))
          newJsonBuffer += `"datatype:"` + `"` + nodeTypesToString(nodeDatatype) + `",`
      case 12: // stream
      default: // 0-9 -> the data types, should not occur here (needs to be separated in C code declarations...
    }
    if (nodeNumofChildren > 0) {
        newJsonBuffer += `"children":` + "{"
    }
    for i := 0 ; i < nodeNumofChildren ; i++ {
        childNode := C.long(C.getChild(nodeHandle, C.int(i)))
        newJsonBuffer += jsonifyTreeNode(childNode, jsonBuffer)
    }
    if (nodeNumofChildren > 0) {
        newJsonBuffer += "}"
    }
    newJsonBuffer += "}"
    return jsonBuffer + newJsonBuffer
}

func synthesizeJsonTree(path string) string {
    var jsonBuffer string
    searchData := [150]searchData_t {}  // vssparserutilities.h: #define MAXFOUNDNODES 150
    matches := searchTree(path, &searchData[0])
    if (matches == 0) {
        return ""
    }
    rootNode := C.long(searchData[0].foundNodeHandle)
    jsonBuffer = jsonifyTreeNode(rootNode, jsonBuffer)
    return "{" + jsonBuffer + "}"
}

func serveRequest(request string, tDChanIndex int, sDChanIndex int) {
    var requestMap = make(map[string]interface{})
    utils.ExtractPayload(request, &requestMap)
    switch requestMap["action"] {
        case "get":
            retrieveServiceResponse(requestMap, tDChanIndex, sDChanIndex)
        case "set":
            retrieveServiceResponse(requestMap, tDChanIndex, sDChanIndex)
        case "subscribe":
            retrieveServiceResponse(requestMap, tDChanIndex, sDChanIndex)
        case "unsubscribe":
            serviceDataChan[sDChanIndex] <- request
            response := <- serviceDataChan[sDChanIndex]
            transportDataChan[tDChanIndex] <- response
        case "getmetadata":
            path := removeQuery(requestMap["path"].(string))
            requestMap["metadata"] = synthesizeJsonTree(path)  //TODO handle error case
            delete(requestMap, "path")
            requestMap["timestamp"] = 1234
            transportDataChan[tDChanIndex] <- finalizeMessage(requestMap)
//        case "authorize":  //TODO
        default:
            utils.Warning.Printf("serveRequest():not implemented/unknown action=%s\n", requestMap["action"])
            errorResponseMap["MgrId"] = 0 //??
            errorResponseMap["ClientId"] = 0 //??
            transportDataChan[tDChanIndex] <- finalizeMessage(errorResponseMap)
    }
}

func updateTransportRoutingTable(mgrId int, portNum int) {
    utils.Info.Printf("Dummy updateTransportRoutingTable, mgrId=%d, portnum=%d\n", mgrId, portNum)
}

func main() {
    logFile := utils.InitLogFile("servercore-log.txt")
    utils.InitLog(logFile, logFile, logFile)
    defer logFile.Close()

    if !initVssFile(){
        utils.Error.Fatal(" Tree file not found")
        return
    }

    initTransportDataServers(transportDataChan, backendChan)
    utils.Info.Printf("main():initTransportDataServers() executed...\n")
    transportRegChan := make(chan int, 2*2)
    go initTransportRegisterServer(transportRegChan)
    utils.Info.Printf("main():initTransportRegisterServer() executed...\n")
    serviceRegChan := make(chan string, 2)
    serviceIndex := 0  // index assigned to registered services
    go initServiceRegisterServer(serviceRegChan, &serviceIndex, backendChan)
    utils.Info.Printf("main():starting loop for channel receptions...\n")
    for {
        select {
        case portNum := <- transportRegChan:  // save port no + transport mgr Id in routing table
            mgrId := <- transportRegChan
            updateTransportRoutingTable(mgrId, portNum)
        case request := <- transportDataChan[0]:  // request from transport0 (=HTTP), verify it, and route matches to servicemgr, or execute and respond if servicemgr not needed
            serveRequest(request, 0, 0)
        case request := <- transportDataChan[1]:  // request from transport1 (=WS), verify it, and route matches to servicemgr, or execute and respond if servicemgr not needed
            serveRequest(request, 1, 0)
//        case xxx := <- transportDataChan[2]:  // implement when there is a 3rd transport protocol mgr
        case portNo := <- serviceRegChan:  // save service data portnum and root node in routing table
            rootNode := <- serviceRegChan
            updateServiceRouting(portNo, rootNode)
//        case xxx := <- serviceDataChan[0]:    // for asynchronous routing, instead of the synchronous above. ToDo?
        default:
            time.Sleep(50*time.Millisecond)
        }
    }
}

