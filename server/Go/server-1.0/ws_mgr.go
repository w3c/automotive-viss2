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
    "bytes"
//    "fmt"
    "io/ioutil"
//    "log"
    "flag"
    "github.com/gorilla/websocket"
    "net/http"
    "net/url"
//    "net"
    "time"
    "encoding/json"
    "strconv"
    "strings"
    "server-1.0/utils"
)
 
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

const isClientLocal = false
var hostIP string


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
        utils.Error.Fatal("registerAsTransportMgr: Error reading request. ", err)
    }

    // Set headers
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Host", "localhost:8081")

    // Set client timeout
    client := &http.Client{Timeout: time.Second * 10}

    // Validate headers are attached
    utils.Info.Println(req.Header)

    // Send request
    resp, err := client.Do(req)
    if err != nil {
        utils.Error.Fatal("registerAsTransportMgr: Error reading response. ", err)
    }
    defer resp.Body.Close()

    utils.Info.Println("response Status:", resp.Status)
    utils.Info.Println("response Headers:", resp.Header)

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        utils.Error.Fatal("Error reading response. ", err)
    }
    utils.Info.Printf("%s\n", body)

    err = json.Unmarshal(body, regData)
    if (err != nil) {
        utils.Error.Fatal("Error JSON decoding of response. ", err)
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
        utils.Error.Fatal("Data session dial error:", err)
        return nil
    }
    return dataConn
}

func frontendWSAppSession(conn *websocket.Conn, clientChannel chan string, clientBackendChannel chan string){
    defer conn.Close()
    for {
        _, msg, err := conn.ReadMessage()
        if err != nil {
            utils.Error.Printf("App client read error:", err)
            break
        }

        payload := utils.UrlToPath(string(msg))  // if path in payload slash delimited, replace with dot delimited
        utils.Info.Printf("%s request: %s, len=%d\n", conn.RemoteAddr(), payload, len(payload))

        clientChannel <- payload // forward to mgr hub, 
        response := <- clientChannel    //  and wait for response

        clientBackendChannel <- response 
    }
}

func backendWSAppSession(conn *websocket.Conn, clientBackendChannel chan string){
    defer conn.Close()
    for {
        message := <- clientBackendChannel  

        utils.Info.Printf("backendWSAppSession(): Message received=%s\n", message)
        // Write message back to app client
        response := []byte(message)

        err := conn.WriteMessage(websocket.TextMessage, response); 
        if err != nil {
           utils.Error.Print("App client write error:", err)
           break
        }
    }
}

func makeappClientHandler(appClientChannel []chan string, clientBackendChannel []chan string, serverIndex *int) func(http.ResponseWriter, *http.Request) {
    return func(w http.ResponseWriter, req *http.Request) {
        if  req.Header.Get("Upgrade") == "websocket" {
            utils.Info.Printf("we are upgrading to a websocket connection. Server index=%d\n", *serverIndex)
            upgrader.CheckOrigin = func(r *http.Request) bool { return true }
            conn, err := upgrader.Upgrade(w, req, nil)
            if err != nil {
                utils.Error.Print("upgrade error:", err)
                return
           }
           if (*serverIndex < len(appClientChannel)) {
               go frontendWSAppSession(conn, appClientChannel[*serverIndex], clientBackendChannel[*serverIndex])
               go backendWSAppSession(conn, clientBackendChannel[*serverIndex])
               *serverIndex += 1
           } else {
               utils.Warning.Printf("not possible to start more app client sessions.\n")
           }
        } else {
            utils.Warning.Printf("Client must set up a Websocket session.\n")
        }
    }
}

func initClientServer(muxServer *http.ServeMux, clientBackendChannel []chan string) {
    serverIndex := 0
    appClientHandler := makeappClientHandler(appClientChan, clientBackendChannel, &serverIndex)
    muxServer.HandleFunc("/", appClientHandler)
    utils.Error.Fatal(http.ListenAndServe(hostIP + ":8080", muxServer))
}

func finalizeResponse(responseMap map[string]interface{}) string {
    response, err := json.Marshal(responseMap)
    if err != nil {
        utils.Error.Printf("WS transport mgr-finalizeResponse: JSON encode failed.\n")
        return "JSON marshal error"   // what to do here?
    }
    return string(response)
}

func transportHubFrontendWSsession(dataConn *websocket.Conn, appClientChannel []chan string, clientBackendChannel []chan string) {
    for {
        _, response, err := dataConn.ReadMessage()
        if err != nil {
            utils.Error.Println("Datachannel read error:", err)
            return  // ??
        }
        utils.Info.Printf("Server hub: Response from server core:%s\n", string(response))
        var responseMap = make(map[string]interface{})
        utils.ExtractPayload(string(response), &responseMap)
        clientId := int(responseMap["ClientId"].(float64))
        delete(responseMap, "MgrId")
        delete(responseMap, "ClientId")
        if (responseMap["action"] == "subscription") {
            clientBackendChannel[clientId] <- finalizeResponse(responseMap)  //subscription notification
        } else {
            appClientChannel[clientId] <- finalizeResponse(responseMap)
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
    logFile := utils.InitLogFile("ws-mgr-log.txt")
    utils.InitLog(logFile, logFile, logFile)
    defer logFile.Close()

    hostIP = utils.GetOutboundIP()
    registerAsTransportMgr(&regData)
    go initClientServer(muxServer[0], clientBackendChan)  // go routine needed due to listenAndServe call...
    utils.Info.Printf("initClientServer() done\n")
    dataConn := initDataSession(muxServer[1], regData)
    go transportHubFrontendWSsession(dataConn, appClientChan, clientBackendChan) // receives messages from server core
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

