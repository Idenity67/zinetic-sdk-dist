package nhi

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestBackoffWithJitter(t *testing.T) {
	for attempt := 1; attempt <= 10; attempt++ {
		delay := backoffWithJitter(attempt)
		if delay <= 0 {
			t.Fatalf("attempt %d: expected positive delay, got %v", attempt, delay)
		}
		if delay > 2*maxBackoffDelay {
			t.Fatalf("attempt %d: delay %v exceeds 2x max %v", attempt, delay, maxBackoffDelay)
		}
	}
}

func TestBackoffWithJitter_Capped(t *testing.T) {
	delay := backoffWithJitter(100)
	if delay > 2*maxBackoffDelay {
		t.Fatalf("high attempt delay %v should be capped near %v", delay, maxBackoffDelay)
	}
}

func TestRenewal_NextRenewalIn_EmptyStore(t *testing.T) {
	s := newStore()
	r := newRenewal(s, nil, renewalThreshold, nil)

	dur := r.nextRenewalIn()
	if dur != renewalCheckMinimum {
		t.Fatalf("expected %v, got %v", renewalCheckMinimum, dur)
	}
}

func TestRenewal_NextRenewalIn_WithCredential(t *testing.T) {
	s := newStore()
	s.Set(&Credential{
		Values:    map[string]string{"k": "v"},
		ExpiresAt: time.Now().Add(60 * time.Second),
	})

	r := newRenewal(s, nil, renewalThreshold, nil)
	dur := r.nextRenewalIn()

	expectedMax := 60 * time.Second
	if dur <= 0 || dur > expectedMax {
		t.Fatalf("expected renewal duration in (0, %v], got %v", expectedMax, dur)
	}

	expectedRenewAt := time.Duration(float64(60*time.Second) * (1.0 - renewalThreshold))
	tolerance := 2 * time.Second
	if dur < expectedRenewAt-tolerance || dur > expectedRenewAt+tolerance {
		t.Fatalf("expected ~%v, got %v", expectedRenewAt, dur)
	}
}

func TestRenewal_NextRenewalIn_Expired(t *testing.T) {
	s := newStore()
	s.Set(&Credential{
		Values:    map[string]string{"k": "v"},
		ExpiresAt: time.Now().Add(-10 * time.Second),
	})

	r := newRenewal(s, nil, renewalThreshold, nil)
	dur := r.nextRenewalIn()
	if dur != 0 {
		t.Fatalf("expected 0 for expired credential, got %v", dur)
	}
}

func TestRenewal_MaxAttempts(t *testing.T) {
	var attempts atomic.Int32
	failingExchange := func(ctx context.Context) (*Credential, error) {
		attempts.Add(1)
		return nil, fmt.Errorf("transient error")
	}

	s := newStore()
	s.Set(&Credential{
		Values:    map[string]string{"k": "v"},
		ExpiresAt: time.Now().Add(-1 * time.Second),
	})
	r := newRenewal(s, failingExchange, renewalThreshold, nil)
	r.backoff = func(int) time.Duration { return 0 }

	ctx := context.Background()
	r.renew(ctx)

	got := int(attempts.Load())
	if got != maxRenewalAttempts {
		t.Fatalf("expected exactly %d attempts, got %d", maxRenewalAttempts, got)
	}
}

func TestRenewal_StopWaitsForGoroutine(t *testing.T) {
	var entered atomic.Bool
	var exited atomic.Bool
	exchange := func(ctx context.Context) (*Credential, error) {
		entered.Store(true)
		<-ctx.Done()
		exited.Store(true)
		return nil, ctx.Err()
	}

	s := newStore()
	s.Set(&Credential{
		Values:    map[string]string{"k": "v"},
		ExpiresAt: time.Now().Add(-1 * time.Second),
	})
	r := newRenewal(s, exchange, renewalThreshold, nil)
	r.backoff = func(int) time.Duration { return 0 }

	r.Start()

	deadline := time.Now().Add(3 * time.Second)
	for !entered.Load() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !entered.Load() {
		t.Fatal("exchange was never called")
	}

	r.Stop()

	if !exited.Load() {
		t.Fatal("goroutine still running after Stop returned")
	}
}

func TestRenewal_StopBeforeStart(t *testing.T) {
	s := newStore()
	r := newRenewal(s, nil, renewalThreshold, nil)
	r.Stop()
}

func TestRenewal_RestartAfterStop(t *testing.T) {
	var calls atomic.Int32
	exchange := func(ctx context.Context) (*Credential, error) {
		calls.Add(1)
		return &Credential{
			Values:    map[string]string{"token": "new"},
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}, nil
	}

	s := newStore()
	s.Set(&Credential{
		Values:    map[string]string{"token": "old"},
		ExpiresAt: time.Now().Add(-1 * time.Second),
	})
	r := newRenewal(s, exchange, renewalThreshold, nil)
	r.backoff = func(int) time.Duration { return 0 }

	r.Start()
	time.Sleep(1500 * time.Millisecond)
	r.Stop()

	before := calls.Load()
	if before == 0 {
		t.Fatal("expected at least one exchange call in first run")
	}

	s.Set(&Credential{
		Values:    map[string]string{"token": "expired-again"},
		ExpiresAt: time.Now().Add(-1 * time.Second),
	})

	r.Start()
	time.Sleep(1500 * time.Millisecond)
	r.Stop()

	after := calls.Load()
	if after <= before {
		t.Fatal("expected exchange calls after restart")
	}
}
