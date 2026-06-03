package credential

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

func (s *Service) Anchor(ctx context.Context, req *AnchorRequest) (*AnchorResponse, error) {
	if req.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if req.SubjectID == "" {
		return nil, fmt.Errorf("subject_id is required")
	}
	if req.CredentialID == "" {
		return nil, fmt.Errorf("credential_id is required")
	}
	if req.PublicKey == "" {
		return nil, fmt.Errorf("public_key is required")
	}

	var resp AnchorResponse
	err := s.transport.Do(ctx, "POST", "/v1/credentials/anchor", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Status(ctx context.Context, anchorID string) (*StatusResponse, error) {
	if anchorID == "" {
		return nil, fmt.Errorf("anchor_id is required")
	}

	path := s.transport.BuildQueryURL("/v1/credentials/status", map[string]string{
		"anchor_id": anchorID,
	})

	var resp StatusResponse
	err := s.transport.Do(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Revoke(ctx context.Context, req *RevokeRequest) (*RevokeResponse, error) {
	if req.AnchorID == "" {
		return nil, fmt.Errorf("anchor_id is required")
	}
	if req.Reason == "" {
		return nil, fmt.Errorf("reason is required")
	}
	if req.Actor == "" {
		return nil, fmt.Errorf("actor is required")
	}

	var resp RevokeResponse
	err := s.transport.Do(ctx, "POST", "/v1/credentials/revoke", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Verify(ctx context.Context, req *VerifyRequest) (*VerifyResponse, error) {
	if req.AnchorID == "" {
		return nil, fmt.Errorf("anchor_id is required")
	}
	if req.MerkleProof == nil {
		return nil, fmt.Errorf("merkle_proof is required")
	}

	var resp VerifyResponse
	err := s.transport.Do(ctx, "POST", "/v1/credentials/verify", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) List(ctx context.Context, subjectID string, cursor string, limit int) (*ListAnchorsResponse, error) {
	if subjectID == "" {
		return nil, fmt.Errorf("subject_id is required")
	}

	params := map[string]string{
		"subject_id": subjectID,
	}
	if cursor != "" {
		params["cursor"] = cursor
	}
	if limit > 0 {
		if limit > 200 {
			limit = 200
		}
		params["limit"] = fmt.Sprintf("%d", limit)
	}

	path := s.transport.BuildQueryURL("/v1/credentials", params)

	var resp ListAnchorsResponse
	err := s.transport.Do(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
