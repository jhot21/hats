package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdminConfig(t *testing.T) {
	os.Setenv("ADMIN_PASSWORD_HASH", "$2a$12$testhash")
	os.Setenv("ADMIN_SESSION_SECRET", "mysecret")
	defer os.Unsetenv("ADMIN_PASSWORD_HASH")
	defer os.Unsetenv("ADMIN_SESSION_SECRET")

	cfg := &HatsConfig{}
	cfg.SetValues(ConfigValueSourceEnv)

	assert.Equal(t, "$2a$12$testhash", cfg.AdminPasswordHash)
	assert.Equal(t, "mysecret", cfg.AdminSessionSecret)
}
