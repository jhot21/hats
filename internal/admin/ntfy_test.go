package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNtfyTokenHandler(t *testing.T) {
	t.Run("returns revealed wrapper containing stored token", func(t *testing.T) {
		adminSecretStore = &mockSecretStore{data: map[string]string{"NTFY_TOKEN": "tk_secret123"}}
		req := httptest.NewRequest(http.MethodGet, "/admin/api/ntfy/token", nil)
		w := httptest.NewRecorder()
		ntfyTokenHandler(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()
		assert.Contains(t, body, "ntfy-token-wrapper")
		assert.Contains(t, body, "tk_secret123")
	})

	t.Run("returns revealed wrapper with empty input when token not stored", func(t *testing.T) {
		adminSecretStore = &mockSecretStore{data: map[string]string{}}
		req := httptest.NewRequest(http.MethodGet, "/admin/api/ntfy/token", nil)
		w := httptest.NewRecorder()
		ntfyTokenHandler(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()
		assert.Contains(t, body, "ntfy-token-wrapper")
		assert.Contains(t, body, "No token stored")
	})
}
