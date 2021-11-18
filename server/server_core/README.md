**(C) 2020 Geotab Inc**<br>
**(C) 2021 Mitsubishi Electric Automotive Europe B.V.**<br>

All files and artifacts in this repository are licensed under the provisions of the license provided by the LICENSE file in this repository.

# VISSv2 server core

At startup the VISSv2 server core reads the vss_vissv2.binary file, which contains the VSS tree in binary format. 
It then generates the file vsspathlist.json in the server parent directory. 
Binary files containing the latest VSS tree on the VSS repo can be generated after cloning the VSS repo, and then issuing the 'make binary' command.

Besides the binary file that the server reads at start up, other binary tree files might be included in this directory. By changing their name to vss_vissv2.binary, the server will start up using the tree defined by that file.<br>
The one having a name mentioning access control have all leaves on the branches Body (read-only) and ADAS (read-write) access controlled. To access any of these nodes, an Access Token must be obtained via following the flow described in the <a href="https://github.com/w3c/automotive/blob/gh-pages/spec/VISSv2_Core.html">W3C VISSv2 CORE spec, Access Control chapter</a>.

# VISSv2 refactored version
Previous version spawened one server for each url, this version uses only a single server to host all of them so the endpoints currently supported are 
/vsspathlist                # same content as vsspathlist.json described above
/service/reg                # registration of the service manager service currently limited to one instance
/transport/reg              # registration of the different types of transport managers, currently supported are HTTP, Websocket and MQTT
/transport/data/{protocol}  # websocket connection endpoint for the transport managers