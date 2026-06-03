package credential

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPTransport_InjectsAuthHeader(t *testing.T) {
	store := NewMemStore()
	defer store.ZeroizeAll()

	store.Store("access_token", []byte("tok-abc123"))

	var capturedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	transport := NewHTTPTransport(http.DefaultTransport, store, "access_token")
	client := &http.Client{Transport: transport}

	resp, err := client.Get(srv.URL + "/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if capturedAuth != "Bearer tok-abc123" {
		t.Fatalf("expected 'Bearer tok-abc123', got %q", capturedAuth)
	}
}

func TestHTTPTransport_MissingCredential(t *testing.T) {
	store := NewMemStore()

	transport := NewHTTPTransport(http.DefaultTransport, store, "nonexistent")
	client := &http.Client{Transport: transport}

	_, err := client.Get("http://localhost:1/test")
	if err == nil {
		t.Fatal("expected error for missing credential")
	}
}

func TestHTTPTransport_CustomTokenType(t *testing.T) {
	store := NewMemStore()
	defer store.ZeroizeAll()

	store.Store("dpop_token", []byte("dpop-tok"))

	var capturedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	transport := NewHTTPTransport(http.DefaultTransport, store, "dpop_token")
	transport.TokenType = "DPoP"
	client := &http.Client{Transport: transport}

	resp, err := client.Get(srv.URL + "/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if capturedAuth != "DPoP dpop-tok" {
		t.Fatalf("expected 'DPoP dpop-tok', got %q", capturedAuth)
	}
}
