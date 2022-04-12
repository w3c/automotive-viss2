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
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/w3c/automotive-viss2/utils"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

/*
* To add support for one more transport manager protocol:
*    - add a component to the transportDataChan array
*    - add a select case in the main loop

 */
/*
* Handler for the vsspathlist server
 */
func (pathList *PathList) vssPathListHandler(w http.ResponseWriter, r *http.Request) {
	bytes, err := json.Marshal(pathList)
	if err != nil {
		utils.Error.Printf("problems with json.Marshal, ", err)
		http.Error(w, "Unable to fetch vsspathlist", http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
	utils.Info.Printf("initVssPathListServer():Response=%s...(truncated to 100 bytes)", bytes[0:101])
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
	muxIndex := 2 + serviceIndex
	utils.Info.Printf("initServiceClientSession: muxIndex=%d", muxIndex)
	dataConn := initServiceDataSession(muxServer[muxIndex], serviceIndex, backendChannel, remoteIp)
	for {
		request := <-serviceDataChannel
		frontendServiceDataComm(dataConn, request)
	}
}
