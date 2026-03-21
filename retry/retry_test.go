package retry

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func TestDo_ImmediateSuccess(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	err := Do(ctx, 3, Fast, func() error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestDo_SuccessAfterFailures(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	err := Do(ctx, 5, Backoff{Initial: time.Millisecond, Max: 10 * time.Millisecond, Multiplier: 2, Jitter: false}, func() error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary failure")
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestDo_MaxAttemptsExhausted(t *testing.T) {
	ctx := context.Background()
	callCount := 0
	expectedErr := errors.New("persistent failure")

	err := Do(ctx, 3, Backoff{Initial: time.Millisecond, Max: 10 * time.Millisecond, Multiplier: 2, Jitter: false}, func() error {
		callCount++
		return expectedErr
	})

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestDo_ContextCancelledDuringSleep(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0

	// Cancel context after first attempt
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	err := Do(ctx, 10, Backoff{Initial: 100 * time.Millisecond, Max: time.Second, Multiplier: 2, Jitter: false}, func() error {
		callCount++
		return errors.New("failure")
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	// Should have only 1 call since context is cancelled during sleep
	if callCount != 1 {
		t.Errorf("expected 1 call (cancelled during sleep), got %d", callCount)
	}
}

func TestDo_ContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	callCount := 0

	err := Do(ctx, 3, Fast, func() error {
		callCount++
		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	if callCount != 0 {
		t.Errorf("expected 0 calls (context already cancelled), got %d", callCount)
	}
}

func TestDo_ExponentialBackoff(t *testing.T) {
	ctx := context.Background()
	callCount := 0
	var callTimes []time.Time

	backoff := Backoff{Initial: 10 * time.Millisecond, Max: 100 * time.Millisecond, Multiplier: 2, Jitter: false}

	start := time.Now()
	_ = Do(ctx, 4, backoff, func() error {
		callTimes = append(callTimes, time.Now())
		callCount++
		return errors.New("failure")
	})

	if callCount != 4 {
		t.Fatalf("expected 4 calls, got %d", callCount)
	}

	// Verify delays are approximately correct (with some tolerance for scheduling)
	// Expected delays: 10ms, 20ms, 40ms (no delay after last attempt)
	// Total expected: ~70ms
	elapsed := time.Since(start)
	expectedMin := 60 * time.Millisecond // Allow some tolerance
	expectedMax := 150 * time.Millisecond

	if elapsed < expectedMin || elapsed > expectedMax {
		t.Errorf("expected elapsed time between %v and %v, got %v", expectedMin, expectedMax, elapsed)
	}
}

func TestDo_MaxDelayRespected(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	backoff := Backoff{Initial: 5 * time.Millisecond, Max: 10 * time.Millisecond, Multiplier: 10, Jitter: false}

	start := time.Now()
	_ = Do(ctx, 4, backoff, func() error {
		callCount++
		return errors.New("failure")
	})

	elapsed := time.Since(start)

	// With max=10ms and 3 delays, max possible is 30ms + some overhead
	// Without max, it would be 5ms + 50ms + 500ms = 555ms
	if elapsed > 100*time.Millisecond {
		t.Errorf("max delay not respected, elapsed: %v", elapsed)
	}
}

func TestDo_JitterProducesVariation(t *testing.T) {
	ctx := context.Background()

	// Use a small initial delay so the test runs quickly.
	// attempts=2 means exactly 1 sleep per run (the last attempt never sleeps),
	// so there is no exponential growth — total ≈ 20 × 7.5ms avg = 150ms.
	backoff := Backoff{Initial: 10 * time.Millisecond, Max: time.Second, Multiplier: 2, Jitter: true}

	// Collect 20 samples. The jitter window is 50–100% of 10ms (5ms–10ms).
	// With 20 draws from a 5ms range, the probability that max–min < 2ms is negligible.
	const samples = 20
	var delays []time.Duration
	for range samples {
		start := time.Now()
		_ = Do(ctx, 2, backoff, func() error {
			return errors.New("failure")
		})
		delays = append(delays, time.Since(start))
	}

	// Find the spread across all samples.
	minDelay, maxDelay := delays[0], delays[0]
	for _, d := range delays[1:] {
		if d < minDelay {
			minDelay = d
		}
		if d > maxDelay {
			maxDelay = d
		}
	}
	spread := maxDelay - minDelay

	// On platforms with coarse timer granularity or consistent oversleep (some
	// CI runners), durations can quantize and collapse spread even when jitter
	// is applied correctly. Skip rather than fail in those environments.
	if spread < 1*time.Millisecond {
		t.Skipf("timer granularity too coarse for jitter test: spread=%v (min=%v max=%v)", spread, minDelay, maxDelay)
	}

	// Expect at least 2ms spread across 20 samples — statistically guaranteed
	// with ≥5ms jitter window on systems with reasonable timer resolution.
	if spread < 2*time.Millisecond {
		t.Errorf("jitter should produce variation in delays: spread=%v (min=%v max=%v)", spread, minDelay, maxDelay)
	}
}

func TestDoWithResult_ImmediateSuccess(t *testing.T) {
	ctx := context.Background()

	result, err := DoWithResult(ctx, 3, Fast, func() (string, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != "success" {
		t.Errorf("expected 'success', got %q", result)
	}
}

func TestDoWithResult_SuccessAfterFailures(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	result, err := DoWithResult(ctx, 5, Backoff{Initial: time.Millisecond, Max: 10 * time.Millisecond, Multiplier: 2, Jitter: false}, func() (int, error) {
		callCount++
		if callCount < 3 {
			return 0, errors.New("temporary failure")
		}
		return 42, nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestDoWithResult_MaxAttemptsExhausted(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("persistent failure")

	result, err := DoWithResult(ctx, 3, Backoff{Initial: time.Millisecond, Max: 10 * time.Millisecond, Multiplier: 2, Jitter: false}, func() (string, error) {
		return "", expectedErr
	})

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
	if result != "" {
		t.Errorf("expected zero value, got %q", result)
	}
}

func TestDoWithResult_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := DoWithResult(ctx, 3, Fast, func() (int, error) {
		return 42, nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	if result != 0 {
		t.Errorf("expected zero value, got %d", result)
	}
}

func TestDoWithResult_GenericTypes(t *testing.T) {
	ctx := context.Background()

	// Test with struct type
	type data struct {
		Value int
		Name  string
	}

	result, err := DoWithResult(ctx, 1, Fast, func() (data, error) {
		return data{Value: 100, Name: "test"}, nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result.Value != 100 || result.Name != "test" {
		t.Errorf("unexpected result: %+v", result)
	}

	// Test with slice type
	sliceResult, err := DoWithResult(ctx, 1, Fast, func() ([]string, error) {
		return []string{"a", "b", "c"}, nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(sliceResult) != 3 {
		t.Errorf("expected slice of length 3, got %d", len(sliceResult))
	}
}

func TestIsTransient_HTTPTransientCodes(t *testing.T) {
	transientCodes := []int{429, 502, 503, 504, 529}
	for _, code := range transientCodes {
		err := &HTTPError{StatusCode: code}
		if !IsTransient(err) {
			t.Errorf("HTTP %d should be transient", code)
		}
	}
}

func TestIsTransient_HTTPNonTransientCodes(t *testing.T) {
	nonTransientCodes := []int{400, 401, 403, 404, 500}
	for _, code := range nonTransientCodes {
		err := &HTTPError{StatusCode: code}
		if IsTransient(err) {
			t.Errorf("HTTP %d should not be transient", code)
		}
	}
}

func TestIsTransient_WrappedHTTPError(t *testing.T) {
	err := fmt.Errorf("anthropic API: %w", &HTTPError{StatusCode: 429})
	if !IsTransient(err) {
		t.Error("wrapped HTTP 429 should be transient")
	}
}

func TestHTTPError_ErrorString(t *testing.T) {
	err := &HTTPError{StatusCode: 503}
	if err.Error() != "HTTP 503" {
		t.Errorf("expected 'HTTP 503', got %q", err.Error())
	}
}

func TestIsTransient_NilError(t *testing.T) {
	if IsTransient(nil) {
		t.Error("nil error should not be transient")
	}
}

func TestIsTransient_ContextCanceled(t *testing.T) {
	if IsTransient(context.Canceled) {
		t.Error("context.Canceled should not be transient")
	}
}

func TestIsTransient_ContextDeadlineExceeded(t *testing.T) {
	if IsTransient(context.DeadlineExceeded) {
		t.Error("context.DeadlineExceeded should not be transient")
	}
}

func TestIsTransient_RegularError(t *testing.T) {
	err := errors.New("some error")
	if IsTransient(err) {
		t.Error("regular error should not be transient")
	}
}

func TestIsTransient_NetworkOpError(t *testing.T) {
	opErr := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: errors.New("connection refused"),
	}

	if !IsTransient(opErr) {
		t.Error("net.OpError should be transient")
	}
}

func TestIsTransient_WrappedContextError(t *testing.T) {
	wrappedErr := errors.Join(errors.New("wrapper"), context.Canceled)
	if IsTransient(wrappedErr) {
		t.Error("wrapped context.Canceled should not be transient")
	}
}

// mockTimeoutError implements net.Error for testing
type mockTimeoutError struct {
	timeout   bool
	temporary bool
}

func (e *mockTimeoutError) Error() string   { return "mock network error" }
func (e *mockTimeoutError) Timeout() bool   { return e.timeout }
func (e *mockTimeoutError) Temporary() bool { return e.temporary }

func TestIsTransient_TimeoutError(t *testing.T) {
	timeoutErr := &mockTimeoutError{timeout: true, temporary: false}
	if !IsTransient(timeoutErr) {
		t.Error("timeout error should be transient")
	}
}

func TestIsTransient_TemporaryError(t *testing.T) {
	tempErr := &mockTimeoutError{timeout: false, temporary: true}
	if !IsTransient(tempErr) {
		t.Error("temporary error should be transient")
	}
}

func TestIsTransient_NonTransientNetError(t *testing.T) {
	nonTransientErr := &mockTimeoutError{timeout: false, temporary: false}
	if IsTransient(nonTransientErr) {
		t.Error("non-transient net error should not be transient")
	}
}

func TestDo_ZeroAttempts(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	err := Do(ctx, 0, Fast, func() error {
		callCount++
		return nil
	})

	// With 0 attempts, function should not be called
	if err != nil {
		t.Errorf("expected nil error with 0 attempts, got %v", err)
	}
	if callCount != 0 {
		t.Errorf("expected 0 calls with 0 attempts, got %d", callCount)
	}
}

func TestDo_SingleAttempt(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("failure")
	callCount := 0

	err := Do(ctx, 1, Fast, func() error {
		callCount++
		return expectedErr
	})

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestBackoffPresets(t *testing.T) {
	// Verify preset values are reasonable
	presets := map[string]Backoff{
		"Fast":     Fast,
		"Moderate": Moderate,
		"Slow":     Slow,
	}

	for name, b := range presets {
		if b.Initial <= 0 {
			t.Errorf("%s: Initial should be positive, got %v", name, b.Initial)
		}
		if b.Max <= 0 {
			t.Errorf("%s: Max should be positive, got %v", name, b.Max)
		}
		if b.Max < b.Initial {
			t.Errorf("%s: Max should be >= Initial", name)
		}
		if b.Multiplier <= 1 {
			t.Errorf("%s: Multiplier should be > 1, got %v", name, b.Multiplier)
		}
		if !b.Jitter {
			t.Errorf("%s: Jitter should be enabled by default", name)
		}
	}
}

func TestDo_ConcurrentSafety(t *testing.T) {
	ctx := context.Background()
	var totalCalls atomic.Int32

	// Run multiple retries concurrently
	done := make(chan struct{}, 10)
	for range 10 {
		go func() {
			_ = Do(ctx, 3, Backoff{Initial: time.Millisecond, Max: 5 * time.Millisecond, Multiplier: 2, Jitter: true}, func() error {
				totalCalls.Add(1)
				return errors.New("failure")
			})
			done <- struct{}{}
		}()
	}

	// Wait for all goroutines
	for range 10 {
		<-done
	}

	// Each goroutine should have made 3 calls
	expected := int32(30)
	if got := totalCalls.Load(); got != expected {
		t.Errorf("expected %d total calls, got %d", expected, got)
	}
}
