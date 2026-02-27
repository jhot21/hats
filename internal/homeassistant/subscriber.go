package homeassistant

import (
	"encoding/json"
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
	quit            chan struct{}
)

const (
	stateChangeEventId   = 1001
	zhaEventId           = 1002
	nfcEventId           = 1003
	zwaveEventId         = 1004
	timerFinishedEventId = 1005
	defaultTimeout       = 250 * time.Millisecond
	pingInterval         = 30 * time.Second
	pingBaseId           = 9000
)

func CloseSubscription() error {
	if haWebsocketConn != nil {
		logger.Debug("Closing Home Assistant subscription")
		haWebsocketConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		close(quit)
		return haWebsocketConn.Close()
	}
	return nil
}

func connect() error {
	url := cfg.GetHomeAssistantWebsocketUrl()
	logger.Debug("Dialing Home Assistant websocket API", "url", url)
	var err error
	haWebsocketConn, _, err = websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("error dialing Home Assistant websocket: %w", err)
	}
	return nil
}

func Subscribe(parentLogger *slog.Logger, parentConfig *config.HatsConfig) error {
	logger = parentLogger
	cfg = parentConfig

	if err := connect(); err != nil {
		return err
	}

	quit = make(chan struct{})
	done = make(chan struct{})
	handleMessages()
	return nil
}

func reconnect() {
	haWebsocketConn.Close()
	for attempts := 1; attempts <= 10; attempts++ {
		time.Sleep(time.Duration(attempts) * 5 * time.Second)
		logger.Info("Trying to reconnect to Home Assistant", "attempt", attempts)
		if err := connect(); err != nil {
			logger.Error("Failed to reconnect to Home Assistant", "attempt", attempts, "error", err)
			continue
		}
		done = make(chan struct{})
		handleMessages()
		return
	}
	logger.Error("Unable to reconnect to Home Assistant after 10 attempts")
}

