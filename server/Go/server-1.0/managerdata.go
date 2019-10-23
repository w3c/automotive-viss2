package main

import (
	"github.com/gorilla/websocket"
	"net/http"
)

var requestTag int

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

var transportErrorMessage string

var regData RegData

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var hostIP string


/********************************************************************** Client response handlers **********************/
type ClientHandler interface{
	makeappClientHandler(appClientChannel []chan string)func(http.ResponseWriter, *http.Request)

}


type HttpChannel struct{
}

type WsChannel struct{
	clientBackendChannel []chan string
	serverIndex *int
}


/**********Client server initialization *******************************************************************************/

type ClientServer interface{
	initClientServer(muxServer *http.ServeMux)
}

type HttpServer struct{

}
type WsServer struct{
	clientBackendChannel []chan string
}


/***********Server Core Communications ********************************************************************************/
type TransportHubFrontendWSSession interface{
	transportHubFrontendWSsession(dataConn *websocket.Conn, appClientChannel []chan string)
}

type HttpWSsession struct{

}

type WsWSsession struct{
	clientBackendChannel []chan string
}

