package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aakamshpm/url-shortener-go/internal/shortener"
	"github.com/aakamshpm/url-shortener-go/internal/storage/memory"
)

func newSvc() *shortener.Service {
	return shortener.NewService(memory.NewStore(1_000_000))
}

func TestShortenHandler_Valid(t *testing.T) {
	svc := newSvc()
	body := `{"url":"https://example.com/docs"}`
	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(body))
	rec := httptest.NewRecorder()
	shortenHandler(rec, req, svc)
	res := rec.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusCreated)
	}
	var got shortenResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Code == "" {
		t.Fatalf("code is empty")
	}
	wantShortURL := shortener.BaseURL + "/" + got.Code
	if got.ShortURL != wantShortURL {
		t.Fatalf("short_url = %q, want %q", got.ShortURL, wantShortURL)
	}
	longURL, ok := svc.Resolve(got.Code)
	if !ok {
		t.Fatalf("code %q not found in store", got.Code)
	}
	if longURL != "https://example.com/docs" {
		t.Fatalf("stored url = %q, want %q", longURL, "https://example.com/docs")
	}
}

func TestShortenHandler_InvalidInput(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "invalid json", body: `{"url":`},
		{name: "missing url", body: `{}`},
		{name: "bad scheme", body: `{"url":"ftp://example.com"}`},
		{name: "multiple objects", body: `{"url":"https://a.com"}{"url":"https://b.com"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := newSvc()
			req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()
			shortenHandler(rec, req, svc)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestShortenHandler_MethodNotAllowed(t *testing.T) {
	svc := newSvc()
	req := httptest.NewRequest(http.MethodGet, "/shorten", nil)
	rec := httptest.NewRecorder()
	shortenHandler(rec, req, svc)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestRedirectHandler_Found(t *testing.T) {
	svc := newSvc()
	code, _, _ := svc.Shorten("https://example.com/docs")

	req := httptest.NewRequest(http.MethodGet, "/"+code, nil)
	rec := httptest.NewRecorder()
	redirectHandler(rec, req, svc)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusFound { // 302
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusFound)
	}
	gotLocation := res.Header.Get("Location")
	if gotLocation != "https://example.com/docs" {
		t.Fatalf("Location = %q, want %q", gotLocation, "https://example.com/docs")
	}
}

func TestRedirectHandler_UnknownCode(t *testing.T) {
	svc := newSvc()
	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	rec := httptest.NewRecorder()
	redirectHandler(rec, req, svc)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestRedirectHandler_RootPath(t *testing.T) {
	svc := newSvc()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	redirectHandler(rec, req, svc)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestRedirectHandler_MethodNotAllowed(t *testing.T) {
	svc := newSvc()
	req := httptest.NewRequest(http.MethodPost, "/abc123", nil)
	rec := httptest.NewRecorder()
	redirectHandler(rec, req, svc)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}