module github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl

go 1.13

//example on how to use replace to point to fork or local path
//replace github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/utils => github.com/MagnusGun/W3C_VehicleSignalInterfaceImpl/utils master
replace github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/utils => ./utils

require (
	github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/server v0.0.0-20191204211610-3716e7ac1e5f // indirect
	github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/utils v0.0.0-20200913070624-f44a7c9498f6
	github.com/PuerkitoBio/goquery v1.5.1 // indirect
	github.com/golang/protobuf v1.4.1 // indirect
	github.com/gorilla/mux v1.7.4 // indirect
	github.com/gorilla/websocket v1.4.2
	github.com/mattn/go-sqlite3 v1.14.3
	github.com/sirupsen/logrus v1.6.0 // indirect
	golang.org/x/net v0.0.0-20200505041828-1ed23360d12c // indirect
	golang.org/x/sys v0.0.0-20200922070232-aee5d888a860 // indirect
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/genproto v0.0.0-20200430143042-b979b6f78d84 // indirect
	google.golang.org/grpc v1.29.1 // indirect
)
