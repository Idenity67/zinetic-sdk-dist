package credential

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestRenewer_RenewsBeforeExpiry(t *testing.T) {
	store := NewMemStore()
	defer store.ZeroizeAll()

	store.Store("token", []byte("initial"))

	var renewCount atomic.Int32

	renewer := NewRenewer(RenewerConfig{
		Key:   "token",
		Store: store,
		TTL:   600 * time.Millisecond,
		RenewFn: func(ctx context.Context) ([]byte, time.Time, error) {
			renewCount.Add(1)
			return []byte("renewed-value"), time.Now().Add(600 * time.Millisecond), nil
		},
		BaseDelay: 10 * time.Millisecond,
		MaxDelay:  50 * time.Millisecond,
	})
	renewer.SetExpiry(time.Now().Add(600 * time.Millisecond))

	ctx := context.Background()
	renewer.Start(ctx)

	time.Sleep(1200 * time.Millisecond)
	renewer.Stop()

	if renewCount.Load() == 0 {
		t.Fatal("expected at least one renewal")
	}

	val, ok := store.Retrieve("token")
	if !ok {
		t.Fatal("expected token in store")
	}
	if string(val) != "renewed-value" {
		t.Fatalf("expected 'renewed-value', got %q", val)
	}
}

func TestRenewer_RetriesOnFailure(t *testing.T) {
	store := NewMemStore()
	defer store.ZeroizeAll()

	store.Store("cred", []byte("original"))

	var attemptCount atomic.Int32

	renewer := NewRenewer(RenewerConfig{
		Key:   "cred",
		Store: store,
		TTL:   600 * time.Millisecond,
		RenewFn: func(ctx context.Context) ([]byte, time.Time, error) {
			n := attemptCount.Add(1)
			if n < 3 {
				return nil, time.Time{}, fmt.Errorf("transient failure %d", n)
			}
			return []byte("success-after-retries"), time.Now().Add(time.Second), nil
		},
		BaseDelay: 5 * time.Millisecond,
		MaxDelay:  20 * time.Millisecond,
	})
	renewer.SetExpiry(time.Now().Add(600 * time.Millisecond))

	ctx := context.Background()
	renewer.Start(ctx)
	time.Sleep(1200 * time.Millisecond)
	renewer.Stop()

	if attemptCount.Load() < 3 {
		t.Fatalf("expected at least 3 attempts, got %d", attemptCount.Load())
	}

	val, ok := store.Retrieve("cred")
	if !ok {
		t.Fatal("expected credential in store after retry")
	}
	if string(val) != "success-after-retries" {
		t.Fatalf("unexpected value: %q", val)
	}
}

func TestRenewer_StopsOnCancel(t *testing.T) {
	store := NewMemStore()
	defer store.ZeroizeAll()

	renewer := NewRenewer(RenewerConfig{
		Key:   "key",
		Store: store,
		TTL:   time.Second,
		RenewFn: func(ctx context.Context) ([]byte, time.Time, error) {
			return []byte("val"), time.Now().Add(time.Second), nil
		},
	})
	renewer.SetExpiry(time.Now().Add(time.Second))

	ctx := context.Background()
	renewer.Start(ctx)

	done := make(chan struct{})
	go func() {
		renewer.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("renewer did not stop in time")
	}
}

func TestRenewer_EventCallbacks(t *testing.T) {
	store := NewMemStore()
	defer store.ZeroizeAll()

	var events []RenewalEvent
	var eventMu atomic.Int32

	renewer := NewRenewer(RenewerConfig{
		Key:   "key",
		Store: store,
		TTL:   600 * time.Millisecond,
		RenewFn: func(ctx context.Context) ([]byte, time.Time, error) {
			n := eventMu.Add(1)
			if n == 1 {
				return nil, time.Time{}, fmt.Errorf("fail once")
			}
			return []byte("ok"), time.Now().Add(time.Second), nil
		},
		BaseDelay: 5 * time.Millisecond,
		MaxDelay:  10 * time.Millisecond,
		OnEvent: func(e RenewalEvent) {
			events = append(events, e)
		},
	})
	renewer.SetExpiry(time.Now().Add(600 * time.Millisecond))

	ctx := context.Background()
	renewer.Start(ctx)
	time.Sleep(1200 * time.Millisecond)
	renewer.Stop()

	if len(events) < 2 {
		t.Fatalf("expected at least 2 events (1 failure + 1 success), got %d", len(events))
	}

	if events[0].Success {
		t.Fatal("first event should be a failure")
	}

	foundSuccess := false
	for _, e := range events {
		if e.Success {
			foundSuccess = true
			break
		}
	}
	if !foundSuccess {
		t.Fatal("expected at least one success event")
	}
}

func TestRenewer_StopBeforeStart(t *testing.T) {
	store := NewMemStore()
	defer store.ZeroizeAll()

	renewer := NewRenewer(RenewerConfig{
		Key:   "key",
		Store: store,
		TTL:   time.Second,
		RenewFn: func(ctx context.Context) ([]byte, time.Time, error) {
			return []byte("val"), time.Now().Add(time.Second), nil
		},
	})

	done := make(chan struct{})
	go func() {
		renewer.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Stop() blocked on never-started renewer")
	}
}

func TestRenewer_DoubleStart(t *testing.T) {
	store := NewMemStore()
	defer store.ZeroizeAll()

	var calls atomic.Int32
	renewer := NewRenewer(RenewerConfig{
		Key:   "key",
		Store: store,
		TTL:   time.Second,
		RenewFn: func(ctx context.Context) ([]byte, time.Time, error) {
			calls.Add(1)
			return []byte("val"), time.Now().Add(time.Hour), nil
		},
	})
	renewer.SetExpiry(time.Now().Add(50 * time.Millisecond))

	ctx := context.Background()
	renewer.Start(ctx)
	renewer.Start(ctx)

	time.Sleep(200 * time.Millisecond)
	renewer.Stop()

	if calls.Load() > 2 {
		t.Fatalf("double Start should be idempotent, got %d calls indicating multiple goroutines", calls.Load())
	}
}
