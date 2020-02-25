/**
* (C) 2020 Geotab Inc
*
* All files and artifacts in the repository at https://github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package main

import (
    "fmt"
    "os"
    "os/exec"
    "flag"
//    "net/http"
    "net/url"
    "github.com/gorilla/websocket"
    "strconv"
    "time"

    "github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/utils"
)

const portNum  = 8080 // WS mgr portnum
const urlPath  = ""
const subscribeCommand  = `{"action":"subscribe", "path":"Vehicle/Cabin/Door/Count", "filter":"$intervalEQ5", "requestId":"999"}`
const subscribePeriod = 5  // set to value X set in "$intervalEQX" above

var addr *string

func initSubscribeSession(addr *string) (dataConn *websocket.Conn) {
	dataSessionUrl := url.URL{Scheme: "ws", Host: *addr, Path: urlPath}
	dataConn, _, err := websocket.DefaultDialer.Dial(dataSessionUrl.String(), nil)
	if err != nil {
		utils.Error.Fatal("Data session dial error:" + err.Error())
	} else{
	    err = dataConn.WriteMessage(websocket.TextMessage, []byte(subscribeCommand))
	    if err != nil {
		    utils.Warning.Println("Watchdog subscribe error:" + err.Error())
                    dataConn = nil
	    }
        }
	return dataConn
}

func receiveSubscriptions(dataConn *websocket.Conn, triggerChannel chan string) {
        for {
                dataConn.SetReadDeadline(time.Now().Add(2*subscribePeriod * time.Second))
		_, response, err := dataConn.ReadMessage()
		if err != nil {
			utils.Error.Println("Watchdog read error:", err)
			return
		}
                triggerChannel <- string(response)
        }
}

func executeScript(command string) {
    bashCmd :=  "./../W3CServer.sh " + command
    _, err := exec.Command("/bin/sh", "-c", bashCmd).Output()  // shell script located in W3CVehicleSignalImpl directory
    if err != nil {
	utils.Error.Println("Script execute error:", err)
	utils.Error.Println("Script execute error command:", command)
    }
    time.Sleep(10 * time.Second)
}

func executeStopme() {
//    bashCmd :=  "./../W3CServer.sh " + command
//    _, err := exec.Command("/bin/sh", "-c", "./../W3CServer.sh", "stopme").Output()  // shell script located in W3CVehicleSignalImpl directory
    out, err := exec.Command("./../W3CServer.sh", "stopme").CombinedOutput()  // shell script located in W3CVehicleSignalImpl directory
    if err != nil {
	utils.Error.Printf("Script execute error:%s", string(out))
	utils.Error.Printf("Script execute error:%v", err)
	utils.Error.Println("Script execute error command: stopme")
    } else {
	utils.Info.Println("Script stopme executed successfully.")
        time.Sleep(10 * time.Second)
    }
}

func executeStartme() {
//    bashCmd :=  "./../W3CServer.sh " + command
//    _, err := exec.Command("/bin/sh", "-c", "./../W3CServer.sh", "stopme").Output()  // shell script located in W3CVehicleSignalImpl directory
    out, err := exec.Command("./../W3CServer.sh", "startme").CombinedOutput()  // shell script located in W3CVehicleSignalImpl directory
    if err != nil {
	utils.Error.Printf("Script execute error:%s", string(out))
	utils.Error.Printf("Script execute error:%v", err)
	utils.Error.Println("Script execute error command: startme")
    } else {
	utils.Info.Println("Script startme executed successfully.")
        time.Sleep(10 * time.Second)
    }
}

func main () {
        ipAddr := os.Args[1]
        if (len(ipAddr) < 10) {
            utils.Error.Println("Program must be provided server IP address as input.")
            return
        }
	triggerChan := make(chan string)
	utils.InitLog("watchdog-log.txt", "./logs")
	addr = flag.String("addr", ipAddr+":"+strconv.Itoa(portNum), "http service address")
	dataConn := initSubscribeSession(addr)
	if dataConn == nil {
                return
	}
        go receiveSubscriptions(dataConn, triggerChan)
        wdTimer := time.NewTimer(2*subscribePeriod * time.Second)
        for {
            select {
            case <- triggerChan:
                wdTimer.Stop()
                wdTimer = time.NewTimer(2*subscribePeriod * time.Second)
            case <- wdTimer.C:
fmt.Println("Gen2 server crashed, restart initiated.")
                utils.Error.Println("Gen2 server crashed, restart initiated.")
                // 1. call stopme script. 2. call startme script 3. wait 30 secs 4. kill receiveSubscriptions() 
                // 5. rerun initSubscribeSession 6. start receiveSubscriptions() 7. set wdTimer
                executeStopme()
                executeStartme()
//                executeScript("stopme")
//                executeScript("startme")
                time.Sleep(30 * time.Second)
                // receiveSubscriptions() killed by its own timeout
                dataConn.Close()
fmt.Println("Restart initSubscribeSession().")
  	        dataConn = initSubscribeSession(addr)
	        if dataConn == nil {
                    return
	        }
fmt.Println("Restart receiveSubscriptions().")
                go receiveSubscriptions(dataConn, triggerChan)
fmt.Println("Restart timer.")
                wdTimer = time.NewTimer(2*subscribePeriod * time.Second)
fmt.Println("Gen2 server crashed, restart completed.")
            default: 
                time.Sleep(1 * time.Second)
            }
        }
}

