# signalbroker.go 

The function 
```
  func getHardCodedSignalSettings(vin string)(*settings)
```

returns the two signal tree subsets from the namespaces BodyCANhs and ChassisCANhs. To add remove signals from these , idenfify can frame name and can signal name and add accordingly:(for more info: https://github.com/volvo-cars/signalbroker-server)


```
...
  {Frameid: "VDDMChasFr01",
	Sigids: []signalid{
		{Identifier: "PtTqAtAxleFrntMaxReq_UB"},
		},
  },
...
```

The api contains two functions:

```
func GetResponseReceiver()(*grpc.ClientConn,base.NetworkService_SubscribeToSignalsClient)
func PrintSignalTree(clientconnection *grpc.ClientConn)
```

GetResponseReceiver returns a gprc client connection and response stream. brokertest.go contains an example on how to use.
PrintSignalTree prints the current signal tree to the console.

The constant broker_adress points to the current signal broker ip adress and must be assigned accordingly.

```
const (
	broker_adress = "10.251.177.181:50051"
)
```
