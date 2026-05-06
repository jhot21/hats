package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"codeberg.org/jhot/hats/internal/secrets"
	"github.com/stretchr/testify/assert"
)

// mockSecretStore is a test double for secrets.SecretStore.
type mockSecretStore struct {
	data map[string]string
}

func (m *mockSecretStore) Get(name string) (*secrets.Secret, error) {
	v, ok := m.data[name]
	if !ok {
		return nil, secrets.ErrSecretNotFound
	}
	return &secrets.Secret{Name: name, Value: v}, nil
}

func (m *mockSecretStore) List() ([]*secrets.SecretMeta, error)                    { return nil, nil }
func (m *mockSecretStore) GetAll() (map[string]string, error)                       { return nil, nil }
func (m *mockSecretStore) Create(name, value string, typ secrets.SecretType) error { return nil }
func (m *mockSecretStore) Update(name, value string) error                          { return nil }
func (m *mockSecretStore) Delete(name string) error                                 { return nil }

func TestHATokenHandler(t *testing.T) {
	t.Run("returns revealed wrapper containing stored token", func(t *testing.T) {
		adminSecretStore = &mockSecretStore{data: map[string]string{"HASS_TOKEN": "eyABCDEF1234"}}
		req := httptest.NewRequest(http.MethodGet, "/admin/api/homeassistant/token", nil)
		w := httptest.NewRecorder()
		haTokenHandler(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()
		assert.Contains(t, body, "ha-token-wrapper")
		assert.Contains(t, body, "eyABCDEF1234")
	})

	t.Run("returns revealed wrapper with empty input when token not stored", func(t *testing.T) {
		adminSecretStore = &mockSecretStore{data: map[string]string{}}
		req := httptest.NewRequest(http.MethodGet, "/admin/api/homeassistant/token", nil)
		w := httptest.NewRecorder()
		haTokenHandler(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()
		assert.Contains(t, body, "ha-token-wrapper")
		assert.Contains(t, body, "No token stored")
	})
}
