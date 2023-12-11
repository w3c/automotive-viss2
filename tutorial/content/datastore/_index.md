---
title: "VISSv2 Data Storage"
---

Two realizations of data storage are available on the [COVESA/CCS-components Github](https://github.com/COVESA/ccs-components),
one using an SQLite database, and the other a Redis database.

The server implements the APIs to both of these databases, which to use is selected by its command line configuration.

The same support should be available on the other SwC that accesses the data storage,
however the current feeder implementation only implements Redis support, hence it is not merged into the master branch but resides on the feeder branch.

It may be a bit confusing that sometimes this is referred to as "data store/storage" and sometimes "state storage".
The latter name is legacy from a previous COVESA project, the Cloud & Connected Services project, while the former has emerged later in the COVESA architecture group work.
An argument for keeping both could be to say that the state storage refers to a storage that only keeps the latest value of a signal,
while the data store refers to a more general database that can also keep time series of values of a signal.
There are two scenarios where the VISSv2 server operates on time series data, [curve logging](https://raw.githack.com/w3c/automotive/gh-pages/spec/VISSv2_Core.html#curvelog-filter-operation),
and [historic data](https://raw.githack.com/w3c/automotive/gh-pages/spec/VISSv2_Core.html#history-filter-operation),
but in this server implementation these data series is temporarily stored within the server, so a "state storage" functionality is sufficient for its needs.

## SQLite state storage
When an SQLite database is used as the state storage it is necessary to prepopulate it with one row for each VSS treeleaf node, having the path name as the key.
To do this there is a [statestorage_mgr](https://github.com/COVESA/ccs-components/tree/master/statestorage/sqlImpl) that takes a file containing a list of all the pathnames,
"vsspathlist.json" as input. This file is generated by the server at startup, taking the paths from the VSS tree that it has access to.

This SQLite DB file then needs to be moved to the WAII/server/visv2server/serviceMgr directory, where it should have the name "statestorage.db"
(if server configuration is not changed to another name).

## Redis state storage
When a Redis database is used as the state storage then there is no explicit database file to handle as the database is managed in-memory by the Redis daemon.
Instead it is necessary to configure and launch the daemon.
The [redisInit.go](https://github.com/COVESA/ccs-components/tree/master/statestorage/redisImpl) file on this link will configure and launch it.
The server uses the same configuration, which needs to be the case also for a feeder to work in concert with the server.
For more information, please check the README on the provided link.

The federclient and serverclient implementations on this link are not needed in this context, they are only meant to make possible some initial testing of using Redis for this,
and as a simple template code base for creating new feeders.

Communication with the Redis daemon is for security reasons configured to use Unix domain sockets. This requires that the socket file, and the directory it is stored in exist.
If not then create it with the commands

$ makedir path-to-socket-file-directory
$ touch socket-file-name