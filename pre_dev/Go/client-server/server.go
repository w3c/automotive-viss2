package main

import (
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
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

func printInComingSession(r *http.Request) {
    log.WithFields(log.Fields{
		"Method": r.Method,
		"Host":r.Host,
		"Proto":r.Proto,
		"Uri":r.RequestURI,
	}).Info("http request fields")

    log.WithFields(log.Fields{
    	"Header-Get":r.Header.Get("Upgrade"),
	}).Trace(" upgraded to ")
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

func wsClientSession(conn *websocket.Conn) {
	defer conn.Close() // dispose when its all over
	for {
		// Read message from browser
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Error("read: ", err)
			break
		}

		var matches int = getMatches(string(msg))
		// log the message
		log.Trace(" request: ", conn.RemoteAddr(), " ", string(msg))

		// Write message back to browser
		message := "Response:Nr of matches = " + strconv.Itoa(matches)
		response := []byte(message)

		err = conn.WriteMessage(msgType, response);
		if err != nil {
			log.Error("write:", err)
			break
		}
	}
}

// removes initial slash, replaces following slashes with dot
func urlToPath(url string) string {
	var path string = strings.TrimPrefix(strings.Replace(url, "/", ".", -1), ".")
	return path
}

func rootServer(w http.ResponseWriter, r *http.Request) {

	printInComingSession(r)
	//check here if we should upgrade our connection to a websocket...
	if r.Header.Get("Upgrade") == "websocket" {
		log.Trace("we are upgrading to a websocket connection")
		upgrader.CheckOrigin = func(r *http.Request) bool { return true }
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error("upgrade:", err)
			return
		}
		go wsClientSession(conn)
		log.Trace("WS client session spawned.")
	} else {

		var path string = urlToPath(r.RequestURI)
		var matches int = getMatches(path)

		// build a JSON string of the response, makes it easier to test against.
		message := `{"response":` + strconv.Itoa(matches) + "}"
		response := []byte(message)

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write(response)
		log.Info("HTTP client request served for path = ", path)
	}
}

func initVssFile() bool{
	filePath := "vss_rel_1.0.cnative"
	cfilePath := C.CString(filePath)
	nodeHandle = C.VSSReadTree(cfilePath)
	C.free(unsafe.Pointer(cfilePath))

	if (nodeHandle == 0) {
		log.Error("Tree file not found")
		return false
	}

	nodeName := C.GoString(C.getName(nodeHandle))
	log.Trace("Root node name = ", nodeName)

	return true
}

func main() {

	initLogger()

	http.HandleFunc("/", rootServer) // register handler

	if !initVssFile(){
		log.Fatal(" Tree file not found")
		return
	}

	log.Fatal(http.ListenAndServe("localhost:8080", nil)) // start server
}
