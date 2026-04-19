package layer1

import (
	"sync"
	"time"
)

// IdempotencyStore tracks processed request IDs to prevent duplicate execution
type IdempotencyStore struct {
	seen map[string]time.Time
	ttl  time.Duration
	mu   sync.RWMutex
}

// NewIdempotencyStore creates a new idempotency store
func NewIdempotencyStore(ttl time.Duration) *IdempotencyStore {
	if ttl <= 0 {
		ttl = 1 * time.Hour
	}
	store := &IdempotencyStore{
		seen: make(map[string]time.Time),
		ttl:  ttl,
	}
	go store.cleanupLoop()
	return store
}

// IsSeen returns true if this request ID has already been processed
func (s *IdempotencyStore) IsSeen(requestID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.seen[requestID]
	return exists
}

// MarkSeen records a request ID as processed
func (s *IdempotencyStore) MarkSeen(requestID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.seen[requestID] = time.Now()
}

// CheckAndMark atomically checks if seen and marks if not.
// Returns true if already seen (duplicate), false if new.
func (s *IdempotencyStore) CheckAndMark(requestID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.seen[requestID]; exists {
		return true // duplicate
	}
	s.seen[requestID] = time.Now()
	return false // new request
}

// Size returns the number of tracked request IDs
func (s *IdempotencyStore) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.seen)
}

// cleanupLoop periodically removes expired entries
func (s *IdempotencyStore) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		cutoff := time.Now().Add(-s.ttl)
		for id, ts := range s.seen {
			if ts.Before(cutoff) {
				delete(s.seen, id)
			}
		}
		s.mu.Unlock()
	}
}
