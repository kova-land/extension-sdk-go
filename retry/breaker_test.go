package retry

import (
	"errors"
	"testing"
	"time"
)

var errTest = errors.New("test error")

func TestBreakerClosedAllowsCalls(t *testing.T) {
	b := NewBreaker(3, time.Second)
	called := false
	err := b.Do(func() error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !called {
		t.Error("expected fn to be called")
	}
	if b.State() != StateClosed {
		t.Errorf("expected StateClosed, got %v", b.State())
	}
}

func TestBreakerOpensAfterThreshold(t *testing.T) {
	b := NewBreaker(3, time.Second)

	for range 3 {
		_ = b.Do(func() error { return errTest })
	}

	if b.State() != StateOpen {
		t.Fatalf("expected StateOpen after %d failures, got %v", 3, b.State())
	}

	// Next call should be rejected
	err := b.Do(func() error { return nil })
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestBreakerHalfOpenAfterCooldown(t *testing.T) {
	b := NewBreaker(2, 10*time.Millisecond)

	_ = b.Do(func() error { return errTest })
	_ = b.Do(func() error { return errTest })

	if b.State() != StateOpen {
		t.Fatal("expected StateOpen")
	}

	time.Sleep(15 * time.Millisecond)

	if b.State() != StateHalfOpen {
		t.Fatal("expected StateHalfOpen after cooldown")
	}
}

func TestBreakerHalfOpenSuccessCloses(t *testing.T) {
	b := NewBreaker(2, 10*time.Millisecond)

	_ = b.Do(func() error { return errTest })
	_ = b.Do(func() error { return errTest })

	time.Sleep(15 * time.Millisecond)

	// Probe call succeeds
	err := b.Do(func() error { return nil })
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if b.State() != StateClosed {
		t.Errorf("expected StateClosed after probe success, got %v", b.State())
	}
}

func TestBreakerHalfOpenFailureReopens(t *testing.T) {
	b := NewBreaker(2, 10*time.Millisecond)

	_ = b.Do(func() error { return errTest })
	_ = b.Do(func() error { return errTest })

	time.Sleep(15 * time.Millisecond)

	// Probe call fails
	_ = b.Do(func() error { return errTest })
	if b.State() != StateOpen {
		t.Errorf("expected StateOpen after probe failure, got %v", b.State())
	}
}

func TestBreakerReset(t *testing.T) {
	b := NewBreaker(2, time.Hour)

	_ = b.Do(func() error { return errTest })
	_ = b.Do(func() error { return errTest })

	if b.State() != StateOpen {
		t.Fatal("expected StateOpen")
	}

	b.Reset()

	if b.State() != StateClosed {
		t.Errorf("expected StateClosed after Reset, got %v", b.State())
	}

	called := false
	err := b.Do(func() error { called = true; return nil })
	if err != nil || !called {
		t.Error("expected call to succeed after Reset")
	}
}

func TestBreakerSuccessResetsFailureCount(t *testing.T) {
	b := NewBreaker(3, time.Second)

	// 2 failures, then success
	_ = b.Do(func() error { return errTest })
	_ = b.Do(func() error { return errTest })
	_ = b.Do(func() error { return nil })

	// 2 more failures should not open (count was reset)
	_ = b.Do(func() error { return errTest })
	_ = b.Do(func() error { return errTest })

	if b.State() != StateClosed {
		t.Error("expected StateClosed — success should have reset failure count")
	}
}
