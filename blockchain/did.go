package blockchain

import (
	"context"
	"fmt"
)

func (s *Service) RegisterDID(ctx context.Context, req *RegisterDIDRequest) (*RegisterDIDResponse, error) {
	var resp RegisterDIDResponse
	if err := s.transport.Do(ctx, "POST", "/blockchain/dids/register", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) RegisterDIDMeta(ctx context.Context, req *RegisterDIDMetaRequest) (*RegisterDIDResponse, error) {
	var resp RegisterDIDResponse
	if err := s.transport.Do(ctx, "POST", "/blockchain/dids/register-meta", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) ResolveDID(ctx context.Context, did string) (*ResolvedDID, error) {
	var resp ResolvedDID
	if err := s.transport.Do(ctx, "GET", fmt.Sprintf("/blockchain/dids/%s", did), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) UpdateDID(ctx context.Context, did string, req *UpdateDIDRequest) error {
	return s.transport.Do(ctx, "PUT", fmt.Sprintf("/blockchain/dids/%s", did), req, nil)
}

func (s *Service) DeactivateDID(ctx context.Context, did string) error {
	return s.transport.Do(ctx, "DELETE", fmt.Sprintf("/blockchain/dids/%s", did), nil, nil)
}

func (s *Service) ReactivateDID(ctx context.Context, did string) (*TxResponse, error) {
	var resp TxResponse
	if err := s.transport.Do(ctx, "POST", fmt.Sprintf("/blockchain/dids/%s/reactivate", did), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetDIDStatus(ctx context.Context, did string) (*DIDStatusResponse, error) {
	var resp DIDStatusResponse
	if err := s.transport.Do(ctx, "GET", fmt.Sprintf("/blockchain/dids/%s/status", did), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
