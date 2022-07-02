**(C) 2019 Volvo Cars**<br>
**(C) 2019 Geotab Inc**<br>
**(C) 2019 Mitsubishi Electric Automotive**<br>

All files and artifacts in this repository are licensed under the provisions of the license provided by the LICENSE file in this repository.

# W3C Automotive Interface Implementation - WAII (pronounced wy-ee)
This project implements the W3C VISS v2 specification under development at <a href="https://github.com/w3c/automotive">W3C Automotive Working Group</a>.<br>


# Starting and building the server

This project requires Go version 1.13 or above, make sure your GOROOT and GOPATH are correctly configured. Since this project uses Go modules all dependencies will automatically download when building the project the first time.

The server can be built and started by running the script below. This will build and start the following components, see Fig. 1.
 - Core server
 - HTTP manager
 - Websocket manager
 - MQTT manager
 - Vehicle service manager
 - Access Grant token server
 - Access Token server 
 
 A subset of these can be started by e. g. removing components from the line below in the W3CServer.sh.
```
 services=(server_core service_mgr at_server agt_server http_mgr ws_mgr mqtt_mgr)
```
Script start / stop commands:
```bash
#start the servers
$ ./W3CServer.sh startme
#stop the servers
$ ./W3CServer.sh stopme
```
Components can be built separately by issuing<br>
$ go build<br>
in their respective directory.<br>
Some of them may be possible to start together with a flag set on the command line. 
To find out about possible flags, they can be started together with the --help flag, e.g.:<br>
$ ./server_core --help


To speed up the first time build you can run the command below in ./ and ./server directory

```bash
$ go mod tidy
```

To update the dependencies to latest run

```bash
$ go get -u ./...
```

If working with a fix or investigating something in a dependency, you can have a local fork by adding a replace directive in the go.mod file, see below examples. 

```
replace example.com/some/dependency => example.com/some/dependency v1.2.3 
replace example.com/original/import/path => /your/forked/import/path
replace example.com/project/foo => ../foo
```

Testing if your fix builds is often easiest to do by the command
$ go build 
in the directory of the fixed go file.

For more information see https://github.com/golang/go/wiki/Modules#when-should-i-use-the-replace-directive

Make sure not to push modified go.mod, go.sum files since that would probably break the master branch.

### Multi-process vs single-process server implementation
The components mentioned above that together realizes the server is available in two different implementations:
- Components are built and deployed as separate processes/binaries, and communicate using Websocket.<br>
- Components are built and deployed as threads within one common process/binary, and communicate using Go channels.<br>

These implementations are found at the branches multi-process and single-process, respectively. 
The master branch is currently identical to the multi-process branch.

### Using Docker-Compose to launch a W3CServer instance

The W3C server can also be built and launched using docker and docker-compose please follow below links to install docker and docker-compose

https://docs.docker.com/install/linux/docker-ce/ubuntu/
https://docs.docker.com/compose/install/

to launch use

```bash
$ docker-compose up -d
Starting w3c_vehiclesignalinterfaceimpl_servercore_1 ... done
Starting w3c_vehiclesignalinterfaceimpl_servicemgr_1 ... done
Starting w3c_vehiclesignalinterfaceimpl_wsmgr_1      ... done
Starting w3c_vehiclesignalinterfaceimpl_httpmgr_1    ... done
```
to stop use 

```bash
$ docker-compose down
Stopping w3c_vehiclesignalinterfaceimpl_servicemgr_1 ... done
Stopping w3c_vehiclesignalinterfaceimpl_httpmgr_1    ... done
Stopping w3c_vehiclesignalinterfaceimpl_wsmgr_1      ... done
Stopping w3c_vehiclesignalinterfaceimpl_servercore_1 ... done
Removing w3c_vehiclesignalinterfaceimpl_servicemgr_1 ... done
Removing w3c_vehiclesignalinterfaceimpl_httpmgr_1    ... done
Removing w3c_vehiclesignalinterfaceimpl_wsmgr_1      ... done
Removing w3c_vehiclesignalinterfaceimpl_servercore_1 ... done
```

if server needs to be rebuilt due to src code modifications

```bash
$ docker-compose up -d --force-recreate --build
```
all log files are stored in a docker volume to find the local path for viewing the logs use the docker inspect command to find the Mountpoint
```bash
$ docker volume ls
DRIVER              VOLUME NAME
local               w3c_vehiclesignalinterfaceimpl_logdata
$ docker volume inspect w3c_vehiclesignalinterfaceimpl_logdata
[
    {
        "CreatedAt": "2019-12-12T15:32:13+01:00",
        "Driver": "local",
        "Labels": {
            "com.docker.compose.project": "w3c_vehiclesignalinterfaceimpl",
            "com.docker.compose.version": "1.25.1-rc1",
            "com.docker.compose.volume": "logdata"
        },
        "Mountpoint": "/var/lib/docker/volumes/w3c_vehiclesignalinterfaceimpl_logdata/_data",
        "Name": "w3c_vehiclesignalinterfaceimpl_logdata",
        "Options": null,
        "Scope": "local"
    }
]

```


# 1. server

