/**
* (C) 2019 Volvo Cars
*
* All files and artifacts in the repository at https://github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package main

import (
    "fmt"
    "flag"
    "github.com/gorilla/websocket"
    "log"
    "math/rand"
    "time"
    "net/http"
    "net/url"
    "strconv"
    "encoding/json"
    "strings"
)

// #include <stdlib.h>
// #include <stdio.h>
// #include <stdbool.h>
// #include "vssparserutilities.h"
import "C"

import "unsafe"
var nodeHandle C.long

type searchData_t struct {  // searchData_t defined in vssparserutilities.h
    responsePath [512]byte  // vssparserutilities.h: #define MAXCHARSPATH 512; typedef char path_t[MAXCHARSPATH];
    foundNodeHandles int64  // defined as long in vssparserutilities.h
}

var transportRegChan chan int
var transportRegPortNum int = 8081
var transportDataPortNum int = 8100  // port number interval [8100-]

// add element if support for new transport protocol is added
var transportDataChan = []chan string {
    make(chan string),
    make(chan string),
}

var supportedProtocols map[int]string

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

var actionList = []string {
    "get",
    "set",
    "subscribe",
    "unsubscribe",
    "getmetadata",
    "authorize",
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
    fmt.Printf("transportRegisterServer():url=%s\n", req.URL.Path)
    protocolIndex := -1
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
    fmt.Printf("transportRegisterServer():POST request=%s\n", payload.Protocol)
        for key, value := range supportedProtocols {
            if (payload.Protocol == value) {
                protocolIndex = key
            }
        }
        if (protocolIndex != -1) {  // communicate: port no + mgr Id to server hub, port no + url path + mgr Id to transport mgr
            select {
                case transportRegChannel <- transportDataPortNum + protocolIndex:  // port no
                default:
            }
            mgrId := rand.Intn(65535)  // [0 -65535], 16-bit value
            select {
                case transportRegChannel <- mgrId:  // mgr id
                default:
            }
	    w.Header().Set("Content-Type", "application/json")
            response := "{ \"Portnum\" : " + strconv.Itoa(transportDataPortNum + protocolIndex) + " , \"Urlpath\" : \"/transport/data/" + strconv.Itoa(protocolIndex) + "\"" + " , \"Mgrid\" : " + strconv.Itoa(mgrId) + " }"
            
            fmt.Printf("transportRegisterServer():POST response=%s\n", response)
            w.Write([]byte(response)) // correct JSON?
        } else {
            http.Error(w, "404 protocol not supported.", 404)
        }
    }
    }
}

func initTransportRegisterServer(transportRegChannel chan int) {
    fmt.Printf("initTransportRegisterServer():localhost:8081/transport/reg\n")
    transportRegisterHandler := maketransportRegisterHandler(transportRegChannel)
    muxServer[0].HandleFunc("/transport/reg", transportRegisterHandler)
    log.Fatal(http.ListenAndServe("localhost:8081", muxServer[0]))
}

func serviceDataRequestAndResponse(dataConn *websocket.Conn, request string) string {
    err := dataConn.WriteMessage(websocket.TextMessage, []byte(request)); 
    if (err != nil) {
        log.Print("Service datachannel write error:", err)
    }
    // receive response from server core, and forward to clientId=0 session
    _, message, err := dataConn.ReadMessage()
    if err != nil {
        log.Println("Service datachannel read error:", err)
        return "Service unavailable"
    }
    fmt.Printf("Server hub: Response from service:%s\n", string(message))
    return string(message)
}

/**
* initServiceDataSession:
* sets up the WS based communication (as client) with a service manager
**/
func initServiceDataSession(muxServer *http.ServeMux, serviceIndex int) (dataConn *websocket.Conn) {
    var addr = flag.String("addr", "localhost:" + strconv.Itoa(serviceDataPortNum+serviceIndex), "http service address")
    dataSessionUrl := url.URL{Scheme: "ws", Host: *addr, Path: "/service/data/"+strconv.Itoa(serviceIndex)}
    fmt.Printf("Connecting to:%s\n", dataSessionUrl.String())
    dataConn, _, err := websocket.DefaultDialer.Dial(dataSessionUrl.String(), http.Header{"Access-Control-Allow-Origin":{"*"}})
//    dataConn, _, err := websocket.DefaultDialer.Dial(dataSessionUrl.String(), nil)
//    defer dataConn.Close() //???
    if err != nil {
        log.Fatal("Service data session dial error:", err)
        return nil
    }
    return dataConn
}

