/**
* (C) 2020 Geotab Inc
*
* All files and artifacts in the repository at https://github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package main

import (
	//    "fmt"
	"os"
	"os/exec"
	"flag"
	//    "net/http"
	"net/url"
	"github.com/gorilla/websocket"
	"strconv"
	"time"

	"github.com/akamensky/argparse"
	"github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/utils"
)

const portNum  = 8080 // WS mgr portnum
const urlPath  = ""
const subscribeCommand  = `{"action":"subscribe", "path":"Vehicle/Cabin/Door/Count", "filter":"$intervalEQ5", "requestId":"999"}`
const subscribePeriod = 5  // set to value X set in "$intervalEQX" above

var addr *string

func initSubscribeSession(addr *string) *websocket.Conn {
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
//                utils.Info.Println("Watchdog subscription received.Message=%s", response)
                triggerChannel <- string(response)
        }
}

func executeStopme() {
//    out, err := exec.Command("/bin/sh", "-c", "./W3CServer.sh", "stopme").CombinedOutput()
    out, err := exec.Command("./W3CServer.sh", "stopme").CombinedOutput()
    if err != nil {
	utils.Error.Printf("Script execute error:%s", string(out))
	utils.Error.Printf("Script execute error:%v", err)
	utils.Error.Println("Script execute error command: stopme")
    } else {
	utils.Info.Println("Script stopme executed successfully.")
        time.Sleep(5 * time.Second)
    }
}

func executeNoScriptStartme() {
    res := startServerCore()
    if (res == true) {
        time.Sleep(5 * time.Second)
        res := startServiceManager()
        if (res == true) {
            time.Sleep(2 * time.Second)
            res := startWSManager()
            if (res == true) {
                time.Sleep(2 * time.Second)
                res := startHTTPManager()
                if (res == true) {
                    time.Sleep(2 * time.Second)
                    res := startAGTServer()
                    if (res == true) {
                        time.Sleep(2 * time.Second)
                        res := startATServer()
                        if (res == true) {
                            utils.Info.Println("Script startme executed successfully.")
                            time.Sleep(10 * time.Second)
                        }
                    }
                }
            }
        }
    }
}

func startServerCore() bool {
    out, err := exec.Command("screen", "-d", "-m", "-t", "serverCore", "sh", "servercore.sh").CombinedOutput()
    if err != nil {
	utils.Error.Printf("Script execute error:%s", string(out))
	utils.Error.Printf("Script execute error:%v", err)
	utils.Error.Println("Script execute error command: screen -d -m -S serverCore")
    } else {
        time.Sleep(1 * time.Second)
        utils.Info.Println("ServerCore started successfully.")
        return true
    }
    return false
}

func startServiceManager() bool {
    out, err := exec.Command("screen", "-d", "-m", "-t", "serviceMgr", "sh", "servicemanager.sh").CombinedOutput()
    if err != nil {
	utils.Error.Printf("Script execute error:%s", string(out))
	utils.Error.Printf("Script execute error:%v", err)
	utils.Error.Println("Script execute error command: screen -d -m -S serviceMgr")
    } else {
        time.Sleep(1 * time.Second)
        utils.Info.Println("Service Manager started successfully.")
        return true
    }
    return false
}

func startWSManager() bool {
    out, err := exec.Command("screen", "-d", "-m", "-t", "wsMgr", "sh", "wsmanager.sh").CombinedOutput()
    if err != nil {
	utils.Error.Printf("Script execute error:%s", string(out))
	utils.Error.Printf("Script execute error:%v", err)
	utils.Error.Println("Script execute error command: screen -d -m -S wsMgr")
    } else {
        time.Sleep(1 * time.Second)
        utils.Info.Println("WebSocket Manager started successfully.")
        return true
    }
    return false
}

func startHTTPManager() bool {
    out, err := exec.Command("screen", "-d", "-m", "-t", "httpMgr", "sh", "httpmanager.sh").CombinedOutput()
    if err != nil {
	utils.Error.Printf("Script execute error:%s", string(out))
	utils.Error.Printf("Script execute error:%v", err)
	utils.Error.Println("Script execute error command: screen -d -m -S httpMgr")
    } else {
        time.Sleep(1 * time.Second)
        utils.Info.Println("HTTP Manager started successfully.")
        return true
    }
    return false
}

func startAGTServer() bool {
    out, err := exec.Command("screen", "-d", "-m", "-t", "agtServer", "sh", "agtserver.sh").CombinedOutput()
    if err != nil {
	utils.Error.Printf("Script execute error:%s", string(out))
	utils.Error.Printf("Script execute error:%v", err)
	utils.Error.Println("Script execute error command: screen -d -m -S agtServer")
    } else {
        time.Sleep(1 * time.Second)
        utils.Info.Println("AGT server started successfully.")
        return true
    }
    return false
}

func startATServer() bool {
    out, err := exec.Command("screen", "-d", "-m", "-t", "atServer", "sh", "atserver.sh").CombinedOutput()
    if err != nil {
	utils.Error.Printf("Script execute error:%s", string(out))
	utils.Error.Printf("Script execute error:%v", err)
	utils.Error.Println("Script execute error command: screen -d -m -S atServer")
    } else {
        time.Sleep(1 * time.Second)
        utils.Info.Println("AT server started successfully.")
        return true
    }
    return false
}

/**
* restartServer:
* 1. call stopme script. 2. call startme script 3. wait 30 secs 4. kill receiveSubscriptions() 
* 5. rerun initSubscribeSession 6. start receiveSubscriptions() 7. set wdTimer
**/
func restartServer(addr *string, triggerChannel chan string) (*websocket.Conn, *time.Timer) {
        executeStopme()
        executeNoScriptStartme()
        time.Sleep(10 * time.Second)
        // receiveSubscriptions() killed by its own timeout
  	dataConn := initSubscribeSession(addr)
	if dataConn != nil {
                go receiveSubscriptions(dataConn, triggerChannel)
	}
        wdTimer := time.NewTimer(2*subscribePeriod * time.Second)
        return dataConn, wdTimer
}

