package ntfy

import (
	"testing"
	"time"

	"codeberg.org/jhot/hats/internal/secrets"
	"github.com/stretchr/testify/assert"
)

// mockStore satisfies secrets.SecretStore using an in-memory map.
type mockStore struct {
	data map[string]string
}

func (m *mockStore) Get(name string) (*secrets.Secret, error) {
	v, ok := m.data[name]
	if !ok {
		return nil, secrets.ErrSecretNotFound
	}
	return &secrets.Secret{Name: name, Type: secrets.TypeBasicString, Value: v, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
}
func (m *mockStore) List() ([]*secrets.SecretMeta, error)                      { return nil, nil }
func (m *mockStore) GetAll() (map[string]string, error)                        { return nil, nil }
func (m *mockStore) Create(n, v string, t secrets.SecretType) error            { return nil }
func (m *mockStore) Update(n, v string) error                                  { return nil }
func (m *mockStore) Delete(n string) error                                     { return nil }

func TestResolveCredentials_FallsBackToBaseValues(t *testing.T) {
	c := New(&mockStore{data: map[string]string{}}, "https://ntfy.sh", "base-token")
	host, token := c.resolveCredentials()
	assert.Equal(t, "https://ntfy.sh", host)
	assert.Equal(t, "base-token", token)
}

func TestResolveCredentials_StoreOverridesBase(t *testing.T) {
	c := New(&mockStore{data: map[string]string{
		"NTFY_HOST":  "https://my-ntfy.example.com",
		"NTFY_TOKEN": "store-token",
	}}, "https://ntfy.sh", "base-token")
	host, token := c.resolveCredentials()
	assert.Equal(t, "https://my-ntfy.example.com", host)
	assert.Equal(t, "store-token", token)
}

func TestResolveCredentials_PartialStoreOverride(t *testing.T) {
	c := New(&mockStore{data: map[string]string{
		"NTFY_HOST": "https://my-ntfy.example.com",
	}}, "https://ntfy.sh", "base-token")
	host, token := c.resolveCredentials()
	assert.Equal(t, "https://my-ntfy.example.com", host)
	assert.Equal(t, "base-token", token)
}
