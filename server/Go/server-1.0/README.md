Project: W3C_VehicleSignalInterfaceImpl: server/server-1.0

Functionality: 
	Long term: Server implementation following the project SwA, with capability to serve multiple clients over both WebSockets and HTTP protocols in parallel.
	Short term limitations: Only the WebSocket protocol is supported, with max two parallel app clients.

Implementation language: Go for server, JS for clients.


Build instructions:
Build server core:
$ go run servercore.go

Build websocket protocol mgr:
$ go run ws_mgr.go

Start websocket app-client:
Click on wsclient.html (or wsclient2.html)

The order of starting the different programs should be:
1. servercore.go
2. ws_mgr.go
3. wsclient(2).html

After the startup sequence above, write any request with correct JSON syntax, e. g.:
{"path":"Vehicle.Cabin"}
{"xxx":123}
and a response starting with "dummy response" followed by the JSON formatted request in which '"Mgrid":xxxx, "ClientId":yyy' has been inserted before the initial request payload, will be returned. 
It is possible to start a second app-client (wsclient2.html) and send request from one or the other client. 
The Mgrid and Clientid are server internal routing data, and should be removed from the response before reching the app-client, but kept here for improved error checking.

Terminate client by closing browser tab.

Terminate core server, and websocket transport manager by Ctrl-C in respective terminal window.

