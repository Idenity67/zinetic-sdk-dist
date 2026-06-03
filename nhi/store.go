package nhi

import (
	"strings"
	"sync"
	"time"
)

type Credential struct {
	Values             map[string]string
	ExpiresAt          time.Time
	TTLSeconds         int
	PolicySignature    string
	PolicyVersion      string
	PolicySigningKeyID string
	AuditID            string
	TransactionHash    string
	LedgerAnchorHash   string
	Target             string
}

type CredentialMetadata struct {
	AuditID            string
	PolicyVersion      string
	PolicySigningKeyID string
	TransactionHash    string
	LedgerAnchorHash   string
	Target             string
	ExpiresAt          time.Time
	TTLSeconds         int
	PolicySignature    string
}

func (c *Credential) Metadata() CredentialMetadata {
	if c == nil {
		return CredentialMetadata{}
	}
	return CredentialMetadata{
		AuditID:            c.AuditID,
		PolicyVersion:      c.PolicyVersion,
		PolicySigningKeyID: c.PolicySigningKeyID,
		TransactionHash:    c.TransactionHash,
		LedgerAnchorHash:   c.LedgerAnchorHash,
		Target:             c.Target,
		ExpiresAt:          c.ExpiresAt,
		TTLSeconds:         c.TTLSeconds,
		PolicySignature:    c.PolicySignature,
	}
}

func (c *Credential) Remaining() time.Duration {
	return time.Until(c.ExpiresAt)
}

func (c *Credential) Expired() bool {
	return time.Now().After(c.ExpiresAt)
}

type store struct {
	mu   sync.RWMutex
	cred *Credential
}

func newStore() *store {
	return &store{}
}

func (s *store) Get() *Credential {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cred == nil || s.cred.Expired() {
		return nil
	}
	return copyCredential(s.cred)
}

func (s *store) Set(c *Credential) {
	s.mu.Lock()
	defer s.mu.Unlock()
	zeroizeCredential(s.cred)
	s.cred = copyCredential(c)
}

func (s *store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	zeroizeCredential(s.cred)
	s.cred = nil
}

func (s *store) ExpiresAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cred == nil {
		return time.Time{}
	}
	return s.cred.ExpiresAt
}

func copyCredential(c *Credential) *Credential {
	if c == nil {
		return nil
	}
	cp := *c
	cp.Values = make(map[string]string, len(c.Values))
	for k, v := range c.Values {
		cp.Values[k] = v
	}
	return &cp
}

func zeroizeCredential(c *Credential) {
	if c == nil {
		return
	}
	for k, v := range c.Values {
		if len(v) > 0 {
			c.Values[k] = strings.Repeat("\x00", len(v))
		}
		delete(c.Values, k)
	}
	c.PolicySignature = ""
	c.PolicyVersion = ""
	c.PolicySigningKeyID = ""
	c.AuditID = ""
	c.TransactionHash = ""
	c.LedgerAnchorHash = ""
	c.Target = ""
	c.ExpiresAt = time.Time{}
	c.TTLSeconds = 0
}
