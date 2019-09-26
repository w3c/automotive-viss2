# Project: W3C_VehicleSignalInterfaceImpl: client/client-1.0

Functionality: <br>
	Long term: Client implementations for communication with the server found at the server/server-1.0 directory.<br>
                   Clients shall be available for the different traansport protocols supported by the server, <br>
                   which currently is Websockets and HTTP. <br>
                   The clients can be executed either locally on the same machine, <br>
                   or remotely towards a publicly accessible deployment of the server. 
	Short term limitations: <br>
		- The following client limitations are due to server limitations. <br>
                - The transport protocols are not the secure versions of Websockets and HTTP. <br>
		- Max two parallel app-clients for each of HTTP and Websoclket protocols. <br>
		- Access restriction not implemented. <br>
		- Responses for error cases may not be correct (or even have JSON format).<br>
		- The server returns dummy values for get.<br>
		- The server does not update values for set.<br>
		- The server returns dummy values every five secs for subscription.<br>

Implementation language: JS. <br>


## Build instructions:
If the server to be used is not the one deployed by the project for public access, see below, then it must be built first, see build instructions in server/server-1.0 directory. <br>
For the clients found here, clicking on any of the client HTML files opens the client in a browser. <br>

## Usage instructions:
Before issuing any requests to the server, the server IP address (or domain name) must be set. <br>
If the server runs on the same machine, write "localhost" (withtout citation marks), and click the Server IP button. <br>
If the server runs on a different machine, the IP address, or domain name, must be obtained, and then written into the Server IP button field.  <br>
If the server to be used is the one deployed by the project for public access, then the URL is !!!Deployment details not yet resolved. Will be shown here when available!! <br>
After the server URL is set, then requests can be issued to the server. <br>
For request examples, see the file appclient_commands.txt that contains request examples for both Websockets and HTTP transport protocols.  <br>
The VSS paths shown in the examples can be replaced by any path that the VSS tree at the VSS Github project supports (https://github.com/GENIVI/vehicle_signal_specification). <br>
As the server uses an instance in the C-native format of that tree, its response will contain data from the addressed parts of that tree. <br>

