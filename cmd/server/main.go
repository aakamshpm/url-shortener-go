package main

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/aakamshpm/url-shortener-go/internal/shortener"
	"github.com/aakamshpm/url-shortener-go/internal/storage/memory"
)

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	Code     string `json:"code"`
	ShortURL string `json:"short_url"`
}

func main() {
	mem := memory.NewStore(1_000_000) // start the counter value at 1_000_000 for non-trivial short codes.
	svc := shortener.NewService(mem)

	mux := http.NewServeMux()

	// route handling for POST /shorten
	mux.HandleFunc("/shorten", func(w http.ResponseWriter, r *http.Request) {
		shortenHandler(w, r, svc)
	})

	// catch all route for GET (/{code})
	// redirects short urls accoridingly
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { redirectHandler(w, r, svc) })

	addr := ":8080"

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}
	log.Printf("server started: %s", addr)

	if err := http.Serve(ln, mux); err != nil {
		log.Printf("server failed: %v", err)
		os.Exit(1)
	}
}

func shortenHandler(w http.ResponseWriter, r *http.Request, svc *shortener.Service) {
	defer r.Body.Close() // body cleanup

	// make sure method is POST
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req shortenRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	// ensure no extra JSON tokens exist aftet the first one which is "url"
	var extra any

	// EOF here means no extra token, which is valid
	if err := dec.Decode(&extra); err != io.EOF {
		http.Error(w, "invalid JSON body; only one JSON object is allowed", http.StatusBadRequest)
		return
	}

	code, shortURL, err := svc.Shorten(req.URL)
	if err != nil {
		switch err {
		case shortener.ErrEmptyURL, shortener.ErrInvalidScheme, shortener.ErrInvalidURL, shortener.ErrMissingHost:
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	}

	resp := shortenResponse{
		Code:     code,
		ShortURL: shortURL,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201

	// serializes resp map into JSON and writes into response body
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to write response", http.StatusInternalServerError)
		return
	}
}

func redirectHandler(w http.ResponseWriter, r *http.Request, svc *shortener.Service) {
	// make sure the request is GET
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// trims the leading slash
	// eg: returns abc123 from /abc123
	code := strings.TrimPrefix(r.URL.Path, "/")

	if code == "" {
		http.NotFound(w, r) // if the url is "/", return 404
	}

	longURL, ok := svc.Resolve(code)
	if !ok {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, longURL, http.StatusFound) // 302
}
