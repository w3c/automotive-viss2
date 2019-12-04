/**
* (C) 2019 Geotab Inc
* (C) 2019 Volvo Cars
*
* All files and artifacts in the repository at https://github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/
package main

import (
	"strconv"
	"strings"

	mgr "github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/server/manager"
	"github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/utils"

	"github.com/gorilla/websocket"
)

// array size same as for manager.AppClientChan
var clientBackendChan = []chan string{
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
}

var serverIndex int

const isClientLocal = false

// add mgrId + clientId to message, forward to server core
func messageUpdateAndForward(reqMessage string, regData mgr.RegData, dataConn *websocket.Conn, clientId int) {
    utils.Info.Printf("Transport server hub: Request from client %d:%s\n", clientId, reqMessage)
    newPrefix := "{ \"MgrId\" : " + strconv.Itoa(regData.Mgrid) + " , \"ClientId\" : " + strconv.Itoa(clientId) + " , "
    request := strings.Replace(reqMessage, "{", newPrefix, 1)
//  utils.Info.Println("WS mgr message to core server:" + request)
    err := dataConn.WriteMessage(websocket.TextMessage, []byte(request))
    if err != nil {
        utils.Error.Printf("Datachannel write error: %s", err)
    }
}

/**
* Websocket transport manager tasks:
*     - register with core server
      - spawn a WS server for every connecting app client
      - forward data between app clients and core server, injecting mgr Id (and appClient Id?) into payloads
**/
func main() {
	mgr.TransportErrorMessage = "WS transport mgr-finalizeResponse: JSON encode failed.\n"

	utils.InitLog("ws-mgr-log.txt")
	regData := mgr.RegData{}

	mgr.HostIP = utils.GetOutboundIP()
	mgr.RegisterAsTransportMgr(&regData, "WebSocket")

	go mgr.WsServer{ClientBackendChannel: clientBackendChan}.InitClientServer(mgr.MuxServer[0], &serverIndex) // go routine needed due to listenAndServe call...

	utils.Info.Printf("initClientServer() done\n")
	dataConn := mgr.InitDataSession(mgr.MuxServer[1], regData)
	go mgr.WsWSsession{clientBackendChan}.TransportHubFrontendWSsession(dataConn, mgr.AppClientChan) // receives messages from server core
	utils.Info.Printf("initDataSession() done\n")

	for {
		select {
		case reqMessage := <-mgr.AppClientChan[0]:
                    messageUpdateAndForward(reqMessage, regData, dataConn, 0)
		case reqMessage := <-mgr.AppClientChan[1]:
                    messageUpdateAndForward(reqMessage, regData, dataConn, 1)
		case reqMessage := <-mgr.AppClientChan[2]:
                    messageUpdateAndForward(reqMessage, regData, dataConn, 2)
		case reqMessage := <-mgr.AppClientChan[3]:
                    messageUpdateAndForward(reqMessage, regData, dataConn, 3)
		case reqMessage := <-mgr.AppClientChan[4]:
                    messageUpdateAndForward(reqMessage, regData, dataConn, 4)
		case reqMessage := <-mgr.AppClientChan[5]:
                    messageUpdateAndForward(reqMessage, regData, dataConn, 5)
		case reqMessage := <-mgr.AppClientChan[6]:
                    messageUpdateAndForward(reqMessage, regData, dataConn, 6)
		case reqMessage := <-mgr.AppClientChan[7]:
                    messageUpdateAndForward(reqMessage, regData, dataConn, 7)
		case reqMessage := <-mgr.AppClientChan[8]:
                    messageUpdateAndForward(reqMessage, regData, dataConn, 8)
		case reqMessage := <-mgr.AppClientChan[9]:
                    messageUpdateAndForward(reqMessage, regData, dataConn, 9)
		case reqMessage := <-mgr.AppClientChan[10]:
                    messageUpdateAndForward(reqMessage, regData, dataConn, 10)
		case reqMessage := <-mgr.AppClientChan[11]:
                    messageUpdateAndForward(reqMessage, regData, dataConn, 11)
		case reqMessage := <-mgr.AppClientChan[12]:
                    messageUpdateAndForward(reqMessage, regData, dataConn, 12)
		case reqMessage := <-mgr.AppClientChan[13]:
                    messageUpdateAndForward(reqMessage, regData, dataConn, 13)
		case reqMessage := <-mgr.AppClientChan[14]:
                    messageUpdateAndForward(reqMessage, regData, dataConn, 14)
		case reqMessage := <-mgr.AppClientChan[15]:
                    messageUpdateAndForward(reqMessage, regData, dataConn, 15)
		case reqMessage := <-mgr.AppClientChan[16]:
                    messageUpdateAndForward(reqMessage, regData, dataConn, 16)
		case reqMessage := <-mgr.AppClientChan[17]:
                    messageUpdateAndForward(reqMessage, regData, dataConn, 17)
		case reqMessage := <-mgr.AppClientChan[18]:
                    messageUpdateAndForward(reqMessage, regData, dataConn, 18)
		case reqMessage := <-mgr.AppClientChan[19]:
                    messageUpdateAndForward(reqMessage, regData, dataConn, 19)
		}
	}
}
