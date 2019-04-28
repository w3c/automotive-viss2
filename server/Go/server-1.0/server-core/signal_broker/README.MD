# signalbroker.go 

The function 
```
  func getHardCodedSignalSettings(vin string)(*settings)
```

returns the two signal tree subsets from the namespaces BodyCANhs and ChassisCANhs. To add remove signals from these , idenfify can frame name and can signal name and add 
```
...
  {Frameid: "VDDMChasFr01",
						Sigids: []signalid{
							{Identifier: "PtTqAtAxleFrntMaxReq_UB"},
						},
					},
...
```
The api contains only one function:

```

```




"10.251.177.181:50051"
