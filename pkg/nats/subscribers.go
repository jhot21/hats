package nats

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	ha "code.jhot.me/jhot/hats/pkg/homeassistant"
)

func GenericStateSubscriber(logger *slog.Logger, natsClient *NatsConnection, entityId string, handler func(ha.StateData) error) {
	if entityId == "" {
		panic(errors.New("entity ID cannot be empty"))
	}
	topic := fmt.Sprintf("homeassistant.states.%s.>", entityId)
	l := logger.With("topic", topic, "entity_id", entityId)
	l.Debug("Subscribing to topic")
	sub, ch, err := natsClient.Subscribe(topic)
	if err != nil {
		l.Error("Error subscribing to topic", "error", err)
		return
	}

	defer sub.Unsubscribe()

	for msg := range ch {
		go msg.Ack()
		var data ha.EventData
		err = json.Unmarshal(msg.Data, &data)
		if err != nil {
			l.Error("Error parsing message", "error", err)
			continue
		}
		l.Debug("Event state " + data.NewState.State)
		err = handler(data.NewState)
		if err != nil {
			l.Error("Error handling state event", "error", err)
			continue
		}
	}
}

func GenericScheduleSubscriber(logger *slog.Logger, natsClient *NatsConnection, scheduleName string, handler func() error) {
	if scheduleName == "" {
		panic(errors.New("schedule name cannot be empty"))
	}
	topic := fmt.Sprintf("schedules.%s", scheduleName)
	l := logger.With("topic", topic, "schedule", scheduleName)
	l.Debug("Subscribing to topic")
	sub, ch, err := natsClient.Subscribe(topic)
	if err != nil {
		l.Error("Error subscribing to topic", "error", err)
		return
	}

	defer sub.Unsubscribe()

	for msg := range ch {
		go msg.Ack()
		l.Debug("Schedule fired", "name", scheduleName)
		err = handler()
		if err != nil {
			l.Error("Error handling schedule event", "error", err)
			continue
		}
	}
}

func GenericCommandSubscriber(logger *slog.Logger, natsClient *NatsConnection, commandName string, handler func([]byte) error) {
	if commandName == "" {
		panic(errors.New("command name cannot be empty"))
	}
	topic := fmt.Sprintf("command.%s", commandName)
	l := logger.With("topic", topic, "command", commandName)
	l.Debug("Subscribing to topic")
	sub, ch, err := natsClient.Subscribe(topic)
	if err != nil {
		l.Error("Error subscribing to topic", "error", err)
		return
	}

	defer sub.Unsubscribe()

	for msg := range ch {
		go msg.Ack()
		l.Debug("Command fired")
		err = handler(msg.Data)
		if err != nil {
			l.Error("Error handling command event", "error", err)
			continue
		}
	}
}

func GenericNfcSubscriber(logger *slog.Logger, natsClient *NatsConnection, tagId string, handler func([]byte) error) {
	if tagId == "" {
		panic(errors.New("tag ID cannot be empty"))
	}
	topic := fmt.Sprintf("homeassistant.nfc.%s", tagId)
	l := logger.With("topic", topic, "tagId", tagId)
	l.Debug("Subscribing to topic")
	sub, ch, err := natsClient.Subscribe(topic)
	if err != nil {
		l.Error("Error subscribing to topic", "error", err)
		return
	}

	defer sub.Unsubscribe()

	for msg := range ch {
		go msg.Ack()
		l.Debug("NFC tag scanned")
		err = handler(msg.Data)
		if err != nil {
			l.Error("Error handling NFC event", "error", err)
			continue
		}
	}
}

func GenericZhaSubscriber(logger *slog.Logger, natsClient *NatsConnection, deviceIeee string, handler func(ha.EventData) error) {
	if deviceIeee == "" {
		panic(errors.New("device IEEE cannot be empty"))
	}
	topic := fmt.Sprintf("homeassistant.zha.%s", deviceIeee)
	l := logger.With("topic", topic, "ieee", deviceIeee)
	l.Debug("Subscribing to topic")
	sub, ch, err := natsClient.Subscribe(topic)
	if err != nil {
		l.Error("Error subscribing to topic", "error", err)
		return
	}

	defer sub.Unsubscribe()

	for msg := range ch {
		go msg.Ack()
		var data ha.EventData
		err = json.Unmarshal(msg.Data, &data)
		if err != nil {
			l.Error("Error parsing message", "error", err)
			continue
		}
		l.Debug("ZHA event fired")
		err = handler(data)
		if err != nil {
			l.Error("Error handling ZHA event", "error", err)
			continue
		}
	}
}

func GenericZwaveLSubscriber(logger *slog.Logger, natsClient *NatsConnection, deviceId string, scene bool, handler func(ha.EventData) error) {
	if deviceId == "" {
		panic(errors.New("device ID cannot be empty"))
	}
	scenePart := ""
	if scene {
		scenePart = "-scene"
	}
	topic := fmt.Sprintf("homeassistant.zwave%s.%s", scenePart, deviceId)
	l := logger.With("topic", topic, "deviceId", deviceId)
	l.Debug("Subscribing to topic")
	sub, ch, err := natsClient.Subscribe(topic)
	if err != nil {
		l.Error("Error subscribing to topic", "error", err)
		return
	}

	defer sub.Unsubscribe()

	for msg := range ch {
		go msg.Ack()
		var data ha.EventData
		err = json.Unmarshal(msg.Data, &data)
		if err != nil {
			l.Error("Error parsing message", "error", err)
			continue
		}
		l.Debug("Zwave event fired")
		err = handler(data)
		if err != nil {
			l.Error("Error handling Zwave event", "error", err)
			continue
		}
	}
}
