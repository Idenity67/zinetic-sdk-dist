package transport

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"sdk.zinetic.net/apierr"
	"sdk.zinetic.net/dpop"
)

func defaultTestConfig() *Config {
	return &Config{
		MaxRetries:     3,
		RetryBaseDelay: 500 * time.Millisecond,
		RetryMaxDelay:  30 * time.Second,
		RequestTimeout: 30 * time.Second,
		EnableTracing:  false,
		ServiceName:    "zinetic-sdk",
		UserAgent:      "zinetic-sdk-go/test",
	}
}

func TestBuildQueryURL_EncodesSpecialCharacters(t *testing.T) {
	cfg := defaultTestConfig()
	tr := New(cfg)

	result := tr.BuildQueryURL("/v1/search", map[string]string{
		"query":  "hello world",
		"filter": "status=active&type=agent",
	})

	if result == "/v1/search?query=hello world&filter=status=active&type=agent" {
		t.Fatal("query params were not URL-encoded")
	}

	if !containsStr(result, "hello+world") && !containsStr(result, "hello%20world") {
		t.Fatalf("expected URL-encoded space in query param, got: %s", result)
	}

	if !containsStr(result, "%26") {
		t.Fatalf("expected URL-encoded ampersand in filter param, got: %s", result)
	}
}

func TestBuildQueryURL_EmptyParams(t *testing.T) {
	cfg := defaultTestConfig()
	tr := New(cfg)

	result := tr.BuildQueryURL("/v1/agents", map[string]string{})
	if result != "/v1/agents" {
		t.Fatalf("expected bare path for empty params, got: %s", result)
	}
}

func TestBuildQueryURL_SingleParam(t *testing.T) {
	cfg := defaultTestConfig()
	tr := New(cfg)

	result := tr.BuildQueryURL("/v1/agents", map[string]string{"limit": "50"})
	if result != "/v1/agents?limit=50" {
		t.Fatalf("expected /v1/agents?limit=50, got: %s", result)
	}
}

func TestDoRequest_SetsIdempotencyKeyForPOST(t *testing.T) {
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	cfg := defaultTestConfig()
	cfg.BaseURL = server.URL
	cfg.IdempotencyKeyPrefix = "test-"
	tr := New(cfg)

	var result map[string]string
	err := tr.Do(context.Background(), "POST", "/v1/test", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	idemKey := receivedHeaders.Get("Idempotency-Key")
	if idemKey == "" {
		t.Fatal("expected Idempotency-Key header for POST request")
	}
	if len(idemKey) < 5 || idemKey[:5] != "test-" {
		t.Fatalf("expected Idempotency-Key to start with prefix 'test-', got: %s", idemKey)
	}
}

func TestDoRequest_NoIdempotencyKeyForGET(t *testing.T) {
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	cfg := defaultTestConfig()
	cfg.BaseURL = server.URL
	cfg.IdempotencyKeyPrefix = "test-"
	tr := New(cfg)

	var result map[string]string
	err := tr.Do(context.Background(), "GET", "/v1/test", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedHeaders.Get("Idempotency-Key") != "" {
		t.Fatal("GET requests should not have Idempotency-Key header")
	}
}

func TestDoWithRetry_RetriesOnServiceUnavailable(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"code": "SERVICE_UNAVAILABLE", "message": "retry"})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	cfg := defaultTestConfig()
	cfg.BaseURL = server.URL
	cfg.MaxRetries = 3
	cfg.RetryBaseDelay = 10 * time.Millisecond
	cfg.RetryMaxDelay = 50 * time.Millisecond
	tr := New(cfg)

	var result map[string]string
	err := tr.Do(context.Background(), "GET", "/v1/test", nil, &result)
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got: %d", attempts)
	}
}

