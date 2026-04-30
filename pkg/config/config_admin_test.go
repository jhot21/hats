package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminConfig(t *testing.T) {
	os.Setenv("ADMIN_PASSWORD_HASH", "$2a$12$testhash")
	os.Setenv("ADMIN_SESSION_SECRET", "mysecret")
	defer os.Unsetenv("ADMIN_PASSWORD_HASH")
	defer os.Unsetenv("ADMIN_SESSION_SECRET")

	srv := &ServerConfig{}
	srv.SetValues(ConfigValueSourceEnv)

	assert.Equal(t, "$2a$12$testhash", srv.AdminPasswordHash)
	assert.Equal(t, "mysecret", srv.AdminSessionSecret)
}

func TestNewServer_AdminFields(t *testing.T) {
	os.Setenv("ADMIN_PASSWORD_HASH", "$2a$12$anotherhash")
	os.Setenv("HATS_TOKEN", "")
	defer os.Unsetenv("ADMIN_PASSWORD_HASH")

	srv, err := NewServer()
	require.NoError(t, err)
	assert.Equal(t, "$2a$12$anotherhash", srv.AdminPasswordHash)
}
