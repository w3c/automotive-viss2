# Project: W3C_VehicleSignalInterfaceImpl: client/client-1.0

Client implementations for communication with the server found at the server/server-1.0 directory.<br>
The functionality in the Gen2 specs shall be supported, with features such as:<br>
 HTTP and WebSockets transport protocols<br>
-- Queries<br>
-- Access restriction <br><br>
The clients can be executed either locally on the same machine as the server, or remotely on a different machine.<br>
The project plans to deploy a publicly accessible server, allowing interested testers to skip building the GO based server. <br>
The client examples in this directory could then just be opened in any browser.<br><br>
Current limitations: <br>
- The following client limitations are due to server limitations. <br>
- The transport protocols are not the secure versions of Websockets and HTTP. <br>
- Max twenty parallel app-clients for the Websocket protocol, any number for HTTP. <br>
- Access restriction is implemented with the following restrictions. <br>
-- Client authentication not verified.<br>
-- Token expiry, and other timing parameters, are not verified.<br>
- Responses for error cases may not be completely according to spec.<br>
- The server returns on get requests (on all valid paths) an incremented integer dummy value.<br>
- The server does not for set requests update with the provided value.<br>
- The server returns subscription notifications containing an integer dummy value from a counter [0-999] that is incrementedevery 50 msec.<br>
- Access control is applied for write requests to all leaf nodes on the branch "Vehicle.Body".
- Access control is applied for read and write requests to all leaf nodes on the branch "Vehicle.ADAS".

Implementation language: JS. <br>


## Build instructions:
If the server to be used is not the one deployed by the project for public access, see below, then it must be built first, see build instructions in server/server-1.0 directory. <br>
The HTML clients found here can be opened in any browser. <br>

## Usage instructions:
Before issuing any requests to the server, the server IP address (or domain name) must be set. <br>
Even if the server runs on the same machine the machine outbound IP must be set, followed by a click on the Server IP button. <br><br>
If the server to be used is the one deployed by the project for public access, then the URL is: puppis.w3.org<br><br>
After the server URL is set, then requests can be issued to the server. <br>
For request examples, see the file appclient_commands.txt that contains request examples for both Websockets and HTTP transport protocols.  <br>
The VSS paths shown in the examples can be replaced by any path that the VSS tree at the VSS Github project supports (https://github.com/GENIVI/vehicle_signal_specification). <br>
The Gen2 access restriction model describes two authorization servers, the Access Grant Token (AGT) server, and the Access Token (AT) server, <br>
as described in https://rawcdn.githack.com/w3c/automotive/63ebd2f57f847f5a59dd508dbafcad81a2eba280/spec/Gen2_Core.html#-a-name-userauth-a-user-authentication-and-authorization. <br>
To obtain an AGT token the agtclient.html is used. The IP address is the same as for the Gen2 server, the path is "agtserver",<br> 
and the request to the agtserver must be a JSON formatted message<br>
{"userid":"XXX","vin":"YYY"}<br>
where XXX can be replaced by any (fictious) user name, and YYY any (fictious) VIN number.<br>
The response contains the AGT token, that is used as input in the following request to the AT-server.<br>
For this request, open the atclient.html in a browser, input the same IP address, and as path "atserver", then the request to the AT-server shall have the following JSON format:<br>
{"scope":"AAA","token":"BBB"}<br>
where AAA must be either "VehicleReadOnly", or "VehicleControl", and BBB is replaced by the AGT token.<br>
If the AGT token is verified as valid, which it is if it comes from the AGT server, the response contains the AT token that can then be used in requests to the Gen2 server.<br>
To enable testing of access restriction, all signals in the subtree "Vehicle.Body" require a token with VehicleControl scope for write requests, 
and all signals in the subtree "Vehicle.ADAS" require a token with scope VehicleReadOnly (or VehicleControl) for read requests, and VehicleControl for write requests.<br>
Please see https://rawcdn.githack.com/w3c/automotive/63ebd2f57f847f5a59dd508dbafcad81a2eba280/spec/Gen2_Core.html#filtering for extending requests with queries. <br>

