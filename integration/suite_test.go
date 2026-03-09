//go:build integration

package integration

import (
	"context"
	"io"
	"log"
	"log/slog"
	"net"
	"net/url"
	"os"
	"testing"
	"time"

	internalha "codeberg.org/jhot/hats/internal/homeassistant"
	internalnats "codeberg.org/jhot/hats/internal/nats"
	"codeberg.org/jhot/hats/integration/haserver"
	"codeberg.org/jhot/hats/pkg/config"
	"github.com/nats-io/nats.go"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/stretchr/testify/require"
)

var (
	natsURL  string
	natsPort string
)

func TestMain(m *testing.M) {
	opts := &natsserver.Options{
		JetStream: true,
		Port:      -1,
		NoLog:     true,
		NoSigs:    true,
		StoreDir:  os.TempDir(),
	}
	ns, err := natsserver.NewServer(opts)
	if err != nil {
		log.Fatalf("failed to create embedded NATS server: %v", err)
	}
	go ns.Start()
	if !ns.ReadyForConnections(5 * time.Second) {
		log.Fatal("embedded NATS server did not become ready")
	}
	natsURL = ns.ClientURL()
	u, _ := url.Parse(natsURL)
	natsPort = u.Port()

	code := m.Run()
	ns.Shutdown()
	os.Exit(code)
}

// setupSubscriber wires the internal HA subscriber and NATS client to a mock HA server.
// It returns a plain *nats.Conn for test-side subscriptions and message assertions.
// Cleanup (CloseSubscription, NATS close) is registered automatically.
func setupSubscriber(t *testing.T, ha *haserver.MockHAServer) *nats.Conn {
	t.Helper()

	host, port, err := net.SplitHostPort(ha.Addr())
	require.NoError(t, err)

	cfg := &config.HatsConfig{
		HomeAssistantHost:  host,
		HomeAssistantPort:  port,
		HomeAssistantToken: "test-token",
		NatsHost:           "127.0.0.1",
		NatsPort:           natsPort,
		NatsClientName:     "test-hats",
	}

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	require.NoError(t, internalnats.JetstreamConnect(ctx, logger, cfg))
	require.NoError(t, internalnats.KvConnect())
	require.NoError(t, internalha.Subscribe(logger, cfg))

	// Wait for the subscriber to authenticate and set up event subscriptions.
	require.Eventually(t, func() bool {
		return ha.ConnectionCount() >= 1
	}, 3*time.Second, 50*time.Millisecond, "subscriber did not connect to mock HA")

	nc, err := nats.Connect(natsURL)
	require.NoError(t, err)

	t.Cleanup(func() {
		internalha.CloseSubscription()
		internalnats.Close()
		nc.Drain()
	})

	return nc
}

// listenAndReply subscribes to subject and responds "got it" to each message.
// Required for subjects where the subscriber uses PublishRequest (state, NFC, timer).
func listenAndReply(t *testing.T, nc *nats.Conn, subject string) *nats.Subscription {
	t.Helper()
	sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		msg.Respond([]byte("got it"))
	})
	require.NoError(t, err)
	t.Cleanup(func() { sub.Unsubscribe() })
	return sub
}

// captureWithReply subscribes to subject, responds "got it", and returns a channel of received messages.
// Use for subjects where the subscriber uses PublishRequest and you want to assert on the payload.
func captureWithReply(t *testing.T, nc *nats.Conn, subject string) <-chan *nats.Msg {
	t.Helper()
	ch := make(chan *nats.Msg, 32)
	sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		msg.Respond([]byte("got it"))
		ch <- msg
	})
	require.NoError(t, err)
	t.Cleanup(func() { sub.Unsubscribe() })
	return ch
}

// capture subscribes to subject and returns a channel of received messages.
// Use for subjects where the subscriber uses Publish (ZHA, Z-Wave) — no reply needed.
func capture(t *testing.T, nc *nats.Conn, subject string) <-chan *nats.Msg {
	t.Helper()
	ch := make(chan *nats.Msg, 32)
	sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		ch <- msg
	})
	require.NoError(t, err)
	t.Cleanup(func() { sub.Unsubscribe() })
	return ch
}

// receiveWithin waits up to d for one message on ch, failing the test if none arrives.
func receiveWithin(t *testing.T, ch <-chan *nats.Msg, d time.Duration) *nats.Msg {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(d):
		t.Fatalf("timed out after %v waiting for NATS message", d)
		return nil
	}
}
