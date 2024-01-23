# Feeder templates
The feeder templates are meant to be used as a starting point for development of feeders for different southbound/vehicle interfaces.
The interface client on the southbound side is all that needs to be modified to support the new interface,
all other functionality in the feeder is already implemented.

There are two versions of the template, mainly due to legacy reasons.
Feeders are built and run as separate executables.<br>
$ go build<br>
$ ./name-of-executable

Input files with other names than the default can be configured as start up command parameters.

Version 1 is limited to support only the signal name mapping, and no scaling.
This is thought to be used only for initial testing.

Version 2 supports both signal name mapping and scaling, and it relies on the Domain Conversion Tool (DCT) to create the mapping and scaling instructions that it uses.

It is the hope that feeders supporting new vehicle interfaces are upstreamed to the repo.

## Version 1
The VehicleVssMapData.json contains an example of the signal name mapping. 
The format must be an array of JSON objects, where each object contains two key-value pairs holding the signal names that are mapped.
This format
```
[{"vssdata":"vssname1","vehicledata":"vehiclename1"}, ..., {"vssdata":"vssnameN","vehicledata":"vehiclenameN"}]
```
must be used to represent the mapping.
Manual editing of this file is needed to update the mapping.
The feeder reads this file a startup.

## Version 2
This version requires that the Domain Conversion Tool is used to create the feeder conversion instructions that the feeder reads at startup.
The two fiels that it reads are:
* VssVehicle.cvt
* VssVehicleScaling.json

The first file contains the primary conversion instructions, aving the format of an array of structs, each with the following format:<br>
```
struct {
	MapIndex uint16
	Name string
	Type int8
	Datatype int8
	ConvertIndex uint16
}
```
Each struct element contains data associated to one signal of any of the two domains.
The MapIndex links it logically to another element in the array, which is the corresponding signal of the other domain tat it is mapped to.
This array can then by the feeder be used to search for signal names (Name element in the struct) of any of the two domains, 
and as the array is sorted on the struct Name element, a binary search algorithm can be used.
The scaling operation is controlled by the ConvertIndex of the struct.
An index of zero is interpreted by the feeder as no scaling needed (one-to-one),
any other number, except 65535 that indicates a DCT mapping error, is set to point to an element of the scaling array that the feeder read from the file VssVehicleScaling.json at startup.
A string element in this array is after being addressed by the ConvertIndex interpreted as either a JSON object containing a list of key-value pairs, or a JSON number array.<br>
In the case of a JSON object, each key-value pair represents the associated scaling values of the "allowed" values from repsective domain.<br>
In the case of a JSON number array, the two elements of the array represents the A and B coefficients of the equation y = A*x + B (or y = (x-B)/A in the other direction).
The struct Datatype can be used to reformat if needed after a linear conversion that is always calculated using float64.
