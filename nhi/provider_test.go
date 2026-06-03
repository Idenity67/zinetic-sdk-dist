package nhi

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewProvider_Validation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ProviderConfig
		wantErr string
	}{
		{
			name:    "missing backend URL",
			cfg:     ProviderConfig{Target: "postgres-prod"},
			wantErr: "backend URL is required",
		},
		{
			name:    "missing target",
			cfg:     ProviderConfig{BackendURL: "https://api.zinetic.io"},
			wantErr: "target resource is required",
		},
		{
			name:    "require policy signature without key",
			cfg:     ProviderConfig{BackendURL: "https://api.zinetic.io", Target: "postgres-prod", RequirePolicySignature: true},
			wantErr: "policy signature verification requires PolicyPublicKey",
		},
		{
			name:    "invalid hardware mode",
			cfg:     ProviderConfig{BackendURL: "https://api.zinetic.io", Target: "postgres-prod", HardwareMode: "maybe"},
			wantErr: "HardwareMode must be one of auto, required, or off",
		},
		{
			name:    "invalid refresh threshold",
			cfg:     ProviderConfig{BackendURL: "https://api.zinetic.io", Target: "postgres-prod", RefreshThreshold: 1.2},
			wantErr: "RefreshThreshold must be greater than 0 and less than 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewProvider(tt.cfg)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Error() != "nhi: "+tt.wantErr {
				t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestNewProvider_GeneratesKey(t *testing.T) {
	p, err := NewProvider(ProviderConfig{
		BackendURL: "https://api.zinetic.io",
		Target:     "postgres-prod",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.cfg.DPoPKey == nil {
		t.Fatal("expected DPoP key to be generated")
	}
}

func TestNewProvider_NormalizesBackendURL(t *testing.T) {
	cases := map[string]string{
		"https://api.zinetic.io":        "https://api.zinetic.io/api/v1/decision/exchange",
		"https://api.zinetic.io/api/v1": "https://api.zinetic.io/api/v1/decision/exchange",
		"https://api.zinetic.io/v1":     "https://api.zinetic.io/api/v1/decision/exchange",
		"http://localhost:8080/v1":      "http://localhost:8080/api/v1/decision/exchange",
	}
	for input, wantEndpoint := range cases {
		t.Run(input, func(t *testing.T) {
			p, err := NewProvider(ProviderConfig{BackendURL: input, Target: "postgres-prod"})
			if err != nil {
				t.Fatalf("NewProvider returned error: %v", err)
			}
			if got := p.exchangeEndpoint(); got != wantEndpoint {
				t.Fatalf("expected endpoint %s, got %s", wantEndpoint, got)
			}
		})
	}
}

func TestNewProvider_RejectsRemoteHTTP(t *testing.T) {
	_, err := NewProvider(ProviderConfig{BackendURL: "http://api.zinetic.io", Target: "postgres-prod"})
	if err == nil {
		t.Fatal("expected remote HTTP backend URL to be rejected")
	}
}

func TestNewProvider_UsesProvidedKey(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	p, err := NewProvider(ProviderConfig{
		BackendURL: "https://api.zinetic.io",
		Target:     "postgres-prod",
		DPoPKey:    key,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.cfg.DPoPKey != key {
		t.Fatal("expected provider to use the provided key")
	}
}

func TestProvider_GetCredential_NoCredential(t *testing.T) {
	p, err := NewProvider(ProviderConfig{
		BackendURL: "https://api.zinetic.io",
		Target:     "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.GetCredential("password")
	if err == nil {
		t.Fatal("expected error when no credential available")
	}
}

func TestProvider_StartUsesEncryptedExchangeV2(t *testing.T) {
	t.Setenv("ZINETIC_ACCESS_TOKEN", "local-session-token")

	var captured exchangeRequest
	var events []ProviderEvent
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/decision/exchange" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("DPoP") == "" {
			t.Fatal("expected DPoP proof")
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		encrypted, err := encryptCredentialsForTest(captured.CredentialEncryptionJWK, map[string]string{
			"username": "app",
			"password": "secret",
		})
		if err != nil {
			t.Fatalf("encrypt test credentials: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(exchangeResponse{
			EncryptedCredentials: encrypted,
			ExpiresAt:            time.Now().Add(time.Hour),
			TTLSeconds:           3600,
			PolicySignature:      "dev",
			PolicyVersion:        "policy-v1",
			PolicySigningKeyID:   "key-1",
			AuditID:              "audit-123",
			TransactionHash:      "tx-abc",
			LedgerAnchorHash:     "ledger-def",
		})
	}))
	defer srv.Close()

	provider, err := NewProvider(ProviderConfig{
		BackendURL:       srv.URL,
		Target:           "postgres-staging",
		TenantID:         "tenant-123",
		Environment:      EnvLocal,
		HardwareMode:     HardwareModeAuto,
		RefreshThreshold: 0.25,
		EventCallback: func(event ProviderEvent) {
			events = append(events, event)
		},
	})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	if err := provider.Start(t.Context()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer provider.Stop()

	if captured.ExchangeVersion != 2 {
		t.Fatalf("expected exchange version 2, got %d", captured.ExchangeVersion)
	}
	if captured.TenantID != "tenant-123" {
		t.Fatalf("expected tenant id, got %q", captured.TenantID)
	}
	if captured.HardwareMode != HardwareModeAuto {
		t.Fatalf("expected hardware mode auto, got %q", captured.HardwareMode)
	}
	if captured.CredentialEncryptionJWK["kty"] != "EC" {
		t.Fatalf("expected credential encryption JWK, got %#v", captured.CredentialEncryptionJWK)
	}
	password, err := provider.GetPassword()
	if err != nil {
		t.Fatalf("GetPassword: %v", err)
	}
	if password != "secret" {
		t.Fatalf("expected decrypted password, got %q", password)
	}
	metadata, err := provider.Metadata()
	if err != nil {
		t.Fatalf("Metadata: %v", err)
	}
	if metadata.AuditID != "audit-123" || metadata.PolicyVersion != "policy-v1" || metadata.TransactionHash != "tx-abc" || metadata.LedgerAnchorHash != "ledger-def" {
		t.Fatalf("unexpected metadata: %#v", metadata)
	}
	if len(events) == 0 || events[0].Type != EventExchangeSucceeded || events[0].AuditID != "audit-123" {
		t.Fatalf("expected exchange success event, got %#v", events)
	}
}

func TestProvider_HardwareRequiredUnavailable(t *testing.T) {
	t.Setenv("ZINETIC_ACCESS_TOKEN", "local-session-token")

	provider, err := NewProvider(ProviderConfig{
		BackendURL:       "http://127.0.0.1:65535",
		Target:           "postgres-staging",
		Environment:      EnvLocal,
		HardwareMode:     HardwareModeRequired,
		HardwareProvider: &noopHardwareProvider{},
	})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	err = provider.Start(t.Context())
	if err == nil || !strings.Contains(err.Error(), ErrHardwareUnavailable.Error()) {
		t.Fatalf("expected hardware unavailable error, got %v", err)
	}
}

func TestProvider_SendsHardwarePayload(t *testing.T) {
	t.Setenv("ZINETIC_ACCESS_TOKEN", "local-session-token")
	var captured exchangeRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(exchangeResponse{
			Credentials:     map[string]string{"password": "secret"},
			ExpiresAt:       time.Now().Add(time.Hour),
			PolicySignature: "dev",
		})
	}))
	defer srv.Close()

	provider, err := NewProvider(ProviderConfig{
		BackendURL:             srv.URL,
		Target:                 "postgres-staging",
		Environment:            EnvLocal,
		AllowPlaintextResponse: true,
		HardwareMode:           HardwareModeAuto,
		HardwareProvider:       &availableHardwareProvider{},
	})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	if err := provider.Start(t.Context()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer provider.Stop()

	if captured.HardwareKeyID != "hw-key-1" {
		t.Fatalf("expected hardware key id, got %q", captured.HardwareKeyID)
	}
	if captured.HardwareAttestation["public_key"] == nil || captured.HardwareAttestation["attestation"] == nil {
		t.Fatalf("expected hardware attestation payload, got %#v", captured.HardwareAttestation)
	}
}

func TestNewHTTPTransport_DefaultsAndOptions(t *testing.T) {
	p, _ := NewProvider(ProviderConfig{
		BackendURL: "https://api.zinetic.io",
		Target:     "test",
	})

	tr := NewHTTPTransport(p)
	if tr.tokenKey != "token" {
		t.Fatalf("expected default tokenKey=token, got %q", tr.tokenKey)
	}

	tr2 := NewHTTPTransport(p, WithTokenKey("password"))
	if tr2.tokenKey != "password" {
		t.Fatalf("expected tokenKey=password, got %q", tr2.tokenKey)
	}
}

func TestNewHTTPClient_ReturnsClient(t *testing.T) {
	p, _ := NewProvider(ProviderConfig{
		BackendURL: "https://api.zinetic.io",
		Target:     "test",
	})

	client := NewHTTPClient(p)
	if client == nil {
		t.Fatal("expected non-nil http.Client")
	}
}

func TestFakeAddr(t *testing.T) {
	var a fakeAddr
	if a.Network() != "nhi" {
		t.Fatalf("expected nhi, got %q", a.Network())
	}
	if a.String() != "nhi-managed" {
		t.Fatalf("expected nhi-managed, got %q", a.String())
	}
}

func TestNewConnector_Validation(t *testing.T) {
	p, _ := NewProvider(ProviderConfig{
		BackendURL: "https://api.zinetic.io",
		Target:     "test",
	})

	cases := []struct {
		name string
		cfg  ConnectorConfig
	}{
		{"nil provider", ConnectorConfig{DSNTemplate: "dsn", BaseDriver: &fakeDriver{}}},
		{"empty DSN", ConnectorConfig{Provider: p, BaseDriver: &fakeDriver{}}},
		{"nil driver", ConnectorConfig{Provider: p, DSNTemplate: "dsn"}},
	}

	for _, c := range cases {
		if _, err := NewConnector(c.cfg); err == nil {
			t.Fatalf("%s: expected error", c.name)
		}
	}
}

func TestNewConnector_DefaultKeys(t *testing.T) {
	p, _ := NewProvider(ProviderConfig{
		BackendURL: "https://api.zinetic.io",
		Target:     "test",
	})

	c, err := NewConnector(ConnectorConfig{
		Provider:    p,
		DSNTemplate: "postgres://{{username}}:{{password}}@host/db",
		BaseDriver:  &fakeDriver{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if c.cfg.PasswordKey != "password" || c.cfg.UsernameKey != "username" {
		t.Fatalf("expected default keys, got password=%q username=%q", c.cfg.PasswordKey, c.cfg.UsernameKey)
	}
}

func TestOpenSQLDB_ReturnsDB(t *testing.T) {
	p, _ := NewProvider(ProviderConfig{
		BackendURL: "https://api.zinetic.io",
		Target:     "test",
	})

	db, err := OpenSQLDB(ConnectorConfig{
		Provider:    p,
		DSNTemplate: "postgres://{{username}}:{{password}}@host/db",
		BaseDriver:  &fakeDriver{},
	})
	if err != nil {
		t.Fatalf("OpenSQLDB: %v", err)
	}
	if db == nil {
		t.Fatal("expected db")
	}
	db.Close()
}

type fakeDriver struct{}

func (d *fakeDriver) Open(_ string) (driver.Conn, error) {
	return nil, fmt.Errorf("not implemented")
}

type availableHardwareProvider struct{}

func (a *availableHardwareProvider) Available(_ context.Context) bool {
	return true
}

func (a *availableHardwareProvider) GenerateKey(_ context.Context, opts KeyOptions) (*PublicKeyInfo, error) {
	if !opts.NonExportable {
		return nil, fmt.Errorf("expected non-exportable key")
	}
	return &PublicKeyInfo{
		Algorithm:    opts.Algorithm,
		PublicKeyPEM: "-----BEGIN PUBLIC KEY-----\nmock\n-----END PUBLIC KEY-----",
		KeyID:        "hw-key-1",
		Metadata:     map[string]string{"provider": "mock"},
	}, nil
}

func (a *availableHardwareProvider) Sign(_ context.Context, challenge []byte) ([]byte, error) {
	return append([]byte("signed:"), challenge...), nil
}

func (a *availableHardwareProvider) Attest(_ context.Context) (*AttestationDocument, error) {
	return &AttestationDocument{
		Format:   "mock",
		Data:     []byte("attestation"),
		Metadata: map[string]string{"provider": "mock"},
	}, nil
}

func encryptCredentialsForTest(clientJWK map[string]string, creds map[string]string) (*EncryptedCredentials, error) {
	clientPub, err := publicKeyFromJWK(clientJWK)
	if err != nil {
		return nil, err
	}
	serverKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	secret, err := serverKey.ECDH(clientPub)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	derivedKey, err := hkdf.Key(sha256.New, secret, nonce, "zinetic-nhi-credential-package-v2", 32)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plaintext, err := json.Marshal(creds)
	if err != nil {
		return nil, err
	}
	serverPub := serverKey.PublicKey().Bytes()
	return &EncryptedCredentials{
		Algorithm:  "ECDH-ES+HKDF-SHA256",
		Encryption: "A256GCM",
		Ephemeral: map[string]string{
			"kty": "EC",
			"crv": "P-256",
			"x":   base64.RawURLEncoding.EncodeToString(serverPub[1:33]),
			"y":   base64.RawURLEncoding.EncodeToString(serverPub[33:65]),
		},
		Nonce:      base64.RawURLEncoding.EncodeToString(nonce),
		Ciphertext: base64.RawURLEncoding.EncodeToString(gcm.Seal(nil, nonce, plaintext, nil)),
	}, nil
}
