package shortener

import (
	"errors"
	"fmt"
	neturl "net/url"
	"strings"
)

const (
	BaseURL = "http://localhost:8080"
	base62  = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

var (
	ErrEmptyURL      = errors.New("url is required")
	ErrInvalidURL    = errors.New("invalid url format")
	ErrInvalidScheme = errors.New("url must start with http:// or https://")
	ErrMissingHost   = errors.New("url host is required")
)

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Shorten(rawURL string) (code string, shortURL string, err error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", "", ErrEmptyURL
	}

	parsed, err := neturl.ParseRequestURI(rawURL)
	if err != nil {
		return "", "", ErrInvalidURL
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", "", ErrInvalidScheme
	}

	if parsed.Host == "" {
		return "", "", ErrMissingHost
	}

	id := s.store.NextID()
	code = encodeBase62(id)

	s.store.Save(code, rawURL)

	return code, fmt.Sprintf("%s/%s", BaseURL, code), nil
}

func (s *Service) Resolve(code string) (string, bool) {
	return s.store.Get(code)
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
		n = n / baseLen                // 1_000_001 / 62 = 16,129
	}

	// the values in 'out' would be in least-significant first order
	// to get the proper base62 representation, we would need to reverse the characters/elements in array
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}

	return string(out)
}
