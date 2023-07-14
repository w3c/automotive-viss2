**(C) 2023 Ford Motor Company**

# gRPC client

To build:

$ go build

To run:

./grpc_client

The gRPC client UI provides a choice of four different request that can be issued:

```
{"action":"get","path":"Vehicle/Cabin/Door/Row1/Right/IsOpen","requestId":"232"}
{"action":"subscribe","path":"Vehicle/Cabin/Door/Row1/Right/IsOpen","filter":{"type":"timebased","parameter":{"period":"3000"}},"requestId":"246"}
{"action":"unsubscribe","subscriptionId":"1","requestId":"240"}
{"action":"set", "path":"Vehicle/Cabin/Door/Row1/Right/IsOpen", "value":"999", "requestId":"245"}
```

These can be issued multiple times, but there is a limitation in that the unsubscribe has a static subscriptionID that only applies to the first started subscription.
