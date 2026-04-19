package policy

import (
	"sync"
	"time"
)

// RateLimiter implements token bucket rate limiting per actor+action
type RateLimiter struct {
	limits  map[string]RateLimitConfig
	buckets map[string]*bucket
	mu      sync.RWMutex
}

type bucket struct {
	tokens    int
	lastReset time.Time
	count     int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limits:  make(map[string]RateLimitConfig),
		buckets: make(map[string]*bucket),
	}
}

// SetLimit sets a rate limit for an action
func (r *RateLimiter) SetLimit(config RateLimitConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.limits[config.Action] = config
}

// Allow checks if an action is allowed under rate limits
func (r *RateLimiter) Allow(actor, action string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	limit, exists := r.limits[action]
	if !exists {
		return true // no limit configured
	}

	key := actor + ":" + action
	b, exists := r.buckets[key]
	if !exists {
		b = &bucket{lastReset: time.Now()}
		r.buckets[key] = b
	}

	// Reset bucket if a minute has passed
	if time.Since(b.lastReset) > time.Minute {
		b.count = 0
		b.lastReset = time.Now()
	}

	if limit.MaxPerMin > 0 && b.count >= limit.MaxPerMin {
		return false
	}

	b.count++
	return true
}
