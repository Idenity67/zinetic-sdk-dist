package zinetic

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestReleaseVersionMatchesExpected(t *testing.T) {
	expected := os.Getenv("ZINETIC_SDK_EXPECT_VERSION")
	if expected == "" {
		return
	}
	if Version != expected {
		t.Fatalf("expected Version=%s, got %s", expected, Version)
	}
	if UserAgent != "zinetic-sdk-go/"+expected {
		t.Fatalf("expected UserAgent to include version %s, got %s", expected, UserAgent)
	}
}

func TestDefaultConfig_HasSaneDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxRetries != 3 {
		t.Fatalf("expected MaxRetries=3, got %d", cfg.MaxRetries)
	}
	if cfg.RetryBaseDelay != 500*time.Millisecond {
		t.Fatalf("expected RetryBaseDelay=500ms, got %v", cfg.RetryBaseDelay)
	}
	if cfg.RetryMaxDelay != 30*time.Second {
		t.Fatalf("expected RetryMaxDelay=30s, got %v", cfg.RetryMaxDelay)
	}
	if cfg.RequestTimeout != 30*time.Second {
		t.Fatalf("expected RequestTimeout=30s, got %v", cfg.RequestTimeout)
	}
	if !cfg.EnableTracing {
		t.Fatal("expected EnableTracing=true by default")
	}
	if cfg.ServiceName != "zinetic-sdk" {
		t.Fatalf("expected ServiceName=zinetic-sdk, got %s", cfg.ServiceName)
	}
	if cfg.UserAgent != UserAgent {
		t.Fatalf("expected UserAgent=%s, got %s", UserAgent, cfg.UserAgent)
	}
	if cfg.APIBasePath != "/api/v1" {
		t.Fatalf("expected APIBasePath=/api/v1, got %s", cfg.APIBasePath)
	}
}

func TestConfig_Validate_RequiresBaseURL(t *testing.T) {
	cfg := DefaultConfig()
	cfg.TenantID = "my-tenant"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for empty BaseURL")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Details[0].Field != "base_url" {
		t.Fatalf("expected field base_url, got %s", apiErr.Details[0].Field)
	}
}

func TestConfig_Validate_RequiresTenantID(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BaseURL = "https://api.zinetic.io"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for empty TenantID")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Details[0].Field != "tenant_id" {
		t.Fatalf("expected field tenant_id, got %s", apiErr.Details[0].Field)
	}
}

func TestConfig_Validate_PassesWithRequiredFields(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BaseURL = "https://api.zinetic.io"
	cfg.TenantID = "my-tenant"

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestConfig_Validate_AllowsLocalHTTP(t *testing.T) {
	for _, rawURL := range []string{"http://localhost:8080", "http://127.0.0.1:8080", "http://[::1]:8080"} {
		cfg := DefaultConfig()
		cfg.BaseURL = rawURL
		cfg.TenantID = "my-tenant"
		if err := cfg.Validate(); err != nil {
			t.Fatalf("expected %s to validate, got %v", rawURL, err)
		}
	}
}

func TestConfig_Validate_RejectsRemoteHTTP(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BaseURL = "http://api.zinetic.io"
	cfg.TenantID = "my-tenant"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected remote HTTP URL to be rejected")
	}
}

func TestNewClient_FailsWithoutBaseURL(t *testing.T) {
	_, err := NewClient(WithTenantID("my-tenant"))
	if err == nil {
		t.Fatal("expected error for missing BaseURL")
	}
}

func TestNewClient_FailsWithoutTenantID(t *testing.T) {
	_, err := NewClient(WithBaseURL("https://api.zinetic.io"))
	if err == nil {
		t.Fatal("expected error for missing TenantID")
	}
}

