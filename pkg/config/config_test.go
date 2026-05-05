package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
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
