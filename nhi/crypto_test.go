package nhi

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"testing"
	"time"
)

func TestParseEd25519PublicKey_PKIXPEM(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatalf("marshal PKIX: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})

	parsed, err := parseEd25519PublicKey(string(pemBytes))
	if err != nil {
		t.Fatalf("parse PKIX PEM: %v", err)
	}
	if !parsed.Equal(pub) {
		t.Fatal("parsed PKIX PEM key does not match original")
	}
}

func TestParseEd25519PublicKey_RawBase64(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	raw := base64.StdEncoding.EncodeToString(pub)
	parsed, err := parseEd25519PublicKey(raw)
	if err != nil {
		t.Fatalf("parse raw base64: %v", err)
	}
	if !parsed.Equal(pub) {
		t.Fatal("parsed raw key does not match original")
	}
}

func TestParseEd25519PublicKey_BareBase64PKIX(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatalf("marshal PKIX: %v", err)
	}
	parsed, err := parseEd25519PublicKey(base64.StdEncoding.EncodeToString(der))
	if err != nil {
		t.Fatalf("parse bare base64 PKIX: %v", err)
	}
	if !parsed.Equal(pub) {
		t.Fatal("parsed bare base64 PKIX key does not match original")
	}
}

func TestVerifyPolicySignature_UsesRawExpiresAt(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	rawExpires := "2026-05-29T12:34:56.123456789Z"
	parsed, err := time.Parse(time.RFC3339Nano, rawExpires)
	if err != nil {
		t.Fatalf("parse time: %v", err)
	}

	envelope := signatureEnvelope{
		CredentialType: "database",
		Credentials:    map[string]string{"username": "svc", "password": "secret"},
		ExpiresAt:      rawExpires,
		TTLSeconds:     900,
		AuditID:        "audit-1",
	}
	signed, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	sig := ed25519.Sign(priv, signed)

	resp := &exchangeResponse{
		CredentialType:  "database",
		Credentials:     map[string]string{"username": "svc", "password": "secret"},
		ExpiresAt:       parsed,
		TTLSeconds:      900,
		AuditID:         "audit-1",
		PolicySignature: base64.RawURLEncoding.EncodeToString(sig),
		rawExpiresAt:    rawExpires,
	}

	if err := verifyPolicySignature(resp, base64.StdEncoding.EncodeToString(pub), true); err != nil {
		t.Fatalf("expected signature to verify against raw expires_at, got: %v", err)
	}

	resp.rawExpiresAt = ""
	if err := verifyPolicySignature(resp, base64.StdEncoding.EncodeToString(pub), true); err == nil {
		t.Fatal("expected signature verification to fail when raw expires_at is reformatted")
	}
}