func initServiceClientSession(serviceDataChan chan string, serviceIndex int) {
    time.Sleep(10*time.Second)  //wait for service data server to be initiated (initiate at first app-client request instead...)
    muxIndex := (len(muxServer) -2)/2 + 1 + (serviceIndex +1)  //could be more intuitive...
    fmt.Printf("initServiceClientSession: muxIndex=%d\n", muxIndex)
    dataConn := initServiceDataSession(muxServer[muxIndex], serviceIndex)
    for {
        select {
            case request := <- serviceDataChan:
                response := serviceDataRequestAndResponse(dataConn, request)
                serviceDataChan <- response
            default:
        }
    }
}

func makeServiceRegisterHandler(serviceRegChannel chan string, serviceIndex *int) func(http.ResponseWriter, *http.Request) {
    return func(w http.ResponseWriter, req *http.Request) {
    fmt.Printf("serviceRegisterServer():url=%s\n", req.URL.Path)
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
        fmt.Printf("serviceRegisterServer(index=%d):received POST request=%s\n", *serviceIndex, payload.Rootnode)
        if (*serviceIndex < 2) {  // communicate: port no + root node to server hub, port no + url path to transport mgr, and start a client session
            select {
                case serviceRegChannel <- strconv.Itoa(serviceDataPortNum + *serviceIndex):
                *serviceIndex += 1
                default:
            }
            select {
                case serviceRegChannel <- payload.Rootnode: 
                default:
            }
	    w.Header().Set("Content-Type", "application/json")
            response := "{ \"Portnum\" : " + strconv.Itoa(serviceDataPortNum + *serviceIndex-1) + " , \"Urlpath\" : \"/service/data/" + strconv.Itoa(*serviceIndex-1) + "\"" + " }"
            
            fmt.Printf("serviceRegisterServer():POST response=%s\n", response)
            w.Write([]byte(response))
            go initServiceClientSession(serviceDataChan[*serviceIndex-1], *serviceIndex-1)
        } else {
            fmt.Printf("serviceRegisterServer():Max number of services already registered.\n")
        }
    }
    }
}

func initServiceRegisterServer(serviceRegChannel chan string, serviceIndex *int) {
    fmt.Printf("initServiceRegisterServer():localhost:8082/service/reg\n")
    serviceRegisterHandler := makeServiceRegisterHandler(serviceRegChannel, serviceIndex)
    muxServer[1].HandleFunc("/service/reg", serviceRegisterHandler)
    log.Fatal(http.ListenAndServe("localhost:8082", muxServer[1]))
}

/*
* createProcolMap:
* To add support for one more transport manager protocol:
*    - add a map entry below
*    - add a komponent to the muxServer array
*    - add a component to the transportDataChan array
*    - add a select case in the main loop
*/
func createProcolMap() {
    supportedProtocols = make(map[int]string)
    supportedProtocols[0] = "HTTP"
    supportedProtocols[1] = "WebSocket"
}

func wsDataSession(conn *websocket.Conn, transportDataChannel chan string){
    defer conn.Close()  // ???
    for {
        msgType, msg, err := conn.ReadMessage()
        if err != nil {
            log.Print("read error data WS protocol.", err)
            break
        }

        fmt.Printf("%s request: %s \n", conn.RemoteAddr(), string(msg))
        transportDataChannel <- string(msg) // send request to server hub
        response := <- transportDataChannel    // wait for response from server hub

        fmt.Printf("%s Server core response: %s \n", conn.RemoteAddr(), string(response))
        err = conn.WriteMessage(msgType, []byte(response)); 
        if err != nil {
            log.Print("write error data WS protocol.", err)
            break
        }
    }
}

