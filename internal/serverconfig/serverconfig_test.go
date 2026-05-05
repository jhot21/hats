package serverconfig_test

import (
	"errors"
	"testing"

	"codeberg.org/jhot/hats/internal/secrets"
	"codeberg.org/jhot/hats/internal/serverconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSecretStore satisfies secrets.SecretStore with controllable GetAll behaviour.
type mockSecretStore struct {
	vals map[string]string
	err  error
}

func (m *mockSecretStore) GetAll() (map[string]string, error)           { return m.vals, m.err }
func (m *mockSecretStore) Get(name string) (*secrets.Secret, error)     { return nil, secrets.ErrSecretNotFound }
func (m *mockSecretStore) List() ([]*secrets.SecretMeta, error)         { return nil, nil }
func (m *mockSecretStore) Create(name, value string, typ secrets.SecretType) error { return nil }
func (m *mockSecretStore) Update(name, value string) error              { return nil }
func (m *mockSecretStore) Delete(name string) error                     { return nil }

func TestNewBootstrap_Defaults(t *testing.T) {
	srv := serverconfig.NewBootstrap()
	assert.Equal(t, "127.0.0.1", srv.HomeAssistantHost)
	assert.Equal(t, "8123", srv.HomeAssistantPort)
	assert.False(t, srv.HomeAssistantSecure)
}

func TestNewBootstrap_EnvOverridesDefault(t *testing.T) {
	t.Setenv("HASS_HOST", "ha.local")
	t.Setenv("HASS_PORT", "8124")
	srv := serverconfig.NewBootstrap()
	assert.Equal(t, "ha.local", srv.HomeAssistantHost)
	assert.Equal(t, "8124", srv.HomeAssistantPort)
}

func TestNewBootstrap_ServerSpecificFields(t *testing.T) {
	t.Setenv("SECRETS_ENCRYPTION_KEY", "abc123")
	srv := serverconfig.NewBootstrap()
	assert.Equal(t, "abc123", srv.SecretsEncryptionKey)
}

func TestApplySecretStore_OverridesBootstrapValues(t *testing.T) {
	t.Setenv("HASS_HOST", "env.host")
	srv := serverconfig.NewBootstrap()
	require.Equal(t, "env.host", srv.HomeAssistantHost)

	ms := &mockSecretStore{vals: map[string]string{"HASS_HOST": "stored.host"}}
	err := srv.ApplySecretStore(ms)
	require.NoError(t, err)
	assert.Equal(t, "stored.host", srv.HomeAssistantHost)
}

func TestApplySecretStore_ErrorPassedThrough(t *testing.T) {
	srv := serverconfig.NewBootstrap()
	ms := &mockSecretStore{err: errors.New("store unavailable")}
	err := srv.ApplySecretStore(ms)
	require.ErrorContains(t, err, "store unavailable")
}

func TestApplySecretStore_EmptyValueDoesNotOverride(t *testing.T) {
	t.Setenv("HASS_HOST", "env.host")
	srv := serverconfig.NewBootstrap()

	// ApplyValues skips empty strings, so an empty stored value must not clear env value.
	ms := &mockSecretStore{vals: map[string]string{"HASS_HOST": ""}}
	err := srv.ApplySecretStore(ms)
	require.NoError(t, err)
	assert.Equal(t, "env.host", srv.HomeAssistantHost)
}
