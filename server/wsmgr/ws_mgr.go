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

	utils "github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/utils"

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
func messageUpdateAndForward(reqMessage string, regData utils.RegData, dataConn *websocket.Conn, clientId int) {
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
	utils.TransportErrorMessage = "WS transport mgr-finalizeResponse: JSON encode failed.\n"
	utils.InitLog("ws-mgr-log.txt", "./logs")

	regData := utils.RegData{}

	utils.RegisterAsTransportMgr(&regData, "WebSocket")

	go utils.WsServer{ClientBackendChannel: clientBackendChan}.InitClientServer(utils.MuxServer[0], &serverIndex) // go routine needed due to listenAndServe call...

	utils.Info.Printf("initClientServer() done\n")
	dataConn := utils.InitDataSession(utils.MuxServer[1], regData)
	go utils.WsWSsession{clientBackendChan}.TransportHubFrontendWSsession(dataConn, utils.AppClientChan) // receives messages from server core
	utils.Info.Printf("initDataSession() done\n")

	for {
		select {
		case reqMessage := <-utils.AppClientChan[0]:
			messageUpdateAndForward(reqMessage, regData, dataConn, 0)
		case reqMessage := <-utils.AppClientChan[1]:
			messageUpdateAndForward(reqMessage, regData, dataConn, 1)
		case reqMessage := <-utils.AppClientChan[2]:
			messageUpdateAndForward(reqMessage, regData, dataConn, 2)
		case reqMessage := <-utils.AppClientChan[3]:
			messageUpdateAndForward(reqMessage, regData, dataConn, 3)
		case reqMessage := <-utils.AppClientChan[4]:
			messageUpdateAndForward(reqMessage, regData, dataConn, 4)
		case reqMessage := <-utils.AppClientChan[5]:
			messageUpdateAndForward(reqMessage, regData, dataConn, 5)
		case reqMessage := <-utils.AppClientChan[6]:
			messageUpdateAndForward(reqMessage, regData, dataConn, 6)
		case reqMessage := <-utils.AppClientChan[7]:
			messageUpdateAndForward(reqMessage, regData, dataConn, 7)
		case reqMessage := <-utils.AppClientChan[8]:
			messageUpdateAndForward(reqMessage, regData, dataConn, 8)
		case reqMessage := <-utils.AppClientChan[9]:
			messageUpdateAndForward(reqMessage, regData, dataConn, 9)
		case reqMessage := <-utils.AppClientChan[10]:
			messageUpdateAndForward(reqMessage, regData, dataConn, 10)
		case reqMessage := <-utils.AppClientChan[11]:
			messageUpdateAndForward(reqMessage, regData, dataConn, 11)
		case reqMessage := <-utils.AppClientChan[12]:
			messageUpdateAndForward(reqMessage, regData, dataConn, 12)
		case reqMessage := <-utils.AppClientChan[13]:
			messageUpdateAndForward(reqMessage, regData, dataConn, 13)
		case reqMessage := <-utils.AppClientChan[14]:
			messageUpdateAndForward(reqMessage, regData, dataConn, 14)
		case reqMessage := <-utils.AppClientChan[15]:
			messageUpdateAndForward(reqMessage, regData, dataConn, 15)
		case reqMessage := <-utils.AppClientChan[16]:
			messageUpdateAndForward(reqMessage, regData, dataConn, 16)
		case reqMessage := <-utils.AppClientChan[17]:
			messageUpdateAndForward(reqMessage, regData, dataConn, 17)
		case reqMessage := <-utils.AppClientChan[18]:
			messageUpdateAndForward(reqMessage, regData, dataConn, 18)
		case reqMessage := <-utils.AppClientChan[19]:
			messageUpdateAndForward(reqMessage, regData, dataConn, 19)
		}
	}
}
