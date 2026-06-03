package token

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

type mockTransport struct {
	method    string
	path      string
	body      interface{}
	result    interface{}
	err       error
	callCount int
}

func (m *mockTransport) Do(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	m.method = method
	m.path = path
	m.body = body
	m.callCount++
	if m.err != nil {
		return m.err
	}
	if m.result != nil && result != nil {
		data, _ := json.Marshal(m.result)
		json.Unmarshal(data, result)
	}
	return nil
}

func (m *mockTransport) DoWithHeaders(ctx context.Context, method, path string, body interface{}, result interface{}, headers map[string]string) error {
	return m.Do(ctx, method, path, body, result)
}

func (m *mockTransport) BuildQueryURL(path string, params map[string]string) string {
	if len(params) == 0 {
		return path
	}
	return path + "?mocked=true"
}

func TestIntrospect_RequiresToken(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.Introspect(context.Background(), &IntrospectRequest{})
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestIntrospect_Success(t *testing.T) {
	mt := &mockTransport{
		result: &IntrospectResponse{
			Active: true,
			Sub:    "agent-123",
			Scope:  "agents:read tools:execute",
		},
	}
	svc := NewService(mt)

	resp, err := svc.Introspect(context.Background(), &IntrospectRequest{Token: "access-token-xyz"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Active {
		t.Fatal("expected active=true")
	}
	if resp.Sub != "agent-123" {
		t.Fatalf("expected sub agent-123, got %s", resp.Sub)
	}
	if mt.path != "/v1/auth/introspect" {
		t.Fatalf("expected /v1/auth/introspect, got %s", mt.path)
	}
}

func TestRevoke_RequiresToken(t *testing.T) {
	svc := NewService(&mockTransport{})
	err := svc.Revoke(context.Background(), &RevokeRequest{})
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestRevoke_Unsupported(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	err := svc.Revoke(context.Background(), &RevokeRequest{Token: "token-to-revoke"})
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported revoke should not call transport")
	}
}

func TestExchange_RequiresSubjectToken(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.Exchange(context.Background(), &ExchangeRequest{
		SubjectTokenType: "urn:ietf:params:oauth:token-type:jwt",
	})
	if err == nil {
		t.Fatal("expected error for missing subject_token")
	}
}

func TestExchange_RequiresSubjectTokenType(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.Exchange(context.Background(), &ExchangeRequest{
		SubjectToken: "jwt-token",
	})
	if err == nil {
		t.Fatal("expected error for missing subject_token_type")
	}
}

func TestExchange_DefaultsGrantType(t *testing.T) {
	mt := &mockTransport{
		result: &ExchangeResponse{AccessToken: "new-token", TokenType: "DPoP"},
	}
	svc := NewService(mt)

	resp, err := svc.Exchange(context.Background(), &ExchangeRequest{
		SubjectToken:     "original-jwt",
		SubjectTokenType: "urn:ietf:params:oauth:token-type:jwt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AccessToken != "new-token" {
		t.Fatalf("expected new-token, got %s", resp.AccessToken)
	}

	bodyData, _ := json.Marshal(mt.body)
	var sent ExchangeRequest
	json.Unmarshal(bodyData, &sent)
	if sent.GrantType != "urn:ietf:params:oauth:grant-type:token-exchange" {
		t.Fatalf("expected default grant_type, got %s", sent.GrantType)
	}
	if mt.path != "/auth/token/exchange" {
		t.Fatalf("expected /auth/token/exchange, got %s", mt.path)
	}
}

func TestExchange_PreservesCustomGrantType(t *testing.T) {
	mt := &mockTransport{
		result: &ExchangeResponse{AccessToken: "x"},
	}
	svc := NewService(mt)

	_, _ = svc.Exchange(context.Background(), &ExchangeRequest{
		SubjectToken:     "jwt",
		SubjectTokenType: "urn:ietf:params:oauth:token-type:jwt",
		GrantType:        "custom_grant",
	})

	bodyData, _ := json.Marshal(mt.body)
	var sent ExchangeRequest
	json.Unmarshal(bodyData, &sent)
	if sent.GrantType != "custom_grant" {
		t.Fatalf("expected custom_grant to be preserved, got %s", sent.GrantType)
	}
}

func TestRefresh_RequiresRefreshToken(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.Refresh(context.Background(), "", "")
	if err == nil {
		t.Fatal("expected error for empty refresh_token")
	}
}

func TestRefresh_Success(t *testing.T) {
	mt := &mockTransport{
		result: &RefreshResponse{
			AccessToken: "refreshed-token",
			ExpiresIn:   3600,
		},
	}
	svc := NewService(mt)

	resp, err := svc.Refresh(context.Background(), "my-refresh-token", "agents:read")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AccessToken != "refreshed-token" {
		t.Fatalf("expected refreshed-token, got %s", resp.AccessToken)
	}

	bodyData, _ := json.Marshal(mt.body)
	var sent RefreshRequest
	json.Unmarshal(bodyData, &sent)
	if sent.GrantType != "refresh_token" {
		t.Fatalf("expected grant_type=refresh_token, got %s", sent.GrantType)
	}
	if sent.Scope != "agents:read" {
		t.Fatalf("expected scope=agents:read, got %s", sent.Scope)
	}
	if mt.path != "/v1/auth/tokens/refresh" {
		t.Fatalf("expected /v1/auth/tokens/refresh, got %s", mt.path)
	}
}

func TestIntrospect_TransportError(t *testing.T) {
	mt := &mockTransport{err: fmt.Errorf("network error")}
	svc := NewService(mt)
	_, err := svc.Introspect(context.Background(), &IntrospectRequest{Token: "tok"})
	if err == nil {
		t.Fatal("expected transport error")
	}
}

func TestClientCredentials_Unsupported(t *testing.T) {
	mt := &mockTransport{
		result: &ClientCredentialsResponse{
			AccessToken: "cc-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		},
	}
	svc := NewService(mt)

	_, err := svc.ClientCredentials(context.Background(), "client-1", "secret", "read", "https://api.example.com")
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported client credentials should not call transport")
	}
}

func TestClientCredentials_MissingClientID(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.ClientCredentials(context.Background(), "", "secret", "", "")
	if err == nil {
		t.Fatal("expected error for missing client_id")
	}
}

func TestClientCredentials_MissingClientSecret(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.ClientCredentials(context.Background(), "client-1", "", "", "")
	if err == nil {
		t.Fatal("expected error for missing client_secret")
	}
}

func TestJWKS_Success(t *testing.T) {
	mt := &mockTransport{
		result: &JWKSResponse{
			Keys: []JWK{{Kty: "EC", Crv: "P-256", Kid: "key-1"}},
		},
	}
	svc := NewService(mt)

	resp, err := svc.JWKS(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(resp.Keys))
	}
	if mt.path != "/.well-known/jwks.json" {
		t.Fatalf("expected jwks path, got %s", mt.path)
	}
}

func TestJWKS_TransportError(t *testing.T) {
	mt := &mockTransport{err: fmt.Errorf("timeout")}
	svc := NewService(mt)
	_, err := svc.JWKS(context.Background())
	if err == nil {
		t.Fatal("expected transport error")
	}
}

func TestOIDCDiscovery_Success(t *testing.T) {
	mt := &mockTransport{
		result: &OIDCDiscovery{
			Issuer:                "https://auth.example.com",
			AuthorizationEndpoint: "https://auth.example.com/authorize",
			TokenEndpoint:         "https://auth.example.com/token",
		},
	}
	svc := NewService(mt)

	resp, err := svc.OIDCDiscovery(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Issuer != "https://auth.example.com" {
		t.Fatalf("expected issuer, got %s", resp.Issuer)
	}
	if mt.path != "/.well-known/openid-configuration" {
		t.Fatalf("expected OIDC path, got %s", mt.path)
	}
}

func TestOIDCDiscovery_TransportError(t *testing.T) {
	mt := &mockTransport{err: fmt.Errorf("timeout")}
	svc := NewService(mt)
	_, err := svc.OIDCDiscovery(context.Background())
	if err == nil {
		t.Fatal("expected transport error")
	}
}

func TestMint_Unsupported(t *testing.T) {
	mt := &mockTransport{
		result: &MintResponse{
			AccessToken: "minted-jwt",
			ExpiresIn:   3600,
		},
	}
	svc := NewService(mt)

	_, err := svc.Mint(context.Background(), &MintRequest{
		SubjectID:   "agent-1",
		SubjectType: "nhi",
	})
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported mint should not call transport")
	}
}

func TestMint_MissingSubjectID(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.Mint(context.Background(), &MintRequest{SubjectType: "nhi"})
	if err == nil {
		t.Fatal("expected error for missing subject_id")
	}
}

func TestMint_MissingSubjectType(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.Mint(context.Background(), &MintRequest{SubjectID: "agent-1"})
	if err == nil {
		t.Fatal("expected error for missing subject_type")
	}
}

func TestMint_TransportError(t *testing.T) {
	mt := &mockTransport{err: fmt.Errorf("server error")}
	svc := NewService(mt)
	_, err := svc.Mint(context.Background(), &MintRequest{SubjectID: "a1", SubjectType: "nhi"})
	if err == nil {
		t.Fatal("expected transport error")
	}
}
