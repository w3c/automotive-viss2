/**
* (C) 2023 Ford Motor Company
*
* All files and artifacts in the repository at https://github.com/w3c/automotive-viss2
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package main

import (
	"fmt"
	"time"
	"strings"
	"net/http"
	"github.com/gorilla/websocket"
	"encoding/json"
)

var muxServer = []*http.ServeMux{
	http.NewServeMux(),
}

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var statusIndex int
var replyStatus [3]string = [3]string{`“status”: “200-OK”`, `“status”: “401-Bad request”`, `“status”: “404-Not found”`}
var postponeTicker *time.Ticker
var postponedRequest string

func initEcfComm(receiveChan chan string, sendChan chan string, muxServer *http.ServeMux) {
	ecfHandler := makeEcfHandler(receiveChan, sendChan)
	muxServer.HandleFunc("/", ecfHandler)
	fmt.Print(http.ListenAndServe(":8445", muxServer))
}

func makeEcfHandler(receiveChan chan string, sendChan chan string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Upgrade") == "websocket" {
			fmt.Printf("Received websocket request: we are upgrading to a websocket connection.\n")
			Upgrader.CheckOrigin = func(r *http.Request) bool { return true }
			h := http.Header{}
			conn, err := Upgrader.Upgrade(w, req, h)
			if err != nil {
				fmt.Print("upgrade error:", err)
				return
			}
			go ecfReceiver(conn, receiveChan)
			go ecfSender(conn, sendChan)
		} else {
			fmt.Printf("Client must set up a Websocket session.\n")
		}
	}
}

func ecfReceiver(conn *websocket.Conn, receiveChan chan string) {
	defer conn.Close()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("ECF server read error: %s\n", err)
			break
		}
		request := string(msg)
//		fmt.Printf("ecfReceiver: request: %s\n", request)
		receiveChan <- request
	}
}

func ecfSender(conn *websocket.Conn, sendChan chan string) {
	defer conn.Close()
	for {
		response := <- sendChan
		err := conn.WriteMessage(websocket.TextMessage, []byte(response))
		if err != nil {
			fmt.Printf("ecfSender: write error: %s\n", err)
			break
		}
	}
}

func dispatchResponse(request string, sendChan chan string) {
	var response string
	var requestMap map[string]interface{}
	err := json.Unmarshal([]byte(request), &requestMap)
	if err != nil {
		fmt.Printf("dispatchResponse:Request unmarshal error=%s", err)
		response = `{"error":"request malformed"}`
	} else {
		response = `{"action":"` + requestMap["action"].(string) + `", "status":"` + replyStatus[statusIndex] + `"}`
	}
	sendChan <- response
}

func uiDialogue(request string) string {
	var actionNum string
	var newstatusIndex int
	fmt.Printf("\nCurrent response to all requests=%s\n", replyStatus[statusIndex])
	fmt.Printf("Change to 0:200-OK / 1:401-Bad request / 2:404-Not found / 4:Keep current response: ")
	fmt.Scanf("%d", &newstatusIndex)
	if newstatusIndex >= 0 && newstatusIndex <= 3 {
		statusIndex = newstatusIndex
	}
	fmt.Printf("\natServer request=%s\n", request)
	fmt.Printf("Select action: 0:Consent reply=YES / 1:Consent reply=NO / 2: Postpone consent reply: ")
	fmt.Scanf("%s", &actionNum)
	switch actionNum {
		case "0":
			return createReply(request, true)
		case "1":
			return createReply(request, false)
		case "2":
			var postponSecs int
			fmt.Printf("Time to postpone in seconds: ")
			fmt.Scanf("%d", &postponSecs)
			postponeTicker.Reset(time.Duration(postponSecs) * time.Second)
			postponedRequest = request
			return ""
		default:
			fmt.Printf("Invalid action.")
			return ""
	}
	return ""
}

func createReply(request string, consent bool) string {
	var reply string
	var requestMap map[string]interface{}
	yesNo := "NO"
	if consent {
		yesNo = "YES"
	}
	err := json.Unmarshal([]byte(request), &requestMap)
	if err != nil {
		fmt.Printf("createReply:Request unmarshal error=%s", err)
		reply = `{"error":"request malformed"}`
	} else {
		reply = `{"action":"consent-reply", "consent":"` + yesNo +  `", "messageId":"` + requestMap["messageId"].(string) + `"}`
	}
	return reply
}

func main() {
	receiveChan := make(chan string)
	sendChan := make(chan string)
	statusIndex = 0
	postponeTicker = time.NewTicker(24 * time.Hour)

	go initEcfComm(receiveChan, sendChan, muxServer[0])
	fmt.Printf("ECF simulator started. Waiting for request from Access Token server...")

	for {
		select {
		  case message := <-receiveChan:
			fmt.Printf("Message received=%s\n", message)
			if !strings.Contains(message, "status\":") {
			  	dispatchResponse(message, sendChan)
				response := uiDialogue(message)
				if response != "" {
					fmt.Printf("Response to atServer=%s\n", response)
					sendChan <- response
				}
			}
		case <-postponeTicker.C:
			fmt.Printf("postpone ticker triggered")
			response := uiDialogue(postponedRequest)
			if response != "" {
				fmt.Printf("Postponed response to atServer=%s\n", response)
				sendChan <- response
			}
		}
	}

}
