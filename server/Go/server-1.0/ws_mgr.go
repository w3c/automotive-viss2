/**
* (C) 2019 Geotab
* (C) 2019 Volvo Cars
*
* All files and artifacts in the repository at https://github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/
package main

import (
    "utils"
    "github.com/gorilla/websocket"
    "strconv"
    "strings"
)


var clientBackendChan = []chan string {
    make(chan string),
    make(chan string),
}

const isClientLocal = false


/**
* Websocket transport manager tasks:
*     - register with core server 
      - spawn a WS server for every connecting app client
      - forward data between app clients and core server, injecting mgr Id (and appClient Id?) into payloads
**/
func main() {
    transportErrorMessage = "WS transport mgr-finalizeResponse: JSON encode failed.\n"

    utils.InitLog("ws-mgr-log.txt")

    hostIP = utils.GetOutboundIP()
    registerAsTransportMgr(&regData)
    go  WsServer{clientBackendChan}.initClientServer(muxServer[0]) // go routine needed due to listenAndServe call...

    utils.Info.Printf("initClientServer() done\n")
    dataConn := initDataSession(muxServer[1], regData)
    go WsWSsession{clientBackendChan}.transportHubFrontendWSsession(dataConn, appClientChan) // receives messages from server core
    utils.Info.Printf("initDataSession() done\n")
    for {
        select {
        case reqMessage := <- appClientChan[0]:
            utils.Info.Printf("Transport server hub: Request from client 0:%s\n", reqMessage)
            // add mgrId + clientId=0 to message, forward to server core
            newPrefix := "{ \"MgrId\" : " + strconv.Itoa(regData.Mgrid) + " , \"ClientId\" : 0 , "
            request := strings.Replace(reqMessage, "{", newPrefix, 1)
            err := dataConn.WriteMessage(websocket.TextMessage, []byte(request)); 
            if (err != nil) {
                utils.Error.Printf("Datachannel write error:", err)
            }
        case reqMessage := <- appClientChan[1]:
            // add mgrId + clientId=1 to message, forward to server core
            newPrefix := "{ MgrId: " + strconv.Itoa(regData.Mgrid) + " , ClientId: 1 , "
            request := strings.Replace(reqMessage, "{", newPrefix, 1)
            err := dataConn.WriteMessage(websocket.TextMessage, []byte(request)); 
            if (err != nil) {
                utils.Error.Printf("Datachannel write error:", err)
            }
        }
    }
}

