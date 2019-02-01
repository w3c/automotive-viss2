package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"math/rand"
	"net/http"
	"strconv"
        "strings"
)

// #include <stdlib.h>
// #include <stdio.h>
// #include <stdbool.h>
// #include "vssparserutilities.h"
import "C"

import "unsafe"

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var nodeHandle C.long

func printInComingSession(r *http.Request){

	fmt.Printf("Method : %s \n", r.Method )
	fmt.Printf("Host : %s \n", r.Host )
	fmt.Printf("Proto : %s \n", r.Proto )
	fmt.Printf("Uri : %s \n", r.RequestURI)

	fmt.Printf("upgrade to : %s \n", r.Header.Get("Upgrade"))

}

func getSpeed() int {
	return rand.Intn(250)
}

func getMatches(path string) int {
// call int VSSSimpleSearch(char* searchPath, long rootNode, bool wildcardAllDepths);
        cpath := C.CString(path)
        var matches C.int = C.VSSSimpleSearch(cpath, nodeHandle, false)
        C.free(unsafe.Pointer(cpath))
	return int(matches)
}

func wsClientSession(conn *websocket.Conn){
        defer conn.Close()  // ???
	for {
		// Read message from browser
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
                        log.Print("read:", err)
			break
		}

                var matches int = getMatches(string(msg)) 
		// Print the message to the console
		fmt.Printf("%s request: %s \n", conn.RemoteAddr(), string(msg))

		// Write message back to browser
		message := "Response:Nr of matches= " + strconv.Itoa(matches)
		response := []byte(message)

		err = conn.WriteMessage(msgType, response); 
                if err != nil {
                        log.Print("write:", err)
			break
		}
	}
}

// removes initial slash, replaces following slashes with dot
func urlToPath( url string) string {
        var path string = strings.TrimPrefix(strings.Replace(url, "/", ".", -1), ".")
        return path
}

func rootServer(w http.ResponseWriter, r *http.Request) {

	printInComingSession(r)
	//check here if we should upgrade our connection to a websocket...
	if  r.Header.Get("Upgrade") == "websocket" {
		fmt.Printf("we are upgrading to a websocket connection\n")
                upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	        conn, err := upgrader.Upgrade(w, r, nil)
	        if err != nil {
                        log.Print("upgrade:", err)
		        return
	        }
                go wsClientSession(conn)
                fmt.Printf("WS client session spawned.\n")
	}else{
                var path string = urlToPath(r.RequestURI)
                var matches int = getMatches(path) 
		message := "Response:Nr of matches= " + strconv.Itoa(matches)
		response := []byte(message)
                w.Header().Set("Access-Control-Allow-Origin", "*")
                w.Write(response)
		fmt.Printf("HTTP client request served for path=%s\n", path)
	}
}

func main() {
	http.HandleFunc("/", rootServer)  // register handler

        // call long VSSReadTree(char* filePath); to read tree into memory
        filePath := "vss_rel_1.0.cnative"
        cfilePath := C.CString(filePath)
        nodeHandle = C.VSSReadTree(cfilePath)
        C.free(unsafe.Pointer(cfilePath))
        if (nodeHandle == 0) {
                fmt.Printf("Tree file not found.\n")
                return
        }
        nodeName := C.GoString(C.getName(nodeHandle))
        fmt.Printf("Root node name=%s\n", nodeName)

	log.Fatal(http.ListenAndServe("localhost:8080", nil)) // start server
}



