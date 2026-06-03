package nhi

import (
	"context"
	"crypto/rand"
	"math/big"
	"sync"
	"time"
)

const (
	renewalThreshold    = 0.33
	maxBackoffDelay     = 60 * time.Second
	baseBackoffDelay    = 500 * time.Millisecond
	renewalCheckMinimum = 1 * time.Second
	maxRenewalAttempts  = 10
)

type renewal struct {
	mu        sync.Mutex
	cancel    context.CancelFunc
	running   bool
	done      chan struct{}
	exchange  func(ctx context.Context) (*Credential, error)
	store     *store
	backoff   func(attempt int) time.Duration
	threshold float64
	event     func(ProviderEvent)
}

func newRenewal(s *store, exchangeFn func(ctx context.Context) (*Credential, error), threshold float64, event func(ProviderEvent)) *renewal {
	if threshold <= 0 || threshold >= 1 {
		threshold = renewalThreshold
	}
	return &renewal{
		store:     s,
		exchange:  exchangeFn,
		backoff:   backoffWithJitter,
		threshold: threshold,
		event:     event,
		done:      make(chan struct{}),
	}
}

func (r *renewal) Start() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.running = true
	go r.loop(ctx)
}

func (r *renewal) Stop() {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return
	}
	r.cancel()
	r.running = false
	done := r.done
	r.mu.Unlock()
	<-done
	r.mu.Lock()
	r.done = make(chan struct{})
	r.mu.Unlock()
}

func (r *renewal) loop(ctx context.Context) {
	defer close(r.done)
	for {
		sleepDur := r.nextRenewalIn()
		if sleepDur < renewalCheckMinimum {
			sleepDur = renewalCheckMinimum
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(sleepDur):
		}

		r.renew(ctx)
	}
}

func (r *renewal) renew(ctx context.Context) {
	var attempt int
	for {
		if ctx.Err() != nil {
			return
		}

		if attempt >= maxRenewalAttempts {
			return
		}

		cred, err := r.exchange(ctx)
		if err == nil && cred != nil {
			r.store.Set(cred)
			r.emit(credentialEvent(EventRenewalSucceeded, cred.Target, cred, attempt, nil, "", false))
			return
		}

		attempt++
		r.emit(ProviderEvent{Type: EventRenewalFailed, Attempt: attempt, Error: err})
		delay := r.backoff(attempt)

		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
	}
}

func (r *renewal) nextRenewalIn() time.Duration {
	expiresAt := r.store.ExpiresAt()
	if expiresAt.IsZero() {
		return renewalCheckMinimum
	}

	remaining := time.Until(expiresAt)
	if remaining <= 0 {
		return 0
	}

	threshold := time.Duration(float64(remaining) * r.threshold)
	renewAt := remaining - threshold
	if renewAt < 0 {
		return 0
	}
	return renewAt
}

func (r *renewal) emit(event ProviderEvent) {
	if r.event != nil {
		r.event(event)
	}
}

func backoffWithJitter(attempt int) time.Duration {
	base := baseBackoffDelay
	for i := 1; i < attempt; i++ {
		if base >= maxBackoffDelay/2 {
			base = maxBackoffDelay
			break
		}
		base *= 2
	}
	jitter := cryptoJitter(base)
	return base + jitter
}

func cryptoJitter(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return max / 2
	}
	return time.Duration(n.Int64())
}
