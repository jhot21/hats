package nats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsConnected_DefaultFalse(t *testing.T) {
	assert.False(t, IsConnected())
}
