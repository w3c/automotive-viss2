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
    "flag"
    "github.com/gorilla/websocket"
    "net/http"
    "net/url"
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

// the number of elements in muxServer and appClientChan arrays sets the max number of parallel app clients
var muxServer = []*http.ServeMux {
    http.NewServeMux(),  // for app client WS sessions on port number 8080
    http.NewServeMux(),  // for data session with core server on port number provided at registration
}

// the number of channel array elements sets the limit for max number of parallel app clients
var appClientChan = []chan string {
    make(chan string),
    make(chan string),
}

var clientBackendChan = []chan string {
    make(chan string),
    make(chan string),
}

type RegData struct {
    Portnum int
    Urlpath string
    Mgrid int
}

var regData RegData

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

/**
* registerAsTransportMgr:
* Registers with servercore as WebSocket protocol manager, and stores response in regData 
**/
func registerAsTransportMgr(regData *RegData) {
    url := "http://localhost:8081/transport/reg"

    data := []byte(`{"protocol": "WebSocket"}`)
//    data := []byte(`{"protocol": "HTTP"}`)  // use in HTTP manager

    req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
    if err != nil {
        log.Fatal("registerAsTransportMgr: Error reading request. ", err)
    }

    // Set headers
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Host", "localhost:8081")

    // Set client timeout
    client := &http.Client{Timeout: time.Second * 10}

    // Validate headers are attached
    fmt.Println(req.Header)

    // Send request
    resp, err := client.Do(req)
    if err != nil {
        log.Fatal("registerAsTransportMgr: Error reading response. ", err)
    }
    defer resp.Body.Close()

    fmt.Println("response Status:", resp.Status)
    fmt.Println("response Headers:", resp.Header)

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        log.Fatal("Error reading response. ", err)
    }
    fmt.Printf("%s\n", body)

    err = json.Unmarshal(body, regData)
    if (err != nil) {
        log.Fatal("Error JSON decoding of response. ", err)
    }
}

/**
* initDataSession:
* sets up the WS based communication (as client) with the core server
**/
func initDataSession(muxServer *http.ServeMux, regData RegData) (dataConn *websocket.Conn) {
    var addr = flag.String("addr", "localhost:" + strconv.Itoa(regData.Portnum), "http service address")
    dataSessionUrl := url.URL{Scheme: "ws", Host: *addr, Path: regData.Urlpath}
    dataConn, _, err := websocket.DefaultDialer.Dial(dataSessionUrl.String(), nil)
//    defer dataConn.Close() //???
    if err != nil {
        log.Fatal("Data session dial error:", err)
        return nil
    }
    return dataConn
}

func frontendWSAppSession(conn *websocket.Conn, clientChannel chan string, clientBackendChannel chan string){
    defer conn.Close()
    for {
        _, msg, err := conn.ReadMessage()
        if err != nil {
            log.Print("App client read error:", err)
            break
        }

        fmt.Printf("%s request: %s \n", conn.RemoteAddr(), string(msg))

        clientChannel <- string(msg) // forward to mgr hub, 
        message := <- clientChannel    //  and wait for response

        clientBackendChannel <- message 
    }
}

func backendWSAppSession(conn *websocket.Conn, clientBackendChannel chan string){
    defer conn.Close()
    for {
        message := <- clientBackendChannel  

        fmt.Printf("backendWSAppSession(): Message received=%s\n", message)
        // Write message back to app client
        response := []byte(message)

        err := conn.WriteMessage(websocket.TextMessage, response); 
        if err != nil {
           log.Print("App client write error:", err)
           break
        }
    }
}