func handleMessages() {
	ch := done
	conn := haWebsocketConn
	q := quit
	go func() {
		defer close(ch)

		type readResult struct {
			data []byte
			err  error
		}
		// Buffered so the reader goroutine can exit cleanly if this goroutine
		// returns (e.g. after a ping write failure) before draining the channel.
		msgCh := make(chan readResult, 1)
		go func() {
			for {
				_, data, err := conn.ReadMessage()
				msgCh <- readResult{data, err}
				if err != nil {
					return
				}
			}
		}()

		pingTicker := time.NewTicker(pingInterval)
		defer pingTicker.Stop()
		pingId := pingBaseId

		for {
			select {
			case <-q:
				return

			case <-pingTicker.C:
				pingId++
				if err := conn.WriteJSON(ha.PingMessage{Type: ha.MessageType.Ping, Id: pingId}); err != nil {
					select {
					case <-q:
						return
					default:
					}
					logger.Error("Failed to send ping", "error", err)
					reconnect()
					return
				}
				logger.Debug("Sent ping", "id", pingId)

			case result := <-msgCh:
				if result.err != nil {
					select {
					case <-q:
						return
					default:
					}
					logger.Error("Error reading Home Assistant websocket message", "error", result.err)
					reconnect()
					return
				}
				if len(result.data) == 0 {
					continue
				}
				var message ha.HassMessage
				if err := json.Unmarshal(result.data, &message); err != nil {
					logger.Error("Error parsing HASS message", "message", string(result.data), "error", err)
					continue
				}

				switch message.Type {
				case ha.MessageType.AuthRequired:
					logger.Debug("Logging in to HomeAssistant websocket API")
					if err := conn.WriteJSON(ha.AuthMessage{Type: "auth", AccessToken: cfg.HomeAssistantToken}); err != nil {
						logger.Error("Failed to send auth message", "error", err)
						reconnect()
						return
					}
				case ha.MessageType.AuthOk:
					logger.Debug("Subscribing to Events")
					subscriptions := []ha.SubscribeEventsMessage{
						{Type: ha.MessageType.SubscribeEvents, EventType: ha.MessageType.StateChanged, Id: stateChangeEventId},
						{Type: ha.MessageType.SubscribeEvents, EventType: ha.MessageType.ZhaEvent, Id: zhaEventId},
						{Type: ha.MessageType.SubscribeEvents, EventType: ha.MessageType.TagScanned, Id: nfcEventId},
						{Type: ha.MessageType.SubscribeEvents, EventType: ha.MessageType.ZwaveEvent, Id: zwaveEventId},
						{Type: ha.MessageType.SubscribeEvents, EventType: ha.MessageType.TimerFinished, Id: timerFinishedEventId},
					}
					for _, sub := range subscriptions {
						if err := conn.WriteJSON(sub); err != nil {
							logger.Error("Failed to send subscribe message", "event_type", sub.EventType, "error", err)
							reconnect()
							return
						}
					}
				case ha.MessageType.AuthInvalid:
					logger.Error("Home Assistant authentication failed, check token")
					return
				case ha.MessageType.Pong:
					logger.Debug("Received pong", "id", message.Id)
				case ha.MessageType.Result:
					if !message.Success {
						logger.Error("Non-Success Result:", "message", message)
						reconnect()
						return
					}
				case ha.MessageType.Event:
					logger.Debug("Event received", "event", message.Event)
					switch message.Id {
					case stateChangeEventId:
						data, marshallErr := json.Marshal(message.Event.Data)
						if marshallErr != nil {
							logger.Error("Error marshalling event data", "error", marshallErr)
						}
						go nats.SetKeyValueString(fmt.Sprintf("homeassistant.states.%s", message.Event.Data.EntityId), message.Event.Data.NewState.State)
						stateReplacer := strings.NewReplacer(" ", "_", ".", "_", "*", "_", ">", "_")
						cleanedState := stateReplacer.Replace(message.Event.Data.NewState.State)
						if message.Event.Data.NewState.State == message.Event.Data.OldState.State {
							logger.Debug("State unchanged, publishing to attributes topic")
							if err := nats.PublishRequest(fmt.Sprintf("homeassistant.attributes.%s.%s", message.Event.Data.EntityId, cleanedState), data, defaultTimeout, 1); err != nil {
								logger.Warn("Failed to publish attributes", "entity", message.Event.Data.EntityId, "error", err)
							}
						} else {
							if err := nats.PublishRequest(fmt.Sprintf("homeassistant.states.%s.%s", message.Event.Data.EntityId, cleanedState), data, defaultTimeout, 2); err != nil {
								logger.Warn("Failed to publish state", "entity", message.Event.Data.EntityId, "error", err)
							}
							if err := nats.PublishRequest(fmt.Sprintf("homeassistant.attributes.%s.%s", message.Event.Data.EntityId, cleanedState), data, defaultTimeout, 1); err != nil {
								logger.Warn("Failed to publish attributes", "entity", message.Event.Data.EntityId, "error", err)
							}
						}
					case zhaEventId:
						data, _ := json.Marshal(message.Event.Data)
						nats.Publish(fmt.Sprintf("homeassistant.zha.%s", message.Event.Data.DeviceIeee), data)
					case nfcEventId:
						data, _ := json.Marshal(message.Event.Data)
						if err := nats.PublishRequest(fmt.Sprintf("homeassistant.nfc.%s", message.Event.Data.TagId), data, defaultTimeout, 3); err != nil {
							logger.Warn("Failed to publish NFC event", "tag", message.Event.Data.TagId, "error", err)
						}
					case zwaveEventId:
						data, _ := json.Marshal(message.Event.Data)
						nats.Publish(fmt.Sprintf("homeassistant.zwave-scene.%s", message.Event.Data.DeviceId), data)
					case timerFinishedEventId:
						if err := nats.PublishRequest(fmt.Sprintf("homeassistant.timers.%s.finished", message.Event.Data.EntityId), []byte("finished"), defaultTimeout, 3); err != nil {
							logger.Warn("Failed to publish timer finished", "entity", message.Event.Data.EntityId, "error", err)
						}
					}
				}
			}
		}
	}()
}
