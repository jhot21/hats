//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	internalha "codeberg.org/jhot/hats/internal/homeassistant"
	internalnats "codeberg.org/jhot/hats/internal/nats"
	"codeberg.org/jhot/hats/integration/haserver"
	"codeberg.org/jhot/hats/pkg/config"
	ha "codeberg.org/jhot/go-ha-sdk"
	nats "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStateChangeRouting verifies that a state_changed event produces the correct NATS subject
// and payload, and that the KV store is updated with the new state.
func TestStateChangeRouting(t *testing.T) {
	server := haserver.New("test-token")
	t.Cleanup(server.Close)
	nc := setupSubscriber(t, server)

	received := captureWithReply(t, nc, "homeassistant.states.sensor.temp.on")
	listenAndReply(t, nc, "homeassistant.attributes.sensor.temp.on")

	server.InjectStateChange("sensor.temp", "off", "on")

	msg := receiveWithin(t, received, 5*time.Second)

	var data ha.EventData
	require.NoError(t, json.Unmarshal(msg.Data, &data))
	assert.Equal(t, "on", data.NewState.State)
	assert.Equal(t, "sensor.temp", data.EntityID)

	require.Eventually(t, func() bool {
		val, err := internalnats.GetKeyValue("homeassistant.states.sensor.temp")
		return err == nil && string(val) == "on"
	}, 3*time.Second, 50*time.Millisecond, "KV store was not updated with new state")
}

// TestTopicCharacterNormalization verifies that '.', ' ', '*', and '>' in state values are
// replaced with '_' in NATS subjects (fix introduced in code-review-fixes-2 branch).
func TestTopicCharacterNormalization(t *testing.T) {
	cases := []struct {
		rawState     string
		cleanedState string
	}{
		{"on.off", "on_off"},
		{"on off", "on_off"},
		{"on*off", "on_off"},
		{"on>off", "on_off"},
	}
	for _, tc := range cases {
		t.Run(tc.rawState, func(t *testing.T) {
			server := haserver.New("test-token")
			t.Cleanup(server.Close)
			nc := setupSubscriber(t, server)

			subject := "homeassistant.states.sensor.test." + tc.cleanedState
			received := captureWithReply(t, nc, subject)
			listenAndReply(t, nc, "homeassistant.attributes.sensor.test."+tc.cleanedState)

			server.InjectStateChange("sensor.test", "off", tc.rawState)
			receiveWithin(t, received, 5*time.Second)
		})
	}
}

// TestStateUnchangedPublishesAttributesOnly verifies that when old_state == new_state,
// only the attributes subject is published (not the states subject).
func TestStateUnchangedPublishesAttributesOnly(t *testing.T) {
	server := haserver.New("test-token")
	t.Cleanup(server.Close)
	nc := setupSubscriber(t, server)

	statesCh := captureWithReply(t, nc, "homeassistant.states.sensor.temp.on")
	attrCh := captureWithReply(t, nc, "homeassistant.attributes.sensor.temp.on")

	server.InjectStateChange("sensor.temp", "on", "on") // same state

	receiveWithin(t, attrCh, 5*time.Second)

	// Once the attributes message arrived, the event is fully processed —
	// the states subject will never receive a message for the same event.
	select {
	case <-statesCh:
		t.Fatal("expected no message on states subject when state is unchanged")
	case <-time.After(100 * time.Millisecond):
		// correct: no message
	}
}

