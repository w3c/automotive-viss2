---
title: "VISSv2 Access Control Servers"
---

The [VISSv2 access control model](https://raw.githack.com/w3c/automotive/gh-pages/spec/VISSv2_Core.html#access-control-model) specifies two authorization servers:
* Access Grant server
* Access Token server

### Access Grant server
This server is in a typical scenario running in the cloud. It is built as a separate executable in the WAII/server/agt_server directory

$ go build

and run by

$ ./agt_server

It exposes an HTTP API according to the VISSv2 specification. However it is currently not TLS protected (which is a must in non-development scenario).
What is also missing in the AGS implementation is authentication of the client, which according to the specification should be an AGT task.

### Access Token server

This server runs as a thread within the vissv2 server, so it is built by the vissv2 build command.
For it to be built, it is necessary to make sure that the "atServer" line in the serverComponents array in the vissv2server.go code is uncommented:
```
var serverComponents []string = []string{
	"serviceMgr",
	"httpMgr",
	"wsMgr",
	"mqttMgr",
	"grpcMgr",
	"atServer",
}
```
If it is part of the vissv2server build, and if a VSS node is access control tagged,
the server will then forward the access token received in the client request to the ATS for validation.

The ATS will as part of the validation also use the VISSv2 specified policy documents if they are found in the working directory.


### VISS web client submodule

This submodule implements a [VISSv2 web client](https://github.com/nicslabdev/viss-web-client/)
that exposes a UI that is considerably more sophisticated than what other clients on the WAII repo exposes,
and it is particularly helpful when it comes to the client interactions with access control involved.
Check out the README on both repos for more information.
