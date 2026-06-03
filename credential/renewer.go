package credential

import (
	"context"
	"crypto/rand"
	"math/big"
	"sync"
	"time"
)

type RenewalFunc func(ctx context.Context) ([]byte, time.Time, error)

type RenewalEvent struct {
	Success bool
	Attempt int
	Error   error
	Renewed time.Time
	NextTTL time.Duration
}

type RenewerConfig struct {
	Key       string
	Store     *MemStore
	RenewFn   RenewalFunc
	TTL       time.Duration
	BaseDelay time.Duration
	MaxDelay  time.Duration
	OnEvent   func(RenewalEvent)
}

type Renewer struct {
	cfg     RenewerConfig
	mu      sync.RWMutex
	expiry  time.Time
	cancel  context.CancelFunc
	done    chan struct{}
	started bool
}

func NewRenewer(cfg RenewerConfig) *Renewer {
	if cfg.BaseDelay == 0 {
		cfg.BaseDelay = time.Second
	}
	if cfg.MaxDelay == 0 {
		cfg.MaxDelay = 30 * time.Second
	}
	return &Renewer{
		cfg:  cfg,
		done: make(chan struct{}),
	}
}

func (r *Renewer) Start(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.started {
		return
	}
	ctx, r.cancel = context.WithCancel(ctx)
	r.started = true
	go r.loop(ctx)
}

func (r *Renewer) Stop() {
	r.mu.Lock()
	if !r.started {
		r.mu.Unlock()
		return
	}
	r.started = false
	r.mu.Unlock()
	if r.cancel != nil {
		r.cancel()
	}
	<-r.done
	r.mu.Lock()
	r.done = make(chan struct{})
	r.mu.Unlock()
}

func (r *Renewer) Expiry() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.expiry
}

func (r *Renewer) SetExpiry(t time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.expiry = t
}

func (r *Renewer) loop(ctx context.Context) {
	defer close(r.done)

	for {
		sleepDur := r.renewalSleep()
		select {
		case <-ctx.Done():
			return
		case <-time.After(sleepDur):
		}

		r.performRenewal(ctx)
	}
}

func (r *Renewer) renewalSleep() time.Duration {
	r.mu.RLock()
	expiry := r.expiry
	r.mu.RUnlock()

	if expiry.IsZero() {
		return r.cfg.TTL / 3
	}

	remaining := time.Until(expiry)
	if remaining <= 0 {
		return 0
	}
	target := remaining / 3
	if target < 100*time.Millisecond {
		target = 100 * time.Millisecond
	}
	return target
}

func (r *Renewer) performRenewal(ctx context.Context) {
	var attempt int
	delay := r.cfg.BaseDelay

	for {
		attempt++

		newCred, newExpiry, err := r.cfg.RenewFn(ctx)
		if err == nil {
			if storeErr := r.cfg.Store.Store(r.cfg.Key, newCred); storeErr != nil {
				err = storeErr
			} else {
				r.SetExpiry(newExpiry)

				if r.cfg.OnEvent != nil {
					r.cfg.OnEvent(RenewalEvent{
						Success: true,
						Attempt: attempt,
						Renewed: time.Now(),
						NextTTL: time.Until(newExpiry),
					})
				}
				return
			}
		}

		if r.cfg.OnEvent != nil {
			r.cfg.OnEvent(RenewalEvent{
				Success: false,
				Attempt: attempt,
				Error:   err,
			})
		}

		if ctx.Err() != nil {
			return
		}

		jitteredDelay := delay/2 + cryptoJitter(delay/2)

		select {
		case <-ctx.Done():
			return
		case <-time.After(jitteredDelay):
		}

		delay *= 2
		if delay > r.cfg.MaxDelay {
			delay = r.cfg.MaxDelay
		}
	}
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
