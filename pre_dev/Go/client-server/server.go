package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"math/rand"
	"net/http"
	"strconv"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

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

func wsClientSession(conn *websocket.Conn){
        defer conn.Close()  // ???
	for {
		// Read message from browser
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
                        log.Print("read:", err)
			break
		}

		// Print the message to the console
		fmt.Printf("%s sent: %s \n", conn.RemoteAddr(), string(msg))

		// Write message back to browser
		message := "speed is " + strconv.Itoa(getSpeed()) + "km/h"
		myMessage := []byte(message)

		err = conn.WriteMessage(msgType, myMessage); 
                if err != nil {
                        log.Print("write:", err)
			break
		}
	}
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
		fmt.Printf("HTTP connection\n")
                w.Header().Set("Access-Control-Allow-Origin", "*")
                w.Write([]byte("Response:XXXX\n"))
		fmt.Printf("HTTP client request served.\n")
	}
}

func main() {
	http.HandleFunc("/", rootServer)  // register handler

	log.Fatal(http.ListenAndServe("localhost:8080", nil)) // start server
}



