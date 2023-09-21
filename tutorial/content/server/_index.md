---
title: "VISSv2 Server"
---

The VISSv2 server is the Sw component that implements the interface that is exposed to the clients, and that must conform to the W3C VISSv2 specification.

### Build the server
Please check the chapter [VISSv2 Build System](/automotive-viss2/build-system) for general Golang information.

To build the server, open a erminal and go to the WAII/server/vissv2 directory and issue the command:

$ go build

### Configure the server

#### VSS tree configuration
The server has a copy of the VSS tree that it uses to verify that client requsts are valid -
that there is a node in the tree that corresponds to the path in a request, if a node requires an access control token, etc.
The tree parser that is used expects the tre to have the 'binary format' that one of the VSS-Tools genertes from the vspec files.
To generate this the [VSS repo](https://github.com/COVESA/vehicle_signal_specification) must be cloned including the VSS-Tools submodule,
and a file containing the binary representation must be created, which is done with the following command issued in the root directory.

$ make binary

This generates a file with a name like 'vss_rel_4.1-dev.binary',
which then needs to be renamed to 'vss_vissv2.binary' and stored in the WAII/server/vissv2server directory.

If you want to configure the tree to include access control, access control tags as described in the
[VISSv2 - Access Control Selection chapter](https://raw.githack.com/w3c/automotive/gh-pages/spec/VISSv2_Core.html#access-control-selection) needs to be added to appropriate tree nodes.
This can either be done by editing vspec files directly, or using the [VSS-Tools](https://github.com/covesa/vss-tools) overlay mechanism.

#### Command line configuration
The server has the following command line configurations:
* Data storage implementation. Select either to use an SQLite implementation (-s sqlite) or a Redis implementation (-s redis). Default is SQLite.
* Data storage file name (--dbfile 'file-name'). Only relevant for SQLite configuration. Default is "serviceMgr/statestorage.db".
* Request the server to generate a pathlist file, then terminate (--dryrun). Default is not to terminate after generating it.
* Pathlist file name (--vssjson 'file-name'. Default is "../vsspathlist.json".
* UDS path for history control (--uds 'file-name'). Name of the Unix domain socket file. Default is "/var/tmp/vissv2/histctrlserver.sock".
* Level of logging (--loglevel levelx). Levelx is one of [trace, debug, info, warn, error, fatal, panic]. Default is "info".
* Whether logging should end up in standard output (false) or in a log file (true) (--logfile false/true). The default is 'false'.

#### Data storage configuration
To be written...

#### Protocol support configuration

The server supports the following protocols:
* HTTP
* Websockets
* MQTT (with the VISSv2 specific application protocol on top)
* gRPC

The message payload is identical for all protocols at the client application level (after protocol specific modifications are restored).

The code is structured to make it reasonably easy to remove any of the protocols if that is desired for reducing the code footprint.
Similarly it should be reasonably straight forward to add new protocols, given that the payload format transformation is not too complicated.
