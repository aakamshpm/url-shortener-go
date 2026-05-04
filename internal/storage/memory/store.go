package memory

import "sync"

type Store struct {
	mu      sync.RWMutex
	data    map[string]string // code -> longUrl
	counter uint64
}

func NewStore(start uint64) *Store {
	return &Store{
		data:    make(map[string]string),
		counter: start,
	}
}

func (s *Store) Save(code, longUrl string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[code] = longUrl
}

func (s *Store) Get(code string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.data[code]
	return v, ok
}

func (s *Store) NextID() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.counter++
	return s.counter
}