func makeappClientHandler(appClientChannel []chan string, clientBackendChannel []chan string, serverIndex *int) func(http.ResponseWriter, *http.Request) {
    return func(w http.ResponseWriter, req *http.Request) {
        if  req.Header.Get("Upgrade") == "websocket" {
            fmt.Printf("we are upgrading to a websocket connection. Server index=%d\n", *serverIndex)
            upgrader.CheckOrigin = func(r *http.Request) bool { return true }
            conn, err := upgrader.Upgrade(w, req, nil)
            if err != nil {
                log.Print("upgrade:", err)
                return
           }
           if (*serverIndex < len(appClientChannel)) {
               go frontendWSAppSession(conn, appClientChannel[*serverIndex], clientBackendChannel[*serverIndex])
               go backendWSAppSession(conn, clientBackendChannel[*serverIndex])
               *serverIndex += 1
           } else {
               fmt.Printf("not possible to start more app client sessions.\n")
           }
        } else {
            fmt.Printf("Client must set up a Websocket session.\n")
        }
    }
}

func initClientServer(muxServer *http.ServeMux, clientBackendChannel []chan string) {
    serverIndex := 0
    appClientHandler := makeappClientHandler(appClientChan, clientBackendChannel, &serverIndex)
    muxServer.HandleFunc("/", appClientHandler)
    log.Fatal(http.ListenAndServe("localhost:8080", muxServer))
}

func getPayloadClientId(request string) int {
    return 0
}

func getPayloadAction(payload string) string {
    for _, element := range actionList {
        if (strings.Contains(payload, element) == true) {
            return element
        }
    }
    return ""
}

func transportHubFrontendWSsession(dataConn *websocket.Conn, appClientChannel []chan string, clientBackendChannel []chan string) {
    for {
        // receive message from server core, and forward to clientId transport session
        
        _, message, err := dataConn.ReadMessage()
        if err != nil {
            log.Println("Datachannel read error:", err)
            return
        }
        fmt.Printf("Server hub: Message from server core:%s\n", string(message))
        clientId := getPayloadClientId(string(message))
        if (getPayloadAction(string(message)) == actionList[4]) {
            clientBackendChannel[clientId] <- string(message)  //subscription notification
        } else {
            appClientChannel[clientId] <- string(message)
        }
    }
}

/**
* Websocket transport manager tasks:
*     - register with core server 
      - spawn a WS server for every connecting app client
      - forward data between app clients and core server, injecting mgr Id (and appClient Id?) into payloads
**/
func main() {
    registerAsTransportMgr(&regData)
    go initClientServer(muxServer[0], clientBackendChan)  // go routine needed due to listenAndServe call...
    fmt.Printf("initClientServer() done\n")
    dataConn := initDataSession(muxServer[1], regData)
    go transportHubFrontendWSsession(dataConn, appClientChan, clientBackendChan) // receives messages from server core
    fmt.Printf("initDataSession() done\n")
    for {
        select {
        case reqMessage := <- appClientChan[0]:
            fmt.Printf("Transport server hub: Request from client 0:%s\n", reqMessage)
            // add mgrId + clientId=0 to message, forward to server core
            newPrefix := "{ \"MgrId\" : " + strconv.Itoa(regData.Mgrid) + " , \"ClientId\" : 0 , "
            request := strings.Replace(reqMessage, "{", newPrefix, 1)
            err := dataConn.WriteMessage(websocket.TextMessage, []byte(request)); 
            if (err != nil) {
                log.Print("Datachannel write error:", err)
            }
/*            // receive response from server core, and forward to clientId=0 session
            _, message, err := dataConn.ReadMessage()
            if err != nil {
                log.Println("Datachannel read error:", err)
                return
            }
            fmt.Printf("Server hub: Response from server core:%s\n", string(message))
            appClientChan[0] <- string(message) */
        case reqMessage := <- appClientChan[1]:
            // add mgrId + clientId=1 to message, forward to server core
            newPrefix := "{ MgrId: " + strconv.Itoa(regData.Mgrid) + " , ClientId: 1 , "
            request := strings.Replace(reqMessage, "{", newPrefix, 1)
            err := dataConn.WriteMessage(websocket.TextMessage, []byte(request)); 
            if (err != nil) {
                log.Print("Datachannel write error:", err)
            }
/*            // receive response from server core, and forward to clientId=1 session
            _, message, err := dataConn.ReadMessage()
            if err != nil {
                log.Println("Datachannel read error:", err)
                return
            }
            fmt.Printf("Server hub: Response from server core:%s\n", string(message))
            appClientChan[1] <- string(message) */
        }
    }
}

