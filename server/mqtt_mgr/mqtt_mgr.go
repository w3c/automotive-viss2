/**
* (C) 2021 Geotab
*
* All files and artifacts in the repository at https://github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/
package main

import (
	"strconv"
	"strings"

	"github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/utils"
	"github.com/gorilla/websocket"
)

//common source for all requestIds

// TODO: check for token in get/set requests. If found, issue authorize-request prior to get/set (the response on this "extra" request needs to be blocked...)

/**
* Websocket transport manager tasks:
*     - register with core server
      - spawn a WS server for every connecting app client
      - forward data between MQTT broker and core server, injecting mgr Id (and appClient Id?) into payloads
**/
func main() {
	utils.TransportErrorMessage = "MQTT transport mgr-finalizeResponse: JSON encode failed.\n"
	utils.InitLog("mqtt-mgr-log.txt", "./logs")

	regData := utils.RegData{}
	utils.RegisterAsTransportMgr(&regData, "MQTT")

	go utils.HttpServer{}.InitClientServer(utils.MuxServer[3]) // go routine needed due to listenAndServe call...
	dataConn := utils.InitDataSession(utils.MuxServer[1], regData)

	go utils.HttpWSsession{}.TransportHubFrontendWSsession(dataConn, utils.AppClientChan) // receives messages from server core
	utils.Info.Println("**** MQTT manager entering server loop... ****")
	// loopIter := 0
	for {
		select {
		case reqMessage := <-utils.AppClientChan[0]:
			utils.Info.Printf("Transport server hub: Request from client 0:%s\n", reqMessage)
			// add mgrId + clientId=0 to message, forward to server core
			newPrefix := "{ \"RouterId\":\"" + strconv.Itoa(regData.Mgrid) + "?0\", "
			request := strings.Replace(reqMessage, "{", newPrefix, 1)
			//utils.Info.Println("HTTP mgr message to core server:" + request)
			err := dataConn.WriteMessage(websocket.TextMessage, []byte(request))
			if err != nil {
				utils.Warning.Println("Datachannel write error:" + err.Error())
			}
		}
	}
}
