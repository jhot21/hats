package admin

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"codeberg.org/jhot/hats/internal/admin/templates"
	"codeberg.org/jhot/hats/internal/secrets"
	"github.com/go-resty/resty/v2"
)

var ntfyReservedNames = map[string]bool{
	"NTFY_HOST":  true,
	"NTFY_TOKEN": true,
}

// ntfyPageHandler renders the ntfy connection config page, pre-populated from the secret store.
func ntfyPageHandler(w http.ResponseWriter, r *http.Request) {
	data := templates.NtfyConnectionData{Host: "https://ntfy.sh"}
	if s, err := adminSecretStore.Get("NTFY_HOST"); err == nil {
		data.Host = s.Value
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.Ntfy(data).Render(r.Context(), w)
}

// ntfySaveHandler saves NTFY_HOST and (optionally) NTFY_TOKEN to the secret store.
// Token is skipped if the field is empty (keep-existing behaviour).
// Returns HX-Redirect on success so HTMX reloads the full page.
func ntfySaveHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	host := r.FormValue("host")
	if host == "" {
		host = "https://ntfy.sh"
	}

	type field struct {
		name  string
		value string
		typ   secrets.SecretType
	}
	fields := []field{
		{"NTFY_HOST", host, secrets.TypeBasicString},
	}
	if token := r.FormValue("token"); token != "" {
		fields = append(fields, field{"NTFY_TOKEN", token, secrets.TypeBasicString})
	}

	for _, f := range fields {
		if err := adminSecretStore.Delete(f.name); err != nil && !errors.Is(err, secrets.ErrSecretNotFound) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			templates.NtfySaveResult(false, fmt.Sprintf("failed to update %s: %s", f.name, err.Error())).Render(r.Context(), w)
			return
		}
		if err := adminSecretStore.Create(f.name, f.value, f.typ); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			templates.NtfySaveResult(false, fmt.Sprintf("failed to save %s: %s", f.name, err.Error())).Render(r.Context(), w)
			return
		}
	}

	w.Header().Set("HX-Redirect", "/admin/ntfy")
	w.WriteHeader(http.StatusOK)
}

// ntfyTestHandler verifies write and read access to a ntfy topic.
// Step 1: POST a test message. Step 2: poll for it with since=<msgID>.
// Falls back to the stored token if the token form field is empty.
func ntfyTestHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	host := r.FormValue("host")
	if host == "" {
		host = "https://ntfy.sh"
	}
	token := r.FormValue("token")
	if token == "" {
		if s, err := adminSecretStore.Get("NTFY_TOKEN"); err == nil {
			token = s.Value
		}
	}
	topic := r.FormValue("test_topic")
	if topic == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.NtfyTestResult(false, "test topic is required", false, "skipped").Render(r.Context(), w)
		return
	}

	rc := resty.New()
	if token != "" {
		rc.SetHeader("Authorization", "Bearer "+token)
	}

	// Step 1: write test — publish a message to the topic.
	var publishResult struct {
		ID string `json:"id"`
	}
	resp, err := rc.R().
		SetBody(map[string]string{
			"topic":   topic,
			"message": "HATS connection test",
			"title":   "HATS Test",
		}).
		SetResult(&publishResult).
		Post(host)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.NtfyTestResult(false, err.Error(), false, "skipped").Render(r.Context(), w)
		return
	}
	if !resp.IsSuccess() {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.NtfyTestResult(false, fmt.Sprintf("HTTP %d", resp.StatusCode()), false, "skipped").Render(r.Context(), w)
		return
	}

	msgID := publishResult.ID

	// Step 2: read test — poll for the message we just published.
	pollResp, err := rc.R().Get(fmt.Sprintf("%s/%s/json?poll=1&since=%s", host, topic, msgID))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err != nil {
		templates.NtfyTestResult(true, "OK", false, err.Error()).Render(r.Context(), w)
		return
	}
	if !pollResp.IsSuccess() {
		templates.NtfyTestResult(true, "OK", false, fmt.Sprintf("HTTP %d", pollResp.StatusCode())).Render(r.Context(), w)
		return
	}

	readMsg := "OK"
	if msgID != "" && !strings.Contains(pollResp.String(), msgID) {
		readMsg = "OK (message may have expired from cache)"
	}
	templates.NtfyTestResult(true, "OK", true, readMsg).Render(r.Context(), w)
}
