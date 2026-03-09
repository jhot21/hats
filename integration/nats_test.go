//go:build integration

package integration

import (
	"testing"
	"time"

	internalnats "codeberg.org/jhot/hats/internal/nats"
	"codeberg.org/jhot/hats/integration/haserver"
	"github.com/nats-io/nats.go"
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
	require.Error(t, err)
	assert.Contains(t, err.Error(), "request retries exceeded")
}

// TestPublishRequestBackoffTiming verifies that linear backoff is applied between retries.
// 3 retries × 50ms timeout + (1+2)×100ms sleep = at least 450ms total; threshold is 300ms.
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

	// Subscribe before launching the goroutine to avoid calling require from a non-test goroutine.
	// Unsubscribe immediately so the subject has no listener yet, then re-subscribe after 200ms.
	sub, err := nc.Subscribe("test.subject.late", func(msg *nats.Msg) {
		msg.Respond([]byte("got it"))
	})
	require.NoError(t, err)
	sub.Unsubscribe()

	// Re-register the listener after a 200ms delay.
	go func() {
		time.Sleep(200 * time.Millisecond)
		nc.Subscribe("test.subject.late", func(msg *nats.Msg) { //nolint:errcheck
			msg.Respond([]byte("got it"))
		})
	}()

	err = internalnats.PublishRequest("test.subject.late", []byte("hello"), 100*time.Millisecond, 5)
	require.NoError(t, err)
}