The server directories contain an implementation of a server acording to the W3C W3C VISS v2 spec, and should align with the project common software architecture, see below. 
Significant deviations should be documented in the README.md file.

# 2. client
The client directories has as a prime target testing of the server implementations. The local README.md file should provide information about client scope and usage.

# Development Process

We will follow a simple process where work packages are create as issues labeled as an implementation issue. The name of the issue is also the name of the branch which is used for implementation. A pull request is created when the task is finished to push the changes to the main branch.

1) Create implementation issue
2) Branch using the name of the issue
3) Implement and test
4) Merge main to local Branch
5) Create PR
6) Gate keepers review and push to main branch.
   - current gate keepers: (@MagnusGun,@UlfBj,@PeterWinzell) 


# Project common software architecture
The server functionality has a complexity level that warrants some thoughts on how a good software architecture could look like. 
Figure 1 is the current proposal on this.<br>
![Software architecture](pics/common_server_swa.png?raw=true)<br>
*Fig 1. Software architecture overview

The design tries to meet the following requirements:
- Abstract the details of how the service managers access the data that is represented in the tree. 
- Support straightforward addition of further transport protocols.
- Support straightforward addition of further service managers. 
- Support both the case of a singleton server through which all access goes, and the option of distributed servers that a client can access directly, after the initial service discovery.
The following describes the components shown in Figure 1. 
## Transport managers and interface.
The transport manager is responsible for that the W3C VISS v2 client-server communication is carried out according to the transport protocol the manager implements. The payload being communicated shall have the format specified in the W3C VISS v2 TRANSPORT document. A transport manager acts as a proxy between a client and the Server core, see Figure 2. The payload communicated over the Transport interface shall be identical regardless of which Transport manager the Core server communicates with. This payload shall therefore follow the format for the Websocket transport as specified in W3C VISS v2 TRANSPORT. This means that all transport managers except the Websocket transport manager must reformat the payload communicated over their respective transport protocol. The transport interface is divided into two components, a registration interface, and a data channel interface. The registration interface is used by a transport manager to register itself with the Server core. The data channel interface is used for exchanging the requests, responses, and notifications between the communication end points. The data channel must support bidirectional asynchronous communication, and should support both local and remote transport manager deployment. 
At the registration with the core server the transport manager receives a transport manager Id. It must then include this in all payloads forwarded to the core server, for the core server to use in its payload routing. The transport manager must also include a client Id in the payload, to enable its own routing to different app-clients. These Ids must not be part of the payload returned to the app-clients. 
![Transport closeup](pics/transport_closeup.png?raw=true)<br>
*Fig 2. Transport SwA closeup
## Tree manager and interface
The tree representing the accessible data points is found on the GENIVI VSS Github. There is also found a basic tree manager from which the basis of this tree manager is cloned. 
The tree manager abstracts the details of the tree, e. g. its format, which through the VSS tool support can be e. g.  JSON, CSV, or c-native (c.f. VSS). It provides a method for searching the tree, to verify that a given path has a match in the tree, or to find all paths matching a path including wild cards. It also supports access to node internal data associated to a node reference provided by a previous search. 
![Tree closeup](pics/tree_closeup.png?raw=true)<br>
*Fig 3. Tree SwA closeup
## Service managers and interface
Although the service manager internal design is anticipated to substantially differ between OEMs, the internal design in this project follows the architecture of the <a href="https://at.projects.genivi.org/wiki/pages/viewpage.action?pageId=34963516">GENIVI CCS project</a> with a "state storage" component as the interface that the service manager interacts with. The service manager interface described here should however be common for all implementations.<br>A service manager must first register with the Server core, in which it provides the path from tree root to the node that is the root node of the branches 'owned' by this service manager. This means that the existing tree must already be populated with the branches of this service manager. A possible later extension is to allow the service manager to provide new branches to be added to the tree. Another extension could be for the service manager to also provide information whether it supports direct client access (distributed server solution), or not. But this is not supported in the current version. The service manager acts as the server in the data channel communication with the server core, so it shall provide the server core with port number and URL path at the registration. 
The service interface payload shall follow the same format as the transport protocol. However, client requests related to authorize and service discovery actions terminate at the Server core, and should not be issued at this interface, i. e. only requests related to get/set/subscribe/unsubscribe should be communicated here. App-client requests may lead to requests on multiple data points, which shall be resolved by the server core, and over this interface be replaced by an array of fully qualified paths. The subsequent response, or subscription notification, shall contain data from all nodes addressed in the array.
The service interface is divided into two components, a registration interface, and a data channel interface. The registration interface is used by a service manager to register itself with the Server core. The data channel interface is used for exchanging the requests, responses, and notifications between the communication end points. The data channel must support bidirectional asynchronous communication, and should support both local and remote service manager deployment. 
In this design it is the state-storage responsibility to map the VSS path expressions to whatever vehicle internal addressing scheme that is required to execute the requested data access. It is the responsibility of the service manager to interpret the filter instructions in a subscription request, and make sure these are followed in subsequent notifications to the client. 
![Service closeup](pics/service_closeup.png?raw=true)<br>
*Fig 4. Service SwA closeup
## Server core
The Server core is the like the spider in the net, tying it all together. Its main tasks are:
 1. Handle registrations by transport managers and service managers.
 2. Payload analysis.
 3. Message routing.
 4. Access restriction management.
 5. Service discovery response.