func makeTransportDataHandler(transportDataChannel chan string) func(http.ResponseWriter, *http.Request) {
    return func(w http.ResponseWriter, req *http.Request) {
        if  req.Header.Get("Upgrade") == "websocket" {
            fmt.Printf("we are upgrading to a websocket connection\n")
            upgrader.CheckOrigin = func(r *http.Request) bool { return true }
            conn, err := upgrader.Upgrade(w, req, nil)
            if err != nil {
                log.Print("upgrade:", err)
                return
            }
            fmt.Printf("WS data session initiated.\n")
            wsDataSession(conn, transportDataChannel)
        }else{
            http.Error(w, "400 protocol must be websocket.", 400)
        }
    }
}

/**
*  All transport data servers implement a WS server which communicates with a transport protocol manager.
**/
func initTransportDataServer(protocolIndex int, muxServer *http.ServeMux, transportDataChan []chan string) {
    fmt.Printf("initTransportDataServer():protocolIndex=%d\n", protocolIndex)
    transportDataHandler := makeTransportDataHandler(transportDataChan[protocolIndex])
    muxServer.HandleFunc("/transport/data/" + strconv.Itoa(protocolIndex), transportDataHandler)
    log.Fatal(http.ListenAndServe("localhost:" + strconv.Itoa(transportDataPortNum+protocolIndex), muxServer))
}

func initTransportDataServers() {
    for key, _ := range supportedProtocols {
        go initTransportDataServer(key, muxServer[key+2], transportDataChan)  //muxelements 0 and one assigned to reg servers
    }
}

func updateRoutingTable(portNum int, mgrId int) {
    fmt.Printf("updateRoutingTable():portnum=%d, mgrid=%d\n", portNum, mgrId)
}

func updateServiceRouting(portNo string, rootNode string) {
    fmt.Printf("updateServiceRouting(): portnum=%s, rootNode=%s\n", portNo, rootNode)
}

func initVssFile() bool{
	filePath := "vss_rel_1.0.cnative"
	cfilePath := C.CString(filePath)
	nodeHandle = C.VSSReadTree(cfilePath)
	C.free(unsafe.Pointer(cfilePath))

	if (nodeHandle == 0) {
//		log.Error("Tree file not found")
		return false
	}

//	nodeName := C.GoString(C.getName(nodeHandle))
//	log.Trace("Root node name = ", nodeName)

	return true
}

// the below impl is *not* robust
func extractPath(request string) string {
    pathValueStart := strings.Index(request, "\"path\":") // colon must follow directly after 'path'
    if (pathValueStart != -1) {
        pathValueStart += 7  // to point to first char after :
        pathValueEnd := strings.Index(request[pathValueStart:], "\",") // '",' must follow directly after the path value
        pathValueEnd += pathValueStart // point before '"'
        fmt.Printf("extractPath(): pathValueStart = %d, pathValueEnd = %d, path=%s\n", pathValueStart, pathValueEnd, request[pathValueStart+1:pathValueEnd])
        return request[pathValueStart+1:pathValueEnd]
    } else {
        return ""
    }
}

func searchTree(request string, searchData *searchData_t) int {
    path := extractPath(request)
    fmt.Printf("getMatches(): path=%s\n", path)
    if (len(path) > 0) {
        // call int VSSSearchNodes(char* searchPath, long rootNode, int maxFound, searchData_t* searchData, bool wildcardAllDepths);
        cpath := C.CString(path)
        var matches C.int = C.VSSSearchNodes(cpath, nodeHandle, 150, (*C.struct_searchData_t)(unsafe.Pointer(searchData)), false)
//        var matches C.int = C.VSSSimpleSearch(cpath, nodeHandle, false)
        C.free(unsafe.Pointer(cpath))
        return int(matches)
    } else {
        return 0
    }
}

