/*
2021 Peter Winzell, Volvo Cars (c)
*/

package paho_mqtt

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"sync"
	"testing"
)

func TestTcpLocalHost(t *testing.T) {
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
