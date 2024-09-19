package nats

import (
	"encoding/json"
	"fmt"

	ha "code.jhot.me/jhot/hats/pkg/homeassistant"
	"github.com/nats-io/nats.go"
)

type TopicType string

const StateTopicType TopicType = "states"
const AttributeTopicType TopicType = "attributes"
const TimerTopicType TopicType = "timers"
const ScheduleTopicType TopicType = "schedules"
const CommandTopicType TopicType = "command"
const NfcTopicType TopicType = "nfc"
const ZhaTopicType TopicType = "zha"
const ZwaveTopicType TopicType = "zwave"
const ZwaveSceneTopicType TopicType = "zwave-scene"

func (t TopicType) GetTopic(id string) string {
	switch t {
	case StateTopicType:
		return fmt.Sprintf("homeassistant.%s.%s.>", t, id)
	case AttributeTopicType:
		return fmt.Sprintf("homeassistant.%s.%s.>", t, id)
	case TimerTopicType:
		return fmt.Sprintf("homeassistant.%s.%s.finished", t, id)
	case ScheduleTopicType, CommandTopicType:
		return fmt.Sprintf("%s.%s", t, id)
	case NfcTopicType, ZhaTopicType, ZwaveTopicType, ZwaveSceneTopicType:
		return fmt.Sprintf("homeassistant.%s.%s", t, id)
	}
	return ""
}

func (n *NatsConnection) addSub(topic string, cb nats.MsgHandler) (*nats.Subscription, error) {
	return n.Conn.Subscribe(topic, cb)
}

func parseEvent(msg *nats.Msg, sendResponse bool) (ha.EventData, error) {
	if sendResponse {
		go msg.Respond([]byte("got it"))
	}
	var data ha.EventData
	err := json.Unmarshal(msg.Data, &data)
	return data, err
}

func (n *NatsConnection) AddEventSub(t TopicType, entityId string, handler func(ha.EventData) error) (*nats.Subscription, error) {
	if t == "" || entityId == "" {
		panic("TopicType and entityId cannot be empty")
	}

	topic := t.GetTopic(entityId)
	l := n.logger.With("topic", topic, "entity_id", entityId)
	l.Debug("Subscribing to topic")

	return n.addSub(topic, func(msg *nats.Msg) {
		data, err := parseEvent(msg, true)
		if err != nil {
			l.Error("Error parsing message", "error", err)
			return
		}
		l.Debug(fmt.Sprintf("%s data event fired", t))
		err = handler(data)
		if err != nil {
			l.Error(fmt.Sprintf("Error handling %s event", t), "error", err)
			return
		}
	})
}

func (n *NatsConnection) AddStateSub(t TopicType, entityId string, handler func(ha.StateData) error) (*nats.Subscription, error) {
	if t == "" || entityId == "" {
		panic("TopicType and entityId cannot be empty")
	}

	topic := t.GetTopic(entityId)
	l := n.logger.With("topic", topic, "entity_id", entityId)
	l.Debug("Subscribing to topic")

	return n.addSub(topic, func(msg *nats.Msg) {
		data, err := parseEvent(msg, true)
		if err != nil {
			l.Error("Error parsing message", "error", err, "message", string(msg.Data))
			return
		}
		l.Debug(fmt.Sprintf("%s state event fired", t), "newState", data.NewState.State)
		err = handler(data.NewState)
		if err != nil {
			l.Error(fmt.Sprintf("Error handling %s event", t), "error", err)
			return
		}
	})
}