func main() {
	// Create new parser object
	parser := argparse.NewParser("print", "watchdog")
	// Create string flag
	logFile := parser.Flag("", "logfile", &argparse.Options{Required: false, Help: "outputs to logfile in ./logs folder"})
	logLevel := parser.Selector("", "loglevel", []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}, &argparse.Options{
		Required: false,
		Help:     "changes log output level",
		Default:  "info"})
	ipAddr := parser.String("", "ip", &argparse.Options{
		Required: true,
		Help:     "Ip adress "})

	// Parse input
	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
	}

	if len(*ipAddr) < 10 {
		utils.Error.Println("Program must be provided server IP address as input.")
		return
	}
	triggerChan := make(chan string)
	utils.InitLog("watchdog-log.txt", "./logs", *logFile, *logLevel)
	addr = flag.String("addr", *ipAddr+":"+strconv.Itoa(portNum), "http service address")
	dataConn := initSubscribeSession(addr)
	if dataConn == nil {
		return
	}
	go receiveSubscriptions(dataConn, triggerChan)
	wdTimer := time.NewTimer(2 * subscribePeriod * time.Second)
	for {
		select {
		case <-triggerChan:
			wdTimer.Stop()
			wdTimer = time.NewTimer(2 * subscribePeriod * time.Second)
		case <-wdTimer.C:
			//fmt.Println("Gen2 server crashed, restart initiated.")
			utils.Error.Println("Gen2 server crashed, restart initiated.")
			dataConn.Close()
			dataConn, wdTimer = restartServer(addr, triggerChan)
			if dataConn == nil {
				utils.Error.Println("Gen2 server failed to restart.")
				return
			}
		default:
			time.Sleep(1 * time.Second)
		}
	}
}
