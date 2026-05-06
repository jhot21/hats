package admin

import (
    "context"
    "net/http"
    "net/http/httptest"
    "net/url"
    "strings"
    "testing"
    "time"

    "codeberg.org/jhot/hats/internal/tokens"
    "github.com/go-chi/chi/v5"
)

// mockTokenStore is a minimal in-memory TokenStore for handler tests.
type mockTokenStore struct {
    byHash  map[string]*tokens.Token
    updated map[string]updateCall
}

type updateCall struct {
    scopes    []string
    expiresAt *time.Time
}

func newMockStore(toks ...*tokens.Token) *mockTokenStore {
    m := &mockTokenStore{
        byHash:  make(map[string]*tokens.Token),
        updated: make(map[string]updateCall),
    }
    for _, t := range toks {
        m.byHash[t.Hash] = t
    }
    return m
}

func (m *mockTokenStore) Create(name string, scopes []string, expiresAt *time.Time) (string, error) {
    return "", nil
}
func (m *mockTokenStore) Validate(rawToken string) (*tokens.Token, error) {
    return nil, nil
}
func (m *mockTokenStore) Revoke(hash string) error {
    delete(m.byHash, hash)
    return nil
}
func (m *mockTokenStore) List() ([]*tokens.Token, error) {
    result := make([]*tokens.Token, 0, len(m.byHash))
    for _, t := range m.byHash {
        result = append(result, t)
    }
    return result, nil
}
func (m *mockTokenStore) Update(hash string, scopes []string, expiresAt *time.Time) error {
    if _, ok := m.byHash[hash]; !ok {
        return tokens.ErrTokenNotFound
    }
    m.updated[hash] = updateCall{scopes: scopes, expiresAt: expiresAt}
    return nil
}

// chiCtx wraps a request with a Chi URL param.
func chiCtx(r *http.Request, key, val string) *http.Request {
    rctx := chi.NewRouteContext()
    rctx.URLParams.Add(key, val)
    return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// --- parseScopes tests ---

func TestParseScopes_NewlineSeparated(t *testing.T) {
    got := parseScopes("read:state:*\nwrite:command:bedtime")
    want := []string{"read:state:*", "write:command:bedtime"}
    if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
        t.Errorf("got %v, want %v", got, want)
    }
}

func TestParseScopes_CommaSeparated(t *testing.T) {
    got := parseScopes("read:state:*,write:command:bedtime")
    want := []string{"read:state:*", "write:command:bedtime"}
    if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
        t.Errorf("got %v, want %v", got, want)
    }
}

func TestParseScopes_BlankLinesDropped(t *testing.T) {
    got := parseScopes("read:state:*\n\n\nwrite:command:bedtime\n")
    if len(got) != 2 {
        t.Errorf("expected 2 scopes, got %d: %v", len(got), got)
    }
}

func TestParseScopes_Deduplication(t *testing.T) {
    got := parseScopes("read:state:*\nread:state:*\nwrite:ntfy:*")
    if len(got) != 2 {
        t.Errorf("expected 2 deduplicated scopes, got %d: %v", len(got), got)
    }
}

func TestParseScopes_MalformedDropped(t *testing.T) {
    got := parseScopes("notascope\nread:state:*")
    if len(got) != 1 || got[0] != "read:state:*" {
        t.Errorf("expected only valid scope, got %v", got)
    }
}

func TestParseScopes_Empty(t *testing.T) {
    got := parseScopes("")
    if len(got) != 0 {
        t.Errorf("expected empty slice, got %v", got)
    }
}

// --- tokenUpdateAPIHandler tests ---

func TestTokenUpdateAPIHandler_Success(t *testing.T) {
    tok := &tokens.Token{Hash: "abc123", Name: "test", Scopes: []string{"read:state:*"}, CreatedAt: time.Now()}
    mock := newMockStore(tok)
    adminTokenStore = mock

    form := url.Values{}
    form.Set("scopes", "read:state:*\nwrite:command:bedtime")
    form.Set("expires_at", "")

    r := httptest.NewRequest(http.MethodPut, "/admin/api/tokens/abc123", strings.NewReader(form.Encode()))
    r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    r = chiCtx(r, "hash", "abc123")
    w := httptest.NewRecorder()

    tokenUpdateAPIHandler(w, r)

    if w.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", w.Code)
    }
    call, ok := mock.updated["abc123"]
    if !ok {
        t.Fatal("expected Update() to be called")
    }
    if len(call.scopes) != 2 {
        t.Errorf("expected 2 scopes, got %v", call.scopes)
    }
    if call.expiresAt != nil {
        t.Errorf("expected nil expiresAt, got %v", call.expiresAt)
    }
}

