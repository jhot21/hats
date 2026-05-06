package tokens

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

const tokenPrefix = "hats_"

const alphanumeric = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateToken returns a new cryptographically random opaque bearer token.
// Format: hats_ + 40 alphanumeric characters.
func GenerateToken() (string, error) {
	b := make([]byte, 40)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = alphanumeric[int(b[i])%len(alphanumeric)]
	}
	return tokenPrefix + string(b), nil
}

// HashToken returns the hex-encoded SHA-256 of rawToken.
// This is used as the NATS KV key — the raw token is never stored.
func HashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}
