package homeassistant

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"log/slog"

	"codeberg.org/jhot/hats/internal/nats"
	"codeberg.org/jhot/hats/pkg/config"
	ha "codeberg.org/jhot/hats/pkg/homeassistant"
	"github.com/gorilla/websocket"
)

var (
	cfg             *config.HatsConfig
	logger          *slog.Logger
	haWebsocketConn *websocket.Conn
	done            chan struct{}
)

const (
	stateChangeEventId   = 1001
	zhaEventId           = 1002
	nfcEventId           = 1003
	zwaveEventId         = 1004
	timerFinishedEventId = 1005
	defaultTimeout       = 250 * time.Millisecond
)

func CloseSubscription() error {
	if haWebsocketConn != nil {
		logger.Debug("Closing Home Assistant subscription")
		haWebsocketConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		close(done)
		return haWebsocketConn.Close()
	}
	return nil
}

func Subscribe(parentLogger *slog.Logger, parentConfig *config.HatsConfig) error {
	logger = parentLogger
	cfg = parentConfig
	var err error

	url := cfg.GetHomeAssistantWebsocketUrl()

	logger.Debug("Dialing Home Assistant websocket API", "url", url)

	haWebsocketConn, _, err = websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("%w: error dialing Home Assistant websocket", err)
	}

	done = make(chan struct{})

	handleMessages()
	return nil
}

func reconnect() {
	haWebsocketConn.Close()
	attempts := 1
	for {
		if attempts > 10 {
			panic(errors.New("unable to reconnect to Home Assistant"))
		}

		time.Sleep(time.Duration(attempts) * 5 * time.Second)
		logger.Info("Trying to reconnect to Home Assistant", "attempt", attempts)

		err := Subscribe(logger, cfg)
		if err == nil {
			break
		}

		attempts += 1
	}
}

func handleMessages() {
	ch := done
	go func() {
		defer close(ch)
		for {
			_, rawMessage, err := haWebsocketConn.ReadMessage()
			if err != nil {
				logger.Error("Error reading Home Assistant websocket message", "error", err)
				reconnect()
				break
			}
			if len(rawMessage) == 0 {
				continue
			}
			var message ha.HassMessage
			err = json.Unmarshal(rawMessage, &message)
			if err != nil {
				logger.Error("Error parsing HASS message", "message", string(rawMessage), "error", err)
				continue
			}

			switch message.Type {
			case ha.MessageType.AuthRequired:
				logger.Debug("Logging in to HomeAssistant websocket API")
				haWebsocketConn.WriteJSON(ha.AuthMessage{Type: "auth", AccessToken: cfg.HomeAssistantToken})
			case ha.MessageType.AuthOk:
				logger.Debug("Subscribing to Events")
				haWebsocketConn.WriteJSON(ha.SubscribeEventsMessage{
					Type:      ha.MessageType.SubscribeEvents,
					EventType: ha.MessageType.StateChanged,
					Id:        stateChangeEventId})
				haWebsocketConn.WriteJSON(ha.SubscribeEventsMessage{
					Type:      ha.MessageType.SubscribeEvents,
					EventType: ha.MessageType.ZhaEvent,
					Id:        zhaEventId})
				haWebsocketConn.WriteJSON(ha.SubscribeEventsMessage{
					Type:      ha.MessageType.SubscribeEvents,
					EventType: ha.MessageType.TagScanned,
					Id:        nfcEventId})
				haWebsocketConn.WriteJSON(ha.SubscribeEventsMessage{
					Type:      ha.MessageType.SubscribeEvents,
					EventType: ha.MessageType.ZwaveEvent,
					Id:        zwaveEventId})
				haWebsocketConn.WriteJSON(ha.SubscribeEventsMessage{
					Type:      ha.MessageType.SubscribeEvents,
					EventType: ha.MessageType.TimerFinished,
					Id:        timerFinishedEventId})
			case ha.MessageType.AuthInvalid:
				logger.Error("Home Assistant authentication failed, check token")
				return
			case ha.MessageType.Result:
				if !message.Success {
					logger.Error("Non-Success Result:", "message", message)
					reconnect()
					return
				}
			case ha.MessageType.Event:
				go func() {
					logger.Debug("Event received", "event", message.Event)
					switch message.Id {
					case stateChangeEventId:
						data, marshallErr := json.Marshal(message.Event.Data)
						if marshallErr != nil {
							logger.Error("Error marshalling event data", "error", marshallErr)
						}
						go nats.SetKeyValueString(fmt.Sprintf("homeassistant.states.%s", message.Event.Data.EntityId), message.Event.Data.NewState.State)
						stateReplacer := strings.NewReplacer(" ", "_", ".", ",", "*", "_", ">", "_")
						cleanedState := stateReplacer.Replace(message.Event.Data.NewState.State)
						if message.Event.Data.NewState.State == message.Event.Data.OldState.State {
							logger.Debug("State unchanged, publishing to attributes topic")
							nats.PublishRequest(fmt.Sprintf("homeassistant.attributes.%s.%s", message.Event.Data.EntityId, cleanedState), data, defaultTimeout, 1)
						} else {
							nats.PublishRequest(fmt.Sprintf("homeassistant.states.%s.%s", message.Event.Data.EntityId, cleanedState), data, defaultTimeout, 2)
							nats.PublishRequest(fmt.Sprintf("homeassistant.attributes.%s.%s", message.Event.Data.EntityId, cleanedState), data, defaultTimeout, 1)
						}
					case zhaEventId:
						data, _ := json.Marshal(message.Event.Data)
						nats.Publish(fmt.Sprintf("homeassistant.zha.%s", message.Event.Data.DeviceIeee), data)
					case nfcEventId:
						data, _ := json.Marshal(message.Event.Data)
						nats.PublishRequest(fmt.Sprintf("homeassistant.nfc.%s", message.Event.Data.TagId), data, defaultTimeout, 3)
					case zwaveEventId:
						data, _ := json.Marshal(message.Event.Data)
						nats.Publish(fmt.Sprintf("homeassistant.zwave-scene.%s", message.Event.Data.DeviceId), data)
					case timerFinishedEventId:
						nats.PublishRequest(fmt.Sprintf("homeassistant.timers.%s.finished", message.Event.Data.EntityId), []byte("finished"), defaultTimeout, 3)
					}
				}()
			}
		}
	}()
}
