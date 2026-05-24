package mqttclient

import (
	"encoding/json"
	"fmt"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/eegfaktura/eegfaktura-backend/model"
	log "github.com/sirupsen/logrus"
)

type TopicType string

var messageBroker *MessageBroker

func (t TopicType) Tenant() string {
	elems := strings.Split(string(t), "/")
	if len(elems) > 4 {
		return elems[2]
	}
	return string(t)
}

func (t TopicType) TypeInfo() (string, string) {
	elems := strings.Split(string(t), "/")
	if len(elems) > 4 {
		return elems[2], elems[4]
	}
	return string(t), ""
}

type InboundMessage struct {
	tenant   string
	protocol model.EdaProtocol
	msg      []byte
}

type MessageBroker struct {
	callbackStore map[model.EdaProtocol]model.SubscribeHandler
	Inbound       chan InboundMessage
	Outbound      chan model.EbmsMessage
	*MQTTStreamer
}

func StartMessageBroker() error {
	in := make(chan InboundMessage)
	out := make(chan model.EbmsMessage)

	streamer, err := NewMqttStreamer()
	if err != nil {
		return err
	}
	messageBroker = &MessageBroker{make(map[model.EdaProtocol]model.SubscribeHandler), in, out, streamer}

	go messageBroker.Listen()

	return nil
}

func NewMessageBroker() (*MessageBroker, error) {
	in := make(chan InboundMessage)
	out := make(chan model.EbmsMessage)

	streamer, err := NewMqttStreamer()
	if err != nil {
		return nil, err
	}
	return &MessageBroker{make(map[model.EdaProtocol]model.SubscribeHandler), in, out, streamer}, nil
}

func (mb *MessageBroker) SendMessage(m model.EbmsMessage, callback func(m string) error) {
	log.WithField("MSG", m.MessageCode).Info("Send Message to MQTT")
	payload, err := json.Marshal(m)
	if err != nil {
		log.WithField("error", err).Error("Marshaling EbmsMessage")
	}
	token := mb.client.Publish("eda/request", 1, false, payload)
	go func() {
		<-token.Done()
		if token.Error() != nil {
			log.Errorf("MQTT ERROR PUBLISHING: %s\n", token.Error())
		}
	}()
	token.Wait()
	callback("message sent")
}

func (mb *MessageBroker) Listen() {
	qos := 0
	token := mb.client.Subscribe("eda/response/+/protocol/#", byte(qos), func(client mqtt.Client, msg mqtt.Message) {
		log.Infof("Message from MQTT: %s [%+v]\n", TopicType(msg.Topic()).Tenant(), msg.Topic())
		tenant, protocol := TopicType(msg.Topic()).TypeInfo()
		mb.Inbound <- InboundMessage{
			strings.ToUpper(tenant),
			model.EdaProtocol(strings.ToUpper(protocol)),
			msg.Payload()}
	})
	token.Wait()
	if token.Error() != nil {
		panic(token.Error())
	}

	for {
		select {
		case msg := <-mb.Inbound:
			log.Infof("Message on topic: %s", msg.protocol)
			mb.received(msg)
		case send := <-mb.Outbound:
			mb.SendMessage(send, func(m string) error {
				fmt.Printf("Callback called: %+v\n", m)
				return nil
			})
		}
	}
}

func (mb *MessageBroker) Subscribe(subscriptions ...model.Subscriptions) {
	for _, s := range subscriptions {
		mb.callbackStore[s.Protocol] = s.Handler
	}
}

func (mb *MessageBroker) Unsubscribe(subscriptions ...model.Subscriptions) {
	for _, s := range subscriptions {
		delete(mb.callbackStore, s.Protocol)
	}
}

func (mb *MessageBroker) received(inbound InboundMessage) {
	msg := model.EbmsMessage{}
	err := json.Unmarshal(inbound.msg, &msg)
	if err != nil {
		log.Errorf("Error from MQTT: (%s) %v - %v", inbound.tenant, inbound.protocol, err)
		return
	}
	c, ok := mb.callbackStore[inbound.protocol]
	if ok {
		c(model.SubscribeMessage{
			Protocol:    inbound.protocol,
			MessageCode: msg.MessageCode,
			Tenant:      inbound.tenant,
			Payload:     msg,
		})
	}
}
