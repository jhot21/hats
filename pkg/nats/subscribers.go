package nats

import (
	"encoding/json"
	"fmt"

	ha "codeberg.org/jhot/hats/pkg/homeassistant"
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

// subscribe is the internal generic helper for all typed subscribers.
// It handles respond, parse, logging, and error handling.
func subscribe[T any](
	n *NatsConnection,
	topic string,
	parse func(*nats.Msg) (T, error),
	handler func(T) error,
) (*nats.Subscription, error) {
	l := n.logger.With("topic", topic)
	l.Debug("Subscribing to topic")
	return n.addSub(topic, func(msg *nats.Msg) {
		go msg.Respond([]byte("got it"))
		data, err := parse(msg)
		if err != nil {
			l.Error("Error parsing message", "error", err)
			return
		}
		if err := handler(data); err != nil {
			l.Error("Error handling event", "error", err)
		}
	})
}

// subscribeNoData is the internal helper for subscribers with no payload (timers, schedules).
func subscribeNoData(n *NatsConnection, topic string, handler func() error) (*nats.Subscription, error) {
	l := n.logger.With("topic", topic)
	l.Debug("Subscribing to topic")
	return n.addSub(topic, func(msg *nats.Msg) {
		go msg.Respond([]byte("got it"))
		if err := handler(); err != nil {
			l.Error("Error handling event", "error", err)
		}
	})
}

// parseEventData parses a NATS message into ha.EventData.
func parseEventData(msg *nats.Msg) (ha.EventData, error) {
	var data ha.EventData
	return data, json.Unmarshal(msg.Data, &data)
}

// parseStateData parses a NATS message into ha.StateData (new state only).
func parseStateData(msg *nats.Msg) (ha.StateData, error) {
	var event ha.EventData
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		return ha.StateData{}, err
	}
	return event.NewState, nil
}

// parseCommandPayload parses a NATS message into a CommandPayload.
func parseCommandPayload(msg *nats.Msg) (CommandPayload, error) {
	var p CommandPayload
	return p, json.Unmarshal(msg.Data, &p)
}

// SubscribeState subscribes to state change events for the given entity.
// The handler receives only the new state.
func (n *NatsConnection) SubscribeState(entityId string, handler func(ha.StateData) error) (*nats.Subscription, error) {
	return subscribe(n, StateTopicType.GetTopic(entityId), parseStateData, handler)
}

// SubscribeStateEvent subscribes to state change events for the given entity.
// The handler receives the full EventData including both old and new state.
func (n *NatsConnection) SubscribeStateEvent(entityId string, handler func(ha.EventData) error) (*nats.Subscription, error) {
	return subscribe(n, StateTopicType.GetTopic(entityId), parseEventData, handler)
}

// SubscribeAttributes subscribes to attribute change events for the given entity.
// The handler receives the full event (old and new state).
func (n *NatsConnection) SubscribeAttributes(entityId string, handler func(ha.EventData) error) (*nats.Subscription, error) {
	return subscribe(n, AttributeTopicType.GetTopic(entityId), parseEventData, handler)
}

// SubscribeZHA subscribes to ZHA device events for the given IEEE address.
func (n *NatsConnection) SubscribeZHA(deviceIeee string, handler func(ha.EventData) error) (*nats.Subscription, error) {
	return subscribe(n, ZhaTopicType.GetTopic(deviceIeee), parseEventData, handler)
}

// SubscribeNFC subscribes to NFC tag scan events for the given tag ID.
func (n *NatsConnection) SubscribeNFC(tagId string, handler func(ha.EventData) error) (*nats.Subscription, error) {
	return subscribe(n, NfcTopicType.GetTopic(tagId), parseEventData, handler)
}

// SubscribeZwave subscribes to Z-Wave device events for the given device ID.
func (n *NatsConnection) SubscribeZwave(deviceId string, handler func(ha.EventData) error) (*nats.Subscription, error) {
	return subscribe(n, ZwaveTopicType.GetTopic(deviceId), parseEventData, handler)
}

// SubscribeZwaveScene subscribes to Z-Wave scene events for the given device ID.
func (n *NatsConnection) SubscribeZwaveScene(deviceId string, handler func(ha.EventData) error) (*nats.Subscription, error) {
	return subscribe(n, ZwaveSceneTopicType.GetTopic(deviceId), parseEventData, handler)
}

// SubscribeTimer subscribes to timer completion events for the given timer name.
func (n *NatsConnection) SubscribeTimer(timerName string, handler func() error) (*nats.Subscription, error) {
	return subscribeNoData(n, TimerTopicType.GetTopic(timerName), handler)
}

// SubscribeSchedule subscribes to schedule fire events for the given schedule name.
func (n *NatsConnection) SubscribeSchedule(scheduleName string, handler func() error) (*nats.Subscription, error) {
	return subscribeNoData(n, ScheduleTopicType.GetTopic(scheduleName), handler)
}

// SubscribeCommand subscribes to command events for the given command name.
// The handler receives the full CommandPayload including HTTP method, headers, query, and body.
func (n *NatsConnection) SubscribeCommand(commandName string, handler func(CommandPayload) error) (*nats.Subscription, error) {
	return subscribe(n, CommandTopicType.GetTopic(commandName), parseCommandPayload, handler)
}