func retrieveServiceResponse(request string, tDChanIndex int, sDChanIndex int) {
    searchData := [150]searchData_t {}  // vssparserutilities.h: #define MAXFOUNDNODES 150
    matches := searchTree(request, &searchData[0])
    fmt.Printf("retrieveServiceResponse():received request from transport manager %d:%s. No of matches=%d\n", tDChanIndex, request, matches)
if (matches > 0) {
    fmt.Printf("retrieveServiceResponse():Match 1:path:%s\n", string(searchData[0].responsePath[:]))
}
    if (matches == 0) {
        transportDataChan[tDChanIndex] <- "No match in tree for requested path."
    } else {
        var aggregatedResponse string
        for i := 0; i < matches; i++ {
            serviceDataChan[sDChanIndex] <- request // this should be preceeded with request verification, and routing analysis (resolve x -> serviceDataChan[x])
            response := <- serviceDataChan[sDChanIndex]
            aggregatedResponse += response + " , "  // not final solution...
        }
        transportDataChan[tDChanIndex] <- aggregatedResponse
    }
    if (matches > 0) {
        if (matches > 1) {
            // add matches to pendingServiceRequestList
        }
        serviceDataChan[sDChanIndex] <- request // this should be preceeded with request verification, and routing analysis (resolve x -> serviceDataChan[x])
        response := <- serviceDataChan[sDChanIndex]
        transportDataChan[tDChanIndex] <- response
    } else {
        transportDataChan[tDChanIndex] <- "No match in tree for requested path."
    }
}

func getPayloadAction(request string) string {
    for _, element := range actionList {
        if (strings.Contains(request, element) == true) {
            return element
        }
    }
    return ""
}

func serveRequest(request string, tDChanIndex int, sDChanIndex int) {
    switch getPayloadAction(request) {
        case actionList[0]: // get
            retrieveServiceResponse(request, tDChanIndex, sDChanIndex)
        case actionList[1]: // set
//        case actionList[2]: // subscribe
//        case actionList[3]: // unsubscribe
//        case actionList[4]: // getmetadata
//        case actionList[5]: // authorise
        default:
            fmt.Printf("serveRequest():not implemented/unknown action=%s\n", getPayloadAction(request))
            transportDataChan[tDChanIndex] <- "Unsupported request."
    }
}

func main() {
    if !initVssFile(){
        log.Fatal(" Tree file not found")
        return
    }

    createProcolMap()
    initTransportDataServers()
    fmt.Printf("main():initTransportDataServers() executed...\n")
    transportRegChan := make(chan int, 2*2)
    go initTransportRegisterServer(transportRegChan)
    fmt.Printf("main():initTransportRegisterServer() executed...\n")
    serviceRegChan := make(chan string, 2)
    serviceIndex := 0  // index assigned to registered services
    go initServiceRegisterServer(serviceRegChan, &serviceIndex)
    fmt.Printf("main():starting loop for channel receptions...\n")
    for {
        select {
        case portNum := <- transportRegChan:  // save port no + transport mgr Id in routing table
            mgrId := <- transportRegChan
            updateRoutingTable(portNum, mgrId)
        case request := <- transportDataChan[0]:  // request from transport0 (=HTTP), verify it, and route matches to servicemgr, or execute and respond if servicemgr not needed
            serveRequest(request, 0, 0)
        case request := <- transportDataChan[1]:  // request from transport1 (=WS), verify it, and route matches to servicemgr, or execute and respond if servicemgr not needed
            serveRequest(request, 1, 0)
//        case xxx := <- transportDataChan[2]:  // implement when there is a 3rd transport protocol mgr
        case portNo := <- serviceRegChan:  // save service data portnum and root node in routing table
            rootNode := <- serviceRegChan
            updateServiceRouting(portNo, rootNode)
//        case xxx := <- serviceDataChan[0]:    // for asynchronous routing, instead of the synchronous above. ToDo?
        default: // what to do here?
        }
    }
}

