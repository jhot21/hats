package tokens

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

const tokenPrefix = "hats_"

// GenerateToken returns a new cryptographically random opaque bearer token.
// Format: hats_ + 32 URL-safe base64 characters (24 random bytes).
func GenerateToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return tokenPrefix + base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken returns the hex-encoded SHA-256 of rawToken.
// This is used as the NATS KV key — the raw token is never stored.
func HashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}
