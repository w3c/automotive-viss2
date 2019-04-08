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
    "fmt"
    "io/ioutil"
    "log"
    "github.com/gorilla/websocket"
    "net/http"
    "time"
    "encoding/json"
    "strconv"
    "strings"
)

var actionList = []string {
    "\"get",
    "\"set",
    "\"subscribe",
    "\"unsubscribe",
    "\"subscription",
    "\"getmetadata",
    "\"authorize",
}

var successResponse = []string {
    "{\"action\": \"get\", \"requestId\": \"AAA\", \"value\": 999, \"timestamp\": 1234}",
    "{\"action\": \"set\", \"requestId\": \"AAA\", \"timestamp\": 1234}",
    "{\"action\": \"subscribe\", \"requestId\": \"AAA\", \"subscriptionId\": \"BBB\", \"timestamp\": 1234}",
    "{\"action\": \"unsubscribe\", \"requestId\": \"AAA\", \"subscriptionId\": \"BBB\", \"timestamp\": 1234}",
}
 
var failureResponse = []string {
    "{\"action\": \"get\", \"requestId\": \"AAA\", \"error\": {\"number\":99, \"reason\": \"BBB\", \"message\": \"CCC\"}, \"timestamp\": 1234}",
    "{\"action\": \"set\", \"requestId\": \"AAA\", \"error\": {\"number\":99, \"reason\": \"BBB\", \"message\": \"CCC\"}, \"timestamp\": 1234}",
    "{\"action\": \"subscribe\", \"requestId\": \"AAA\", \"error\": {\"number\":99, \"reason\": \"BBB\", \"message\": \"CCC\"}, \"timestamp\": 1234}",
    "{\"action\": \"unsubscribe\", \"requestId\": \"AAA\", \"subscriptionId\": \"BBB\", \"error\": {\"number\":99, \"reason\": \"CCC\", \"message\": \"DDD\"}, \"timestamp\": 1234}",
    "{\"action\": \"unsupported\", \"requestId\": \"AAA\", \"error\": {\"number\":99, \"reason\": \"BBB\", \"message\": \"CCC\"}, \"timestamp\": 1234}",
}

var subscriptionNotification string = "{\"action\": \"subscription\", \"subscriptionId\": \"BBB\", \"value\": 999, \"timestamp\": 1234}"
 
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
        log.Fatal("registerAsServiceMgr: Error creating request. ", err)
    }

    // Set headers
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Host", "localhost:8082")

    // Set client timeout
    client := &http.Client{Timeout: time.Second * 10}

    // Validate headers are attached
    fmt.Println(req.Header)

    // Send request
    resp, err := client.Do(req)
    if err != nil {
        log.Fatal("registerAsServiceMgr: Error reading response. ", err)
    }
    defer resp.Body.Close()

    fmt.Println("response Status:", resp.Status)
    fmt.Println("response Headers:", resp.Header)

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        log.Fatal("Error reading response. ", err)
    }
    fmt.Printf("%s\n", body)

    err = json.Unmarshal(body, regResponse)
    if (err != nil) {
        log.Fatal("Service mgr: Error JSON decoding of response. ", err)
    }
    if (regResponse.Portnum <= 0) {
        fmt.Printf("Service registration denied.\n")
        return 0
    }
    return 1
}

func frontendWSdataSession(conn *websocket.Conn, clientChannel chan string, backendChannel chan string){
    defer conn.Close()
    for {
        _, msg, err := conn.ReadMessage()
        if err != nil {
            log.Print("Service data read error:", err)
            break
        }
        fmt.Printf("%s request: %s \n", conn.RemoteAddr(), string(msg))

        clientChannel <- string(msg) // forward to mgr hub, 
        message := <- clientChannel    //  and wait for response

        backendChannel <- message 
    }
}

func backendWSdataSession(conn *websocket.Conn, backendChannel chan string){
    defer conn.Close()
    for {
        message := <- backendChannel  

        fmt.Printf("Service:backendWSdataSession(): message received=%s\n", message)
        // Write message back to server core
        response := []byte(message)

        err := conn.WriteMessage(websocket.TextMessage, response)
        if err != nil {
           log.Print("Service data write error:", err)
           break
        }
    }
}

