**(C) 2023 Ford Motor Company**
**(C) 2023 Volvo Cars**

# gRPC client

To build:

$ go build

To run:

./grpc_client

The gRPC client UI provides a choice of four different request that can be issued:

```
{"action":"get","path":"Vehicle/Chassis/Accelerator/PedalPosition","requestId":"232"}
{"action":"subscribe","path":"Vehicle/Speed","filter":{"type":"timebased","parameter":{"period":"100"}},"requestId":"246"}
{"action":"unsubscribe","subscriptionId":"1","requestId":"240"}
{"action":"set", "path":"Vehicle/Body/Lights/IsLeftIndicatorOn", "value":"999", "requestId":"245"}
```

These can be issued multiple times, but there is a limitation in that the unsubscribe has a static subscriptionID that only applies to the first started subscription.
