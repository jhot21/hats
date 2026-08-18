package tokens_test

import (
	"strings"
	"testing"

	"github.com/jhot21/hats/internal/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateToken_HasPrefix(t *testing.T) {
	raw, err := tokens.GenerateToken()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(raw, "hats_"), "token must start with hats_")
}

func TestGenerateToken_Length(t *testing.T) {
	raw, err := tokens.GenerateToken()
	require.NoError(t, err)
	// "hats_" (5) + 40 random chars = 45 total
	assert.Equal(t, 45, len(raw))
}

func TestGenerateToken_Alphanumeric(t *testing.T) {
	raw, err := tokens.GenerateToken()
	require.NoError(t, err)
	suffix := raw[len("hats_"):]
	for _, c := range suffix {
		assert.True(t, (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9'), "suffix must be alphanumeric, got %q", c)
	}
}

func TestGenerateToken_Unique(t *testing.T) {
	a, err := tokens.GenerateToken()
	require.NoError(t, err)
	b, err := tokens.GenerateToken()
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
}

func TestHashToken_Deterministic(t *testing.T) {
	raw := "hats_testtoken12345678901234567"
	h1 := tokens.HashToken(raw)
	h2 := tokens.HashToken(raw)
	assert.Equal(t, h1, h2)
}

func TestHashToken_DifferentInputs(t *testing.T) {
	assert.NotEqual(t, tokens.HashToken("hats_aaa"), tokens.HashToken("hats_bbb"))
}

func TestHashToken_IsHex(t *testing.T) {
	h := tokens.HashToken("hats_testtoken12345678901234567")
	// SHA-256 hex = 64 chars
	assert.Len(t, h, 64)
	for _, c := range h {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'), "must be lowercase hex")
	}
}
