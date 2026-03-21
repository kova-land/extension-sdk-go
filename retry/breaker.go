package retry

import (
	"errors"
	"sync"
	"time"
)

// ErrCircuitOpen is returned when the circuit breaker is in the open state
// and the operation is not attempted.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// BreakerState represents the current state of the circuit breaker.
type BreakerState int

const (
	StateClosed   BreakerState = iota // normal operation
	StateOpen                         // failing, reject requests
	StateHalfOpen                     // testing if recovery succeeded
)

// Breaker implements a simple circuit breaker pattern.
// After `threshold` consecutive failures, the breaker opens for `cooldown`
// duration. After the cooldown, it allows one probe request (half-open).
// If the probe succeeds, the breaker closes. If it fails, it re-opens.
type Breaker struct {
	mu        sync.Mutex
	state     BreakerState
	failures  int
	threshold int
	cooldown  time.Duration
	openedAt  time.Time
}

// NewBreaker creates a circuit breaker that opens after `threshold`
// consecutive failures and stays open for `cooldown` before probing.
func NewBreaker(threshold int, cooldown time.Duration) *Breaker {
	return &Breaker{
		threshold: threshold,
		cooldown:  cooldown,
	}
}

// State returns the current breaker state.
func (b *Breaker) State() BreakerState {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.currentState()
}

// currentState returns the effective state, transitioning from open to
// half-open when cooldown has elapsed. Must be called with b.mu held.
func (b *Breaker) currentState() BreakerState {
	if b.state == StateOpen && time.Since(b.openedAt) >= b.cooldown {
		b.state = StateHalfOpen
	}
	return b.state
}

// Do executes fn if the circuit is closed or half-open.
// Returns ErrCircuitOpen without calling fn if the circuit is open.
func (b *Breaker) Do(fn func() error) error {
	b.mu.Lock()
	state := b.currentState()
	if state == StateOpen {
		b.mu.Unlock()
		return ErrCircuitOpen
	}
	b.mu.Unlock()

	err := fn()

	b.mu.Lock()
	defer b.mu.Unlock()

	if err != nil {
		b.failures++
		if b.failures >= b.threshold {
			b.state = StateOpen
			b.openedAt = time.Now()
		}
		return err
	}

	// Success: reset to closed.
	b.failures = 0
	b.state = StateClosed
	return nil
}

// Reset manually closes the breaker (e.g. after a successful health check).
func (b *Breaker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	b.state = StateClosed
}
