/**
* (C) 2019 Geotab Inc
* (C) 2019 Volvo Cars
*
* All files and artifacts in the repository at https://github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package manager

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var requestTag int

var MuxServer = []*http.ServeMux{
	http.NewServeMux(), // for app client HTTP sessions on port number 8888
	http.NewServeMux(), // for data session with core server on port number provided at registration
}

// the number of channel array elements sets the limit for max number of parallel WS app clients
var AppClientChan = []chan string{
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
	make(chan string),
}

type RegData struct {
	Portnum int
	Urlpath string
	Mgrid   int
}

var TransportErrorMessage string

//var RegisterData RegData

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var HostIP string

/********************************************************************** Client response handlers **********************/
type ClientHandler interface {
	makeappClientHandler(appClientChannel []chan string) func(http.ResponseWriter, *http.Request)
}

type HttpChannel struct {
}

type WsChannel struct {
	clientBackendChannel []chan string
	serverIndex          *int
}

/**********Client server initialization *******************************************************************************/

type ClientServer interface {
	InitClientServer(muxServer *http.ServeMux)
}

type HttpServer struct {
}
type WsServer struct {
	ClientBackendChannel []chan string
}

/***********Server Core Communications ********************************************************************************/
type TransportHubFrontendWSSession interface {
	transportHubFrontendWSsession(dataConn *websocket.Conn, appClientChannel []chan string)
}

type HttpWSsession struct {
}

type WsWSsession struct {
	ClientBackendChannel []chan string
}
