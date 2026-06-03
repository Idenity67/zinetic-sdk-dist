package blockchain

import (
	"context"
	"fmt"
)

func (s *Service) RegisterIssuer(ctx context.Context, req *RegisterIssuerRequest) (*TrustedIssuer, error) {
	var resp TrustedIssuer
	if err := s.transport.Do(ctx, "POST", "/blockchain/trust/issuers", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) UpdateIssuer(ctx context.Context, did string, req *UpdateIssuerRequest) (*TrustedIssuer, error) {
	var resp TrustedIssuer
	if err := s.transport.Do(ctx, "PUT", fmt.Sprintf("/blockchain/trust/issuers/%s", did), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) RevokeIssuer(ctx context.Context, did string, req *RevokeIssuerRequest) error {
	return s.transport.Do(ctx, "DELETE", fmt.Sprintf("/blockchain/trust/issuers/%s", did), req, nil)
}

func (s *Service) GetIssuer(ctx context.Context, did string) (*TrustedIssuer, error) {
	var resp TrustedIssuer
	if err := s.transport.Do(ctx, "GET", fmt.Sprintf("/blockchain/trust/issuers/%s", did), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) IsIssuerTrusted(ctx context.Context, did, credentialType string) (*IssuerTrustResponse, error) {
	path := s.transport.BuildQueryURL(fmt.Sprintf("/blockchain/trust/issuers/%s/trusted", did), map[string]string{
		"credential_type": credentialType,
	})
	var resp IssuerTrustResponse
	if err := s.transport.Do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) RegisterVerifier(ctx context.Context, req *RegisterVerifierRequest) (*TrustedVerifier, error) {
	var resp TrustedVerifier
	if err := s.transport.Do(ctx, "POST", "/blockchain/trust/verifiers", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetVerifier(ctx context.Context, did string) (*TrustedVerifier, error) {
	var resp TrustedVerifier
	if err := s.transport.Do(ctx, "GET", fmt.Sprintf("/blockchain/trust/verifiers/%s", did), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) RegisterSchema(ctx context.Context, req *RegisterSchemaRequest) (*TrustedSchema, error) {
	var resp TrustedSchema
	if err := s.transport.Do(ctx, "POST", "/blockchain/trust/schemas", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetSchema(ctx context.Context, schemaID string) (*TrustedSchema, error) {
	var resp TrustedSchema
	if err := s.transport.Do(ctx, "GET", fmt.Sprintf("/blockchain/trust/schemas/%s", schemaID), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) EvaluateTrust(ctx context.Context, issuerDID, credentialType string) (*EvaluateTrustResponse, error) {
	path := s.transport.BuildQueryURL("/blockchain/trust/evaluate", map[string]string{
		"issuer_did":      issuerDID,
		"credential_type": credentialType,
	})
	var resp EvaluateTrustResponse
	if err := s.transport.Do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetTrustStats(ctx context.Context) (*TrustStats, error) {
	var resp TrustStats
	if err := s.transport.Do(ctx, "GET", "/blockchain/trust/stats", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
