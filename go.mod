module github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl

go 1.13

//example on how to use replace to point to fork or local path
//replace github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/utils => github.com/MagnusGun/W3C_VehicleSignalInterfaceImpl/utils master
//replace github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/utils => ./utils 

require (
	github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/utils v0.0.0-20191209075010-f1ec4a6fa397
	github.com/golang/protobuf v1.3.2
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/websocket v1.4.1
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/sirupsen/logrus v1.4.2
	golang.org/x/net v0.0.0-20191209160850-c0dbc17a3553 // indirect
	golang.org/x/sys v0.0.0-20191210023423-ac6580df4449 // indirect
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/genproto v0.0.0-20191206224255-0243a4be9c8f // indirect
	google.golang.org/grpc v1.25.1
)
