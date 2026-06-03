// internal/admin/auth.go
package admin

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"codeberg.org/jhot/hats/internal/admin/templates"
	"codeberg.org/jhot/hats/internal/serverconfig"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const cookieName = "hats_admin_session"

var (
	adminCfg    *serverconfig.ServerConfig
	adminLogger *slog.Logger
	sessionKey  []byte
)

func initSessionKey() {
	if adminCfg.AdminSessionSecret != "" {
		sessionKey = []byte(adminCfg.AdminSessionSecret)
		return
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic("admin: failed to generate session key: " + err.Error())
	}
	sessionKey = key
	adminLogger.Warn("ADMIN_SESSION_SECRET not set; admin sessions will be invalidated on restart")
}

func generateToken(secret []byte) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   "admin",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

func validateToken(tokenStr string, secret []byte) (*jwt.RegisteredClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}

func setSessionCookie(w http.ResponseWriter, tokenStr string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    tokenStr,
		Path:     "/admin",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400,
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/admin",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if adminCfg.AdminPasswordHash == "" {
		http.Error(w, "admin not configured: set ADMIN_PASSWORD_HASH", http.StatusServiceUnavailable)
		return
	}

	next := r.FormValue("next")
	if !strings.HasPrefix(next, "/admin") {
		next = ""
	}

	if r.Method == http.MethodGet {
		templates.Login("", next).Render(r.Context(), w)
		return
	}

	password := r.FormValue("password")
	if err := bcrypt.CompareHashAndPassword([]byte(adminCfg.AdminPasswordHash), []byte(password)); err != nil {
		templates.Login("Invalid password", next).Render(r.Context(), w)
		return
	}

	tokenStr, err := generateToken(sessionKey)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	setSessionCookie(w, tokenStr)

	dest := next
	if dest == "" {
		dest = "/admin"
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	clearSessionCookie(w)
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			redirectToLogin(w, r)
			return
		}
		if _, err := validateToken(cookie.Value, sessionKey); err != nil {
			clearSessionCookie(w)
			redirectToLogin(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	target := "/admin/login?next=" + url.QueryEscape(r.URL.RequestURI())
	http.Redirect(w, r, target, http.StatusSeeOther)
}
