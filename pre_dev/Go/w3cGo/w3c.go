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

func printInComingMessage(r *http.Request){

	fmt.Printf("Method : %s \n", r.Method )
	fmt.Printf("Host : %s \n", r.Host )
	fmt.Printf("Proto : %s \n", r.Proto )
	fmt.Printf("Uri : %s \n", r.RequestURI)

	fmt.Printf("upgrade to : %s \n", r.Header.Get("Upgrade"))

}

func getSpeed() int {
	return rand.Intn(250)
}

func weSocketUpgradeHandler(w http.ResponseWriter, r *http.Request){
	conn, _ := upgrader.Upgrade(w, r, nil)
	for {
		// Read message from browser
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		// Print the message to the console
		fmt.Printf("%s sent: %s \n", conn.RemoteAddr(), string(msg))

		// Write message back to browser
		message := "speed is " + strconv.Itoa(getSpeed()) + "km/h"
		myMessage := []byte(message)

		if err = conn.WriteMessage(msgType, myMessage); err != nil {
			return
		}
	}
}

func main() {
	/*http.HandleFunc("/reply", func(w http.ResponseWriter, r *http.Request) {
		printInComingMessage(r)

		conn, _ := upgrader.Upgrade(w, r, nil) // error ignored for sake of simplicity

		for {
			// Read message from browser
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			// Print the message to the console
			fmt.Printf("%s sent: %s \n", conn.RemoteAddr(), string(msg))

			// Write message back to browser
			if err = conn.WriteMessage(msgType, msg); err != nil {
				return
			}
		}
	})*/


	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		printInComingMessage(r)
		//check here if we should upgrade our connection to a websocket...
		if  r.Header.Get("Upgrade") == "websocket" {
			fmt.Printf("we are upgrading to a websocket connection\n")
			weSocketUpgradeHandler(w,r)
		}else{
			http.ServeFile(w, r, "Websockets.html")
			fmt.Printf("websockets.html template served\n")
		}
	})

	http.HandleFunc("/Vehicle/Media/Artists/Abba/Songs/Waterloo",func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "My My At Waterloo Napoleon did surrender %s!", r.URL.Path[1:])
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}