func makeServiceDataHandler(dataChannel chan string, backendChannel chan string) func(http.ResponseWriter, *http.Request) {
    return func(w http.ResponseWriter, req *http.Request) {
        if  req.Header.Get("Upgrade") == "websocket" {
            fmt.Printf("we are upgrading to a websocket connection.\n")
            upgrader.CheckOrigin = func(r *http.Request) bool { return true }
            conn, err := upgrader.Upgrade(w, req, nil)
            if err != nil {
                log.Print("upgrade:", err)
                return
           }
           go frontendWSdataSession(conn, dataChannel, backendChannel)
           go backendWSdataSession(conn, backendChannel)
        } else {
            fmt.Printf("Client must set up a Websocket session.\n")
        }
    }
}

func initDataServer(muxServer *http.ServeMux, dataChannel chan string, backendChannel chan string, regResponse RegResponse) {
    serviceDataHandler := makeServiceDataHandler(dataChannel, backendChannel)
    muxServer.HandleFunc(regResponse.Urlpath, serviceDataHandler)
    fmt.Printf("initDataServer: URL:%s, Portno:%d\n", regResponse.Urlpath, regResponse.Portnum)
    log.Fatal(http.ListenAndServe("localhost:"+strconv.Itoa(regResponse.Portnum), muxServer))
}

func getPayloadAction(request string) string {
    for _, element := range actionList {
        if (strings.Contains(request, element) == true) {
            return element
        }
    }
    return ""
}

func updateResponseValue(response string, value string) string {
    valueStart := strings.Index(response, "\"value\":") // colon must follow directly after 'value'
    if (valueStart == -1) {
        return response
    }
    valueStart += 8  // to point to first char after :
    valueEnd := strings.Index(response[valueStart:], "\",") // '",' must follow directly after the 'value'
    valueEnd += valueStart // point before '"'
    return response[:valueStart] + value + response[valueEnd+5:]  // 5->': 999'

}

var subscriptionTrigger time.Duration = 5000 // used for triggering subscription events every 5000 ms
var subscriptionTicker *time.Ticker

func activateSubscription(subscriptionChannel chan int) {
    subscriptionTicker = time.NewTicker(subscriptionTrigger*time.Millisecond)
    go func() {
        for range subscriptionTicker.C {
            subscriptionChannel <- 1
        }
    }()
}

func deactivateSubscription() {
    subscriptionTicker.Stop()
}

func checkSubscription(subscriptionChannel chan int, backendChannel chan string, subscReq string) {
    select {
        case <- subscriptionChannel:
            backendChannel <- prependResponse(subscReq, subscriptionNotification)
        default: // no subscription, so return
    }
}

// prepends response with'{"MgrID" : xxx , "ClientId" : x ,' copied from request
func prependResponse(request string, response string) string {
    cutIndex := strings.Index(request, "\"ClientId\" :") // one space between 'ClientId' and colon
    if (cutIndex != -1) {
        cutIndex += 17  // cutIndex points to C, move it to point after comma (x is one digit)
        return request[:cutIndex] + response[1:]
    }
    return "prependResponse() failed"
}

var dummyValue int  // used as return value in get

func main() {
    var regResponse RegResponse
    dataChan := make(chan string)
    backendChan := make(chan string)
    regRequest := RegRequest{Rootnode: "Vehicle"}
    subscriptionChan := make(chan int)

    if (registerAsServiceMgr(regRequest, &regResponse) == 0) {
        return
    }
    go initDataServer(muxServer[1], dataChan, backendChan, regResponse)
    fmt.Printf("initDataServer() done\n")
    var response, subscribeRequest string
    for {
        select {
        case request := <- dataChan:
            fmt.Printf("Service manager: Request from Server core 0:%s\n", request)
            // use template as response  TODO: 1. update template, 2. include error handling, 3. connect to a vehicle data source
            switch getPayloadAction(request) {
                case actionList[0]: // get
                    response = updateResponseValue(successResponse[0], strconv.Itoa(dummyValue))
                    dummyValue++
                case actionList[1]: // set
                    response = successResponse[1]
                case actionList[2]: // subscribe
                    activateSubscription(subscriptionChan)
                    response = successResponse[2]
                    subscribeRequest = request
                case actionList[3]: // unsubscribe
                    deactivateSubscription()
                    response = successResponse[3]
                default:
                    response = failureResponse[4] // improvement needed
            }
            fmt.Printf("Service mgr response:%s\n", response)
            dataChan <- prependResponse(request, response)
        default:
            checkSubscription(subscriptionChan, backendChan, subscribeRequest)
        } // select
    } // for
}

