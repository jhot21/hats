package tokens_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/jhot21/hats/internal/tokens"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func startTestNATS(t *testing.T) jetstream.JetStream {
	t.Helper()
	srv, err := natsserver.NewServer(&natsserver.Options{
		JetStream: true,
		Port:      -1,
		StoreDir:  t.TempDir(),
	})
	require.NoError(t, err)
	go srv.Start()
	require.True(t, srv.ReadyForConnections(5*time.Second))

	nc, err := nats.Connect(srv.ClientURL())
	require.NoError(t, err)

	js, err := jetstream.New(nc)
	require.NoError(t, err)

	t.Cleanup(func() {
		nc.Close()
		srv.Shutdown()
	})
	return js
}

func TestStore_CreateAndValidate(t *testing.T) {
	js := startTestNATS(t)
	store, err := tokens.NewStore(context.Background(), slog.Default(), js)
	require.NoError(t, err)

	raw, err := store.Create("test-token", []string{"read:states:*"}, nil)
	require.NoError(t, err)
	require.NotEmpty(t, raw)

	// Watch propagation takes a moment
	time.Sleep(50 * time.Millisecond)

	tok, err := store.Validate(raw)
	require.NoError(t, err)
	assert.Equal(t, "test-token", tok.Name)
	assert.Equal(t, []string{"read:states:*"}, tok.Scopes)
	assert.Nil(t, tok.ExpiresAt)
}

func TestStore_Validate_WrongToken(t *testing.T) {
	js := startTestNATS(t)
	store, err := tokens.NewStore(context.Background(), slog.Default(), js)
	require.NoError(t, err)

	_, err = store.Validate("hats_doesnotexist12345678901234567890123456")
	require.ErrorIs(t, err, tokens.ErrTokenNotFound)
}

func TestStore_Validate_Expired(t *testing.T) {
	js := startTestNATS(t)
	store, err := tokens.NewStore(context.Background(), slog.Default(), js)
	require.NoError(t, err)

	past := time.Now().Add(-time.Hour)
	raw, err := store.Create("expired", []string{"*:*:*"}, &past)
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	_, err = store.Validate(raw)
	require.ErrorIs(t, err, tokens.ErrTokenExpired)
}

func TestStore_Revoke(t *testing.T) {
	js := startTestNATS(t)
	store, err := tokens.NewStore(context.Background(), slog.Default(), js)
	require.NoError(t, err)

	raw, err := store.Create("to-revoke", []string{"*:*:*"}, nil)
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	tok, err := store.Validate(raw)
	require.NoError(t, err)

	err = store.Revoke(tok.Hash)
	require.NoError(t, err)

	// Revoke clears the in-memory cache immediately
	_, err = store.Validate(raw)
	require.ErrorIs(t, err, tokens.ErrTokenNotFound)
}

func TestStore_Revoke_NonExistent(t *testing.T) {
	js := startTestNATS(t)
	store, err := tokens.NewStore(context.Background(), slog.Default(), js)
	require.NoError(t, err)

	err = store.Revoke("nonexistenthash")
	require.NoError(t, err) // should not error
}

func TestStore_List(t *testing.T) {
	js := startTestNATS(t)
	store, err := tokens.NewStore(context.Background(), slog.Default(), js)
	require.NoError(t, err)

	_, err = store.Create("alpha", []string{"read:states:*"}, nil)
	require.NoError(t, err)
	_, err = store.Create("beta", []string{"write:command:bedtime"}, nil)
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	list, err := store.List()
	require.NoError(t, err)
	assert.Len(t, list, 2)

	names := make([]string, len(list))
	for i, t := range list {
		names[i] = t.Name
	}
	assert.ElementsMatch(t, []string{"alpha", "beta"}, names)
}

func TestStore_ContextCancellation(t *testing.T) {
	js := startTestNATS(t)
	ctx, cancel := context.WithCancel(context.Background())
	_, err := tokens.NewStore(ctx, slog.Default(), js)
	require.NoError(t, err)

	// Cancel the context and give goroutines a moment to exit.
	cancel()
	time.Sleep(50 * time.Millisecond)
	// The race detector will catch any goroutine accessing freed state after cancel.
}
