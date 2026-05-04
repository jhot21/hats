package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCustomSetting_EnvVarTakesPrecedence(t *testing.T) {
	os.Setenv("MY_SETTING", "from-env")
	defer os.Unsetenv("MY_SETTING")

	cfg := &HatsConfig{secrets: map[string]string{"MY_SETTING": "from-cache"}}
	assert.Equal(t, "from-env", cfg.GetCustomSetting("MY_SETTING", "default"))
}

func TestGetCustomSetting_FallsBackToSecretsCache(t *testing.T) {
	os.Unsetenv("MY_SETTING")
	cfg := &HatsConfig{secrets: map[string]string{"MY_SETTING": "from-cache"}}
	assert.Equal(t, "from-cache", cfg.GetCustomSetting("MY_SETTING", "default"))
}

func TestGetCustomSetting_FallsBackToDefault(t *testing.T) {
	os.Unsetenv("MY_SETTING")
	cfg := &HatsConfig{}
	assert.Equal(t, "default", cfg.GetCustomSetting("MY_SETTING", "default"))
}

func TestServerConfig_SetValues_PopulatesEmbeddedAndOwnFields(t *testing.T) {
	os.Setenv("LOG_LEVEL", "DEBUG")
	os.Setenv("ADMIN_PASSWORD_HASH", "$2a$12$testhash")
	defer os.Unsetenv("LOG_LEVEL")
	defer os.Unsetenv("ADMIN_PASSWORD_HASH")

	srv := &ServerConfig{}
	srv.SetValues(ConfigValueSourceEnv)

	assert.Equal(t, "DEBUG", srv.LogLvl)
	assert.Equal(t, "$2a$12$testhash", srv.AdminPasswordHash)
}

func TestNewServer_ReturnsServerConfig(t *testing.T) {
	os.Setenv("ADMIN_PASSWORD_HASH", "$2a$12$testhash")
	os.Setenv("HATS_TOKEN", "") // ensure HATS overlay is skipped
	defer os.Unsetenv("ADMIN_PASSWORD_HASH")

	srv, err := NewServer()
	require.NoError(t, err)
	assert.Equal(t, "$2a$12$testhash", srv.AdminPasswordHash)
}

func TestHATSOverlaySkippedWhenTokenEmpty(t *testing.T) {
	os.Unsetenv("HATS_TOKEN")
	cfg := &HatsConfig{HatsHost: "hats", HatsToken: ""}
	err := cfg.applyHATSSecrets()
	assert.NoError(t, err)
	assert.Nil(t, cfg.secrets)
}

func TestGetHomeAssistantToken_ReturnsToken(t *testing.T) {
	cfg := &HatsConfig{HomeAssistantToken: "mytoken"}
	assert.Equal(t, "mytoken", cfg.GetHomeAssistantToken())
}

func TestGetHomeAssistantToken_EmptyWhenNotSet(t *testing.T) {
	cfg := &HatsConfig{}
	assert.Equal(t, "", cfg.GetHomeAssistantToken())
}
