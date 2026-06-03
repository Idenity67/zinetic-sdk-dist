package token

import (
	"context"
	"fmt"
)

type Transport interface {
	Do(ctx context.Context, method, path string, body interface{}, result interface{}) error
	DoWithHeaders(ctx context.Context, method, path string, body interface{}, result interface{}, headers map[string]string) error
	BuildQueryURL(path string, params map[string]string) string
}

type Service struct {
	transport Transport
}

func NewService(t Transport) *Service {
	return &Service{transport: t}
}

func (s *Service) Introspect(ctx context.Context, req *IntrospectRequest) (*IntrospectResponse, error) {
	if req.Token == "" {
		return nil, fmt.Errorf("token is required")
	}

	var resp IntrospectResponse
	err := s.transport.Do(ctx, "POST", "/v1/auth/introspect", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Revoke(ctx context.Context, req *RevokeRequest) error {
	if req.Token == "" {
		return fmt.Errorf("token is required")
	}

	return fmt.Errorf("token revoke is not supported by the current backend OpenAPI contract")
}

func (s *Service) Exchange(ctx context.Context, req *ExchangeRequest) (*ExchangeResponse, error) {
	if req.SubjectToken == "" {
		return nil, fmt.Errorf("subject_token is required")
	}
	if req.SubjectTokenType == "" {
		return nil, fmt.Errorf("subject_token_type is required")
	}

	if req.GrantType == "" {
		req.GrantType = "urn:ietf:params:oauth:grant-type:token-exchange"
	}

	var resp ExchangeResponse
	err := s.transport.Do(ctx, "POST", "/auth/token/exchange", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string, scope string) (*RefreshResponse, error) {
	if refreshToken == "" {
		return nil, fmt.Errorf("refresh_token is required")
	}

	req := &RefreshRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
		Scope:        scope,
	}

	var resp RefreshResponse
	err := s.transport.Do(ctx, "POST", "/v1/auth/tokens/refresh", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) ClientCredentials(ctx context.Context, clientID, clientSecret, scope, resource string) (*ClientCredentialsResponse, error) {
	if clientID == "" {
		return nil, fmt.Errorf("client_id is required")
	}
	if clientSecret == "" {
		return nil, fmt.Errorf("client_secret is required")
	}

	return nil, fmt.Errorf("client credentials token grant is not supported by the current backend OpenAPI contract")
}

func (s *Service) JWKS(ctx context.Context) (*JWKSResponse, error) {
	var resp JWKSResponse
	err := s.transport.Do(ctx, "GET", "/.well-known/jwks.json", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) OIDCDiscovery(ctx context.Context) (*OIDCDiscovery, error) {
	var resp OIDCDiscovery
	err := s.transport.Do(ctx, "GET", "/.well-known/openid-configuration", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Mint(ctx context.Context, req *MintRequest) (*MintResponse, error) {
	if req.SubjectID == "" {
		return nil, fmt.Errorf("subject_id is required")
	}
	if req.SubjectType == "" {
		return nil, fmt.Errorf("subject_type is required")
	}

	return nil, fmt.Errorf("token mint is not supported by the current backend OpenAPI contract")
}
