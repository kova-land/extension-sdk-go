// Package retry provides utilities for retrying operations with exponential backoff.
package retry

import (
	"context"
	"errors"
	"math/rand/v2"
	"net"
	"strconv"
	"time"
)

// Backoff configures the retry behavior.
type Backoff struct {
	Initial    time.Duration // Initial delay between retries
	Max        time.Duration // Maximum delay between retries
	Multiplier float64       // Multiplier for exponential growth
	Jitter     bool          // Add randomization to prevent thundering herd
}

// Presets for common retry scenarios.
var (
	// Fast is for operations that should recover quickly (e.g., local calls).
	Fast = Backoff{Initial: 100 * time.Millisecond, Max: time.Second, Multiplier: 2, Jitter: true}

	// Moderate is for operations with moderate latency tolerance (e.g., API calls).
	Moderate = Backoff{Initial: 500 * time.Millisecond, Max: 30 * time.Second, Multiplier: 2, Jitter: true}

	// Slow is for operations that can wait longer (e.g., reconnection attempts).
	Slow = Backoff{Initial: time.Second, Max: 2 * time.Minute, Multiplier: 1.5, Jitter: true}
)

// Do retries the given function until it succeeds or the maximum attempts are reached.
// It respects context cancellation and applies exponential backoff between attempts.
// Returns the last error if all attempts fail.
func Do(ctx context.Context, attempts int, b Backoff, fn func() error) error {
	var lastErr error
	delay := b.Initial

	for i := range attempts {
		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		// Don't sleep after the last attempt
		if i == attempts-1 {
			break
		}

		// Apply jitter if enabled (50-100% of computed delay)
		sleepDuration := delay
		if b.Jitter {
			// G404: math/rand is appropriate for jitter - crypto/rand overkill for backoff timing
			sleepDuration = time.Duration(float64(delay) * (0.5 + rand.Float64()*0.5)) //nolint:gosec
		}

		// Wait with context cancellation support
		timer := time.NewTimer(sleepDuration)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}

		// Increase delay for next attempt
		delay = min(time.Duration(float64(delay)*b.Multiplier), b.Max)
	}

	return lastErr
}

// DoWithResult retries the given function until it succeeds or the maximum attempts are reached.
// Returns the result and nil error on success, or zero value and last error on failure.
func DoWithResult[T any](ctx context.Context, attempts int, b Backoff, fn func() (T, error)) (T, error) {
	var zero T
	var lastErr error
	var result T
	delay := b.Initial

	for i := range attempts {
		if err := ctx.Err(); err != nil {
			return zero, err
		}

		result, lastErr = fn()
		if lastErr == nil {
			return result, nil
		}

		// Don't sleep after the last attempt
		if i == attempts-1 {
			break
		}

		// Apply jitter if enabled
		sleepDuration := delay
		if b.Jitter {
			// G404: math/rand is appropriate for jitter - crypto/rand overkill for backoff timing
			sleepDuration = time.Duration(float64(delay) * (0.5 + rand.Float64()*0.5)) //nolint:gosec
		}

		// Wait with context cancellation support
		timer := time.NewTimer(sleepDuration)
		select {
		case <-ctx.Done():
			timer.Stop()
			return zero, ctx.Err()
		case <-timer.C:
		}

		// Increase delay for next attempt
		delay = min(time.Duration(float64(delay)*b.Multiplier), b.Max)
	}

	return zero, lastErr
}

// HTTPError wraps an HTTP status code as an error for transience checking.
type HTTPError struct {
	StatusCode int
}

func (e *HTTPError) Error() string {
	return "HTTP " + strconv.Itoa(e.StatusCode)
}

// IsTransient returns true if the error is likely transient and worth retrying.
// This includes network timeouts, connection errors, temporary failures, and
// transient HTTP status codes (429, 502, 503, 504, 529).
func IsTransient(err error) bool {
	if err == nil {
		return false
	}

	// Check for context errors (not transient - deliberate cancellation)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Check for HTTP status code errors
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		switch httpErr.StatusCode {
		case 429, 502, 503, 504, 529: // rate limit, bad gateway, unavailable, gateway timeout, overloaded
			return true
		default:
			return false
		}
	}

	// Check for specific network operation errors first (connection refused, etc.)
	// These are always transient - worth retrying even if not timeout/temporary
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	// Check for network errors that implement net.Error interface
	// Note: Temporary() is deprecated since Go 1.18, but we check it for
	// backward compatibility with error types that may still use it.
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary() //nolint:staticcheck // SA1019: backward compat
	}

	return false
}
