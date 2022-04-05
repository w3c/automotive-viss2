/**
* (C) 2020 Mitsubishi Electrics Automotive
* (C) 2019 Geotab Inc
* (C) 2019 Volvo Cars
*
* All files and artifacts in the repository at https://github.com/w3c/automotive-viss2
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/w3c/automotive-viss2/utils"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

/*
* To add support for one more transport manager protocol:
*    - add a map entry to supportedProtocols
*    - add a component to the muxServer array
*    - add a component to the transportDataChan array
*    - add a select case in the main loop

 */
var supportedProtocols = map[transferProtocol]int{
	HTTP: 0,
	WS:   1,
	MQTT: 2,
}

/*
* enum for simplifying the lookup of the protocol during registration requests
 */
type transferProtocol string

const (
	HTTP transferProtocol = "HTTP"
	WS   transferProtocol = "WebSocket"
	MQTT transferProtocol = "MQTT"
)

/*
* Handler for the vsspathlist server
 */
func (pathList *PathList) vssPathListHandler(w http.ResponseWriter, r *http.Request) {
	bytes, err := json.Marshal(pathList)
	if err != nil {
		utils.Error.Printf("problems with json.Marshal, ", err)
		http.Error(w, "Unable to fetch vsspathlist", http.StatusInternalServerError)
	} else {
	}

	maxChars := len(bytes)
	if (maxChars > 99) {
		maxChars = 99
	}
	utils.Info.Printf("initVssPathListServer():Response=%s...(truncated to max 100 bytes)", bytes[0:maxChars])
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}

func transportRegisterHandler(w http.ResponseWriter, r *http.Request) {
	type Payload struct {
		Protocol string
	}
	var payload Payload
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&payload)
	if err != nil {
		panic(err)
	}

	protocol := transferProtocol(payload.Protocol)
	utils.Info.Printf("transportRegisterServer():POST protocol registration request=%s", protocol)

	switch protocol {
	case HTTP, WS, MQTT:
		mgrId := rand.Intn(65535) // [0 -65535], 16-bit value
		w.Header().Set("Content-Type", "application/json")

		response, err := json.Marshal(utils.TranspRegResponse{
			Portnum: 8081, //TODO magic number should be changed
			Urlpath: fmt.Sprintf("/transport/data/%s", protocol),
			Mgrid:   mgrId,
		})

		if err != nil {
			utils.Error.Println(err)
		}

		utils.Info.Printf("transportRegisterServer():POST response=%s", response)
		w.Write([]byte(response))
		routerTableAdd(mgrId, supportedProtocols[protocol])
	default:
		http.Error(w, "404 protocol not supported.", 404)
	}
}

func makeTransportDataHandler(transportDataChannel []chan string, backendChannel []chan string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		utils.Info.Printf("makeTransportDataHandler:: protocol: %s", vars["protocol"])

		upgrader.CheckOrigin = func(r *http.Request) bool {
			// TODO should be changed to below?
			//	origin := r.Header.Get("Origin")
			//	return origin == "http://127.0.0.1:8080"
			return true
		}
		utils.Info.Printf("we are upgrading to a websocket connection.")
		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			utils.Error.Print("upgrade:", err)
			return
		}
		utils.Info.Printf("WS data session initiated.")

		protocol := transferProtocol(vars["protocol"])
		switch protocol {
		case HTTP, MQTT, WS:
			go frontendWSDataSession(conn, transportDataChannel[supportedProtocols[protocol]])
			go backendWSDataSession(conn, backendChannel[supportedProtocols[protocol]])
		default:
			http.Error(w, vars["protocol"]+" is currently not a supported protocol.", 404)
		}
	}
}

func makeServiceRegisterHandler(serviceIndex *int, backendChannel []chan string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		var re = regexp.MustCompile(`^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}`)
		remoteIp := re.FindString(req.RemoteAddr)
		utils.Info.Printf("serviceRegisterServer():remoteIp=%s, path=%s", remoteIp, req.URL.Path)

		type Payload struct {
			Rootnode string
		}

		var payload Payload
		err := json.NewDecoder(req.Body).Decode(&payload)
		if err != nil {
			panic(err)
		}

		utils.Info.Printf("serviceRegisterServer(index=%d):received POST request=%s", *serviceIndex, payload.Rootnode)
		if *serviceIndex < len(serviceDataChan) {
			w.Header().Set("Content-Type", "application/json")

			response, err := json.Marshal(utils.SvcRegResponse{
				Portnum: serviceDataPortNum + *serviceIndex,
				Urlpath: "/service/data/" + strconv.Itoa(*serviceIndex),
			})

			if err != nil {
				utils.Error.Println(err)
			}

			utils.Info.Printf("serviceRegisterServer():POST response=%s", response)
			w.Write(response)
			go initServiceClientSession(serviceDataChan[*serviceIndex], *serviceIndex, backendChannel, remoteIp)
			*serviceIndex += 1
		} else {
			utils.Info.Printf("serviceRegisterServer():Max number of services already registered.")
		}
	}
}

/**
* initServiceDataSession:
* sets up the WS based communication (as client) with a service manager
**/
func initServiceDataSession(muxServer *http.ServeMux, serviceIndex int, backendChannel []chan string, remoteIp string) (dataConn *websocket.Conn) {
	var addr = flag.String("addr", remoteIp+":"+strconv.Itoa(serviceDataPortNum+serviceIndex), "http service address")
	dataSessionUrl := url.URL{Scheme: "ws", Host: *addr, Path: "/service/data/" + strconv.Itoa(serviceIndex)}
	utils.Info.Printf("Connecting to:%s", dataSessionUrl.String())
	dataConn, _, err := websocket.DefaultDialer.Dial(dataSessionUrl.String(), http.Header{"Access-Control-Allow-Origin": {"*"}})
	if err != nil {
		utils.Error.Fatal("Service data session dial error:", err)
		return nil
	}
	go backendServiceDataComm(dataConn, backendChannel, serviceIndex)
	return dataConn
}

func initServiceClientSession(serviceDataChannel chan string, serviceIndex int, backendChannel []chan string, remoteIp string) {
	time.Sleep(3 * time.Second) //wait for service data server to be initiated (initiate at first app-client request instead...)
	muxIndex := 2 + len(supportedProtocols) + serviceIndex
	utils.Info.Printf("initServiceClientSession: muxIndex=%d", muxIndex)
	dataConn := initServiceDataSession(muxServer[muxIndex], serviceIndex, backendChannel, remoteIp)
	for {
		request := <-serviceDataChannel
		frontendServiceDataComm(dataConn, request)
	}
}