func TestNewClient_Success(t *testing.T) {
	c, err := NewClient(
		WithBaseURL("https://api.zinetic.io"),
		WithTenantID("my-tenant"),
		WithTracing(false),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.Agents == nil {
		t.Fatal("expected Agents service to be initialized")
	}
	if c.Credentials == nil {
		t.Fatal("expected Credentials service to be initialized")
	}
	if c.Decisions == nil {
		t.Fatal("expected Decisions service to be initialized")
	}
	if c.Tokens == nil {
		t.Fatal("expected Tokens service to be initialized")
	}
	if c.Health == nil {
		t.Fatal("expected Health service to be initialized")
	}
}

func TestNewClient_WithDPoPKey(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	c, err := NewClient(
		WithBaseURL("https://api.zinetic.io"),
		WithTenantID("my-tenant"),
		WithDPoPKey(key),
		WithTracing(false),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.config.DPoPPrivateKey == nil {
		t.Fatal("expected DPoP key to be set in config")
	}
}

func TestNewClient_WithAllOptions(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	c, err := NewClient(
		WithBaseURL("https://api.zinetic.io"),
		WithTenantID("my-tenant"),
		WithDPoPKey(key),
		WithAccessToken("at-123"),
		WithRefreshToken("rt-456"),
		WithAPIBasePath("/gateway/v1"),
		WithAttestationToken("attestation-789"),
		WithMaxRetries(5),
		WithRetryDelay(1*time.Second, 60*time.Second),
		WithRequestTimeout(10*time.Second),
		WithTracing(false),
		WithServiceName("test-service"),
		WithUserAgent("test-agent/1.0"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg := c.GetConfig()
	if cfg.AccessToken != "at-123" {
		t.Fatalf("expected AccessToken=at-123, got %s", cfg.AccessToken)
	}
	if cfg.RefreshToken != "rt-456" {
		t.Fatalf("expected RefreshToken=rt-456, got %s", cfg.RefreshToken)
	}
	if c.tcfg.APIBasePath != "/gateway/v1" {
		t.Fatalf("expected APIBasePath=/gateway/v1, got %s", c.tcfg.APIBasePath)
	}
	if c.tcfg.AttestationToken != "attestation-789" {
		t.Fatal("expected attestation token to be mapped")
	}
	if cfg.MaxRetries != 5 {
		t.Fatalf("expected MaxRetries=5, got %d", cfg.MaxRetries)
	}
	if cfg.RetryBaseDelay != 1*time.Second {
		t.Fatalf("expected RetryBaseDelay=1s, got %v", cfg.RetryBaseDelay)
	}
	if cfg.RetryMaxDelay != 60*time.Second {
		t.Fatalf("expected RetryMaxDelay=60s, got %v", cfg.RetryMaxDelay)
	}
	if cfg.RequestTimeout != 10*time.Second {
		t.Fatalf("expected RequestTimeout=10s, got %v", cfg.RequestTimeout)
	}
	if cfg.ServiceName != "test-service" {
		t.Fatalf("expected ServiceName=test-service, got %s", cfg.ServiceName)
	}
	if cfg.UserAgent != "test-agent/1.0" {
		t.Fatalf("expected UserAgent=test-agent/1.0, got %s", cfg.UserAgent)
	}
}

func TestClient_SetAccessToken(t *testing.T) {
	c, _ := NewClient(
		WithBaseURL("https://api.zinetic.io"),
		WithTenantID("my-tenant"),
		WithTracing(false),
	)

	c.SetAccessToken("new-token")
	if c.GetConfig().AccessToken != "new-token" {
		t.Fatal("expected access token to be updated")
	}
}

func TestClient_SetAccessToken_UpdatesTransport(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	c, err := NewClient(
		WithBaseURL(server.URL),
		WithTenantID("my-tenant"),
		WithTracing(false),
	)
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	c.SetAccessToken("new-token")
	if _, err := c.DoRaw(context.Background(), "GET", "/users/me", nil, nil); err != nil {
		t.Fatalf("DoRaw error: %v", err)
	}
	if gotAuth != "Bearer new-token" {
		t.Fatalf("expected transport to use updated token, got %q", gotAuth)
	}
}

func TestClient_SetRefreshToken(t *testing.T) {
	c, _ := NewClient(
		WithBaseURL("https://api.zinetic.io"),
		WithTenantID("my-tenant"),
		WithTracing(false),
	)

	c.SetRefreshToken("new-refresh")
	if c.GetConfig().RefreshToken != "new-refresh" {
		t.Fatal("expected refresh token to be updated")
	}
}

func TestWithClientCredentials(t *testing.T) {
	c, _ := NewClient(
		WithBaseURL("https://api.zinetic.io"),
		WithTenantID("my-tenant"),
		WithClientCredentials("client-id", []byte("client-secret")),
		WithTracing(false),
	)

	cfg := c.GetConfig()
	if cfg.ClientID != "client-id" {
		t.Fatalf("expected ClientID=client-id, got %s", cfg.ClientID)
	}
	if !bytes.Equal(cfg.ClientSecret, []byte("client-secret")) {
		t.Fatalf("expected ClientSecret=client-secret, got %s", cfg.ClientSecret)
	}
}

func TestWithTokenEndpoint(t *testing.T) {
	c, _ := NewClient(
		WithBaseURL("https://api.zinetic.io"),
		WithTenantID("my-tenant"),
		WithTokenEndpoint("https://auth.zinetic.io/token"),
		WithTracing(false),
	)

	if c.GetConfig().TokenEndpoint != "https://auth.zinetic.io/token" {
		t.Fatal("expected token endpoint to be set")
	}
}
