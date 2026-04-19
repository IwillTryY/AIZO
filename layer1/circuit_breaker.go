package layer1

import (
	"fmt"
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	CircuitClosed   CircuitState = iota // normal operation
	CircuitOpen                         // failing, reject fast
	CircuitHalfOpen                     // testing recovery
)

// CircuitBreaker prevents cascading failures by fast-failing
// when a downstream dependency is unhealthy
type CircuitBreaker struct {
	name          string
	state         CircuitState
	failCount     int
	successCount  int
	threshold     int           // failures before opening
	resetTimeout  time.Duration // how long to stay open before half-open
	lastFailTime  time.Time
	mu            sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, threshold int, resetTimeout time.Duration) *CircuitBreaker {
	if threshold <= 0 {
		threshold = 5
	}
	if resetTimeout <= 0 {
		resetTimeout = 30 * time.Second
	}
	return &CircuitBreaker{
		name:         name,
		state:        CircuitClosed,
		threshold:    threshold,
		resetTimeout: resetTimeout,
	}
}

// Execute runs fn if the circuit allows it.
// Returns error immediately if circuit is open.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.AllowRequest() {
		return fmt.Errorf("circuit breaker '%s' is open", cb.name)
	}

	err := fn()

	if err != nil {
		cb.RecordFailure()
	} else {
		cb.RecordSuccess()
	}

	return err
}

// AllowRequest returns true if the circuit allows a request through
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if reset timeout has elapsed
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			return true // allow one test request (half-open)
		}
		return false
	case CircuitHalfOpen:
		return true // allow test request
	}
	return true
}

// RecordSuccess records a successful operation
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.successCount++

	if cb.state == CircuitHalfOpen || cb.state == CircuitOpen {
		// Recovery — close the circuit
		cb.state = CircuitClosed
		cb.failCount = 0
	}
}

// RecordFailure records a failed operation
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failCount++
	cb.lastFailTime = time.Now()

	if cb.state == CircuitHalfOpen {
		// Failed during test — reopen
		cb.state = CircuitOpen
		return
	}

	if cb.failCount >= cb.threshold {
		cb.state = CircuitOpen
	}
}

// State returns the current circuit state
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset manually resets the circuit breaker to closed
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = CircuitClosed
	cb.failCount = 0
	cb.successCount = 0
}

// Stats returns circuit breaker statistics
func (cb *CircuitBreaker) Stats() (state string, failures, successes int) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case CircuitClosed:
		state = "closed"
	case CircuitOpen:
		state = "open"
	case CircuitHalfOpen:
		state = "half-open"
	}
	return state, cb.failCount, cb.successCount
}
