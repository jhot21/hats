package serverconfig

import (
	"reflect"

	"codeberg.org/jhot/hats/internal/secrets"
	"codeberg.org/jhot/hats/pkg/config"
)

// ServerConfig extends HatsConfig with fields only needed by the HATS server process.
// It lives in internal/ so it can read directly from the secret store.
type ServerConfig struct {
	config.HatsConfig
	SecretsEncryptionKey string `config:"SECRETS_ENCRYPTION_KEY" default:""`
	AdminPasswordHash    string `config:"ADMIN_PASSWORD_HASH" default:""`
	AdminSessionSecret   string `config:"ADMIN_SESSION_SECRET" default:""`
}

// NewBootstrap reads from defaults and env vars only.
// Call this before NATS connects — it returns enough config to bootstrap the server.
func NewBootstrap() *ServerConfig {
	srv := &ServerConfig{}
	config.ApplyValues(reflect.ValueOf(srv).Elem(), config.ConfigValueSourceDefault)
	config.ApplyValues(reflect.ValueOf(srv).Elem(), config.ConfigValueSourceEnv)
	return srv
}

// ApplySecretStore overlays all secrets from the store onto the config.
// Returns an error if the store is unavailable; callers should log and continue.
func (s *ServerConfig) ApplySecretStore(ss secrets.SecretStore) error {
	vals, err := ss.GetAll()
	if err != nil {
		return err
	}
	config.ApplyValues(reflect.ValueOf(s).Elem(), config.ConfigValueSourceHATS, vals)
	return nil
}
