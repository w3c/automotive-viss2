Project: W3C_VehicleSignalInterfaceImpl

Functionality: Server with capability to serve multiple clients over both WebSockets and HTTP protocols in parallel.

Implementation language: Go for server, JS for clients.


Build instructions:
Start server:
$ go run server.go

Terminate server by Ctrl-C.

Start a client by clicking on any of the HTML-files, which then opens in browser. 
Then for WS clients, write payload in input field, and push Send button.
For HTTP clients, push Send button.

Terminate client by closing browser tab.


