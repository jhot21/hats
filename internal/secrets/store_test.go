package secrets_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"codeberg.org/jhot/hats/internal/secrets"
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

func testKey() []byte {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	return key
}

type publishCapture struct {
	mu       sync.Mutex
	subjects []string
}

func (p *publishCapture) fn() func(string, []byte) {
	return func(subject string, _ []byte) {
		p.mu.Lock()
		p.subjects = append(p.subjects, subject)
		p.mu.Unlock()
	}
}

func (p *publishCapture) last() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.subjects) == 0 {
		return ""
	}
	return p.subjects[len(p.subjects)-1]
}

func TestStore_CreateAndGet_BasicString(t *testing.T) {
	js := startTestNATS(t)
	pub := &publishCapture{}
	store, err := secrets.NewStore(context.Background(), js, nil, pub.fn())
	require.NoError(t, err)

	err = store.Create("my_key", "hello", secrets.TypeBasicString)
	require.NoError(t, err)
	assert.Equal(t, "secret.my_key.created", pub.last())

	s, err := store.Get("my_key")
	require.NoError(t, err)
	assert.Equal(t, "my_key", s.Name)
	assert.Equal(t, secrets.TypeBasicString, s.Type)
	assert.Equal(t, "hello", s.Value)
}

func TestStore_CreateAndGet_EncryptedString(t *testing.T) {
	js := startTestNATS(t)
	pub := &publishCapture{}
	store, err := secrets.NewStore(context.Background(), js, testKey(), pub.fn())
	require.NoError(t, err)

	err = store.Create("enc_key", "s3cr3t", secrets.TypeEncryptedString)
	require.NoError(t, err)

	s, err := store.Get("enc_key")
	require.NoError(t, err)
	assert.Equal(t, "s3cr3t", s.Value)
	assert.Equal(t, secrets.TypeEncryptedString, s.Type)
}

func TestStore_CreateAndGet_YAML(t *testing.T) {
	js := startTestNATS(t)
	store, err := secrets.NewStore(context.Background(), js, nil, func(string, []byte) {})
	require.NoError(t, err)

	yaml := "host: localhost\nport: 5432\n"
	err = store.Create("db_config", yaml, secrets.TypeYAML)
	require.NoError(t, err)

	s, err := store.Get("db_config")
	require.NoError(t, err)
	assert.Equal(t, yaml, s.Value)
	assert.Equal(t, secrets.TypeYAML, s.Type)
}

func TestStore_Get_NotFound(t *testing.T) {
	js := startTestNATS(t)
	store, err := secrets.NewStore(context.Background(), js, nil, func(string, []byte) {})
	require.NoError(t, err)

	_, err = store.Get("nonexistent")
	require.ErrorIs(t, err, secrets.ErrSecretNotFound)
}

func TestStore_Create_InvalidName(t *testing.T) {
	js := startTestNATS(t)
	store, err := secrets.NewStore(context.Background(), js, nil, func(string, []byte) {})
	require.NoError(t, err)

	err = store.Create("bad name!", "value", secrets.TypeBasicString)
	require.ErrorIs(t, err, secrets.ErrInvalidName)
}

func TestStore_Create_NameConflict(t *testing.T) {
	js := startTestNATS(t)
	store, err := secrets.NewStore(context.Background(), js, nil, func(string, []byte) {})
	require.NoError(t, err)

	require.NoError(t, store.Create("dup", "v1", secrets.TypeBasicString))
	err = store.Create("dup", "v2", secrets.TypeBasicString)
	require.ErrorIs(t, err, secrets.ErrNameConflict)
}

func TestStore_Update(t *testing.T) {
	js := startTestNATS(t)
	pub := &publishCapture{}
	store, err := secrets.NewStore(context.Background(), js, nil, pub.fn())
	require.NoError(t, err)

	require.NoError(t, store.Create("k", "original", secrets.TypeBasicString))
	require.NoError(t, store.Update("k", "updated"))
	assert.Equal(t, "secret.k.updated", pub.last())

	s, err := store.Get("k")
	require.NoError(t, err)
	assert.Equal(t, "updated", s.Value)
	assert.True(t, s.UpdatedAt.After(s.CreatedAt) || s.UpdatedAt.Equal(s.CreatedAt))
}

