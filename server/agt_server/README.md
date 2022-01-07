/**
* (C) 2021 Geotab Inc
*
* All files and artifacts in the repository at https://github.com/MEAE-GOT/WAII
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

# Access Grant Token Server (AGTS)

The AGTS is deployed of the vehicle, and has two major tasks.

1. Authenticating requesting clients.
2. Provision access grant tokens to requesting clients.

The tasks are supported by the ATS through that it exposes a service to clients over an HTTP transport.

A client shall issue a POST message with the path = agtserver, on the port number 7500.

The POST payload shall use JSON format, and contain the following:
{"vin":"GEO001", "context":"Independent+OEM+Cloud", "proof":"ABC", "key":"DEF"}

where the "vin" value shall be the vehicle identity, which typically would be a pseudo-VIN, 
the "context" value is the role triplet, the "proof" value contains the credentials that the client refer to for its "context" claim, and the "key" value, which may be omitted, is used for the proof-of-possession concept as described in the VISSv2 specification, in the access control chapter in the CORE document. 



