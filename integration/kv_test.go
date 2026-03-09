//go:build integration

package integration

import (
	"testing"
	"time"

	internalnats "codeberg.org/jhot/hats/internal/nats"
	"codeberg.org/jhot/hats/integration/haserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestKVStateCache verifies that a state_changed event causes the KV store to be updated
// with the new state string.
func TestKVStateCache(t *testing.T) {
	server := haserver.New("test-token")
	t.Cleanup(server.Close)
	nc := setupSubscriber(t, server)

	listenAndReply(t, nc, "homeassistant.states.sensor.kv_test.on")
	listenAndReply(t, nc, "homeassistant.attributes.sensor.kv_test.on")

	server.InjectStateChange("sensor.kv_test", "off", "on")

	require.Eventually(t, func() bool {
		val, err := internalnats.GetKeyValue("homeassistant.states.sensor.kv_test")
		return err == nil && string(val) == "on"
	}, 5*time.Second, 100*time.Millisecond, "KV store was not updated")
}

// TestKVRoundTrip verifies that SetKeyValue followed by GetKeyValue returns the same bytes.
func TestKVRoundTrip(t *testing.T) {
	server := haserver.New("test-token")
	t.Cleanup(server.Close)
	_ = setupSubscriber(t, server) // initializes KvConnect(); nc not needed for this test

	key := "test.kv.roundtrip"
	value := []byte(`{"hello":"world"}`)

	require.NoError(t, internalnats.SetKeyValue(key, value))

	got, err := internalnats.GetKeyValue(key)
	require.NoError(t, err)
	assert.Equal(t, value, got)
}
