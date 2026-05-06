package admin

import (
	"errors"
	"fmt"
	"net/http"

	ha "codeberg.org/jhot/go-ha-sdk"
	"codeberg.org/jhot/hats/internal/admin/templates"
	"codeberg.org/jhot/hats/internal/secrets"
)

var haReservedNames = map[string]bool{
	"HASS_HOST":   true,
	"HASS_PORT":   true,
	"HASS_SECURE": true,
	"HASS_TOKEN":  true,
}

// haPageHandler renders the HA connection config page, pre-populated from the secret store.
func haPageHandler(w http.ResponseWriter, r *http.Request) {
	data := templates.HAConnectionData{Port: "8123"}
	if s, err := adminSecretStore.Get("HASS_HOST"); err == nil {
		data.Host = s.Value
	}
	if s, err := adminSecretStore.Get("HASS_PORT"); err == nil {
		data.Port = s.Value
	}
	if s, err := adminSecretStore.Get("HASS_SECURE"); err == nil {
		data.Secure = s.Value == "true"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.HomeAssistant(data).Render(r.Context(), w)
}

// haSaveHandler saves the four HASS_* secrets using delete-then-recreate.
// It skips HASS_TOKEN if the token field is empty (keep existing).
// Returns HX-Redirect on success so HTMX reloads the full page.
func haSaveHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	if adminCfg.SecretsEncryptionKey == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.HATestResult(false, "HASS_TOKEN cannot be saved: SECRETS_ENCRYPTION_KEY is not configured").Render(r.Context(), w)
		return
	}

	secureVal := "false"
	if r.FormValue("secure") == "true" {
		secureVal = "true"
	}

	type field struct {
		name  string
		value string
		typ   secrets.SecretType
	}
	fields := []field{
		{"HASS_HOST", r.FormValue("host"), secrets.TypeBasicString},
		{"HASS_PORT", r.FormValue("port"), secrets.TypeBasicString},
		{"HASS_SECURE", secureVal, secrets.TypeBasicString},
	}
	if token := r.FormValue("token"); token != "" {
		fields = append(fields, field{"HASS_TOKEN", token, secrets.TypeEncryptedString})
	}

	for _, f := range fields {
		if err := adminSecretStore.Delete(f.name); err != nil && !errors.Is(err, secrets.ErrSecretNotFound) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			templates.HATestResult(false, fmt.Sprintf("failed to update %s: %s", f.name, err.Error())).Render(r.Context(), w)
			return
		}
		if err := adminSecretStore.Create(f.name, f.value, f.typ); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			templates.HATestResult(false, fmt.Sprintf("failed to save %s: %s", f.name, err.Error())).Render(r.Context(), w)
			return
		}
	}

	w.Header().Set("HX-Redirect", "/admin/homeassistant")
	w.WriteHeader(http.StatusOK)
}

// haTokenHandler fetches the stored HASS_TOKEN and returns the revealed token wrapper fragment.
func haTokenHandler(w http.ResponseWriter, r *http.Request) {
	var token string
	if s, err := adminSecretStore.Get("HASS_TOKEN"); err == nil {
		token = s.Value
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.HATokenReveal(token).Render(r.Context(), w)
}

// haTestHandler tests a connection to Home Assistant using the submitted form values.
// If the token field is empty, it falls back to the stored token so users can test
// the current configuration without re-entering it.
func haTestHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	host := r.FormValue("host")
	port := r.FormValue("port")
	secure := r.FormValue("secure") == "true"
	token := r.FormValue("token")
	if token == "" {
		if s, err := adminSecretStore.Get("HASS_TOKEN"); err == nil {
			token = s.Value
		}
	}

	protocol := "http"
	if secure {
		protocol = "https"
	}
	baseURL := fmt.Sprintf("%s://%s:%s", protocol, host, port)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ha.New(baseURL, token).Ping(); err != nil {
		templates.HATestResult(false, err.Error()).Render(r.Context(), w)
		return
	}
	templates.HATestResult(true, "Connected successfully.").Render(r.Context(), w)
}
