package zinetic

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewRawClient_AllowsMissingTenantForPublicRoutes(t *testing.T) {
	var gotPath, gotTenant string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotTenant = r.Header.Get("X-Tenant-ID")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	client, err := NewRawClient(
		WithBaseURL(srv.URL),
		WithTracing(false),
	)
	if err != nil {
		t.Fatalf("NewRawClient: %v", err)
	}
	if _, err := client.DoRaw(context.Background(), http.MethodGet, "/health", nil, nil); err != nil {
		t.Fatalf("DoRaw: %v", err)
	}
	if gotPath != "/health" {
		t.Fatalf("expected /health, got %s", gotPath)
	}
	if gotTenant != "" {
		t.Fatalf("public route should not send tenant header, got %q", gotTenant)
	}
}

func rawClientTestJWT(t *testing.T, exp time.Time) string {
	t.Helper()
	header, err := json.Marshal(map[string]string{"alg": "none", "typ": "JWT"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]int64{"exp": exp.Unix()})
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload) + "."
}

func TestRawClient_RequiresTenantForScopedRoutes(t *testing.T) {
	client, err := NewRawClient(
		WithBaseURL("https://api.zinetic.io"),
		WithTracing(false),
	)
	if err != nil {
		t.Fatalf("NewRawClient: %v", err)
	}
	_, err = client.DoRaw(context.Background(), http.MethodGet, "/agents", nil, nil)
	if err == nil {
		t.Fatal("expected tenant validation error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if !strings.Contains(apiErr.Message, "tenant ID is required") {
		t.Fatalf("unexpected message: %s", apiErr.Message)
	}
}

func TestRawClient_SendsTenantForScopedRoutes(t *testing.T) {
	var gotPath, gotTenant, gotSource string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotTenant = r.Header.Get("X-Tenant-ID")
		gotSource = r.Header.Get("X-Source")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	client, err := NewRawClient(
		WithBaseURL(srv.URL),
		WithTenantID("tenant-1"),
		WithTracing(false),
	)
	if err != nil {
		t.Fatalf("NewRawClient: %v", err)
	}
	if _, err := client.DoRaw(context.Background(), http.MethodGet, "/agents", nil, map[string]string{"X-Source": "cli"}); err != nil {
		t.Fatalf("DoRaw: %v", err)
	}
	if gotPath != "/api/v1/agents" {
		t.Fatalf("expected /api/v1/agents, got %s", gotPath)
	}
	if gotTenant != "tenant-1" {
		t.Fatalf("expected tenant header, got %q", gotTenant)
	}
	if gotSource != "cli" {
		t.Fatalf("expected X-Source=cli, got %q", gotSource)
	}
}

func TestRawClient_DPoPNonceRetry(t *testing.T) {
	var attempts int
	var proofs []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		proofs = append(proofs, r.Header.Get("DPoP"))
		if attempts == 1 {
			w.Header().Set("DPoP-Nonce", "nonce-1")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"use_dpop_nonce","message":"nonce required"}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	client, err := NewRawClient(
		WithBaseURL(srv.URL),
		WithTenantID("tenant-1"),
		WithDPoPKey(key),
		WithAccessToken("access-token"),
		WithTracing(false),
	)
	if err != nil {
		t.Fatalf("NewRawClient: %v", err)
	}
	if _, err := client.DoRaw(context.Background(), http.MethodGet, "/agents", nil, nil); err != nil {
		t.Fatalf("DoRaw: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected nonce retry, got %d attempts", attempts)
	}
	if proofs[0] == "" || proofs[1] == "" || proofs[0] == proofs[1] {
		t.Fatalf("expected distinct DPoP proofs, got %#v", proofs)
	}
}

func TestRawClient_BackendJSONRefresh(t *testing.T) {
	expired := rawClientTestJWT(t, time.Now().Add(-time.Hour))
	var refreshContentType string
	var refreshBody map[string]string
	var gotAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/agents":
			if r.Header.Get("Authorization") == "Bearer "+expired {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"code":"AUTH_TOKEN_EXPIRED","message":"expired"}`))
				return
			}
			gotAuth = r.Header.Get("Authorization")
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/api/v1/auth/tokens/refresh":
			refreshContentType = r.Header.Get("Content-Type")
			if err := json.NewDecoder(r.Body).Decode(&refreshBody); err != nil {
				t.Fatalf("decode refresh body: %v", err)
			}
			_, _ = w.Write([]byte(`{"access_token":"fresh-token","refresh_token":"next-refresh"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client, err := NewRawClient(
		WithBaseURL(srv.URL),
		WithTenantID("tenant-1"),
		WithAccessToken(expired),
		WithRefreshToken("refresh-token"),
		WithTokenEndpoint(srv.URL+"/api/v1/auth/tokens/refresh"),
		WithTracing(false),
	)
	if err != nil {
		t.Fatalf("NewRawClient: %v", err)
	}
	if _, err := client.DoRaw(context.Background(), http.MethodGet, "/agents", nil, nil); err != nil {
		t.Fatalf("DoRaw: %v", err)
	}
	if refreshContentType != "application/json" {
		t.Fatalf("expected JSON refresh, got %s", refreshContentType)
	}
	if refreshBody["refresh_token"] != "refresh-token" {
		t.Fatalf("unexpected refresh body: %#v", refreshBody)
	}
	if gotAuth != "Bearer fresh-token" {
		t.Fatalf("expected refreshed token, got %q", gotAuth)
	}
}
