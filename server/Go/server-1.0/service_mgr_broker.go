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
    //"log"
    "github.com/gorilla/websocket"
    "net/http"
    "time"
    "encoding/json"
    "strconv"
    "strings"
    log "github.com/sirupsen/logrus"
    "./server-core/util"
    "./server-core/signal_broker"
    //"strconv"
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
 
func removeQuery(path string) string {
    pathEnd := strings.Index(path, "?$spec")
    if (pathEnd != -1) {
        return path[:pathEnd]
    }
    return path
}
 
type SubscribePath struct {
    path string
    index int
}
 

var subscribePaths = []SubscribePath{{"Vehicle.Drivetrain.Transmission.Speed", 0}, {"Vehicle.x.y.z", 1}, {"Vehicle.a.b.c", 2}}
var inSubscription = false
 
var subscriptionId int
func activateSubscription(requestPath string, subscriptionChannel chan string) int {
    if (inSubscription == true) {
        return -1
    }
    
    path := removeQuery(requestPath)
    signalIndex := -1
    for i := 0 ; i < len(subscribePaths) ; i++ {
        if (subscribePaths[i].path == path) {
            signalIndex = subscribePaths[i].index
            inSubscription = true
        }
    }

    if (signalIndex != -1) {
        go func() {

            conn, response := signal_broker.GetResponseReceiver();
            defer conn.Close();

            for {
                if (inSubscription == false) {
                    break
                }
                //log.Info("WAIT FOR RESPONSE...")
                msg, err := response.Recv(); // wait for a subscription msg

                if (err != nil) {
                    log.Debug(" error ", err);
                    break;
                }
 
                values := msg.GetSignal();
                asig := values[signalIndex];

                var val = strconv.FormatInt(int64(asig.GetInteger()), 10)

                //log.Info(asig.Id.Namespace);
		        //log.Info(asig.Id.Name);
		        //log.Info( "val: ", val );
 
                subscriptionChannel <- val
            }
        }()
 
        subscriptionId++
        return subscriptionId-1
    }
    return -1
}
 
func deactivateSubscription() {
    inSubscription = false  // breaks after next receive from signal broker
}
 
func checkSubscription(subscriptionChannel chan string, backendChannel chan string, subscriptionMap map[string]interface{}) {
    select {
       case value := <- subscriptionChannel:
            //log.Debug("checkSubscription - we got new value!!!!");
            subscriptionMap["value"] = value
            backendChannel <- finalizeResponse(subscriptionMap, true)
        default: // no subscription, so return
    }
}
 
func extractPayload(request string, rMap *map[string]interface{}) {
    decoder := json.NewDecoder(strings.NewReader(request))
    err := decoder.Decode(rMap)
    if err != nil {
        fmt.Printf("Service manager-extractPayload: JSON decode failed for request:%s\n", request)
        return 
    }
}
 
func finalizeResponse(responseMap map[string]interface{}, responseStatus bool) string {
    if (responseStatus == false) {
    responseMap["error"] = "{\"number\":99, \"reason\": \"BBB\", \"message\": \"CCC\"}" // TODO
    }
    responseMap["timestamp"] = 1234
    response, err := json.Marshal(responseMap)
    if err != nil {
        fmt.Printf("Server core-finalizeResponse: JSON encode failed.\n")
        return ""
    }
    return string(response)
}
 
var dummyValue int  // used as return value in get
 
func main() {
    var regResponse RegResponse
    dataChan := make(chan string)
    backendChan := make(chan string)
    regRequest := RegRequest{Rootnode: "Vehicle"}
    subscriptionChan := make(chan string)
 
    util.InitLogger()
    if (registerAsServiceMgr(regRequest, &regResponse) == 0) {
        return
    }
    go initDataServer(muxServer[1], dataChan, backendChan, regResponse)
    fmt.Printf("initDataServer() done\n")
    var subscriptionMap = make(map[string]interface{})  // only one subscription is supported!

    for {
        select {
        case request := <- dataChan:  // request from server core
            fmt.Printf("Service manager: Request from Server core:%s\n", request)
            // use template as response  TODO: 1. update template, 2. include error handling, 3. connect to a vehicle data source
            var requestMap = make(map[string]interface{})
            var responseMap = make(map[string]interface{})
            extractPayload(request, &requestMap)
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
                    subscrId := activateSubscription(requestMap["path"].(string), subscriptionChan)
                    // if subscrId == -1, then send error response
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