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
    "github.com/gorilla/websocket"
    "strconv"
    "strings"
    "utils"
)
 


 //common source for all requestIds

// TODO: check for token in get/set requests. If found, issue authorize-request prior to get/set (the response on this "extra" request needs to be blocked...)



/**
* Websocket transport manager tasks:
*     - register with core server 
      - spawn a WS server for every connecting app client
      - forward data between app clients and core server, injecting mgr Id (and appClient Id?) into payloads
**/
func main() {
    transportErrorMessage = "HTTP transport mgr-finalizeResponse: JSON encode failed.\n"
    utils.InitLog("http-mgr-log.txt")

    hostIP = utils.GetOutboundIP()
    registerAsTransportMgr(&regData)

    go  HttpServer{}.initClientServer(muxServer[0])  // go routine needed due to listenAndServe call...
    dataConn := initDataSession(muxServer[1], regData)

    go HttpWSsession{}.transportHubFrontendWSsession(dataConn,appClientChan) // receives messages from server core
    utils.Info.Println("**** HTTP manager entering server loop... ****")
    loopIter := 0
    for {
        select {
        case reqMessage := <- appClientChan[0]:
            utils.Info.Println(" bloody var ", testvar)
            utils.Info.Printf("Transport server hub: Request from client 0:%s\n", reqMessage)
            // add mgrId + clientId=0 to message, forward to server core
            newPrefix := "{ \"MgrId\" : " + strconv.Itoa(regData.Mgrid) + " , \"ClientId\" : 0 , "
            request := strings.Replace(reqMessage, "{", newPrefix, 1)
            utils.Info.Println(" client channel 0 request ", request)
            err := dataConn.WriteMessage(websocket.TextMessage, []byte(request)); 
            if (err != nil) {
                utils.Warning.Println("Datachannel write error:" + err.Error())
            }
        case reqMessage := <- appClientChan[1]:
            // add mgrId + clientId=1 to message, forward to server core
            newPrefix := "{ MgrId: " + strconv.Itoa(regData.Mgrid) + " , ClientId: 1 , "
            request := strings.Replace(reqMessage, "{", newPrefix, 1)
            utils.Info.Println(" client channel 1 request ", request)
            err := dataConn.WriteMessage(websocket.TextMessage, []byte(request)); 
            if (err != nil) {
                utils.Warning.Println("Datachannel write error:" + err.Error())
            }
        }
        loopIter++
        if (loopIter%1000 == 0) {
            utils.TrimLogFile(utils.Logfile)
        }
    }
}

