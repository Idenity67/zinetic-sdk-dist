package secrets

import (
	"context"
	"fmt"
	"strings"

	"sdk.zinetic.net/internal/pathutil"
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

func (s *Service) ListMounts(ctx context.Context) (*ListMountsResponse, error) {
	var resp ListMountsResponse
	if err := s.transport.Do(ctx, "GET", "/v1/secrets/sys/mounts", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Mount(ctx context.Context, req *MountEngineRequest) (*MountEngineStatus, error) {
	var resp MountEngineStatus
	if err := s.transport.Do(ctx, "POST", "/v1/secrets/sys/mounts", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Write(ctx context.Context, engine, path string, req *KVWriteRequest) (*KVWriteResponse, error) {
	var resp KVWriteResponse
	endpoint, err := kvDataEndpoint(engine, path)
	if err != nil {
		return nil, err
	}
	if err := s.transport.Do(ctx, "PUT", endpoint, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Read(ctx context.Context, engine, path string, version int) (*KVReadResponse, error) {
	endpoint, err := kvDataEndpoint(engine, path)
	if err != nil {
		return nil, err
	}
	if version > 0 {
		endpoint = s.transport.BuildQueryURL(endpoint, map[string]string{"version": fmt.Sprintf("%d", version)})
	}
	var resp KVReadResponse
	if err := s.transport.Do(ctx, "GET", endpoint, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Delete(ctx context.Context, engine, path string) error {
	return s.DeleteWithReason(ctx, engine, path, "sdk requested secret deletion")
}

func (s *Service) DeleteWithReason(ctx context.Context, engine, path string, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return fmt.Errorf("reason is required")
	}
	endpoint, err := kvDataEndpoint(engine, path)
	if err != nil {
		return err
	}
	endpoint = s.transport.BuildQueryURL(endpoint, map[string]string{"reason": reason})
	return s.transport.Do(ctx, "DELETE", endpoint, nil, nil)
}

func (s *Service) Encrypt(ctx context.Context, engine string, req *TransitEncryptRequest) (*TransitResponse, error) {
	var resp TransitResponse
	engineEscaped, err := pathutil.Segment("engine", engine)
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("/v1/secrets/%s/encrypt", engineEscaped)
	if err := s.transport.Do(ctx, "POST", endpoint, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Decrypt(ctx context.Context, engine string, req *TransitDecryptRequest) (*TransitResponse, error) {
	var resp TransitResponse
	engineEscaped, err := pathutil.Segment("engine", engine)
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("/v1/secrets/%s/decrypt", engineEscaped)
	if err := s.transport.Do(ctx, "POST", endpoint, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) CreateLease(ctx context.Context, engine, path string, req *LeaseRequest) (*LeaseResponse, error) {
	var resp LeaseResponse
	engineEscaped, err := pathutil.Segment("engine", engine)
	if err != nil {
		return nil, err
	}
	pathEscaped, err := pathutil.SlashPath("lease path", path)
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("/v1/secrets/%s/lease/%s", engineEscaped, pathEscaped)
	if err := s.transport.Do(ctx, "POST", endpoint, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) RenewLease(ctx context.Context, req *RenewLeaseRequest) (*LeaseResponse, error) {
	var resp LeaseResponse
	if err := s.transport.Do(ctx, "PUT", "/v1/secrets/sys/leases/renew", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) RevokeLease(ctx context.Context, req *RevokeLeaseRequest) error {
	return s.transport.Do(ctx, "PUT", "/v1/secrets/sys/leases/revoke", req, nil)
}

func kvDataEndpoint(engine, secretPath string) (string, error) {
	engineEscaped, err := pathutil.Segment("engine", engine)
	if err != nil {
		return "", err
	}
	pathEscaped, err := pathutil.SlashPath("secret path", secretPath)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/v1/secrets/%s/data/%s", engineEscaped, pathEscaped), nil
}