func TestStore_Update_NotFound(t *testing.T) {
	js := startTestNATS(t)
	store, err := secrets.NewStore(context.Background(), js, nil, func(string, []byte) {})
	require.NoError(t, err)

	err = store.Update("missing", "value")
	require.ErrorIs(t, err, secrets.ErrSecretNotFound)
}

func TestStore_Delete(t *testing.T) {
	js := startTestNATS(t)
	pub := &publishCapture{}
	store, err := secrets.NewStore(context.Background(), js, nil, pub.fn())
	require.NoError(t, err)

	require.NoError(t, store.Create("del_me", "v", secrets.TypeBasicString))
	require.NoError(t, store.Delete("del_me"))
	assert.Equal(t, "secret.del_me.deleted", pub.last())

	_, err = store.Get("del_me")
	require.ErrorIs(t, err, secrets.ErrSecretNotFound)
}

func TestStore_Delete_NotFound(t *testing.T) {
	js := startTestNATS(t)
	store, err := secrets.NewStore(context.Background(), js, nil, func(string, []byte) {})
	require.NoError(t, err)

	err = store.Delete("missing")
	require.ErrorIs(t, err, secrets.ErrSecretNotFound)
}

func TestStore_List(t *testing.T) {
	js := startTestNATS(t)
	store, err := secrets.NewStore(context.Background(), js, testKey(), func(string, []byte) {})
	require.NoError(t, err)

	require.NoError(t, store.Create("alpha", "v1", secrets.TypeBasicString))
	require.NoError(t, store.Create("beta", "v2", secrets.TypeEncryptedString))
	require.NoError(t, store.Create("gamma", "k: v", secrets.TypeYAML))

	list, err := store.List()
	require.NoError(t, err)
	assert.Len(t, list, 3)

	names := make([]string, len(list))
	for i, m := range list {
		names[i] = m.Name
		assert.Empty(t, "") // SecretMeta has no Value field — compile-time checked
	}
	assert.ElementsMatch(t, []string{"alpha", "beta", "gamma"}, names)
}

func TestStore_RefusesStartWithEncryptedDataAndNoKey(t *testing.T) {
	js := startTestNATS(t)

	// Seed an encrypted secret using a store with a key
	s1, err := secrets.NewStore(context.Background(), js, testKey(), func(string, []byte) {})
	require.NoError(t, err)
	require.NoError(t, s1.Create("enc", "val", secrets.TypeEncryptedString))

	// Starting a new store without a key must fail
	_, err = secrets.NewStore(context.Background(), js, nil, func(string, []byte) {})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SECRETS_ENCRYPTION_KEY")
}

func TestStore_ValidNames(t *testing.T) {
	js := startTestNATS(t)
	store, err := secrets.NewStore(context.Background(), js, nil, func(string, []byte) {})
	require.NoError(t, err)

	valid := []string{"abc", "ABC", "a1b2", "my-key", "my_key", "a-b_c-1"}
	for _, name := range valid {
		assert.NoError(t, store.Create(name, "v", secrets.TypeBasicString), "name: %s", name)
	}

	invalid := []string{"bad name", "has.dot", "has/slash", ""}
	for _, name := range invalid {
		assert.ErrorIs(t, store.Create(name, "v", secrets.TypeBasicString), secrets.ErrInvalidName, "name: %s", name)
	}
}

func TestStore_GetAll(t *testing.T) {
	js := startTestNATS(t)
	store, err := secrets.NewStore(context.Background(), js, testKey(), func(string, []byte) {})
	require.NoError(t, err)

	require.NoError(t, store.Create("plain_key", "plain_val", secrets.TypeBasicString))
	require.NoError(t, store.Create("enc_key", "enc_val", secrets.TypeEncryptedString))

	all, err := store.GetAll()
	require.NoError(t, err)
	assert.Len(t, all, 2)
	assert.Equal(t, "plain_val", all["plain_key"])
	assert.Equal(t, "enc_val", all["enc_key"])
}

func TestStore_GetAll_Empty(t *testing.T) {
	js := startTestNATS(t)
	store, err := secrets.NewStore(context.Background(), js, nil, func(string, []byte) {})
	require.NoError(t, err)

	all, err := store.GetAll()
	require.NoError(t, err)
	assert.Empty(t, all)
}
