//go:build integration

package integration

import (
	"encoding/json"
	"testing"
	"time"

	internalnats "codeberg.org/jhot/hats/internal/nats"
	"codeberg.org/jhot/hats/integration/haserver"
	ha "codeberg.org/jhot/hats/pkg/homeassistant"
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
	assert.Equal(t, "sensor.temp", data.EntityId)

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
