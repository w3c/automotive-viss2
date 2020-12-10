**(C) 2020 Geotab Inc**<br>

All files and artifacts in this repository are licensed under the provisions of the license provided by the LICENSE file in this repository.

# VISv2 server core

At startup the VISSv2 server core reads the vss_gen2.cnative file, which contains the VSS tree in cnative format. 
It then generates the file vsspathlist.json in the server parent directory. 
Cnative files containing the latest VSS tree on the VSS repo can be generated after cloning the VSS repo, and then issuing the 'make cnative' command.

Besides the cnative file that the server reads at start up, three cnative files are included in this directory. By changing their name to vss_gen2.cnative, the server will start up using the tree defined by that file.<br>
The one having a name mentioning access control have all leaves on the branches Body (read-only) and ADAS (read-write) access controlled. To access any of these nodes, an Access Token must be obtained via following the flow described in the <a href="https://github.com/w3c/automotive/blob/gh-pages/spec/Gen2_Core.html">W3C Gen2 CORE spec, Access Control chapter</a>.
