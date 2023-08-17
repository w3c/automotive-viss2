**(C) 2023 Ford Motor Company**<br>

# VISSv2 feeder

The feeder is designed to work together with the VISSv2 server and a state storage as shown in the figure below.
![Software architecture](feeder-sw-arch.jpg?raw=true)<br>
*Fig 1. Software architecture overview

The feeder has two interface clients, each running on a thread, one exercises the interface towards the server, and the other the underlying vehicle interface.
When either client reads data from he interface it forwards it to the Map & Scale component.
This component uses mapping instructions that it reads from a config file at startup to swwitch the name of the data to the name of the other domain, and scales the value if necessary.
When that is done it forwards the mapped data to the other interface client that then uses the interface to send it into the domain.

The interface towards the VISSv2 server consists of two parts, one for reading data and one for writing data. When data is to be written to the server, the feeder writes to the statestorage from which the server then can read it. Data coming from the server is read over a Unix domain socket connection, where the feeder acts as a server.
The VISSv2 server, which on this Unix connection acts as the client, reads a "feeder registration" file at startup that provides the socket address and
the root node name of the tree that the feeder manages (VSS currently only defines one tree, but that may change).

The vehicle interface client exercises the vehicle interface. This interface can be an interface towards a CAN bus, a Flexray bus, etc., and the details of it is OEM proprietary.
Therefore this feeder implements a simulation of a data exchange over the interface, code that will have to be replaced by the OEM before deployment.

The current implementation does not scale the data value and the information needed for that is not present in the map and scale instructions read at startup.
A solution for this is in the planning.

The feeder expects the statestorage to be implemented using a Redis database, the details of this can be found at
<a href="https://github.com/COVESA/ccs-components/tree/master/statestorage">COVESA CCS components</a>.
