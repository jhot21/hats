package admin

import (
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"codeberg.org/jhot/hats/internal/admin/templates"
	"codeberg.org/jhot/hats/internal/tokens"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// tokenListHandler renders the full Tokens page.
func tokenListHandler(w http.ResponseWriter, r *http.Request) {
	toks, err := adminTokenStore.List()
	if err != nil {
		http.Error(w, "failed to list tokens", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.Tokens(tokenDataList(toks)).Render(r.Context(), w)
}

// tokenTableHandler returns just the token table fragment for HTMX refresh.
func tokenTableHandler(w http.ResponseWriter, r *http.Request) {
	toks, err := adminTokenStore.List()
	if err != nil {
		http.Error(w, "failed to list tokens", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.TokenList(tokenDataList(toks)).Render(r.Context(), w)
}

// tokenNewHandler returns the create token modal form fragment.
func tokenNewHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.TokenCreateForm("", "").Render(r.Context(), w)
}

// tokenEditHandler returns the edit token modal form fragment.
func tokenEditHandler(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	toks, err := adminTokenStore.List()
	if err != nil {
		http.Error(w, "failed to list tokens", http.StatusInternalServerError)
		return
	}
	var found *tokens.Token
	for _, t := range toks {
		if t.Hash == hash {
			found = t
			break
		}
	}
	if found == nil {
		http.Error(w, "token not found", http.StatusNotFound)
		return
	}
	data := tokenDataList([]*tokens.Token{found})[0]
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.TokenEditForm(data, "").Render(r.Context(), w)
}

// tokenUpdateAPIHandler updates a token's scopes and expiry.
func tokenUpdateAPIHandler(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	scopesRaw := r.FormValue("scopes")
	expiryRaw := r.FormValue("expires_at")

	scopes := parseScopes(scopesRaw)
	if len(scopes) == 0 {
		http.Error(w, "at least one valid scope is required", http.StatusBadRequest)
		return
	}

	var expiresAt *time.Time
	if expiryRaw != "" {
		t, err := time.Parse("2006-01-02", expiryRaw)
		if err != nil {
			http.Error(w, "invalid expiry date (use YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		t = t.UTC()
		expiresAt = &t
	}

	if err := adminTokenStore.Update(hash, scopes, expiresAt); err != nil {
		if errors.Is(err, tokens.ErrTokenNotFound) {
			http.Error(w, "token not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to update token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	toks, err := adminTokenStore.List()
	if err != nil {
		http.Error(w, "failed to list tokens", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.TokenEditSuccess(tokenDataList(toks)).Render(r.Context(), w)
}

// tokenCreateAPIHandler creates a new token and returns the "token created" fragment.
func tokenCreateAPIHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	scopesRaw := r.FormValue("scopes")
	expiryRaw := r.FormValue("expires_at")

	if name == "" || scopesRaw == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.TokenCreateForm("name and scopes are required", "").Render(r.Context(), w)
		return
	}

	scopes := parseScopes(scopesRaw)
	if len(scopes) == 0 {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.TokenCreateForm("at least one valid scope is required", scopesRaw).Render(r.Context(), w)
		return
	}

	var expiresAt *time.Time
	if expiryRaw != "" {
		t, err := time.Parse("2006-01-02", expiryRaw)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			templates.TokenCreateForm("invalid expiry date (use YYYY-MM-DD)", scopesRaw).Render(r.Context(), w)
			return
		}
		t = t.UTC()
		expiresAt = &t
	}

	rawToken, err := adminTokenStore.Create(name, scopes, expiresAt)
	if err != nil {
		http.Error(w, "failed to create token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.TokenCreated(rawToken).Render(r.Context(), w)
}

// tokenRevokeAPIHandler revokes a token and returns the refreshed token table fragment.
func tokenRevokeAPIHandler(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	if err := adminTokenStore.Revoke(hash); err != nil {
		http.Error(w, "failed to revoke token: "+err.Error(), http.StatusInternalServerError)
		return
	}
	toks, err := adminTokenStore.List()
	if err != nil {
		http.Error(w, "failed to list tokens", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.TokenList(tokenDataList(toks)).Render(r.Context(), w)
}

// tokenJSONListHandler returns all tokens as JSON (for programmatic access).
func tokenJSONListHandler(w http.ResponseWriter, r *http.Request) {
	toks, err := adminTokenStore.List()
	if err != nil {
		http.Error(w, "failed to list tokens", http.StatusInternalServerError)
		return
	}
	render.JSON(w, r, tokenDataList(toks))
}

func tokenDataList(toks []*tokens.Token) []templates.TokenData {
	sort.Slice(toks, func(i, j int) bool {
		return toks[i].Name < toks[j].Name
	})
	data := make([]templates.TokenData, len(toks))
	for i, t := range toks {
		expiry := "never"
		if t.ExpiresAt != nil {
			expiry = t.ExpiresAt.Format("2006-01-02")
		}
		lastUsed := "never"
		if t.LastUsedAt != nil {
			lastUsed = t.LastUsedAt.Format("2006-01-02 15:04")
		}
		data[i] = templates.TokenData{
			Hash:       t.Hash,
			Name:       t.Name,
			Scopes:     t.Scopes,
			ExpiresAt:  expiry,
			LastUsedAt: lastUsed,
		}
	}
	return data
}

// parseScopes splits a comma-or-newline separated scope string into a deduplicated slice.
func parseScopes(raw string) []string {
	raw = strings.ReplaceAll(raw, "\n", ",")
	raw = strings.ReplaceAll(raw, "\r", "")
	parts := strings.Split(raw, ",")
	seen := make(map[string]struct{})
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, dup := seen[p]; dup {
			continue
		}
		if strings.Count(p, ":") != 2 {
			continue // skip malformed scopes — silently drop rather than error
		}
		seen[p] = struct{}{}
		result = append(result, p)
	}
	return result
}
