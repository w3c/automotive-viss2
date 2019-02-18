# W3C_VehicleSignalInterfaceImpl
Playground for evaluating the second iteration of the W3C automotive specification.

In order to keep some structure on this playground, the players should try to follow the guidelines below.

Playing shall be done within any of the directories
	1. server
	2. client
	3. pre_dev
These directories each kontain language specific directories for respective playing dialect. If directory for dialect of interest is missing, just create it.

# 3. pre_dev
The real playing should be done under this directory, where new ideas, etc can be evaluated. Please create a directory for the play, and create a README.md file from the template.

# 1. server
Playing under the server directory should align with the project common software architecture. Significant deviations should be documented in the README.md file.

# 2. client
The playing under the client directory has as a prime target testing of the server implementations. To play, create a directory if a new testscope is addressed. In the README.md file, please describe the testscope, build and test intructions.

# Development Process

We will follow a simple process where work packages are create as issues labeled as an implementation issue. The name of the issue is also the name of the branch which is used for implementation. A pull request is created when the task is finished to push the changes to the main branch.

1) Create implementation issue
2) Branch using the name of the issue
3) Implement and test
4) Merge main to local Branch
5) Create PR
6) Gate keepers review and push to main branch.



# Project common software architecture
The server functionlity has a complexity level that warrants some thoughts on how a good software architecture could look like. 
The figure below is the current proposal on this. Improvements are welcome, which typically starts with some playing in pre_dev, and a discussion on the issue list, before a pull request may be issued.
![Software architecture](pics/common_server_swa.png?raw=true)<br>
*Fig 1. Software architecture overview

The design tries to meet the following requirements:
- Abstract the details of how the service managers access the data that is represented in the tree. 
- Support straightforward addition of further transport protocols.
- Support straightforward addition of further service managers. 
- Support both the case of a singleton server through which all acces goes, and the option of distributed servers that a client can access directly, after the initial service discovery.
The following describes the components shown in Figure 1. 
## Transport managers and interface.
The transport manager is responsible for that the Gen2 client-server communication is carried out according to the transport protocol the manager implements. The payload being communicated shall have the format specified in the GEN2 TRANSPORT document. A transport manager acts as a proxy between a client and the Server core, see Figure 2. The payload communicated over the Transport interface shall be identical regardless of which Transport manager the Core server communicates with. This payload shall therefore follow the format for the Websocket transport as specified in GEN2 TRANSPORT. This means that all transport managers except the Websocket transport manager must reformat the payload communicated over their respective transport protocol. The transport interface is divided into two components, a registration interface, and a data channel interface. The registration interface is used by a transport manager to register itself with the Server core. The data channel interface is used for exchanging the requests, responses, and notifications between the communication end points. The data channel must support bidirectional asynchronous communication, and should support both local and remote transport manager deployment. 
At the registration with the core server the transport manager receives a transport manager Id. It must then include this in all payloads forwarded to the core server, for the core server to use in its payload routing. The transport manager must also include a client Id in the payload, to enable its own routing to different app-clients. These Ids must not be part of the payload returned to the app-clients. 
![Transport closeup](pics/transport_closeup.png?raw=true)<br>
*Fig 2. Transport SwA closeup
## Tree manager and interface
The tree representing the accessible data points is found on the GENIVI VSS Github. There is also found a basic tree manager from which the basis of this tree manager is cloned. 
The tree manager abstracts the details of the tree, e. g. its format, which through the VSS tool support can be e. g.  JSON, CSV, or c-native (c.f. VSS). It provides a method for searching the tree, to verify that a given path has a match in the tree, or to find all paths matching a path including wild cards. It also supports access to node internal data associated to a node reference provided by a previous search. 
![Tree closeup](pics/tree_closeup.png?raw=true)<br>
*Fig 3. Tree SwA closeup
## Service managers and interface
The service manager internal design is not part of this project scope, as it is anticipated that this may substantially differ between OEMs. Therefore only the service manager interface is described here, and some service manager behaviour related to this interface. A service manager must first register with the Server core, in which it provides the path from tree root to the node that is the root node of the branches owned by this service manager. This means that the existing tree must already be populated with the branches of this service manager. A possible later extension is to allow the service manager to provide new branches to be added to the tree. Another extension could be for the service manager to also provide information whether it supports direct client access (distributed server solution), or not. But this is not supported in this first version. The service manager acts as the server in the data channel communication with the server core, so it shall provide the server core with port number and URL path at the registration. 
The service interface payload shall follow the same format as the transport protocol. However, client requests related to authorize and service discovery actions terminate at the Server core, and should not be issued at this interface, i. e. only requests related to get/set/subscribe/unsubscribe should be communicated here. App-client requests may lead to requests on multiple data points, which shall be resolved by the server core, and over this interface lead to multiple requests, one per data point.
The service interface is divided into two components, a registration interface, and a data channel interface. The registration interface is used by a service manager to register itself with the Server core. The data channel interface is used for exchanging the requests, responses, and notifications between the communication end points. The data channel must support bidirectional asynchronous communication, and should support both local and remote service manager deployment. 
It is the responsibility of the service manager to map the VSS path expressions to whatever internal addressing scheme that is required to execute the requested data access. It is also the responsibility of the service manager to interpret the filter instructions in a subscription request, and make sure these are followed in subsequent notifications to the client. Further, the provided requestId must be included in responses and notifications as described in the GEN2 specification. 
![Service closeup](pics/service_closeup.png?raw=true)<br>
*Fig 4. Service SwA closeup
## Server core
The Server core is the like the spider in the net, tieing it all together. Its main tasks are:
 1. Handle registrations by transport managers and service managers.
 2. Payloadd analysis.
 3. Message routing.
 4. Access restriction management.
 5. Service discovery response.
#### 1. Payload analysis
As not all messages shall lead to a forwarded request to a related service manager, the Server core must analyse the requested action, and act according to it. 
#### 2. Message routing
The Server core must keep track of from which transport manager a request message was received, in order to return the response to the same transport manager. The core server therefore provides the transport manager with an Id at the registration, which it then will embed in all requests forwarded to the server core. As the transport manager itself needs to route payloads back to respsctive app-client, it also embeds a client Id in the payload. These two Ids shall follow the payload to the service manager, which then must embed it in its responses and notifications sent back to the core server. It is the responsibility of the transport manager to remove the Ids before returning payloads to an app-client. 
#### 3. Access restriction management
The Server core shall always check a tree node for which a client is requesting access to find out whether there is access restrictions tied to it. If so, it shall act as described in the VSI CORE document. 
The authorisation server used in this project may not meet the security robustness required in a real life deployment. Initially, it may simply have fixed yes/no response that may be toggled during testing. 
#### 4. Service discovery response
The response of a service discovery client request shall contain a JSON formatted tree containing all nodes under the tree node pointed to by the path. It is the responsibility of the Server core to use the tree interface to read node data for all nodes of this subtree, and format it into a JSON tree object to be returned to the client.
![Server core closeup](pics/server_core_closeup.png?raw=true)<br>
*Fig 2. Server core SwA closeup

