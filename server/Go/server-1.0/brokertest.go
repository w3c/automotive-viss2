package main

import(
	log "github.com/sirupsen/logrus"
	"W3C_VehicleSignalInterfaceImpl/server/Go/server-1.0/server-core/util"
	"W3C_VehicleSignalInterfaceImpl/server/Go/server-1.0/server-core/signal_broker"
)

func main() {
	util.InitLogger()
	// Test connection
	conn,response := signal_broker.GetResponseReceiver();
	defer conn.Close();

	for { // infinit loop
		msg, err := response.Recv(); // wait for a subscription msg
		if (err != nil) {
			log.Debug(" error ", err);
			break;
		}

		values := msg.GetSignal();
		asig := values[0];

		// print some signal data to the log ...
		log.Info(asig.Id.Namespace);
		log.Info(asig.Id.Name);
		log.Info( asig.GetDouble()," ", asig.String());
	}
}