package homeassistant

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"log/slog"

	"codeberg.org/jhot/hats/internal/nats"
	"codeberg.org/jhot/hats/pkg/config"
	ha "codeberg.org/jhot/go-ha-sdk"
)

var (
	wsClient *ha.WSClient
	wg       sync.WaitGroup
	logger   *slog.Logger
	cfg      *config.HatsConfig
)

const defaultTimeout = 250 * time.Millisecond

// PingInterval is the interval between WebSocket ping messages sent to Home Assistant.
// It is a var (not a const) to allow integration tests to reduce it for faster keepalive testing.
var PingInterval = 30 * time.Second

// MaxReconnectAttempts is the maximum number of SDK-level reconnect attempts per connect cycle.
// Set to a lower value in integration tests to exhaust reconnects quickly.
var MaxReconnectAttempts = 10

// ReconnectBackoff is the base duration for SDK-level reconnect backoff.
// Set to a lower value in integration tests.
var ReconnectBackoff = 5 * time.Second

// SupervisorDelay is how long the supervisor waits between subscription restart attempts.
// Set to a lower value in integration tests.
var SupervisorDelay = 15 * time.Second

// Subscribe connects to Home Assistant WebSocket, subscribes to all event types,
// and starts goroutines that route events to NATS subjects.
func Subscribe(parentLogger *slog.Logger, parentConfig *config.HatsConfig) error {
	logger = parentLogger
	cfg = parentConfig

	var err error
	wsClient, err = ha.NewWSClient(
		cfg.GetHomeAssistantBaseUrl(),
		cfg.HomeAssistantToken,
		ha.WithPingInterval(PingInterval),
		ha.WithMaxReconnectAttempts(10),
		ha.WithReconnectBackoff(5*time.Second),
	)
	if err != nil {
		return fmt.Errorf("connecting to Home Assistant: %w", err)
	}

	ctx := context.Background()

	stateCh, err := wsClient.Subscribe(ctx, ha.EventStateChanged)
	if err != nil {
		wsClient.Close()
		return fmt.Errorf("subscribing to state changes: %w", err)
	}
	zhaCh, err := wsClient.Subscribe(ctx, ha.EventZHA)
	if err != nil {
		wsClient.Close()
		return fmt.Errorf("subscribing to ZHA events: %w", err)
	}
	nfcCh, err := wsClient.Subscribe(ctx, ha.EventTagScanned)
	if err != nil {
		wsClient.Close()
		return fmt.Errorf("subscribing to NFC events: %w", err)
	}
	zwaveCh, err := wsClient.Subscribe(ctx, ha.EventZWave)
	if err != nil {
		wsClient.Close()
		return fmt.Errorf("subscribing to Z-Wave events: %w", err)
	}
	timerCh, err := wsClient.Subscribe(ctx, ha.EventTimerFinished)
	if err != nil {
		wsClient.Close()
		return fmt.Errorf("subscribing to timer events: %w", err)
	}

	wg.Add(5)
	go handleStateChanges(stateCh)
	go handleZHA(zhaCh)
	go handleNFC(nfcCh)
	go handleZWave(zwaveCh)
	go handleTimerFinished(timerCh)

	return nil
}

// CloseSubscription shuts down the WebSocket client and waits for all event handlers to exit.
func CloseSubscription() error {
	if wsClient == nil {
		return nil
	}
	wsClient.Close()
	wsClient = nil
	wg.Wait()
	return nil
}

func handleStateChanges(ch <-chan ha.Event) {
	defer wg.Done()
	stateReplacer := strings.NewReplacer(" ", "_", ".", "_", "*", "_", ">", "_")
	for event := range ch {
		data, err := json.Marshal(event.Data)
		if err != nil {
			logger.Error("Error marshalling state change event", "error", err)
			continue
		}
		go nats.SetKeyValueString(fmt.Sprintf("homeassistant.states.%s", event.Data.EntityID), event.Data.NewState.State)
		cleanedState := stateReplacer.Replace(event.Data.NewState.State)
		if event.Data.NewState.State == event.Data.OldState.State {
			logger.Debug("State unchanged, publishing to attributes topic")
			if err := nats.PublishRequest(fmt.Sprintf("homeassistant.attributes.%s.%s", event.Data.EntityID, cleanedState), data, defaultTimeout, 1); err != nil {
				logger.Warn("Failed to publish attributes", "entity", event.Data.EntityID, "error", err)
			}
		} else {
			if err := nats.PublishRequest(fmt.Sprintf("homeassistant.states.%s.%s", event.Data.EntityID, cleanedState), data, defaultTimeout, 2); err != nil {
				logger.Warn("Failed to publish state", "entity", event.Data.EntityID, "error", err)
			}
			if err := nats.PublishRequest(fmt.Sprintf("homeassistant.attributes.%s.%s", event.Data.EntityID, cleanedState), data, defaultTimeout, 1); err != nil {
				logger.Warn("Failed to publish attributes", "entity", event.Data.EntityID, "error", err)
			}
		}
	}
}

func handleZHA(ch <-chan ha.Event) {
	defer wg.Done()
	for event := range ch {
		data, _ := json.Marshal(event.Data)
		nats.Publish(fmt.Sprintf("homeassistant.zha.%s", event.Data.DeviceIEEE), data)
	}
}

func handleNFC(ch <-chan ha.Event) {
	defer wg.Done()
	for event := range ch {
		data, _ := json.Marshal(event.Data)
		if err := nats.PublishRequest(fmt.Sprintf("homeassistant.nfc.%s", event.Data.TagID), data, defaultTimeout, 3); err != nil {
			logger.Warn("Failed to publish NFC event", "tag", event.Data.TagID, "error", err)
		}
	}
}

func handleZWave(ch <-chan ha.Event) {
	defer wg.Done()
	for event := range ch {
		data, _ := json.Marshal(event.Data)
		nats.Publish(fmt.Sprintf("homeassistant.zwave-scene.%s", event.Data.DeviceID), data)
	}
}

func handleTimerFinished(ch <-chan ha.Event) {
	defer wg.Done()
	for event := range ch {
		if err := nats.PublishRequest(fmt.Sprintf("homeassistant.timers.%s.finished", event.Data.EntityID), []byte("finished"), defaultTimeout, 3); err != nil {
			logger.Warn("Failed to publish timer finished", "entity", event.Data.EntityID, "error", err)
		}
	}
}
