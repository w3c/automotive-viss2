**(C) 2020 Geotab Inc**<br>

All files and artifacts in this repository are licensed under the provisions of the license provided by the LICENSE file in this repository.

# VISS v2 service manager

At statup the VISSv2 service manager tries to read a DB file named statestorage.db.
If successful, then the signal value(s) being requested will first be searched for in this DB, if not found then dummy values will be returned instead. 
Dummy values are always an integer, taken from a counter that is incremented every 37 msec, and wrapping to stay within the values 0 to 999.

If a request contains an array of paths, then the response/notification will include values related to all elements of the array. 

The service manager will do its best to interpret subscription filter expressions, but if unsuccessful it will return an error response without activating a subscription session.

The figure shows the internal architecture of the service manager when it comes to handling of request for historic data and use of curve logic.
![Service manager time-series architecure](servicemgr-timeseries-architecture.jpg)<br>

Each request for a curve logic subscription instantiates a Go routine that handles the request. An unsubscribe request kills the Go routine.<br>

A Go routine for handling of historic data is spawned at server start up. The vehicle system can via the History control interface control the saving of data for one o more signals via a Unix Domain Socket command with the socket address /tmp/vissv2/histctrlserver.sock.<br>
The write commands available are:<br>
1. {"action":"create", "path": X, "buf-size":"Y"}<br>
2. {"action":"start", "path": X, "freq":"Z"}<br>
3. {"action":"stop", "path": X}<br>
4. {"action":"delete", "path": X}<br>
where X can be a single path "x.y.z", or an array of paths ["a.b.c", ..., "x.y.z"], Y is the max number of samples that can be buffered, which must be less than 65535, and Z is the capture frequency in captures per hour, which must be less than 65535.<br>

The create request leads to the creation of a buffer of the size requested.<br>
The start request initates capture of samples at the set frequency until the buffer is full.<br>
The stop request halts the capture of samples.<br>
the delete request discards the buffer.<br>

Data is captured from the statestorage, and it is only saved in the buffer if the timestamp differs from the previously latest saved. This polling paradigm may be replaces by an event driven paradigm if/when the statestorage supports it. With this polling paradigm, the capture frequency to be set must be higher than the actual update frequency of the signal in the statestorage. Other system latencies should also be taken into account when selecting this frequency as the frequency sets the sleep time in the capture loop.

If a client issues a request for historic data, specifying a period from now and backwards in time, then the service manager will check if there is historic data saved, and select the part that matches the requested period. If there is no dat saved, then the response will only contain the latest data point. 

This architecture supports a use case where a high frequency capture rate is applied to the battery voltage during cranking of the starter motor. The vehile can then stat saving of this data at a high capture frequency, and then issue a stop command when the motor has started. This data can then be available for some time so that a client has a resonable time to issue a request for it.<br>

Another use case could be that the vehicle temporarily loses its connection, maybe due to passage through a tunnel. If this is detected by the vehicle telematics unit, it may issue a request over the History control interface to start saving multiple selected signals, but with buf-size set to zero. The later means that the buffer size is automatically increased when it becomes full. 
The saving of data will automatically stop at some max limit if no stop command is issued before that.

A third use case could be that data related to electrical charging shall be saved, the vehicle system then uses the start and stop commands to record the appropriate signals during the charging session.

