module github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl

go 1.13

//example on how to use replace to point to fork or local path
//replace github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/utils => github.com/MagnusGun/W3C_VehicleSignalInterfaceImpl/utils master
//replace github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/utils => ./utils

require (
	github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/utils v0.0.0-20200503093621-4e8edac3582d
	github.com/golang/protobuf v1.4.0
	github.com/gorilla/mux v1.7.4
	github.com/gorilla/websocket v1.4.2
	github.com/sirupsen/logrus v1.6.0
	golang.org/x/net v0.0.0-20200501053045-e0ff5e5a1de5 // indirect
	golang.org/x/sys v0.0.0-20200501145240-bc7a7d42d5c3 // indirect
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/genproto v0.0.0-20200430143042-b979b6f78d84 // indirect
	google.golang.org/grpc v1.29.1
)
