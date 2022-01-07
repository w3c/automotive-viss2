module github.com/MEAE-GOT/WAII

go 1.16

//example on how to use replace to point to fork or local path
//replace github.com/MEAE-GOT/WAII/utils => github.com/MagnusGun/WAII/utils master
//replace github.com/MEAE-GOT/WAII/utils => ./utils
//replace (
//	github.com/COVESA/vss-tools/binary/go_parser/datamodel => /home/ulfbjorkengren/Proj/covesa/vss-tools/binary/go_parser/datamodel
//	github.com/COVESA/vss-tools/binary/go_parser/parserlib => /home/ulfbjorkengren/Proj/covesa/vss-tools/binary/go_parser/parserlib
//)

//replace github.com/MEAE-GOT/WAII/protobuf/protoc-out => ./protobuf/protoc-out

require (
	github.com/COVESA/vss-tools/binary/go_parser/datamodel v0.0.0-20211207094201-7208d48f32b6
	github.com/COVESA/vss-tools/binary/go_parser/parserlib v0.0.0-20211207094201-7208d48f32b6
	github.com/MEAE-GOT/WAII/protobuf/protoc-out v0.0.0-20220107173150-7b0d0a7a09f7 
	github.com/akamensky/argparse v1.3.1
	github.com/eclipse/paho.mqtt.golang v1.3.5
	github.com/golang/protobuf v1.5.2
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/mattn/go-sqlite3 v1.14.9
	github.com/sirupsen/logrus v1.8.1
	golang.org/x/net v0.0.0-20211020060615-d418f374d309 // indirect
	golang.org/x/sys v0.0.0-20211117180635-dee7805ff2e1 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20211019152133-63b7e35f4404 // indirect
	google.golang.org/grpc v1.41.0
)
