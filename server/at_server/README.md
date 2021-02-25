/**
* (C) 2021 Geotab Inc
*
* All files and artifacts in the repository at https://github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

# Access Token Server (ATS)

The access token server is deployed in the vehicle, and has two major tasks.

1. Provision access tokens to requesting clients.
2. Support the VISSv2 server with validation of client requests to access restricted VSS nodes.

The first task is supported by the ATS through that it exposes a service to clients over the transports:
A. HTTP
B. MQTT

A client using HTTP shall issue a POST message with the path = atserver, on the port number 8600.
The POST payload shall use JSON format, and contain the following:
{"token":"ag-token", "purpose":"fuel-status", "pop":"GHI"}
where pop is included only if a key was present in AGT request, the token value must be replaced by the AG token, and purpose must be on the Purpose list. For more information, see the VISSv2 specification, the access control chapter in the CORE document.

A client using the MQTT transport must apply the application level protocol which is described in the <a href="https://github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/tree/master/server/mqtt_mgr">MQTT manager directory</a>, with the difference that the ATS is subscribing to the following topic:
VIN/access-control


