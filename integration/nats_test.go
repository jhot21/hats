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

// TestPublishRequestSuccess verifies that PublishRequest returns nil when a listener
// is already subscribed and responds.
func TestPublishRequestSuccess(t *testing.T) {
	server := haserver.New("test-token")
	t.Cleanup(server.Close)
	nc := setupSubscriber(t, server)

	listenAndReply(t, nc, "test.subject.success")

	err := internalnats.PublishRequest("test.subject.success", []byte("hello"), 250*time.Millisecond, 3)
	require.NoError(t, err)
}

// TestPublishRequestRetryExhausted verifies that PublishRequest returns an error
// when no listener exists after all retries are exhausted.
func TestPublishRequestRetryExhausted(t *testing.T) {
	server := haserver.New("test-token")
	t.Cleanup(server.Close)
	_ = setupSubscriber(t, server)

	err := internalnats.PublishRequest("test.subject.nobody", []byte("hello"), 100*time.Millisecond, 3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request retries exceeded")
}

// TestPublishRequestBackoffTiming verifies that linear backoff is applied between retries.
// 3 retries × 100ms timeout + (1+2)×100ms sleep = at least 300ms total sleep.
func TestPublishRequestBackoffTiming(t *testing.T) {
	server := haserver.New("test-token")
	t.Cleanup(server.Close)
	_ = setupSubscriber(t, server)

	start := time.Now()
	internalnats.PublishRequest("test.subject.backoff", []byte("hello"), 50*time.Millisecond, 3)
	elapsed := time.Since(start)

	assert.GreaterOrEqual(t, elapsed, 300*time.Millisecond,
		"expected at least 300ms for linear backoff across 3 retries")
}

// TestPublishRequestSucceedsOnRetry verifies that PublishRequest returns nil if a listener
// arrives mid-retry (simulates a slow consumer starting up).
func TestPublishRequestSucceedsOnRetry(t *testing.T) {
	server := haserver.New("test-token")
	t.Cleanup(server.Close)
	nc := setupSubscriber(t, server)

	// Start listener 200ms after PublishRequest begins.
	go func() {
		time.Sleep(200 * time.Millisecond)
		listenAndReply(t, nc, "test.subject.late")
	}()

	err := internalnats.PublishRequest("test.subject.late", []byte("hello"), 100*time.Millisecond, 5)
	require.NoError(t, err)
}
