// +build integration

/**
Integration tests , tests parts of the system which includes one or more functions.
Ex: Websocket upgrade from http.
*****/

package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// VSS tree matches Vehicle.* should be 25
const vehicle_star_vss_tree_response = 25

type response_j struct {
	Response int `json:"Response"`
}

// Testing GET request handler
func TestHttpHandling(t *testing.T){

	//testing HTTP request connection
	req, err := http.NewRequest("GET", "http://localhost:8080", nil)
    req.RequestURI = "/Vehicle/*"

	if err != nil {
		t.Errorf(" request connection creation failed ")
	}

	// Setup a response recorder to record the response
	rr := httptest.NewRecorder()
	// we will test the rootServer
	if !initVssFile(){
		t.Errorf("initialization of Vss file failed, expected %t got %t", true,false)
	}
	handler := http.HandlerFunc(rootServer)
	// handle the request
	handler.ServeHTTP(rr,req)
	status := rr.Code;

	if status != http.StatusOK{
		t.Errorf("handler returned wrong status code: %v want %v", status, http.StatusOK)
	}

	// parse JSON response
	resp := rr.Result()
	body,_ := ioutil.ReadAll(resp.Body)
	res := response_j{}
    json.Unmarshal([]byte(string(body)), &res)

    if res.Response != vehicle_star_vss_tree_response{
    	t.Errorf(" URI path /Vehicle/* expects %d matches got %d",vehicle_star_vss_tree_response,res.Response)
	}

}

// var upgrader = websocket.Upgrader{}


//Testing upgrading to websocket protocol from HTTP.
func TestSocketHandling(t *testing.T){
	//setup for testing socket connection
	server := httptest.NewServer(http.HandlerFunc(rootServer))
	defer server.Close()

	uri := "ws" + strings.TrimPrefix(server.URL,"http")

	// connect to server, test upgrage
	_, _, err := websocket.DefaultDialer.Dial(uri,nil)
    if err != nil {
    	t.Fatalf("%v", err)
	}
	// defer ws.Close()

    // maybe add response test later...
	/*for i := 0; i < 10; i++ {
		if err := ws.WriteMessage(websocket.TextMessage, []byte("hello")); err != nil {
			t.Fatalf("%v", err)
		}
		_, p, err := ws.ReadMessage()
		if err != nil {
			t.Fatalf("%v", err)
		}
		if string(p) != "hello" {
			t.Fatalf("bad message")
		}
	}*/
}

