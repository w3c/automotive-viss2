module github.com/w3c/automotive-viss2

go 1.18

//example on how to use replace to point to fork or local path
//replace github.com/w3c/automotive-viss2/utils => github.com/MagnusGun/WAII/utils master
replace github.com/w3c/automotive-viss2/utils => ./utils

replace (
	github.com/COVESA/vss-tools/binary/go_parser/datamodel => github.com/UlfBj/vss-tools/binary/go_parser/datamodel v0.0.0-20220524163944-c753a539973f
	github.com/COVESA/vss-tools/binary/go_parser/parserlib => github.com/UlfBj/vss-tools/binary/go_parser/parserlib v0.0.0-20220524163944-c753a539973f
	github.com/w3c/automotive-viss2/grpc_pb => ./grpc_pb
	github.com/w3c/automotive-viss2/server/vissv2server/atServer => ./server/vissv2server/atServer
	github.com/w3c/automotive-viss2/server/vissv2server/grpcMgr => ./server/vissv2server/grpcMgr
	github.com/w3c/automotive-viss2/server/vissv2server/httpMgr => ./server/vissv2server/httpMgr
	github.com/w3c/automotive-viss2/server/vissv2server/mqttMgr => ./server/vissv2server/mqttMgr
	github.com/w3c/automotive-viss2/server/vissv2server/serviceMgr => ./server/vissv2server/serviceMgr
	github.com/w3c/automotive-viss2/server/vissv2server/wsMgr => ./server/vissv2server/wsMgr
)

//replace github.com/w3c/automotive-viss2/protobuf/protoc-out => ./protobuf/protoc-out

require (
	github.com/COVESA/vss-tools/binary/go_parser/datamodel v0.0.0-20220104185813-cad8492de65f
	github.com/COVESA/vss-tools/binary/go_parser/parserlib v0.0.0-20220104185813-cad8492de65f
	github.com/akamensky/argparse v1.3.1
	github.com/eclipse/paho.mqtt.golang v1.3.5
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/go-redis/redis/v8 v8.11.5
	// github.com/golang/protobuf v1.5.3
	github.com/google/uuid v1.3.1
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.4.2
	github.com/mattn/go-sqlite3 v1.14.14
	github.com/petervolvowinz/viss-rl-interfaces v0.0.8
	github.com/sirupsen/logrus v1.9.3
	google.golang.org/grpc v1.60.0
	google.golang.org/protobuf v1.31.0
)

require github.com/golang/protobuf v1.5.3

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	golang.org/x/net v0.16.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231002182017-d307bd883b97 // indirect
)
