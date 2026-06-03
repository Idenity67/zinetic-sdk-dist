package did

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

func (s *Service) Create(ctx context.Context, req *CreateRequest) (*DID, error) {
	var resp DID
	if err := s.transport.Do(ctx, "POST", "/v1/dids/create", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Resolve(ctx context.Context, didID string) (*DID, error) {
	var resp DID
	if err := s.transport.Do(ctx, "GET", fmt.Sprintf("/v1/dids/%s", didID), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) List(ctx context.Context, userID string, cursor string, limit int) (*ListResponse, error) {
	params := map[string]string{"user_id": userID}
	if cursor != "" {
		params["cursor"] = cursor
	}
	if limit > 0 {
		if limit > 200 {
			limit = 200
		}
		params["limit"] = fmt.Sprintf("%d", limit)
	}
	path := s.transport.BuildQueryURL("/v1/dids", params)
	var resp ListResponse
	if err := s.transport.Do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Revoke(ctx context.Context, didID string) error {
	return s.transport.Do(ctx, "DELETE", fmt.Sprintf("/v1/dids/%s", didID), nil, nil)
}

func (s *Service) RotateKey(ctx context.Context, didID string, req *RotateKeyRequest) (*RotateKeyResponse, error) {
	var resp RotateKeyResponse
	if err := s.transport.Do(ctx, "POST", fmt.Sprintf("/v1/dids/%s/rotate", didID), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetRotations(ctx context.Context, didID string) (*RotationHistory, error) {
	var resp RotationHistory
	if err := s.transport.Do(ctx, "GET", fmt.Sprintf("/v1/dids/%s/rotations", didID), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) IssueConsent(ctx context.Context, req *IssueConsentRequest) (*ConsentReceipt, error) {
	var resp ConsentReceipt
	if err := s.transport.Do(ctx, "POST", "/v1/consent/receipts", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetConsent(ctx context.Context, id string) (*ConsentReceipt, error) {
	var resp ConsentReceipt
	if err := s.transport.Do(ctx, "GET", fmt.Sprintf("/v1/consent/receipts/%s", id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) ListConsents(ctx context.Context) (*ConsentListResponse, error) {
	var resp ConsentListResponse
	if err := s.transport.Do(ctx, "GET", "/v1/consent/receipts", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) WithdrawConsent(ctx context.Context, id string) error {
	return s.transport.Do(ctx, "POST", fmt.Sprintf("/v1/consent/receipts/%s/withdraw", id), nil, nil)
}

func (s *Service) GetDIDCommIdentity(ctx context.Context) (*DIDCommIdentity, error) {
	var resp DIDCommIdentity
	if err := s.transport.Do(ctx, "GET", "/v1/didcomm/identity", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) SendDIDCommMessage(ctx context.Context, msg *DIDCommMessage) error {
	headers := map[string]string{"Content-Type": "application/didcomm-plain+json"}
	return s.transport.DoWithHeaders(ctx, "POST", "/v1/didcomm/inbox", msg, nil, headers)
}