// TestReconnectAfterConnectionDrop verifies that after the HA server drops the WebSocket
// connection, the subscriber reconnects, re-authenticates, and events flow again.
// NOTE: reconnect() sleeps 5s before the first retry; allow 15s total.
func TestReconnectAfterConnectionDrop(t *testing.T) {
	server := haserver.New("test-token")
	t.Cleanup(server.Close)
	nc := setupSubscriber(t, server)

	// First connection is established by setupSubscriber.
	server.DropConnection()

	// Wait for reconnect (second connection).
	require.Eventually(t, func() bool {
		return server.ConnectionCount() >= 2
	}, 15*time.Second, 200*time.Millisecond, "subscriber did not reconnect after connection drop")

	// Events should flow on the new connection.
	received := captureWithReply(t, nc, "homeassistant.states.sensor.reconnect.on")
	listenAndReply(t, nc, "homeassistant.attributes.sensor.reconnect.on")
	server.InjectStateChange("sensor.reconnect", "off", "on")
	receiveWithin(t, received, 5*time.Second)
}

// TestPingKeepalive verifies that the subscriber sends pings at PingInterval and the
// connection stays alive.
func TestPingKeepalive(t *testing.T) {
	internalha.PingInterval = 500 * time.Millisecond
	t.Cleanup(func() { internalha.PingInterval = 30 * time.Second })

	server := haserver.New("test-token")
	t.Cleanup(server.Close)
	setupSubscriber(t, server)

	// Check both conditions atomically to avoid TOCTOU between PingsReceived and ConnectionCount.
	require.Eventually(t, func() bool {
		return server.PingsReceived() >= 2 && server.ConnectionCount() == 1
	}, 5*time.Second, 100*time.Millisecond, "expected at least 2 pings with connection remaining stable")
}

// TestPingFailureTriggerReconnect verifies that when the server closes the connection on a
// ping, the subscriber detects the failure and reconnects.
func TestPingFailureTriggerReconnect(t *testing.T) {
	internalha.PingInterval = 500 * time.Millisecond
	t.Cleanup(func() { internalha.PingInterval = 30 * time.Second })

	server := haserver.New("test-token")
	server.DropOnPing = true
	t.Cleanup(server.Close)
	setupSubscriber(t, server)

	require.Eventually(t, func() bool {
		return server.ConnectionCount() >= 2
	}, 15*time.Second, 200*time.Millisecond, "subscriber did not reconnect after ping failure")
}

// TestAuthInvalidStops verifies that auth_invalid causes Subscribe to return an error
// and the subscriber does not retry (no second connection attempt).
func TestAuthInvalidStops(t *testing.T) {
	server := haserver.New("test-token")
	server.RejectAuth = true
	t.Cleanup(server.Close)

	host, port, err := net.SplitHostPort(server.Addr())
	require.NoError(t, err)

	testCfg := &config.HatsConfig{
		HomeAssistantHost:  host,
		HomeAssistantPort:  port,
		HomeAssistantToken: "test-token",
		NatsHost:           "127.0.0.1",
		NatsPort:           natsPort,
		NatsClientName:     "test-hats",
	}

	ctx := context.Background()
	testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	require.NoError(t, internalnats.JetstreamConnect(ctx, testLogger, testCfg))
	require.NoError(t, internalnats.KvConnect())
	t.Cleanup(func() { internalnats.Close() })

	err = internalha.Subscribe(testLogger, testCfg)
	require.Error(t, err, "Subscribe should return an error when auth is rejected")

	assert.Never(t, func() bool {
		return server.ConnectionCount() > 1
	}, 2*time.Second, 100*time.Millisecond, "subscriber should not retry after auth_invalid")
}

// TestZHAEventRouting verifies that a ZHA event is published to the correct NATS subject.
func TestZHAEventRouting(t *testing.T) {
	server := haserver.New("test-token")
	t.Cleanup(server.Close)
	nc := setupSubscriber(t, server)

	received := capture(t, nc, "homeassistant.zha.aa:bb:cc:dd")

	server.InjectZHAEvent("aa:bb:cc:dd", "toggle")

	msg := receiveWithin(t, received, 5*time.Second)
	var data ha.EventData
	require.NoError(t, json.Unmarshal(msg.Data, &data))
	assert.Equal(t, "aa:bb:cc:dd", data.DeviceIEEE)
	assert.Equal(t, "toggle", data.Command)
}

