/*
2021 Peter Winzell, Volvo Cars (c)
*/

package paho_mqtt

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestTcpLocalHost(t *testing.T){
	const TOPIC = "mytopic/test"

	opts := mqtt.NewClientOptions().AddBroker("ssl://localhost:8883")
	opts.SetUsername("homesecurity").SetPassword("rocktheworld")

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		t.Error(token.Error(), "the image needs to run locally for this test to pass")
	}

	var wg sync.WaitGroup
	wg.Add(1)

	if token := client.Subscribe(TOPIC, 0, func(client mqtt.Client, msg mqtt.Message) {
		if string(msg.Payload()) != "mymessage" {
			t.Error("want mymessage, got", msg.Payload())
		}
		wg.Done()
	}); token.Wait() && token.Error() != nil {
		t.Error(token.Error())
	}

	if token := client.Publish(TOPIC, 0, false, "mymessage"); token.Wait() && token.Error() != nil {
		t.Error(token.Error())
	}
}



var r = rand.New(rand.NewSource(time.Now().UnixNano()))

const requestTopic = "vehicle.speed"
const vehicle_id_topic =    "mqttVIN111"

var cloudClientID   = "IMACLOUD"
var vehicleClientID = "IMAVEHICLE"

var vehicle_speed = ""
var done * chan bool

func cloudSimClient(done *chan bool,t *testing.T){
	// subcribe to


	opts := mqtt.NewClientOptions().AddBroker("tcp://localhost:1883")
	opts.SetUsername("homesecurity").SetPassword("rocktheworld")
	opts.SetClientID(cloudClientID)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		t.Error(token.Error(), "the image needs to run locally for this test to pass")
	}

	var wg sync.WaitGroup
	wg.Add(10)

	if token := client.Subscribe(requestTopic, 0, func(client mqtt.Client, msg mqtt.Message) {

		t.Log("speed is ",string(msg.Payload())," topic was", msg.Topic())

		wg.Done() // if we get this 10 times the test is over
	}); token.Wait() && token.Error() != nil {
		t.Error(token.Error())
	}

	//
	if token := client.Publish(vehicle_id_topic, 0, false, requestTopic); token.Wait() && token.Error() != nil {
		t.Error(token.Error())
	}

	wg.Wait()
	*done <- true
}

func vehicleSimClient(done *chan bool,t *testing.T) {

	opts := mqtt.NewClientOptions().AddBroker("tcp://localhost:1883")
	opts.SetUsername("homesecurity").SetPassword("rocktheworld")
	opts.SetClientID(vehicleClientID)

	client := mqtt.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		t.Error(token.Error(), "the image needs to run locally for this test to pass")
	}

	var wg sync.WaitGroup
	wg.Add(1)

	if token := client.Subscribe(vehicle_id_topic, 0, func(client mqtt.Client, msg mqtt.Message) {
		// this checks that the message is the request topic, no need for variable since we are waiting below
		if string(msg.Payload()) == requestTopic{
			t.Log("got ", string(msg.Payload()))
			wg.Done()
		}else{
			t.Error(string(msg.Payload()))
		}
	}); token.Wait() && token.Error() != nil {
		t.Error(token.Error())
	}
	wg.Wait() // we hold here until we get the request...

	func() {
		i := 0
		for {
			if i >10{
				break
			}

			if token := client.Publish(requestTopic, 0, false, vehicle_speed); token.Wait() && token.Error() != nil {
				t.Error(token.Error())
			}
			time.Sleep(1000 * time.Millisecond)
			i++
		}

	}()

}

func canBus() string{
	return fmt.Sprintf("%f",r.Float64() * 200)
}

func getVehicleBusData(){
	for {
		vehicle_speed = canBus()
		time.Sleep(100*time.Millisecond) // update speed every 100 ms
	}
}

func TestVissMqttProtocol(t *testing.T){
	done := make(chan bool, 1)
	go getVehicleBusData()

	go cloudSimClient(&done,t)
	go vehicleSimClient(&done,t)

	<- done
}

