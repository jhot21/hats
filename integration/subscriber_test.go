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

func TestStateChangeRouting(t *testing.T) {
	server := haserver.New("test-token")
	defer server.Close()
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
		tc := tc
		t.Run(tc.rawState, func(t *testing.T) {
			server := haserver.New("test-token")
			defer server.Close()
			nc := setupSubscriber(t, server)

			subject := "homeassistant.states.sensor.test." + tc.cleanedState
			received := captureWithReply(t, nc, subject)
			listenAndReply(t, nc, "homeassistant.attributes.sensor.test."+tc.cleanedState)

			server.InjectStateChange("sensor.test", "off", tc.rawState)
			receiveWithin(t, received, 5*time.Second)
		})
	}
}

func TestStateUnchangedPublishesAttributesOnly(t *testing.T) {
	server := haserver.New("test-token")
	defer server.Close()
	nc := setupSubscriber(t, server)

	statesCh := captureWithReply(t, nc, "homeassistant.states.sensor.temp.on")
	attrCh := captureWithReply(t, nc, "homeassistant.attributes.sensor.temp.on")

	server.InjectStateChange("sensor.temp", "on", "on") // same state

	receiveWithin(t, attrCh, 5*time.Second)

	select {
	case <-statesCh:
		t.Fatal("expected no message on states subject when state is unchanged")
	case <-time.After(500 * time.Millisecond):
		// correct: no message
	}
}
