/**
* (C) 2022 Geotab Inc
* (C) 2019 Volvo Cars
*
* All files and artifacts in the repository at https://github.com/w3c/automotive-viss2
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package utils

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var requestTag int

var trSecConfigPath string = "../transport_sec/"  // relative path to the directory containing the transportSec.json file
type SecConfig struct {
    TransportSec string  `json:"transportSec"`// "yes" or "no"
    HttpSecPort string   `json:"httpSecPort"`// HTTPS port number
    WsSecPort string     `json:"wsSecPort"`// WSS port number
    MqttSecPort string   `json:"mqttSecPort"`// MQTTS port number
    AgtsSecPort string   `json:"agtsSecPort"`// AGTS port number
    AtsSecPort string    `json:"atsSecPort"`// ATS port number
    CaSecPath string     `json:"caSecPath"`// relative path from the directory containing the transportSec.json file
    ServerSecPath string `json:"serverSecPath"`// relative path from the directory containing the transportSec.json file
    ServerCertOpt string `json:"serverCertOpt"`// one of  "NoClientCert"/"ClientCertNoVerification"/"ClientCertVerification"
    ClientSecPath string `json:"clientSecPath"`// relative path from the directory containing the transportSec.json file
}
var secConfig SecConfig

type Compression int

const (
	NONE Compression = 0
	PROPRIETARY      = 1
	PB_LEVEL1        = 2  // path has string format, e. g. "Vehicle.Acceleration.Longitudinal"
	PB_LEVEL2        = 3  // path is represented by integer index, retrieved from vsspathlist.json
)

var MuxServer = []*http.ServeMux{
	http.NewServeMux(), // for app client HTTP sessions
	http.NewServeMux(), // for app client WS sessions
	http.NewServeMux(), // for history control HTTP sessions
//	http.NewServeMux(), // for X transport sessions
}

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var HostIP string

/************ Client response handlers ********************************************************************************/
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

