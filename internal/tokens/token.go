package tokens

import (
	"context"
	"time"
)

// Token is the in-memory representation of an API token.
type Token struct {
	Hash       string     // hex-encoded SHA-256 of raw token; used as KV key and UI identifier
	Name       string
	Scopes     []string
	ExpiresAt  *time.Time
	CreatedAt  time.Time
	LastUsedAt *time.Time
}

// tokenRecord is the JSON shape stored in NATS KV (no hash — it's the key).
type tokenRecord struct {
	Name       string     `json:"name"`
	Scopes     []string   `json:"scopes"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

type contextKey struct{}

// TokenFromContext retrieves the authenticated token from the request context.
// Returns nil if no token is present (unauthenticated request or non-API route).
func TokenFromContext(ctx context.Context) *Token {
	t, _ := ctx.Value(contextKey{}).(*Token)
	return t
}

// ContextWithToken stores t in ctx for retrieval by TokenFromContext.
func ContextWithToken(ctx context.Context, t *Token) context.Context {
	return context.WithValue(ctx, contextKey{}, t)
}