func TestTokenUpdateAPIHandler_WithExpiry(t *testing.T) {
    tok := &tokens.Token{Hash: "abc123", Name: "test", Scopes: []string{"*:*:*"}, CreatedAt: time.Now()}
    mock := newMockStore(tok)
    adminTokenStore = mock

    form := url.Values{}
    form.Set("scopes", "*:*:*")
    form.Set("expires_at", "2027-01-01")

    r := httptest.NewRequest(http.MethodPut, "/admin/api/tokens/abc123", strings.NewReader(form.Encode()))
    r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    r = chiCtx(r, "hash", "abc123")
    w := httptest.NewRecorder()

    tokenUpdateAPIHandler(w, r)

    if w.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", w.Code)
    }
    call := mock.updated["abc123"]
    if call.expiresAt == nil {
        t.Fatal("expected non-nil expiresAt")
    }
    if call.expiresAt.Format("2006-01-02") != "2027-01-01" {
        t.Errorf("expected 2027-01-01, got %s", call.expiresAt.Format("2006-01-02"))
    }
}

func TestTokenUpdateAPIHandler_EmptyScopes(t *testing.T) {
    tok := &tokens.Token{Hash: "abc123", Name: "test", Scopes: []string{"*:*:*"}, CreatedAt: time.Now()}
    mock := newMockStore(tok)
    adminTokenStore = mock

    form := url.Values{}
    form.Set("scopes", "")

    r := httptest.NewRequest(http.MethodPut, "/admin/api/tokens/abc123", strings.NewReader(form.Encode()))
    r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    r = chiCtx(r, "hash", "abc123")
    w := httptest.NewRecorder()

    tokenUpdateAPIHandler(w, r)

    if w.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", w.Code)
    }
    if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
        t.Errorf("expected text/html, got %s", ct)
    }
    if _, ok := mock.updated["abc123"]; ok {
        t.Error("Update() should not have been called")
    }
}

func TestTokenUpdateAPIHandler_InvalidExpiry(t *testing.T) {
    tok := &tokens.Token{Hash: "abc123", Name: "test", Scopes: []string{"*:*:*"}, CreatedAt: time.Now()}
    mock := newMockStore(tok)
    adminTokenStore = mock

    form := url.Values{}
    form.Set("scopes", "*:*:*")
    form.Set("expires_at", "not-a-date")

    r := httptest.NewRequest(http.MethodPut, "/admin/api/tokens/abc123", strings.NewReader(form.Encode()))
    r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    r = chiCtx(r, "hash", "abc123")
    w := httptest.NewRecorder()

    tokenUpdateAPIHandler(w, r)

    if w.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", w.Code)
    }
    if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
        t.Errorf("expected text/html, got %s", ct)
    }
    if _, ok := mock.updated["abc123"]; ok {
        t.Error("Update() should not have been called")
    }
}

func TestTokenUpdateAPIHandler_TokenNotFound(t *testing.T) {
    mock := newMockStore() // empty
    adminTokenStore = mock

    form := url.Values{}
    form.Set("scopes", "read:state:*")

    r := httptest.NewRequest(http.MethodPut, "/admin/api/tokens/notexist", strings.NewReader(form.Encode()))
    r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    r = chiCtx(r, "hash", "notexist")
    w := httptest.NewRecorder()

    tokenUpdateAPIHandler(w, r)

    if w.Code != http.StatusNotFound {
        t.Errorf("expected 404, got %d", w.Code)
    }
}

func TestTokenEditHandler_Found(t *testing.T) {
    tok := &tokens.Token{Hash: "abc123", Name: "test", Scopes: []string{"read:state:*"}, CreatedAt: time.Now()}
    mock := newMockStore(tok)
    adminTokenStore = mock

    r := httptest.NewRequest(http.MethodGet, "/admin/tokens/abc123/edit", nil)
    r = chiCtx(r, "hash", "abc123")
    w := httptest.NewRecorder()

    tokenEditHandler(w, r)

    if w.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", w.Code)
    }
    if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
        t.Errorf("expected text/html, got %s", ct)
    }
}

func TestTokenEditHandler_NotFound(t *testing.T) {
    mock := newMockStore()
    adminTokenStore = mock

    r := httptest.NewRequest(http.MethodGet, "/admin/tokens/notexist/edit", nil)
    r = chiCtx(r, "hash", "notexist")
    w := httptest.NewRecorder()

    tokenEditHandler(w, r)

    if w.Code != http.StatusNotFound {
        t.Errorf("expected 404, got %d", w.Code)
    }
}
