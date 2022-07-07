# Access Token Server (ATS)

  The access token server is deployed in the vehicle, and has two major tasks:
  
1. Providing access tokens to requesting clients.

2. Support the VISSv2 server with validation of client requests to access restricted VSS nodes.

The first task is supported by the ATS through that it exposes a service to clients over the transports:

- HTTP
- MQTT


## HTTP Requests
 
A client using HTTP shall issue a POST message with the path "/ats", on the port number 8600.

The POST message, in case of using a Short Term Access Grant Token should contain the following:

```
POST /ats HTTP/1.1
...
{
"purpose":"pay-as-you-drive",
"token":"eyJhbGciON . . . J4OjAKsltT7x"
}
```

In case of being using a Long Term Access Grant Token (the difference is that the long term includes the client key), a proof of possession of the key must be issued.

```
POST /ats HTTP/1.1
...
{
"purpose":"pay-as-you-drive",
"token":"eyJhbGciON . . . J4OjAKsltT7x",
"pop": "eyJ0eXAiOiJ. . . GAdLinsCffKKEA"
}
```

## MQTT Requests

A client using the MQTT transport must apply the application level protocol which is described in the <a  href="https://github.com/MEAE-GOT/WAII/tree/master/server/mqtt_mgr">MQTT manager directory</a>, with the difference that the ATS is subscribing to the following topic:

VIN/access-control

> Note: Not developed yet.

## Request Validation
An AT Request must contains the following claims that must be validated:

- **Purpose**: It must be on the Purposelist file holded by the AT server. The purpose represents a set of signals that a client can access and its permissions.
- **Token**: The Access Grant Token that authenticates the client and allows to check the requested purpose with its context.  The following claims are validated:
	- Expiration times
	- Client context: must be valid with the purpose
	- Audience claim: must be set to "w3org/gen2"
	- JWT Identifier: must be valid
	- Public Key: Included in case of long term. Must match the key in the Proof of Possession.
	- Token Signature: The AGT must be signed by the AGT Server.
- **Proof of Possession**: The proof of possession must match the public key in the AGT received. The PoP token must be valid.