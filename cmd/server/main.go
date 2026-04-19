package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	neturl "net/url"
	"os"
	"strings"
	"sync"
)

const (
	baseURL = "http://localhost:8080"
	base62  = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	Code     string `json:"code"`
	ShortURL string `json:"short_url"`
}

type store struct {
	mu      sync.RWMutex
	data    map[string]string // code -> longUrl
	counter uint64
}

func newStore(start uint64) *store {
	return &store{
		data:    make(map[string]string),
		counter: start,
	}
}

func (s *store) Save(code, longUrl string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[code] = longUrl
}

func (s *store) Get(code string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.data[code]
	return v, ok
}

func (s *store) NextID() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.counter++
	return s.counter
}

func main() {
	st := newStore(1_000_000) // start the counter value at 1_000_000 for non-trivial short codes.
	mux := http.NewServeMux()

	// route handling for POST /shorten
	mux.HandleFunc("/shorten", func(w http.ResponseWriter, r *http.Request) {
		shortenHandler(w, r, st)
	})

	// catch all route for GET (/{code})
	// redirects short urls accoridingly
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { redirectHandler(w, r, st) })

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

func shortenHandler(w http.ResponseWriter, r *http.Request, st *store) {
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

	req.URL = strings.TrimSpace(req.URL)
	if req.URL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}

	parsed, err := neturl.ParseRequestURI(req.URL)
	if err != nil {
		http.Error(w, "invalid url format", http.StatusBadRequest)
		return
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		http.Error(w, "url must start with http:// or https://", http.StatusBadRequest)
		return
	}

	if parsed.Host == "" {
		http.Error(w, "url host is required", http.StatusBadRequest)
		return
	}

	// counter based short code generation
	id := st.NextID()
	code := encodeBase62(id)

	// save mapping
	st.Save(code, req.URL)

	resp := shortenResponse{
		Code:     code,
		ShortURL: fmt.Sprintf("%s/%s", baseURL, code),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201

	// serializes resp map into JSON and writes into response body
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to write response", http.StatusInternalServerError)
		return
	}
}

func redirectHandler(w http.ResponseWriter, r *http.Request, st *store) {
	// make sure the request is GET
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// trims the leading slash
	// eg: returns abc123 from /abc123
	code := strings.TrimPrefix(r.URL.Path, "/")

	if v, ok := st.Get(code); ok {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "value: %s\n", v)
		return
	}

	http.NotFound(w, r)
}

func encodeBase62(n uint64) string {
	if n == 0 {
		return string(base62[0])
	}

	baseLen := uint64(len(base62)) //62
	out := make([]byte, 0, 11)     // uint64 in base62 fits within 11 characters

	// convert the uint64 value to its base62 representation
	for n > 0 {
		rem := n % baseLen             // 1_000_001 % 62 would give 1 as remainder
		out = append(out, base62[rem]) // take the base62 character based on remainder
		n = n / baseLen                // 1_000_001 / 62 = 16.129
	}

	// the values in 'out' would be in least-significant first order
	// to get the proper base62 representation, we would need to reverse the characters/elements in array
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}

	return string(out)
}
