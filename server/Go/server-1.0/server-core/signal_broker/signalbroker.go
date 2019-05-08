package signal_broker

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"W3C_VehicleSignalInterfaceImpl/server/Go/server-1.0/server-core/proto_files"
	log "github.com/sirupsen/logrus"
	// "W3C_VehicleSignalInterfaceImpl/server/Go/server-1.0/server-core/util"
)

const (
	broker_adress = "10.251.177.181:50051"
)
// print current configuration to the console
func PrintSignalTree(clientconnection *grpc.ClientConn) {
	systemServiceClient := base.NewSystemServiceClient(clientconnection);
	configuration,err := systemServiceClient.GetConfiguration(context.Background(),&base.Empty{})

	infos := configuration.GetNetworkInfo();
	for _,element := range infos{
		printSignals(element.Namespace.Name,clientconnection);
	}

	if err != nil{
		log.Debug("could not retrieve configuration " , err);
	}

}

// print signal tree(s) to console , using fmt for this.
func printSpaces(number int){
	for k := 1; k < number; k++ {
		fmt.Print(" ");
	}
}

func printTreeBranch(){
	fmt.Print("|");
}

func getFirstNameSpace(frames []*base.FrameInfo) string{
	element := frames[0];
	return element.SignalInfo.Id.Name;
}

func printSignals(zenamespace string,clientconnection *grpc.ClientConn){
	systemServiceClient := base.NewSystemServiceClient(clientconnection)
	signallist, err := systemServiceClient.ListSignals(context.Background(),&base.NameSpace{Name : zenamespace})

	frames := signallist.GetFrame();

	rootstring := "|[" + zenamespace + "]---|";
	rootstringlength := len(rootstring);
	fmt.Println(rootstring);

	for _,element := range frames{

		printTreeBranch();
		printSpaces(rootstringlength -1);

		framestring := "|---[" + element.SignalInfo.Id.Name + "]---|";
		framestringlength := len(framestring);

		fmt.Println(framestring);
		childs := element.ChildInfo;

		for _,childelement := range childs{
			outstr := childelement.Id.Name;
			printTreeBranch();
			printSpaces(rootstringlength -1);
			printTreeBranch();
			printSpaces(framestringlength - 1);
			fmt.Println("|---{", outstr, "}");
		}
	}

	if err != nil {
		log.Debug(" could not list signals ", err);
	}
}

type signalid struct{
	Identifier string
}

type framee struct{
	Frameid string `json:frameid`
	Sigids []signalid `json:sigids`
}

type spaces struct{
	Name  string `json:name`
	Frames []framee `json:framee`
}

type settings struct{
	Namespaces []spaces `json:namespaces`
}

type VehiclesList struct{
	Vehicles []settings `json:vehicles`
}

func getHardCodedSignalSettings(vin string)(*settings){
	data := &settings{
		Namespaces: []spaces{
			{Name: " BodyCANhs",
				Frames: []framee{
					{Frameid: "DDMBodyFr01",
						Sigids: []signalid{
							{Identifier: "ChdLockgProtnFailrStsToHmi_UB"},
							{Identifier: "ChdLockgProtnStsToHmi_UB"},
							{Identifier: "DoorDrvrLockReSts_UB"},
							{Identifier: "ChdLockgProtnFailrStsToHmi"},
							{Identifier: "WinPosnStsAtDrvrRe"},
						}},
					{Frameid: "PAMDevBodyFr09",
						Sigids: []signalid{
							{Identifier: "DevDataForPrkgAssi9Byte0"},
							{Identifier: "DevDataForPrkgAssi9Byte1"},
							{Identifier: "DevDataForPrkgAssi9Byte2"},
						},
					},
				},
			},
			{Name: "ChassisCANhs",
				Frames: []framee{
					{Frameid: "SASChasFr01",
						Sigids: []signalid{
							{Identifier: "SteerWhlAgSafe"},
						}},
					{Frameid: "VDDMChasFr01",
						Sigids: []signalid{
							{Identifier: "PtTqAtAxleFrntMaxReq_UB"},
						},
					},
					{Frameid: "VDDMChasFr09",
						Sigids: []signalid{
							{Identifier: "TqRednDurgCllsnMtgtnByBrkg_UB"},
						},
					},
					{Frameid: "PSCMChasFr01",
						Sigids: []signalid{
							{Identifier: "PinionSteerAg1"},
						},
					},
					{Frameid: "VDDMChasFr23",
						Sigids: []signalid{
							{Identifier: "SteerWhlHeatOnReq"},
							{Identifier: "SteerWhlHeatOnReq_UB"},
						},
					},
					{Frameid: "EM_ChasFr05",
						Sigids: []signalid{
							{Identifier: "EngSpdDispd"},
						},
					},
					{Frameid: "VDDMChasFr06",
						Sigids: []signalid{
							{Identifier: "DoorPassSts"},
						},
					},

				},
			},
		},
	}

	return data;
}

func getSignaId(signalName string,namespaceName string) *base.SignalId{
	return &base.SignalId{
		Name: signalName,
		Namespace:&base.NameSpace{
			Name:namespaceName},
	}
}

func readVehicleSettingsFromDB(vin string)(string,*base.SubscriberConfig){

	data := getHardCodedSignalSettings(vin); // this should be replaced by an actual call to a vehicle settings db.


	var signalids []*base.SignalId;
	var namespacename string;

	// loop over namespace indices and get namespaces and finally the signal subscribing set.
	for cindex := 0; cindex < len(data.Namespaces); cindex++{
		namespacename = data.Namespaces[cindex].Name;
		for _,frameelement := range data.Namespaces[cindex].Frames{
			for _,sigelement := range frameelement.Sigids{
				log.Info("subscribing signals " , sigelement);
				signalids = append(signalids,getSignaId(sigelement.Identifier,namespacename));
			}
		}
	}

	log.Info(signalids)
	// special test

	signals := &base.SubscriberConfig{
		ClientId: &base.ClientId{
			Id: "app_identifier",
		},
		Signals: &base.SignalIds{
			SignalId:signalids,
		},
		OnChange: false,
	}

	// assign namespace to first
	namespacename = data.Namespaces[0].Name;
	return namespacename, signals
}

func GetResponseReceiver()(*grpc.ClientConn,base.NetworkService_SubscribeToSignalsClient){
	conn, err := grpc.Dial(broker_adress,grpc.WithInsecure())
	if (err != nil){
		log.Debug("could not connect to broker", err);
	}

	_, signals := readVehicleSettingsFromDB("sommevin");
	PrintSignalTree(conn);
	c := base.NewNetworkServiceClient(conn);

	response, err := c.SubscribeToSignals(context.Background(),signals);
	return conn,response;
}

