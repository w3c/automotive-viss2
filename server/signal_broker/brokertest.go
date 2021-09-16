package main

import (
	"fmt"
	"os"

	"github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/utils"
	"github.com/akamensky/argparse"
	log "github.com/sirupsen/logrus"
	//	"github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/server/Go/server-1.0/server-core/signal_broker"
)

func main() {

	parser := argparse.NewParser("print", "Prints provided string to stdout")
	// Create string flag
	logFile := parser.Flag("", "logfile", &argparse.Options{Required: false, Help: "outputs to logfile in ./logs folder"})
	logLevel := parser.Selector("", "loglevel", []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}, &argparse.Options{
		Required: false,
		Help:     "changes log output level",
		Default:  "info"})
	// Parse input
	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
	}

	utils.InitLog("brokerlog", "", *logFile, *logLevel)
	// Test connection
	conn, response := GetResponseReceiver()
	defer conn.Close()

	for { // infinit loop
		msg, err := response.Recv() // wait for a subscription msg
		if err != nil {
			log.Debug(" error ", err)
			break
		}

		values := msg.GetSignal()
		asig := values[0]

		// print some signal data to the log ...
		log.Info(asig.Id.Namespace)
		log.Info(asig.Id.Name)
		log.Info(asig.GetDouble(), " ", asig.String())
	}
}