#### 1. Payload analysis
As not all messages shall lead to a forwarded request to a related service manager, the Server core must analyze the requested action, and act according to it. 
#### 2. Message routing
The Server core must keep track of from which transport manager a request message was received, in order to return the response to the same transport manager. The core server therefore provides the transport manager with an Id at the registration, which it then will embed in all requests forwarded to the server core. As the transport manager itself needs to route payloads back to respective app-client, it also embeds a client Id in the payload. These two Ids shall follow the payload to the service manager, which then must embed it in its responses and notifications sent back to the core server. It is the responsibility of the transport manager to remove the Ids before returning payloads to an app-client. 
#### 3. Access restriction management
The Server core shall always check a tree node for which a client is requesting access to find out whether there is access restrictions tied to it. If so, it shall act as described in the VISSv2 CORE document. 
The authorization solution used in this project may not meet the security robustness required in a real life deployment. It shall however follow the access control model as described in the VISSvs specification. 
#### 4. Service discovery response
The response of a service discovery client request shall contain a JSON formatted tree containing all nodes under the tree node pointed to by the path. It is the responsibility of the Server core to use the tree interface to read node data for all nodes of this sub-tree, and format it into a JSON tree object to be returned to the client.
![Server core closeup](pics/server_core_closeup.png?raw=true)<br>
*Fig 2. Server core SwA closeup

# Server configurations

## VSS tree
The vehicle signals that the W3C VISS v2 server manages are defined in the "vss_W3C VISS v2.cnative" file in the server_core directory. New cnative files containing the latest verision on the VSS repo can be generated by cloning the <a href="https://github.com/GENIVI/vehicle_signal_specification">Vehicle Signal Specification</a> repo, and then issuing a "make cnative" command, see <a href="https://genivi.github.io/vehicle_signal_specification/tools/usage/">Tools usage</a>.<br>
To use other formats. e. g. JSON, the vssparserutilities.c found in the c_native directory at the <a href="https://github.com/GENIVI/vss-tools">VSS Tools</a> repo would have to implemented for that format (instead of the cnative format that it currently implements). The major parts to be reimplemented are the file read/write methods, and the "atomic" data access methods. Other methods, like the search method use these atomic methods for actual data access. This file would then have to replace the current vssparserutilities.c in the server_core directory. 

## VSS data sources
The service manager implementation tries to open the file "statestorage.db" in the service_mgr directory. If this file exists, the service manager will then try to read the signals being addressed by the paths in client requests from this file. The file is an SQL database containing a table with a column for VSS paths, and a column for the data associated with the path. If there is no match, or if the database file was not found at server startup, then the service manager will instead generate a dummy value to be returned in the response. Dummy values are always an integer in the range from 0 to 999, from a counter that is incremented every 37 msec.<br>
New statestorage.db files can be generated by cloning the <a href="https://github.com/GENIVI/ccs-w3c-client">CCS-W3C-Client</a> repo, and then run the statestorage manager, see the statestorage directory. It is then important that the "vsspathlist.json" file being read by the statestorage manager is copied from the server directoy of this repo, where it becomes generated by the W3C VISS v2 server at startup (from the data in the "vss_W3C VISS v2.cnative" file, and that the new statestorage database is populated with actual data, either in real time when running the W3C VISS v2 server, or preloaded with static data. The statestorage architecture allows one or more "feeders" to write data into the database, and also provides a translation table that can be preloaded for translating from a "non-VSS" address space to the VSS addres space (=VSS paths).

## Payload encoding
A reference payload encoding is implemented that compresses the W3C VISS v2 transport payloads with a ratio of around 450% to 700%.<br>
The encoding is currently only possible to activate over the WebSocket transport protocol, but it would also be possible to invoke over the HTTP protocol. 
To invoke it a client must set the sub-protocol to "W3C VISS v2c", in JS something like<br>
```
 socket = new WebSocket("ws://url-to-server:8080", "W3C VISS v2c");
```
For unencoded WebSocket sessions, the sub-protocol shall be set to "W3C VISS v2", or left out completely.<br>
The encoding uses both a lookup table to replace known data (paths, and other fixed key-values), removal of rule-based JSON reserved characters, and binary compression of actual data.<br>
The mechanism of using the sub-protocol parameter to invoke encoding scheme can easily be extended to use other compression schemes.
For more information about the reference encoding, see README in the utils directory.

## Access control
The access control model in the W3C VISS v2 specification is supported, with the exception of the authentication step. 
The implementation would not pass a security check, for eample the shared key for token signature verification is hardcoded with a redable text as value. 
The access control model architecture is shown below.
![Access control architecture](pics/W3C_VISS_v2_access_control_model.png?raw=true)
The README in the client/client-1.0/Javascript directory describes the requests a client must issue first to the Access Grant Token server, 
and then to the Access Token server in order to obtain an Access token.<br>
HTML client implementations for respective access can also be found in the directory.
