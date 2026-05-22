package mqttclient

import (
	"errors"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/eegfaktura/eegfaktura-backend/model"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type MQTTStreamer struct {
	client mqtt.Client
}
type Error string

var (
	MqttBrokerNotStarted = errors.New("Broker not running")
)

func NewMqttStreamer() (*MQTTStreamer, error) {
	opts := mqtt.NewClientOptions()

	brokerHost := viper.GetString("mqtt.host")
	brokerId := viper.GetString("mqtt.id")

	log.Infof("Use MQTT broker with address %s and Id %s", brokerHost, brokerId)

	opts.AddBroker(brokerHost)
	opts.SetClientID(brokerId)

	opts.SetOrderMatters(true)        // Allow out of order messages (use this option unless in order delivery is essential)
	opts.ConnectTimeout = time.Second // Minimal delays on connect
	opts.WriteTimeout = time.Second   // Minimal delays on writes
	opts.KeepAlive = 10               // Keepalive every 10 seconds so we quickly detect network outages
	opts.PingTimeout = time.Second    // local broker so response should be quick

	// Automate connection management (will keep trying to connect and will reconnect if network drops)
	opts.ConnectRetry = true
	opts.AutoReconnect = true

	// Log events
	opts.OnConnectionLost = func(cl mqtt.Client, err error) {
		log.Info("connection lost")
	}
	opts.OnConnect = func(mqtt.Client) {
		log.Info("MQTT connection established")
	}
	opts.OnReconnecting = func(mqtt.Client, *mqtt.ClientOptions) {
		log.Info("attempting to reconnect")
	}

	client := mqtt.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	return &MQTTStreamer{client: client}, nil
}

func Subscribe(subscriptions ...model.Subscriptions) error {
	if messageBroker != nil {
		messageBroker.Subscribe(subscriptions...)
		return nil
	}
	return MqttBrokerNotStarted
}

func Unsubscribe(subscriptions ...model.Subscriptions) error {
	if messageBroker != nil {
		messageBroker.Unsubscribe(subscriptions...)
		return nil
	}
	return MqttBrokerNotStarted
}

// SendEbmsMessage dispatches an EBMS message to the configured MQTT
// broker. Indirected through a package-level var so tests can swap in
// a capture-mock without touching the real broker — see how the
// ebmsProcessProcessor helpers use the `dispatch` var for the same
// reason.
var SendEbmsMessage = func(msg model.EbmsMessage) error {
	if messageBroker != nil {
		messageBroker.Outbound <- msg

		return nil
	}
	return MqttBrokerNotStarted
}
