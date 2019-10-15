/**
* (C) 2019 Volvo Cars
*
* All files and artifacts in the repository at https://github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package main

import (
    "bytes"
//    "fmt"
    "io/ioutil"
//    "log"
    "github.com/gorilla/websocket"
    "net/http"
    "time"
    "encoding/json"
    "strconv"
//    "strings"
    "server-1.0/utils"
)

// one muxServer component for service registration, one for the data communication
var muxServer = []*http.ServeMux {
    http.NewServeMux(),  // 0 = for registration
    http.NewServeMux(),  // 1 = for data session
}


type RegRequest struct {
    Rootnode string
}

type RegResponse struct {
    Portnum int
    Urlpath string
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func registerAsServiceMgr(regRequest RegRequest, regResponse *RegResponse) int {

    url := "http://localhost:8082/service/reg"

    data := []byte(`{"Rootnode": "` + regRequest.Rootnode + `"}`)

    req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
    if err != nil {
        utils.Error.Fatal("registerAsServiceMgr: Error creating request. ", err)
    }

    // Set headers
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Host", "localhost:8082")

    // Set client timeout
    client := &http.Client{Timeout: time.Second * 10}

    // Validate headers are attached
    utils.Info.Println(req.Header)

    // Send request
    resp, err := client.Do(req)
    if err != nil {
        utils.Error.Fatal("registerAsServiceMgr: Error reading response. ", err)
    }
    defer resp.Body.Close()

    utils.Info.Println("response Status:", resp.Status)
    utils.Info.Println("response Headers:", resp.Header)

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        utils.Error.Fatal("Error reading response. ", err)
    }
    utils.Info.Printf("%s\n", body)

    err = json.Unmarshal(body, regResponse)
    if (err != nil) {
        utils.Error.Fatal("Service mgr: Error JSON decoding of response. ", err)
    }
    if (regResponse.Portnum <= 0) {
        utils.Warning.Printf("Service registration denied.\n")
        return 0
    }
    return 1
}

func frontendWSdataSession(conn *websocket.Conn, clientChannel chan string, backendChannel chan string){
    defer conn.Close()
    for {
        _, msg, err := conn.ReadMessage()
        if err != nil {
            utils.Error.Printf("Service data read error:", err)
            break
        }
        utils.Info.Printf("%s request: %s \n", conn.RemoteAddr(), string(msg))

        clientChannel <- string(msg) // forward to mgr hub, 
        message := <- clientChannel    //  and wait for response

        backendChannel <- message 
    }
}

func backendWSdataSession(conn *websocket.Conn, backendChannel chan string){
    defer conn.Close()
    for {
        message := <- backendChannel  

        utils.Info.Printf("Service:backendWSdataSession(): message received=%s\n", message)
        // Write message back to server core
        response := []byte(message)

        err := conn.WriteMessage(websocket.TextMessage, response)
        if err != nil {
           utils.Error.Printf("Service data write error:", err)
           break
        }
    }
}

func makeServiceDataHandler(dataChannel chan string, backendChannel chan string) func(http.ResponseWriter, *http.Request) {
    return func(w http.ResponseWriter, req *http.Request) {
        if  req.Header.Get("Upgrade") == "websocket" {
            utils.Info.Printf("we are upgrading to a websocket connection.\n")
            upgrader.CheckOrigin = func(r *http.Request) bool { return true }
            conn, err := upgrader.Upgrade(w, req, nil)
            if err != nil {
                utils.Error.Printf("upgrade:", err)
                return
           }
           go frontendWSdataSession(conn, dataChannel, backendChannel)
           go backendWSdataSession(conn, backendChannel)
        } else {
            utils.Warning.Printf("Client must set up a Websocket session.\n")
        }
    }
}

func initDataServer(muxServer *http.ServeMux, dataChannel chan string, backendChannel chan string, regResponse RegResponse) {
    serviceDataHandler := makeServiceDataHandler(dataChannel, backendChannel)
    muxServer.HandleFunc(regResponse.Urlpath, serviceDataHandler)
    utils.Info.Printf("initDataServer: URL:%s, Portno:%d\n", regResponse.Urlpath, regResponse.Portnum)
    utils.Error.Fatal(http.ListenAndServe("localhost:"+strconv.Itoa(regResponse.Portnum), muxServer))
}

var subscriptionTrigger time.Duration = 5000 // used for triggering subscription events every 5000 ms
var subscriptionTicker *time.Ticker

var subscriptionId int
func activateSubscription(subscriptionChannel chan int) int {
    subscriptionTicker = time.NewTicker(subscriptionTrigger*time.Millisecond)
    go func() {
        for range subscriptionTicker.C {
            subscriptionChannel <- 1
        }
    }()
    subscriptionId++
    return subscriptionId-1
}

func deactivateSubscription() {
    subscriptionTicker.Stop()
}

func checkSubscription(subscriptionChannel chan int, backendChannel chan string, subscriptionMap map[string]interface{}) {
    select {
        case <- subscriptionChannel:
            backendChannel <- finalizeResponse(subscriptionMap, true)
        default: // no subscription, so return
    }
}

func finalizeResponse(responseMap map[string]interface{}, responseStatus bool) string {
    if (responseStatus == false) {
    responseMap["error"] = "{\"number\":99, \"reason\": \"BBB\", \"message\": \"CCC\"}" // TODO
    }
    responseMap["timestamp"] = 1234
    response, err := json.Marshal(responseMap)
    if err != nil {
        utils.Error.Printf("Server core-finalizeResponse: JSON encode failed.\n")
        return ""
    }
    return string(response)
}

var dummyValue int  // used as return value in get

func main() {
    logFile := utils.InitLogFile("service-mgr-log.txt")
    utils.InitLog(logFile, logFile, logFile)
    defer logFile.Close()

    var regResponse RegResponse
    dataChan := make(chan string)
    backendChan := make(chan string)
    regRequest := RegRequest{Rootnode: "Vehicle"}
    subscriptionChan := make(chan int)

    if (registerAsServiceMgr(regRequest, &regResponse) == 0) {
        return
    }
    go initDataServer(muxServer[1], dataChan, backendChan, regResponse)
    utils.Info.Printf("initDataServer() done\n")
    var subscriptionMap = make(map[string]interface{})  // only one subscription is supported!
    for {
        select {
        case request := <- dataChan:  // request from server core
            utils.Info.Printf("Service manager: Request from Server core:%s\n", request)
            // use template as response  TODO: 1. update template, 2. include error handling, 3. connect to a vehicle data source
            var requestMap = make(map[string]interface{})
            var responseMap = make(map[string]interface{})
            utils.ExtractPayload(request, &requestMap)
            responseMap["MgrId"] = requestMap["MgrId"]
            responseMap["ClientId"] = requestMap["ClientId"]
            responseMap["action"] = requestMap["action"]
            responseMap["requestId"] = requestMap["requestId"]
            var responseStatus bool
            switch requestMap["action"] {
                case "get":
                    responseMap["value"] = strconv.Itoa(dummyValue)
                    dummyValue++
                    responseStatus = true
                case "set":
                    // interact with underlying subsystem to set the value
                    responseStatus = true
                case "subscribe":
                    subscrId := activateSubscription(subscriptionChan)
                    for k, v := range responseMap {
                        subscriptionMap[k] = v
                    }
                    subscriptionMap["action"] = "subscription"
                    subscriptionMap["subscriptionId"] = strconv.Itoa(subscrId)
                    responseMap["subscriptionId"] = strconv.Itoa(subscrId)
                    responseStatus = true
                case "unsubscribe":
                    deactivateSubscription()
                    responseStatus = true
                default:
                    responseStatus = false
            }
            dataChan <- finalizeResponse(responseMap, responseStatus)
        default:
            checkSubscription(subscriptionChan, backendChan, subscriptionMap)
            time.Sleep(50*time.Millisecond)
        } // select
    } // for
}

