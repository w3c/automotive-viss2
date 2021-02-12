/**
* (C) 2021 Geotab
*
* All files and artifacts in the repository at https://github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/
package main

import (
	"strconv"
	"os"
	"fmt"

	"github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/utils"
  MQTT  "github.com/eclipse/paho.mqtt.golang"
)

var requestNo int
var uniqueTopicName string


func getBrokerSocket(isSecure bool) string {
//	FVTAddr := os.Getenv("MQTT_BROKER_ADDR")
        FVTAddr := "test.mosquitto.org"   // does it work for testing?
//        FVTAddr := "mqtt.flespi.io"
	if FVTAddr == "" {
		FVTAddr = "127.0.0.1"
	}
	if (isSecure == true) {
	    return "ssl://" + FVTAddr + ":8883"
        } 
	return "tcp://" + FVTAddr + ":1883"
}

var publishHandler MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
    fmt.Printf("Topic=%s\n", msg.Topic())
    fmt.Printf("Payload=%s\n", string(msg.Payload()))
}

func mqttSubscribe(brokerSocket string, topic string) MQTT.Client {
    fmt.Printf("mqttSubscribe:Topic=%s\n", topic)
    opts := MQTT.NewClientOptions().AddBroker(brokerSocket)
//    opts.SetClientID("VIN001-Client")
    opts.SetDefaultPublishHandler(publishHandler)

    c := MQTT.NewClient(opts)
    if token := c.Connect(); token.Wait() && token.Error() != nil {
        panic(token.Error())
    }
    if token := c.Subscribe(topic, 0, nil); token.Wait() && token.Error() != nil {
        utils.Error.Println(token.Error())
        os.Exit(1)
    }
    return c
}

func mqttUnsubscribe(mqttClient *(MQTT.Client)) {
    (*mqttClient).Disconnect(250)
}

func publishMessage(brokerSocket string , topic string, payload string) {   
    fmt.Printf("publishMessage:Topic=%s, Payload=%s\n", topic, payload)
    opts := MQTT.NewClientOptions().AddBroker(brokerSocket)
//    opts.SetClientID("VIN001")

    c := MQTT.NewClient(opts)
    if token := c.Connect(); token.Wait() && token.Error() != nil {
        utils.Error.Println(token.Error())
        os.Exit(1)
    }
    token := c.Publish(topic, 0, false, payload)
    token.Wait()
    c.Disconnect(250)
}

func getVissV2Response(brokerSocket string, vin string, request string) *(MQTT.Client) {
    uniqueTopic := uniqueTopicName + strconv.Itoa(requestNo)
    client := mqttSubscribe(brokerSocket, uniqueTopic)
    
    payload := `{"topic":"` + uniqueTopic + `", "request":` + request + `}`
//    payload := `{"topic":"` + uniqueTopic + `", "request":"` + request + `"}`
    publishMessage(brokerSocket, "/" + vin + "/Vehicle", payload)
    requestNo++
    return &client
}

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("MQTT client command line: ./mqtt_client vin\n")
		os.Exit(1)
	}
	vin := os.Args[1]
	utils.TransportErrorMessage = "MQTT client-finalizeResponse: JSON encode failed.\n"
	utils.InitLog("mqtt-client-log.txt", "./logs")

	brokerSocket := getBrokerSocket(false)
	
    var request string
    clientSubscription := make([]*(MQTT.Client), 25)
    i := 0
    continueLoop := true
    fmt.Printf("\nSet unique topic name:")
    fmt.Scanf("%s", &uniqueTopicName)
    for continueLoop {
        fmt.Printf("\nVISSv2-request (or q to quit):")
        fmt.Scanf("%s", &request)
        switch request[0] {
          case 'q': continueLoop = false
          default:
	      clientSubscription[i] = getVissV2Response(brokerSocket, vin, request)
        }
        i++
        if (i == 25) {
            fmt.Printf("Max number of requests reached. Goodbye.\n")
            continueLoop = false
        }
    }
    for j := 0 ; j < i ; j++ {
        mqttUnsubscribe(clientSubscription[j])
    }
}
