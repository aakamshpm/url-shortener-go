package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestShortenHandler_Valid(t *testing.T) {
	st := newStore(1_000_000)
	body := `{"url":"https://example.com/docs"}`
	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(body))
	rec := httptest.NewRecorder()
	shortenHandler(rec, req, st)
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
	wantShortURL := baseURL + "/" + got.Code
	if got.ShortURL != wantShortURL {
		t.Fatalf("short_url = %q, want %q", got.ShortURL, wantShortURL)
	}
	longURL, ok := st.Get(got.Code)
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
			st := newStore(1_000_000)
			req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()
			shortenHandler(rec, req, st)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestShortenHandler_MethodNotAllowed(t *testing.T) {
	st := newStore(1_000_000)
	req := httptest.NewRequest(http.MethodGet, "/shorten", nil)
	rec := httptest.NewRecorder()
	shortenHandler(rec, req, st)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}
