package idempotency

import "sync"

type Store struct {
	mu     sync.Mutex
	events map[string]bool
}

func NewStore() *Store {
	return &Store{
		events: make(map[string]bool),
	}
}

func (s *Store) IsProcessed(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.events[id]
}

func (s *Store) MarkProcessed(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events[id] = true
}
