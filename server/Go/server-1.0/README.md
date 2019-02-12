Project: W3C_VehicleSignalInterfaceImpl

Functionality: Server with capability to serve multiple clients over both WebSockets and HTTP protocols in parallel.

Implementation language: Go for server, JS for clients.


Build instructions:
Build server:
$ go install
Executable is stored in $GOPATH/bin
Store the file containing the tree, vss_rel_1.0.cnative, in $GOPATH/bin.
Run executable
$ ./w3cImpl_Go

Start a client by clicking on any of the HTML-files, which then opens in browser. 
Then for WS clients, write dot-notated search path in input field, and push Send button.
For HTTP clients, write slash-notated search path in input field, and push Send button 
The server will return the number of nodes matching the search path. 
See Wildcard chapter in Gen2_Core.html on https://github.com/w3c/automotive/tree/gh-pages/spec for rules on wildcard usage. 
The boolean parameter in the call to VSSSimpleSearch is currently hardcoded to returning only the subtree of the depth gien by the number of trailing wildcards. Changing it to true leads to that the complete subtree below is returned. 

Terminate client by closing browser tab.

Terminate server by Ctrl-C in terminal window.

