**(C) 2020 Geotab Inc**<br>

All files and artifacts in this repository are licensed under the provisions of the license provided by the LICENSE file in this repository.

# Gen2 server core

At statup the Gen2 server reads the vss_gen2.cnative file, which contains the VSS tree in cnative format. 
It then generates the file vsspathlist.json in the server parent directory. 

Besides the cnative file that the server starts up reading, three cnative files are included in this directory. By changing their name to vss_gen2.cnative, the server will start up using the tree defined by that file.<br>
The one having a name mentioning access control have all leaves on the branches Body (read-only) and ADAS (read-write) acces controlled. To access any of these nodes, an Access Token must be obtained via following the flow described in the <a href="https://github.com/w3c/automotive/blob/gh-pages/spec/Gen2_Core.html">W3C Gen2 CORE spec, Access Control chapter</a>.
