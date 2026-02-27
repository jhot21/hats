package util

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
)

func restyGet(t *testing.T, url string) (*resty.Response, error) {
	t.Helper()
	return resty.New().R().Get(url)
}

func TestCheckSuccess_200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	resp, err := restyGet(t, srv.URL)
	resp, err = CheckSuccess(resp, err)
	if err != nil {
		t.Fatalf("expected nil error for 200, got: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestCheckSuccess_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	resp, err := restyGet(t, srv.URL)
	_, err = CheckSuccess(resp, err)
	if err == nil {
		t.Fatal("expected non-nil error for 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("expected error to contain %q, got: %v", "404", err)
	}
}

func TestCheckSuccess_500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	resp, err := restyGet(t, srv.URL)
	_, err = CheckSuccess(resp, err)
	if err == nil {
		t.Fatal("expected non-nil error for 500")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected error to contain %q, got: %v", "500", err)
	}
}

func TestCheckSuccess_PassthroughErr(t *testing.T) {
	original := errors.New("original")
	_, err := CheckSuccess(nil, original)
	if !errors.Is(err, original) {
		t.Fatalf("expected original error to be returned unchanged, got: %v", err)
	}
}
