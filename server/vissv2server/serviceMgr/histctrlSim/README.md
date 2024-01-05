**(C) 2023 Ford Motor Company**<br>
**(C) 2020 Geotab Inc**<br>

All files and artifacts in this repository are licensed under the provisions of the license provided by the LICENSE file in this repository.

# Historic data
A Go routine for handling of historic data is spawned at server start up. The Unix domain socket server that it realizes can be used by a vehicle subsystem to control the recording of data for one o more signals. The Unix domain socket file address is /var/tmp/vissv2/histctrlserver.sock.<br>
The payload structures that is available for a client to issue to the server are:<br>
1. {"action":"create", "path": X, "buf-size":"Y"}<br>
2. {"action":"start", "path": X, "freq":"Z"}<br>
3. {"action":"stop", "path": X}<br>
4. {"action":"delete", "path": X}<br>
where X can be a single path "x.y.z", or an array of paths ["a.b.c", ..., "x.y.z"], Y is the max number of samples that can be buffered, which must be less than 65535, and Z is the capture frequency in captures per hour, which must be less than 65535.<br>

The create request leads to the creation of a buffer of the size requested.<br>
The start request initates capture of samples at the set frequency until the buffer is full.<br>
The stop request halts the capture of samples.<br>
The delete request discards the buffer.<br>

If a VISSv2 client issues a request for historic data, specifying a period from now and backwards in time, then the service manager will check if there is historic data saved, and select the part that matches the requested period. If there is no data saved, then the response will only contain the latest data point. 

This architecture supports a use case where a high frequency capture rate is applied to the battery voltage during cranking of the starter motor. The vehile can then start the saving of this data at a high capture frequency, and then issue a stop command when the motor has started. This data can then be available for some time so that a client has a resonable time to issue a request for it.<br>

Another use case could be that the vehicle temporarily loses its connection, maybe due to passage through a tunnel. If this is detected by the vehicle telematics unit, it may issue a request over the History control interface to start saving multiple selected signals, but with buf-size set to zero. The later means that the buffer size is automatically increased when it becomes full. 
The saving of data will automatically stop at some max limit if no stop command is issued before that.

A third use case could be that data related to electrical charging shall be saved, the vehicle system then uses the start and stop commands to record the appropriate signals during the charging session.

# History control simulator
The hist_ctrl_client.go can be used to simulate a vehicle subsystem that is responsible for controlling the VISSv2 server's functionality for recording historic data.
Its UI can be used to issue the commands described above to control the server to record historic data that can then be retrieved by a client issuing a historic data request.

To use the History control simulator, open a terminal window, go to the directory where it resides, build it by issuing<br>
go build<br>
and then start it by the command<br>
./histctrlSim
