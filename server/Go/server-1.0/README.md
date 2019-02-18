# Project: W3C_VehicleSignalInterfaceImpl: server/server-1.0

Functionality: 
	Long term: Server implementation following the project SwA, with capability to serve multiple app-clients over both WebSockets and HTTP protocols in parallel.
	Short term limitations: Only the WebSocket protocol is supported, with max two parallel app-clients.

Implementation language: Go for server, JS for app-clients.


## Build instructions:
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

## Software implementation
Figures 1 and 2 shows the design of the core server and the Websocket transport manager, respectively. The design is based on the high level Sw Architecture description found in the README of the root directory.<br>
The core server is partitioned in the following logical components:<br>
- Core server hub - the manager, tying it all together,<br>
- The transport manager registration server, managing the registration of transport protocol managers over HTTP,<br>
- Transport data channel server, exist in multiple instances, one for each registered transport manager, managing the data communiction,<br>
- The service manager registration server, managing the registration of service managers over HTTP,<br>
- Service data channel client, exist in multiple instances, one for each registered service manager, managing the data communiction,<br>
- The tree manager, providing access to the tree, abstracting the actual format of the tree, and more complex operations such as tree search, tree initiation and termination.<br>
The core server hub, running in the main context, spawns the following Go routines:<br>
- The transport manager registration server<br>
- Transport data channel servers<br>
- The service manager registration server<br>
The Go routines communicate with the server hub using Go channels.<br>
The communication with the transport protocol and service managers is realized using the Websocket protocol.<br>
![Core server design](./pics/Core server SwA.jpg?raw=true)<br>
* Fig. 1 Core server design<br>
The Websocket transport protocol manager is partitioned in the following logical components:<br>
- Websocket manager hub, the manager, responsible for registration with the core server, spawning of Websocket servers for connecting app-clients, and routing of messages to/from app-clients, etc.,<br>
- Websocket server,  exist in multiple instances, one for each app-client that connects to it.<br>
![Transport manager design](./pics/WS manager SwA.jpg?raw=true)<br>
* Fig. 2 Websocket transport manager design<br>
The Websocket servers run in separate Go routines, and communicate with the manager hub via Go channels.<br>
The data communication with the core server uses the Websocket protocol.<br>
