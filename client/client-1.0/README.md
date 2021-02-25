**(C) 2021 Geotab Inc**<br>

# MQTT client

The MQTT client is a rudimentary implementation to allow testing VISS v2 request/response communication over MQTT using the proposed application level protocol that is described in the <a href="https://github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/tree/master/server/mqtt_mgr">server/mqtt_mgr</a> directory.<br>
The MQTT client is started as shown below<br>
$ ./mqtt_mgr VIN<br>
where VIN is the VIN number associated to the VSS path "Vehicle.VehicleIdentification.VIN".<br>
This VIN should probably only be used in test cases like this, otherwise a "pseudo-VIN" is probably more appropriate from a privacy point of view.<br>
In the current implementation the VISSv2 server mqtt manager reads the VIN using the path above, and then issues a subcribe on the topic "VIN/Vehicle". This topic must be known by the MQTT client for the protocol to work.<br>
In the current server implementation a random dummy value is returned if the service manager at start up does not find a "statestorage.db" database, AND that this database has a value stored for the addressed path. So, to simplify testing with the MQTT client, it is recommended that a statestorage.db is created, with at least a value, plus corresponding timestamp, written into it. Then this value is what shall be used when starting as mentioned above.<br>
To create such a statestorage database, please clone the <a href="https://github.com/GENIVI/ccs-w3c-client">ccs-w3-client</a> repo, and follow the instructions <a href="https://github.com/GENIVI/ccs-w3c-client/tree/master/statestorage">statestorage</a> directory.

