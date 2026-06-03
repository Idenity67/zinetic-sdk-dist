package nhi

import (
	"testing"
	"time"
)

func TestStore_SetAndGet(t *testing.T) {
	s := newStore()

	if got := s.Get(); got != nil {
		t.Fatalf("expected nil credential, got %v", got)
	}

	cred := &Credential{
		Values:    map[string]string{"password": "secret123"},
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	s.Set(cred)

	got := s.Get()
	if got == nil {
		t.Fatal("expected credential, got nil")
	}
	if got.Values["password"] != "secret123" {
		t.Fatalf("expected password 'secret123', got %q", got.Values["password"])
	}
}

func TestStore_ExpiredCredential(t *testing.T) {
	s := newStore()
	cred := &Credential{
		Values:    map[string]string{"token": "expired"},
		ExpiresAt: time.Now().Add(-1 * time.Minute),
	}
	s.Set(cred)

	if got := s.Get(); got != nil {
		t.Fatalf("expected nil for expired credential, got %v", got)
	}
}

func TestStore_Clear(t *testing.T) {
	s := newStore()
	s.Set(&Credential{
		Values:    map[string]string{"x": "y"},
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})

	s.Clear()
	if got := s.Get(); got != nil {
		t.Fatalf("expected nil after clear, got %v", got)
	}
}

func TestStore_ExpiresAt(t *testing.T) {
	s := newStore()

	if !s.ExpiresAt().IsZero() {
		t.Fatal("expected zero time for empty store")
	}

	exp := time.Now().Add(30 * time.Minute)
	s.Set(&Credential{
		Values:    map[string]string{"k": "v"},
		ExpiresAt: exp,
	})

	if s.ExpiresAt() != exp {
		t.Fatalf("expected %v, got %v", exp, s.ExpiresAt())
	}
}

func TestCredential_Remaining(t *testing.T) {
	cred := &Credential{
		Values:    map[string]string{"k": "v"},
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	remaining := cred.Remaining()
	if remaining < 9*time.Minute || remaining > 10*time.Minute {
		t.Fatalf("expected ~10m remaining, got %v", remaining)
	}
}

func TestCredential_Expired(t *testing.T) {
	future := &Credential{ExpiresAt: time.Now().Add(1 * time.Hour)}
	past := &Credential{ExpiresAt: time.Now().Add(-1 * time.Second)}

	if future.Expired() {
		t.Fatal("expected future credential to not be expired")
	}
	if !past.Expired() {
		t.Fatal("expected past credential to be expired")
	}
}

func TestStore_GetReturnsCopy(t *testing.T) {
	s := newStore()
	s.Set(&Credential{
		Values:    map[string]string{"token": "original"},
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})

	got := s.Get()
	got.Values["token"] = "mutated"
	got.Values["injected"] = "bad"

	internal := s.Get()
	if internal.Values["token"] != "original" {
		t.Fatalf("mutation leaked: got %q, want %q", internal.Values["token"], "original")
	}
	if _, exists := internal.Values["injected"]; exists {
		t.Fatal("injected key leaked into store")
	}
}

func TestStore_ReplacesAndZeroizesPreviousCredential(t *testing.T) {
	s := newStore()
	s.Set(&Credential{
		Values:             map[string]string{"password": "old-secret"},
		ExpiresAt:          time.Now().Add(1 * time.Hour),
		PolicySignature:    "sig-old",
		PolicyVersion:      "policy-old",
		PolicySigningKeyID: "key-old",
		AuditID:            "audit-old",
		TransactionHash:    "tx-old",
		LedgerAnchorHash:   "ledger-old",
		Target:             "old-target",
	})
	previousStored := s.cred

	s.Set(&Credential{
		Values:    map[string]string{"password": "new-secret"},
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Target:    "new-target",
	})

	if len(previousStored.Values) != 0 {
		t.Fatalf("expected previous credential values to be cleared, got %#v", previousStored.Values)
	}
	if previousStored.PolicySignature != "" || previousStored.AuditID != "" || previousStored.Target != "" {
		t.Fatalf("expected previous metadata to be cleared, got %#v", previousStored)
	}
	got := s.Get()
	if got.Values["password"] != "new-secret" || got.Target != "new-target" {
		t.Fatalf("unexpected replacement credential: %#v", got)
	}
}

func TestCredential_Metadata(t *testing.T) {
	exp := time.Now().Add(time.Hour)
	cred := &Credential{
		ExpiresAt:          exp,
		TTLSeconds:         3600,
		PolicySignature:    "sig",
		PolicyVersion:      "policy-v1",
		PolicySigningKeyID: "key-1",
		AuditID:            "audit-1",
		TransactionHash:    "tx-1",
		LedgerAnchorHash:   "ledger-1",
		Target:             "postgres",
	}
	meta := cred.Metadata()
	if meta.AuditID != "audit-1" || meta.PolicyVersion != "policy-v1" || meta.Target != "postgres" || meta.ExpiresAt != exp {
		t.Fatalf("unexpected metadata: %#v", meta)
	}
}

func TestStore_GetCopyIndependence(t *testing.T) {
	s := newStore()
	s.Set(&Credential{
		Values:    map[string]string{"a": "1", "b": "2"},
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})

	c1 := s.Get()
	c2 := s.Get()

	c1.Values["a"] = "modified"

	if c2.Values["a"] != "1" {
		t.Fatal("modifying one copy affected another")
	}
}
