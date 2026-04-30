package secrets

import (
	"errors"
	"time"
)

// SecretType identifies storage and display type.
type SecretType string

const (
	TypeBasicString     SecretType = "basic_string"
	TypeEncryptedString SecretType = "encrypted_string"
	TypeYAML            SecretType = "yaml"
	TypeJSON            SecretType = "json"
)

// Secret is the full representation — Value is always plaintext.
type Secret struct {
	Name      string
	Type      SecretType
	Value     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SecretMeta is safe for listing — no value.
type SecretMeta struct {
	Name      string
	Type      SecretType
	CreatedAt time.Time
	UpdatedAt time.Time
}

var (
	ErrSecretNotFound = errors.New("secret not found")
	ErrInvalidName    = errors.New("secret name must match [a-zA-Z0-_-]+")
	ErrNameConflict   = errors.New("secret already exists")
)