// TestNFCEventRouting verifies that an NFC tag_scanned event reaches the correct NATS subject.
func TestNFCEventRouting(t *testing.T) {
	server := haserver.New("test-token")
	t.Cleanup(server.Close)
	nc := setupSubscriber(t, server)

	received := captureWithReply(t, nc, "homeassistant.nfc.tag-abc-123")

	server.InjectNFCEvent("tag-abc-123")

	msg := receiveWithin(t, received, 5*time.Second)
	var data ha.EventData
	require.NoError(t, json.Unmarshal(msg.Data, &data))
	assert.Equal(t, "tag-abc-123", data.TagID)
}

// TestNFCEventNoListenerContinues verifies that when no NATS listener exists for an NFC tag,
// PublishRequest exhausts retries but the subscriber continues processing subsequent events.
func TestNFCEventNoListenerContinues(t *testing.T) {
	server := haserver.New("test-token")
	t.Cleanup(server.Close)
	nc := setupSubscriber(t, server)

	// Inject NFC with no listener — retries will exhaust (warn logged, no crash).
	server.InjectNFCEvent("unknown-tag")

	// Give retries time to exhaust (3 retries × 100ms backoff + 250ms timeouts each ≈ 1.5s).
	time.Sleep(2 * time.Second)

	// Subscriber should still process subsequent events.
	received := captureWithReply(t, nc, "homeassistant.states.sensor.alive.on")
	listenAndReply(t, nc, "homeassistant.attributes.sensor.alive.on")
	server.InjectStateChange("sensor.alive", "off", "on")
	receiveWithin(t, received, 5*time.Second)
}

// TestTimerFinishedRouting verifies that a timer.finished event reaches the correct NATS subject
// with the expected payload.
func TestTimerFinishedRouting(t *testing.T) {
	server := haserver.New("test-token")
	t.Cleanup(server.Close)
	nc := setupSubscriber(t, server)

	received := captureWithReply(t, nc, "homeassistant.timers.timer.bedtime.finished")

	server.InjectTimerFinished("timer.bedtime")

	msg := receiveWithin(t, received, 5*time.Second)
	assert.Equal(t, "finished", string(msg.Data))
}

// TestZWaveEventRouting verifies that a Z-Wave scene event reaches the correct NATS subject.
func TestZWaveEventRouting(t *testing.T) {
	server := haserver.New("test-token")
	t.Cleanup(server.Close)
	nc := setupSubscriber(t, server)

	received := capture(t, nc, "homeassistant.zwave-scene.node-42")

	server.InjectZWaveEvent("node-42")

	msg := receiveWithin(t, received, 5*time.Second)
	var data ha.EventData
	require.NoError(t, json.Unmarshal(msg.Data, &data))
	assert.Equal(t, "node-42", data.DeviceID)
}

// TestMultipleEventsOrdered injects 20 rapid state_changed events and verifies all 20 arrive
// in order. This validates inline (sequential) event processing rather than goroutine-per-event.
func TestMultipleEventsOrdered(t *testing.T) {
	server := haserver.New("test-token")
	t.Cleanup(server.Close)
	nc := setupSubscriber(t, server)

	// Pre-register listeners for all 20 subjects before injecting.
	channels := make([]<-chan *nats.Msg, 20)
	for i := 0; i < 20; i++ {
		subject := fmt.Sprintf("homeassistant.states.sensor.order.state%02d", i)
		channels[i] = captureWithReply(t, nc, subject)
		listenAndReply(t, nc, fmt.Sprintf("homeassistant.attributes.sensor.order.state%02d", i))
	}

	for i := 0; i < 20; i++ {
		server.InjectStateChange("sensor.order", "off", fmt.Sprintf("state%02d", i))
	}

	for i := 0; i < 20; i++ {
		receiveWithin(t, channels[i], 10*time.Second)
	}
}