func TestDoWithRetry_ReusesIdempotencyKeyForPOSTRetries(t *testing.T) {
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys = append(keys, r.Header.Get("Idempotency-Key"))
		if len(keys) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"code": "SERVICE_UNAVAILABLE", "message": "retry"})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	cfg := defaultTestConfig()
	cfg.BaseURL = server.URL
	cfg.MaxRetries = 3
	cfg.RetryBaseDelay = time.Millisecond
	cfg.RetryMaxDelay = 5 * time.Millisecond
	cfg.IdempotencyKeyPrefix = "test-"
	tr := New(cfg)

	var result map[string]string
	if err := tr.Do(context.Background(), http.MethodPost, "/v1/test", map[string]string{"x": "y"}, &result); err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if len(keys) != 3 {
		t.Fatalf("expected 3 attempts, got %d", len(keys))
	}
	if keys[0] == "" {
		t.Fatal("expected idempotency key")
	}
	for _, key := range keys[1:] {
		if key != keys[0] {
			t.Fatalf("expected stable idempotency key, got %#v", keys)
		}
	}
	if !strings.HasPrefix(keys[0], "test-") {
		t.Fatalf("expected prefixed key, got %q", keys[0])
	}
}

func TestDoStream_RetriesOnServiceUnavailable(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"code": "SERVICE_UNAVAILABLE", "message": "retry"})
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	cfg := defaultTestConfig()
	cfg.BaseURL = server.URL
	cfg.MaxRetries = 3
	cfg.RetryBaseDelay = 10 * time.Millisecond
	cfg.RetryMaxDelay = 50 * time.Millisecond
	tr := New(cfg)

	resp, err := tr.DoStream(context.Background(), "GET", "/v1/test", nil, nil)
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	resp.Body.Close()
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got: %d", attempts)
	}
}

func TestDoWithRetry_RespectsContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"code": "SERVICE_UNAVAILABLE", "message": "down"})
	}))
	defer server.Close()

	cfg := defaultTestConfig()
	cfg.BaseURL = server.URL
	cfg.MaxRetries = 10
	cfg.RetryBaseDelay = 100 * time.Millisecond
	tr := New(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	var result map[string]string
	err := tr.Do(ctx, "GET", "/v1/test", nil, &result)
	if err == nil {
		t.Fatal("expected error due to context cancellation")
	}
}

func TestDoRequest_ParsesRateLimitHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "100")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"code": "AUTH_RATE_LIMITED", "message": "rate limited"})
	}))
	defer server.Close()

	cfg := defaultTestConfig()
	cfg.BaseURL = server.URL
	cfg.MaxRetries = 0
	tr := New(cfg)

	var result map[string]string
	err := tr.Do(context.Background(), "GET", "/v1/test", nil, &result)
	if err == nil {
		t.Fatal("expected rate limit error")
	}

	rateErr, ok := err.(*apierr.RateLimitError)
	if !ok {
		t.Fatalf("expected *apierr.RateLimitError, got %T", err)
	}
	if rateErr.Limit != 100 {
		t.Fatalf("expected limit 100, got %d", rateErr.Limit)
	}
	if rateErr.Remaining != 0 {
		t.Fatalf("expected remaining 0, got %d", rateErr.Remaining)
	}
	if rateErr.RetryAfter != 5*time.Second {
		t.Fatalf("expected retry after 5s, got %v", rateErr.RetryAfter)
	}
}

func TestBackoffDelay(t *testing.T) {
	cfg := defaultTestConfig()
	cfg.RetryBaseDelay = 500 * time.Millisecond
	cfg.RetryMaxDelay = 5 * time.Second
	tr := New(cfg)

	d1 := tr.BackoffDelay(1)
	if d1 < 250*time.Millisecond || d1 > 500*time.Millisecond {
		t.Fatalf("expected 250ms-500ms for attempt 1 (with jitter), got %v", d1)
	}

	d2 := tr.BackoffDelay(2)
	if d2 < 500*time.Millisecond || d2 > 1*time.Second {
		t.Fatalf("expected 500ms-1s for attempt 2 (with jitter), got %v", d2)
	}

	d10 := tr.BackoffDelay(10)
	if d10 > 5*time.Second {
		t.Fatalf("expected max delay cap at 5s, got %v", d10)
	}
}

func TestGenerateRequestID(t *testing.T) {
	id := GenerateRequestID()
	if len(id) != 32 {
		t.Fatalf("expected 32 hex chars, got %d: %s", len(id), id)
	}

	id2 := GenerateRequestID()
	if id == id2 {
		t.Fatal("expected unique request IDs")
	}
}

