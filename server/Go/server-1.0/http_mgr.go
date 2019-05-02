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
 
// the number of elements in muxServer and appClientChan arrays sets the max number of parallel app clients
var muxServer = []*http.ServeMux {
    http.NewServeMux(),  // for app client HTTP sessions on port number 8888
    http.NewServeMux(),  // for data session with core server on port number provided at registration
}

// the number of channel array elements sets the limit for max number of parallel app clients
var appClientChan = []chan string {
    make(chan string),
    make(chan string),
}

type RegData struct {
    Portnum int
    Urlpath string
    Mgrid int
}

var regData RegData

var requestTag int  //common source for all requestIds

/**
* registerAsTransportMgr:
* Registers with servercore as WebSocket protocol manager, and stores response in regData 
**/
func registerAsTransportMgr(regData *RegData) {
    url := "http://localhost:8081/transport/reg"

//    data := []byte(`{"protocol": "WebSocket"}`)  // use in Websocket manager
    data := []byte(`{"protocol": "HTTP"}`)

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

func urlToPath(url string) string {
	var path string = strings.TrimPrefix(strings.Replace(url, "/", ".", -1), ".")
	return path[:]
}

func pathToUrl(path string) string {  // not needed?
	var url string = strings.Replace(path, ".", "/", -1)
	return "/" + url
}

// TODO: check for token in get/set requests. If found, issue authorize-request prior to get/set (the response on this "extra" request needs to be blocked...)
func frontendHttpAppSession(w http.ResponseWriter, req *http.Request, clientChannel chan string){
    path := urlToPath(req.RequestURI)
    fmt.Printf("HTTP method:%s, path: %s\n", req.Method, path)
    var requestMap = make(map[string]interface{})
    switch req.Method {
      case "GET":  // get/getmetadata
           if (strings.Contains(path, "$spec")) {
               requestMap["action"] = "getmetadata"
           } else {
               requestMap["action"] = "get"
           }
           requestMap["path"] = path
           requestMap["requestId"] = strconv.Itoa(requestTag)
           requestTag++
      case "POST":  // set
           requestMap["action"] = "set"
           requestMap["path"] = path
           body,_ := ioutil.ReadAll(req.Body)
           requestMap["value"] = string(body)
           requestMap["requestId"] = strconv.Itoa(requestTag)
           requestTag++
      default:
          http.Error(w, "400 Unsupported method", http.StatusBadRequest)
          fmt.Printf("Only GET and POST methods are supported.")
          return
    }
    clientChannel <- finalizeResponse(requestMap) // forward to mgr hub, 
    response := <- clientChannel    //  and wait for response

    backendHttpAppSession(response, &w)
}

func extractPayload(message string, rMap *map[string]interface{}) {
    decoder := json.NewDecoder(strings.NewReader(message))
    err := decoder.Decode(rMap)
    if err != nil {
        fmt.Printf("HTTP transport mgr-extractPayload: JSON decode failed for message:%s\n", message)
        return 
    }
}

func backendHttpAppSession(message string, w *http.ResponseWriter){
        fmt.Printf("backendWSAppSession(): Message received=%s\n", message)

        var responseMap = make(map[string]interface{})
        extractPayload(message, &responseMap)
        var response string
        if (responseMap["error"] != nil) {
            http.Error(*w, "400 Error", http.StatusBadRequest)  // TODO select error code from responseMap-error:number
            return
        }
        switch responseMap["action"] {
          case "get":
              response = responseMap["value"].(string)
          case "getmetadata":
              response = responseMap["metadata"].(string)
          case "set":
              response = "200 OK"  //??
          default:
              http.Error(*w, "500 Internal error", http.StatusInternalServerError)  // TODO select error code from responseMap-error:number
              return

        }
        resp := []byte(response)
        (*w).Header().Set("Content-Length", strconv.Itoa(len(resp)))
        written, err := (*w).Write(resp)
        if (err != nil) {
            fmt.Printf("HTTP manager error on response write.Written bytes=%d. Error=%s\n", written, err.Error())
        }
}

func makeappClientHandler(appClientChannel []chan string) func(http.ResponseWriter, *http.Request) {
    return func(w http.ResponseWriter, req *http.Request) {
        if  req.Header.Get("Upgrade") == "websocket" {
            http.Error(w, "400 Incorrect port number", http.StatusBadRequest)
            fmt.Printf("Client call to incorrect port number for websocket connection.\n")
            return
        }
        /*go*/ frontendHttpAppSession(w, req, appClientChannel[0])  // array not needed
    }
}

func initClientServer(muxServer *http.ServeMux) {
    appClientHandler := makeappClientHandler(appClientChan)
    muxServer.HandleFunc("/", appClientHandler)
    log.Fatal(http.ListenAndServe("localhost:8888", muxServer))
}

func finalizeResponse(responseMap map[string]interface{}) string {
    response, err := json.Marshal(responseMap)
    if err != nil {
        fmt.Printf("WS transport mgr-finalizeResponse: JSON encode failed.\n")
        return "JSON marshal error"   // what to do here?
    }
    return string(response)
}

func transportHubFrontendWSsession(dataConn *websocket.Conn, appClientChannel []chan string) {
    for {
        _, response, err := dataConn.ReadMessage()
        if err != nil {
            log.Println("Datachannel read error:", err)
            return  // ??
        }
        fmt.Printf("Server hub: Response from server core:%s\n", string(response))
        var responseMap = make(map[string]interface{})
        extractPayload(string(response), &responseMap)
        clientId := int(responseMap["ClientId"].(float64))
        delete(responseMap, "MgrId")
        delete(responseMap, "ClientId")
        appClientChannel[clientId] <- finalizeResponse(responseMap)  // no need for clientBackendChannel as subscription notifications not supported
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
    go initClientServer(muxServer[0])  // go routine needed due to listenAndServe call...
    fmt.Printf("initClientServer() done\n")
    dataConn := initDataSession(muxServer[1], regData)
    go transportHubFrontendWSsession(dataConn, appClientChan) // receives messages from server core
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
        case reqMessage := <- appClientChan[1]:
            // add mgrId + clientId=1 to message, forward to server core
            newPrefix := "{ MgrId: " + strconv.Itoa(regData.Mgrid) + " , ClientId: 1 , "
            request := strings.Replace(reqMessage, "{", newPrefix, 1)
            err := dataConn.WriteMessage(websocket.TextMessage, []byte(request)); 
            if (err != nil) {
                log.Print("Datachannel write error:", err)
            }
        }
    }
}

