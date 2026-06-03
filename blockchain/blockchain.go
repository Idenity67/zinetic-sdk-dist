package blockchain

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

func (s *Service) AnchorCredential(ctx context.Context, req *AnchorRequest) (*AnchorResponse, error) {
	var resp AnchorResponse
	if err := s.transport.Do(ctx, "POST", "/blockchain/credentials/anchor", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) RevokeCredential(ctx context.Context, req *RevokeCredentialRequest) (*TxResponse, error) {
	var resp TxResponse
	if err := s.transport.Do(ctx, "POST", "/blockchain/credentials/revoke", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) BatchRevoke(ctx context.Context, req *BatchRevokeRequest) (*TxResponse, error) {
	var resp TxResponse
	if err := s.transport.Do(ctx, "POST", "/blockchain/credentials/batch-revoke", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetCredentialStatus(ctx context.Context, issuerDID string, credentialIndex int) (*CredentialStatusResponse, error) {
	path := s.transport.BuildQueryURL("/blockchain/credentials/status", map[string]string{
		"issuer_did":       issuerDID,
		"credential_index": fmt.Sprintf("%d", credentialIndex),
	})
	var resp CredentialStatusResponse
	if err := s.transport.Do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetCredentialHash(ctx context.Context, issuerDID string, credentialIndex int) (*CredentialHashResponse, error) {
	path := s.transport.BuildQueryURL("/blockchain/credentials/hash", map[string]string{
		"issuer_did":       issuerDID,
		"credential_index": fmt.Sprintf("%d", credentialIndex),
	})
	var resp CredentialHashResponse
	if err := s.transport.Do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetRevocationStatus(ctx context.Context, issuerDID string, credentialIndex int) (*RevocationStatusResponse, error) {
	path := s.transport.BuildQueryURL("/blockchain/credentials/revocation-status", map[string]string{
		"issuer_did":       issuerDID,
		"credential_index": fmt.Sprintf("%d", credentialIndex),
	})
	var resp RevocationStatusResponse
	if err := s.transport.Do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetRevocations(ctx context.Context, issuerDID string) (*RevocationsResponse, error) {
	path := s.transport.BuildQueryURL("/blockchain/credentials/revocations", map[string]string{
		"issuer_did": issuerDID,
	})
	var resp RevocationsResponse
	if err := s.transport.Do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) SuspendCredential(ctx context.Context, req *SuspendRequest) (*TxResponse, error) {
	var resp TxResponse
	if err := s.transport.Do(ctx, "POST", "/blockchain/credentials/suspend", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) UnsuspendCredential(ctx context.Context, req *UnsuspendRequest) (*TxResponse, error) {
	var resp TxResponse
	if err := s.transport.Do(ctx, "POST", "/blockchain/credentials/unsuspend", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) BatchSuspend(ctx context.Context, req *BatchSuspendRequest) (*TxResponse, error) {
	var resp TxResponse
	if err := s.transport.Do(ctx, "POST", "/blockchain/credentials/batch-suspend", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetSuspensionStatus(ctx context.Context, issuerDID string, credentialIndex int) (*SuspensionStatusResponse, error) {
	path := s.transport.BuildQueryURL("/blockchain/credentials/suspension-status", map[string]string{
		"issuer_did":       issuerDID,
		"credential_index": fmt.Sprintf("%d", credentialIndex),
	})
	var resp SuspensionStatusResponse
	if err := s.transport.Do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetSuspensions(ctx context.Context, issuerDID string) (*SuspensionsResponse, error) {
	path := s.transport.BuildQueryURL("/blockchain/credentials/suspensions", map[string]string{
		"issuer_did": issuerDID,
	})
	var resp SuspensionsResponse
	if err := s.transport.Do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Status(ctx context.Context) (*BlockchainStatus, error) {
	var resp BlockchainStatus
	if err := s.transport.Do(ctx, "GET", "/blockchain/status", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Metadata(ctx context.Context) (*BlockchainMetadata, error) {
	var resp BlockchainMetadata
	if err := s.transport.Do(ctx, "GET", "/blockchain/metadata", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetIssuerInfo(ctx context.Context, did string) (*IssuerInfo, error) {
	var resp IssuerInfo
	if err := s.transport.Do(ctx, "GET", fmt.Sprintf("/blockchain/issuers/%s", did), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
