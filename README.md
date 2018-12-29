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
A transport manager is responsible for that the VSI client-server communication is carried out according to the transport protocol the manager implements. The payload being communicated shall have the format specified in the VSI TRANSPORT document. 
A transport manager acts as a proxy between a client and the Server core, see Figure 2. 
The payload communicated over the Transport interface shall be identical regardless of which Transport manager the Core server communicates with. This payload shall therefore follow the format for the Websocket transport as specified in VSI TRANSPORT. This means that all transport managers except the Websocket transport manager must reformat the payload communicated over their respective transport protocol. 
The transport interface is implemented as a socket interface, as either a UNIX domain socket, or a network socket, depending on the IPC scenario. 
![Transport closeup](pics/transport_closeup.png?raw=true)<br>
*Fig 2. Transport SwA closeup
## Tree manager and interface
The tree representing the accessible data points is found on the GENIVI VSS Github. There is also found a basic tree manager from which the basis of this tree manager is cloned. 
The tree manager abstracts the details of the tree, e. g. its format, which through the VSS tool support can be e. g.  JSON, CSV, or c-native (c.f. VSS). It provides a method for searching the tree, to verify that a given path has a match in the tree, or to find all paths matching a path including wild cards. It also supports access to node internal data associated to a node reference provided by a previous search. 
![Tree closeup](pics/tree_closeup.png?raw=true)<br>
*Fig 3. Tree SwA closeup
## Service managers and interface
The service manager internal design is not part of this project scope, as it is anticipated that this may substantially differ between OEMs. Therefore only the serice manager interface is described here, and some service manager behaviour related to this interface. 
A service manager must first register with the Server core, in which it provides the tree branches containing the services it manages. It also provides information whether it supports direct client access (distributed server solution), or not. It also provides a path, starting at the VSS root, to the node it want to branch out from. The resulting tree should align with the tree as defined in VSS, but there is no verification by the Server core. Non-alignment will most likely lead to interoperability issues, with possible consequences for the registered service manager. 
The service interface payload shall follow the same format as the transport protocol. Client requests related to the authorize and service discovery actions terminate at the Server core, and should not be issued at this interface. Requests containing paths including wildcards may lead to multiple requests on this interface, one per matching path. The payload is transported on a socket interface, similar to the transport interface. 
In the registration phase the service manager(s) act as clients to the Server core. After the registration, the service managers acts as servers, receiving request messages from the Server core. This switch of roles in the client server model leads to a somewhat more complicated registration call where callback methods are established in both directions. 
It is the responsibility of the service manager to map the VSS paths to whatever internal addressing scheme that is required to execute the requested data access. 
It is also the responsibility of the service manager to interpret the filter instructions in a subscription request, and make sure these are followed in subsequent notifications to the client. 
![Service closeup](pics/service_closeup.png?raw=true)<br>
*Fig 4. Service SwA closeup
## Server core
The Server core is the like the spider in the net, tieing it all together. Its main tasks are:
 1. Payloadd analysis.
 2. Message routing.
 3. Access restriction management.
 4. Service discovery response.
#### 1. Payload analysis
As not all messages shall lead to a forwarded request to a related service manager, the Server core must analyse the requested action, and act according to it. 
#### 2. Message routing
The Server core must keep track of from which transport manager a request message ws received, in order to return the response to the same transport manager. The unique identity tied to this message must follow it also in the forwarding of this message to a service manager, so that when its response is returned, it can be routed back to the original transport manager, and from there back to the originating client. A message id must be unique over all client sessions, and in an efficient protocol is needs to be set by the client. This could be accomplished by the server providing the client with a set of message ids at registration, e. g. a million ids. With a 32 bit id then about 4k clients is the max number of clients. A client may reuse an id provided it there is no outstanding responses to this id. Client sessions are typically finite in time.
#### 3. Access restriction management
The Server core shall always check a tree node for which a client is requesting access to find out whether there is access restrictions tied to it. If so, it shall act as described in the VSI CORE document. 
The authorisation server used in this project may not meet the security robustness required in a real life deployment. Initially, it may simply have fixed yes/no response that may be toggled during testing. 
#### 4. Service discovery response
The response of a service discovery client request shall contain a JSON formatted tree containing all nodes under the tree node pointed to by the path. It is the responsibility of the Server core to use the tree interface to read node data for all nodes of this subtree, and format it into a JSON tree object to be returned to the client.
![Server core closeup](pics/server_core_closeup.png?raw=true)<br>
*Fig 2. Server core SwA closeup

