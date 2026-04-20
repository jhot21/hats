package homeassistant

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsConnected_DefaultFalse(t *testing.T) {
	setConnected(false)
	assert.False(t, IsConnected())
}

func TestSetConnected(t *testing.T) {
	t.Cleanup(func() { setConnected(false) })
	setConnected(true)
	assert.True(t, IsConnected())
	setConnected(false)
	assert.False(t, IsConnected())
}