func TestDoRequest_UsesBearerScheme(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	cfg := defaultTestConfig()
	cfg.BaseURL = server.URL
	cfg.AccessToken = "test-access-token-xyz"
	cfg.TenantID = "t-1"
	tr := New(cfg)

	var result map[string]string
	err := tr.Do(context.Background(), "GET", "/v1/agents", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedAuth != "Bearer test-access-token-xyz" {
		t.Fatalf("expected 'Bearer test-access-token-xyz', got '%s'", receivedAuth)
	}
}

func TestDoRequest_DPoPProofIncludesATH(t *testing.T) {
	var receivedDPoP string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedDPoP = r.Header.Get("DPoP")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	key, err := dpop.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate DPoP key: %v", err)
	}

	cfg := defaultTestConfig()
	cfg.BaseURL = server.URL
	cfg.AccessToken = "my-access-token"
	cfg.TenantID = "t-1"
	cfg.DPoPPrivateKey = key
	tr := New(cfg)

	var result map[string]string
	err = tr.Do(context.Background(), "GET", "/v1/agents", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedDPoP == "" {
		t.Fatal("expected DPoP proof header to be set")
	}

	parts := strings.Split(receivedDPoP, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 JWT parts, got %d", len(parts))
	}

	payload := decodeJWTPayload(t, parts[1])
	if _, ok := payload["ath"]; !ok {
		t.Fatal("expected 'ath' claim in DPoP proof when access token is present")
	}
}

func TestDoRequest_NoDPoPWithoutKey(t *testing.T) {
	var receivedDPoP string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedDPoP = r.Header.Get("DPoP")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	cfg := defaultTestConfig()
	cfg.BaseURL = server.URL
	cfg.AccessToken = "some-token"
	cfg.TenantID = "t-1"
	tr := New(cfg)

	var result map[string]string
	err := tr.Do(context.Background(), "GET", "/v1/agents", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedDPoP != "" {
		t.Fatal("expected no DPoP header when DPoPPrivateKey is nil")
	}
}

func TestAttemptTokenRefresh_SendsTenantRequestIDAndDPoP(t *testing.T) {
	var refreshTenant string
	var refreshRequestID string
	var refreshDPoP string
	var refreshBody map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshTenant = r.Header.Get("X-Tenant-ID")
		refreshRequestID = r.Header.Get("X-Request-ID")
		refreshDPoP = r.Header.Get("DPoP")
		if err := json.NewDecoder(r.Body).Decode(&refreshBody); err != nil {
			t.Fatalf("decode refresh body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"access_token":  "fresh-token",
			"refresh_token": "next-refresh",
		})
	}))
	defer server.Close()

	key, err := dpop.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate DPoP key: %v", err)
	}

	cfg := defaultTestConfig()
	cfg.BaseURL = server.URL
	cfg.TenantID = "tenant-1"
	cfg.RefreshToken = "refresh-token"
	cfg.TokenEndpoint = server.URL + "/api/v1/auth/tokens/refresh"
	cfg.DPoPPrivateKey = key
	tr := New(cfg)

	if err := tr.attemptTokenRefresh(context.Background()); err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if refreshTenant != "tenant-1" {
		t.Fatalf("expected tenant header, got %q", refreshTenant)
	}
	if refreshRequestID == "" {
		t.Fatal("expected request ID header")
	}
	if refreshDPoP == "" {
		t.Fatal("expected DPoP header")
	}
	if refreshBody["refresh_token"] != "refresh-token" {
		t.Fatalf("unexpected refresh body: %#v", refreshBody)
	}
	if tr.getToken() != "fresh-token" {
		t.Fatalf("expected refreshed token, got %q", tr.getToken())
	}
}

func TestAttemptTokenRefresh_MapsOAuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-ID", "req-refresh")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_grant",
			"error_description": "refresh token expired",
		})
	}))
	defer server.Close()

	cfg := defaultTestConfig()
	cfg.BaseURL = server.URL
	cfg.RefreshToken = "refresh-token"
	cfg.TokenEndpoint = server.URL + "/api/v1/auth/tokens/refresh"
	tr := New(cfg)

	err := tr.attemptTokenRefresh(context.Background())
	if err == nil {
		t.Fatal("expected refresh error")
	}
	apiErr, ok := err.(*apierr.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", apiErr.HTTPStatus)
	}
	if apiErr.Message != "invalid_grant: refresh token expired" {
		t.Fatalf("unexpected message: %s", apiErr.Message)
	}
	if apiErr.RequestID != "req-refresh" {
		t.Fatalf("expected request ID, got %q", apiErr.RequestID)
	}
}

