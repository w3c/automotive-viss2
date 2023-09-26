---
title: "VISSv2 peripheral components"
---

A few other software components that can be useful when setting up a VISSv2 communication tech stack exists:
* Open Vehicle Data Set, a relational database with a table configuration that enables it to store time series of VSS data from multiple vehicles.
* A "live simulator" that can read vehicle trip data stored in an OVDS database , and replay it so that it appears as live data from the recorded trip.

## Open Vehicle Data Set (OVDS)
The code to realize an OVDS database is found [here](https://github.com/COVESA/ccs-components/tree/master/ovds).
The database is realized using SQLite, so it is possible to use the SQLite API to read and write from it.

However, an [OVDS server](https://github.com/COVESA/ccs-components/tree/master/ovds/server) is available that exposes a small set of methods for this, over HTTP.
For more details, please check the README on the link.

There is as well an [OVDS client](https://github.com/COVESA/ccs-components/tree/master/ovds/client)
available that connects to a VISSv2 server to fetch data that it then writes into the OVDS using the OVDS server interface.

## Live simulator
The [live simulator](https://github.com/COVESA/ccs-components/tree/master/livesim) reads data from an OVDS containing recorded trip data,
and then writes it into a state storage timed by the data time stamps so that it appears timing wise as when it was recorded.
For more details, please check the README on the link.

The the test_vehicles.db file in the [OVDS server](https://github.com/COVESA/ccs-components/tree/master/ovds/server)
directory contains trip data generously provided by Geotab Inc.
It can be used as input to the live simulator, as well as the sawtooth_trip.db for simple testing.

The live simulator needs a copy of the list of leaf node paths (vsspathlist.json),
which needs to contain at least all the paths that are to be replayed from the OVDS, and are also to be found in the VSS tree that the VISSv2 server uses.
