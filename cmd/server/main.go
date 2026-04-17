package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

func main() {
	mux := http.NewServeMux()

	// route handling for POST /shorten
	mux.HandleFunc("/shorten", shortenHandler)

	// catch all route for GET (/{code})
	// redirects short urls accoridingly
	mux.HandleFunc("/", redirectHandler)

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

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	// make sure method is POST
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201

	// temporary code to return response
	resp := map[string]string{
		"code": "abc123",
	}

	// serializes resp map into JSON and writes into response body
	_ = json.NewEncoder(w).Encode(resp)
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
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
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "todo redirect for code: %s\n", code)
}
