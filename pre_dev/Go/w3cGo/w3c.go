package main

import "C"
import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// #cgo CFLAGS:
// #include <stdlib.h>
// #include "test.h"
import "C"

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func printInComingMessage(r *http.Request) {

	fmt.Printf("Method : %s \n", r.Method)
	fmt.Printf("Host : %s \n", r.Host)
	fmt.Printf("Proto : %s \n", r.Proto)
	fmt.Printf("Uri : %s \n", r.RequestURI)

	fmt.Printf("upgrade to : %s \n", r.Header.Get("Upgrade"))

}

func getSpeed() int {
	return rand.Intn(250)
}

func weSocketUpgradeHandler(w http.ResponseWriter, r *http.Request) {
	conn, _ := upgrader.Upgrade(w, r, nil)
	for {
		// Read message from browser
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		// Print the message to the console
		fmt.Printf("%s sent: %s \n", conn.RemoteAddr(), string(msg))

		// run speed collection as separate thread sleeping 500 ms in between each.
		go func() {
			for {
				speed := rand.Intn(250)
				message := strconv.Itoa(speed)
				myMessage := []byte(message)
				conn.WriteMessage(msgType, myMessage)
				time.Sleep(time.Millisecond * 500)
			}

		}()
		// Write message back to browser

		/*if err = conn.WriteMessage(msgType, myMessage); err != nil {
			return
		}*/
	}
}

func main() {

	i := C._function_f(10)
	fmt.Println(i)

	r := mux.NewRouter()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		printInComingMessage(r)
		//check here if we should upgrade our connection to a websocket...
		if r.Header.Get("Upgrade") == "websocket" {
			fmt.Printf("we are upgrading to a websocket connection\n")
			weSocketUpgradeHandler(w, r)
		} else {
			http.ServeFile(w, r, "Websockets.html")
			fmt.Printf("websockets.html template served\n")
		}
	})

	r.HandleFunc("/Vehicle/Media/Artists/{Art_ID}/Songs/{Song_name}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "My My At Waterloo Napoleon did surrender %s!", r.URL.Path[1:])
	})

	r.HandleFunc("/Vehicle/{testID}", func(w http.ResponseWriter, r *http.Request) {

		params := mux.Vars(r)
		str := params["testID"]
		fmt.Printf("param is : %s \n", str)

		fmt.Fprintf(w, "did surrender %s!", r.URL.Path[1:])
	})

	log.Fatal(http.ListenAndServe(":8080", r))
}
