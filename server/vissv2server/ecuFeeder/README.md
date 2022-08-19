# ECU feeder

The ECU feeder is configured by reading the feederServicesList.json. 
This file declares the services that it will access, out of what is available via the ECU HAL API. 
For each service it declares how the signals that the service provides access to are mapped to corresponding VSS signals, by providing the VSS path for each signal. 
It is important the the order that the signals are provided aligns with the order that the corresponding interface handler in ECUFeeder.go processes them.<br>

The ECU feeder initiates the service, and then writes the signal stream into the state storage as signals are received from the vehicle system. 
Both the Redis and the SQLite state storage implementations are supported.
