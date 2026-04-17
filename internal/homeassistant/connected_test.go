package homeassistant

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsConnected_DefaultFalse(t *testing.T) {
	// Package starts unconnected
	assert.False(t, IsConnected())
}

func TestSetConnected(t *testing.T) {
	setConnected(true)
	assert.True(t, IsConnected())
	setConnected(false)
	assert.False(t, IsConnected())
}