func TestDoRequest_NormalizesAPIBasePath(t *testing.T) {
	var gotPaths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPaths = append(gotPaths, r.URL.Path)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	cfg := defaultTestConfig()
	cfg.BaseURL = server.URL + "/api/v1"
	tr := New(cfg)

	var result map[string]string
	for _, path := range []string{"/v1/agents", "/agents", "/api/v1/agents", "/health"} {
		if err := tr.Do(context.Background(), "GET", path, nil, &result); err != nil {
			t.Fatalf("Do(%s) error: %v", path, err)
		}
	}

	want := []string{"/api/v1/agents", "/api/v1/agents", "/api/v1/agents", "/health"}
	if strings.Join(gotPaths, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected paths: got %v want %v", gotPaths, want)
	}
}

func TestDoRaw_ReturnsRawResponseAndAttestationHeader(t *testing.T) {
	var gotAttestation string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAttestation = r.Header.Get("X-Attestation-Token")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"raw":true}`))
	}))
	defer server.Close()

	cfg := defaultTestConfig()
	cfg.BaseURL = server.URL
	cfg.AttestationToken = "attestation-token"
	tr := New(cfg)

	raw, err := tr.DoRaw(context.Background(), "GET", "/agents", nil, nil)
	if err != nil {
		t.Fatalf("DoRaw error: %v", err)
	}
	if string(raw) != `{"raw":true}` {
		t.Fatalf("unexpected raw response: %s", string(raw))
	}
	if gotAttestation != "attestation-token" {
		t.Fatalf("expected attestation header, got %q", gotAttestation)
	}
}

func TestDoRequest_ResponseSizeLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strings.Repeat("a", 32)))
	}))
	defer server.Close()

	cfg := defaultTestConfig()
	cfg.BaseURL = server.URL
	cfg.MaxResponseBytes = 8
	tr := New(cfg)

	var result map[string]string
	err := tr.Do(context.Background(), "GET", "/v1/test", nil, &result)
	if err == nil {
		t.Fatal("expected response size error")
	}
	apiErr, ok := err.(*apierr.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Code != apierr.ErrCodeResponseTooLarge {
		t.Fatalf("expected RESPONSE_TOO_LARGE, got %s", apiErr.Code)
	}
}

func TestDoRequest_RetriesWithDPoPNonce(t *testing.T) {
	attempts := 0
	var retryProof string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("DPoP-Nonce", "server-nonce")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "use_dpop_nonce"})
			return
		}
		retryProof = r.Header.Get("DPoP")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	key, err := dpop.GenerateKey()
	if err != nil {
		t.Fatalf("generate DPoP key: %v", err)
	}
	cfg := defaultTestConfig()
	cfg.BaseURL = server.URL
	cfg.AccessToken = "access-token"
	cfg.DPoPPrivateKey = key
	cfg.MaxRetries = 0
	tr := New(cfg)

	var result map[string]string
	if err := tr.Do(context.Background(), "GET", "/agents", nil, &result); err != nil {
		t.Fatalf("Do error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected nonce retry, got %d attempts", attempts)
	}
	parts := strings.Split(retryProof, ".")
	if len(parts) != 3 {
		t.Fatalf("expected retry DPoP proof, got %q", retryProof)
	}
	payload := decodeJWTPayload(t, parts[1])
	if payload["nonce"] != "server-nonce" {
		t.Fatalf("expected retry proof nonce, got %+v", payload)
	}
}

func containsStr(s, substr string) bool {
	return strings.Contains(s, substr)
}

func decodeJWTPayload(t *testing.T, encoded string) map[string]interface{} {
	t.Helper()
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("failed to decode JWT payload: %v", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("failed to unmarshal JWT payload: %v", err)
	}
	return payload
}
