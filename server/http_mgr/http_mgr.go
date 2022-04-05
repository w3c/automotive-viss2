/**
* (C) 2019 Geotab
* (C) 2019 Volvo Cars
*
* All files and artifacts in the repository at https://github.com/w3c/automotive-viss2
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/w3c/automotive-viss2/utils"
	"github.com/akamensky/argparse"
	"github.com/gorilla/websocket"
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
	// Create new parser object
	parser := argparse.NewParser("print", "http manager")
	// Create string flag
	logFile := parser.Flag("", "logfile", &argparse.Options{Required: false, Help: "outputs to logfile in ./logs folder"})
	logLevel := parser.Selector("", "loglevel", []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}, &argparse.Options{
		Required: false,
		Help:     "changes log output level",
		Default:  "info"})

	// Parse input
	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
	}

	utils.TransportErrorMessage = "HTTP transport mgr-finalizeResponse: JSON encode failed.\n"
	utils.InitLog("http-mgr-log.txt", "./logs", *logFile, *logLevel)

	regData := utils.RegData{}
	utils.RegisterAsTransportMgr(&regData, "HTTP")

	utils.ReadTransportSecConfig()

	go utils.HttpServer{}.InitClientServer(utils.MuxServer[0]) // go routine needed due to listenAndServe call...
	dataConn := utils.InitDataSession(utils.MuxServer[1], regData)

	go utils.HttpWSsession{}.TransportHubFrontendWSsession(dataConn, utils.AppClientChan) // receives messages from server core
	utils.Info.Println("**** HTTP manager entering server loop... ****")
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
