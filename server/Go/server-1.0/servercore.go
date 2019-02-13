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
    "github.com/gorilla/websocket"
    "log"
    "math/rand"
    "net/http"
    "strconv"
    "encoding/json"
//    "strings"
)

// #cgo CFLAGS: -I/home/ubjorken/goDev/src/w3cImpl_Go
// #include <stdlib.h>
// #include <stdio.h>
// #include <stdbool.h>
// #include "vssparserutilities.h"
import "C"

//import "unsafe"

var transportRegChan chan int
var transportRegPortNum int = 8081
var transportDataPortNum int = 8100  // port number interval [8100-]

// add element if support for new transport protocol is added
var transportDataChan = []chan string {
    make(chan string),
    make(chan string),
}

// muxServer[0] is assigned to transport registration server, the following for transport data servers
var muxServer = []*http.ServeMux {
    http.NewServeMux(),
    http.NewServeMux(),
    http.NewServeMux(),
}

var serviceRegChan chan<- string
var serviceRegPortNum int = 8082
var serviceDataPortNum int = 8200  // port number interval [8200-8299]
var supportedProtocols map[int]string
var MAXPENDINGDATAREQUESTS = 150  // same limit as in search calls to tree parser

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
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
        go initTransportDataServer(key, muxServer[key+1], transportDataChan)
    }
}

func updateRoutingTable(portNum int, mgrId int) {
    fmt.Printf("updateRoutingTable():portnum=%d, mgrid=%d\n", portNum, mgrId)
}

func main() {
    createProcolMap()
    initTransportDataServers()
    fmt.Printf("main():initTransportDataServers() executed...\n")
    transportRegChan := make(chan int, 2*2)
    go initTransportRegisterServer(transportRegChan)
    fmt.Printf("main():initTransportRegisterServer() executed...\n")
//    serviceRegChan := make(chan<- string, 2)
//    go initServiceRegisterServer()
    fmt.Printf("main():starting loop for channel receptions...\n")
    for {
        select {
        case portNum := <- transportRegChan:  // save port no + transport mgr Id in routing table
            mgrId := <- transportRegChan
            updateRoutingTable(portNum, mgrId)
        case request := <- transportDataChan[0]:  // request from transport0, verify it, and route matches to servicemgr, or execute and respond if servicemgr not needed
            fmt.Printf("main():received request from tramsport manager 0:%s\n", request)
            transportDataChan[0] <- "dummy response" + request // should not be here but when response from serviceDataChan below is received, and contains mgr Id linked to protocol index 0
        case request := <- transportDataChan[1]:  // request from transport1, verify it, and route matches to servicemgr, or execute and respond if servicemgr not needed
            transportDataChan[1] <- "dummy response" + request
//        case xxx := <- transportDataChan[2]:  // implement when there is a transport2
//        case xxx := <- serviceRegChan:  // new service registered, add to tree, etc.
//        case xxx := <- serviceDataChan:    // response from service, route it to transportmgr
//        default: // what to do here?
//    fmt.Printf("? ")
        default:
        }
    }
}

