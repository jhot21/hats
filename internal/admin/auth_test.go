// internal/admin/auth_test.go
package admin

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"codeberg.org/jhot/hats/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestGenerateToken(t *testing.T) {
	key := []byte("test-secret-key-32-bytes-padding!")
	token, err := generateToken(key)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, 2, countDots(token))
}

func TestValidateToken(t *testing.T) {
	key := []byte("test-secret-key-32-bytes-padding!")

	t.Run("valid token", func(t *testing.T) {
		token, _ := generateToken(key)
		claims, err := validateToken(token, key)
		require.NoError(t, err)
		assert.Equal(t, "admin", claims.Subject)
	})

	t.Run("wrong key rejected", func(t *testing.T) {
		token, _ := generateToken(key)
		_, err := validateToken(token, []byte("different-key"))
		assert.Error(t, err)
	})

	t.Run("malformed token rejected", func(t *testing.T) {
		_, err := validateToken("not.a.valid.jwt", key)
		assert.Error(t, err)
	})
}

func TestLoginHandler(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.MinCost)
	adminCfg = &config.HatsConfig{AdminPasswordHash: string(hash)}
	sessionKey = []byte("test-session-key-32-bytes-padded")

	t.Run("GET renders login page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/login", nil)
		w := httptest.NewRecorder()
		loginHandler(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "HATS Admin")
	})

	t.Run("POST correct password sets cookie and redirects", func(t *testing.T) {
		form := url.Values{"password": {"testpassword"}}
		req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		loginHandler(w, req)
		assert.Equal(t, http.StatusSeeOther, w.Code)
		assert.Equal(t, "/admin", w.Header().Get("Location"))
		var sessionCookie *http.Cookie
		for _, c := range w.Result().Cookies() {
			if c.Name == cookieName {
				sessionCookie = c
			}
		}
		require.NotNil(t, sessionCookie)
		assert.NotEmpty(t, sessionCookie.Value)
	})

	t.Run("POST wrong password re-renders with error", func(t *testing.T) {
		form := url.Values{"password": {"wrongpassword"}}
		req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		loginHandler(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid password")
	})

	t.Run("POST with next param redirects to /admin/status", func(t *testing.T) {
		form := url.Values{"password": {"testpassword"}}
		req := httptest.NewRequest(http.MethodPost, "/admin/login?next=%2Fadmin%2Fstatus", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		loginHandler(w, req)
		assert.Equal(t, http.StatusSeeOther, w.Code)
		assert.Equal(t, "/admin/status", w.Header().Get("Location"))
	})

	t.Run("POST with non-admin next falls back to /admin", func(t *testing.T) {
		form := url.Values{"password": {"testpassword"}}
		req := httptest.NewRequest(http.MethodPost, "/admin/login?next=%2Fevil", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		loginHandler(w, req)
		assert.Equal(t, http.StatusSeeOther, w.Code)
		assert.Equal(t, "/admin", w.Header().Get("Location"))
	})
}

func TestRequireAuth(t *testing.T) {
	sessionKey = []byte("test-session-key-32-bytes-padded")

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := requireAuth(next)

	t.Run("no cookie redirects to login", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusSeeOther, w.Code)
		assert.Contains(t, w.Header().Get("Location"), "/admin/login")
	})

	t.Run("valid cookie passes through", func(t *testing.T) {
		token, _ := generateToken(sessionKey)
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.AddCookie(&http.Cookie{Name: cookieName, Value: token})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid cookie redirects to login", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.AddCookie(&http.Cookie{Name: cookieName, Value: "bad.token.value"})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusSeeOther, w.Code)
		assert.Contains(t, w.Header().Get("Location"), "/admin/login")
	})
}

func countDots(s string) int {
	count := 0
	for _, c := range s {
		if c == '.' {
			count++
		}
	}
	return count
}
