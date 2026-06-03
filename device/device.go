package device

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

func (s *Service) List(ctx context.Context, cursor string, limit int) (*ListResponse, error) {
	params := map[string]string{}
	if cursor != "" {
		params["cursor"] = cursor
	}
	if limit > 0 {
		if limit > 200 {
			limit = 200
		}
		params["limit"] = fmt.Sprintf("%d", limit)
	}
	path := s.transport.BuildQueryURL("/v1/devices", params)
	var resp ListResponse
	if err := s.transport.Do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Get(ctx context.Context, id string) (*Device, error) {
	var resp Device
	if err := s.transport.Do(ctx, "GET", fmt.Sprintf("/v1/devices/%s", id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Revoke(ctx context.Context, id string) error {
	return s.transport.Do(ctx, "POST", fmt.Sprintf("/v1/devices/%s/revoke", id), nil, nil)
}

func (s *Service) GetPosture(ctx context.Context, id string) (*PostureReport, error) {
	var resp PostureReport
	if err := s.transport.Do(ctx, "GET", fmt.Sprintf("/v1/devices/%s/posture", id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetHistory(ctx context.Context, id string) (*HistoryResponse, error) {
	var resp HistoryResponse
	if err := s.transport.Do(ctx, "GET", fmt.Sprintf("/v1/devices/%s/history", id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetTrend(ctx context.Context, id string) (*TrendResponse, error) {
	var resp TrendResponse
	if err := s.transport.Do(ctx, "GET", fmt.Sprintf("/v1/devices/%s/trend", id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetTrust(ctx context.Context, id string) (*TrustResponse, error) {
	var resp TrustResponse
	if err := s.transport.Do(ctx, "GET", fmt.Sprintf("/v1/devices/%s/trust", id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Verify(ctx context.Context, id string) (*VerifyResponse, error) {
	var resp VerifyResponse
	if err := s.transport.Do(ctx, "GET", fmt.Sprintf("/v1/devices/%s/verify", id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) ComplianceSummary(ctx context.Context) (*ComplianceSummary, error) {
	var resp ComplianceSummary
	if err := s.transport.Do(ctx, "GET", "/v1/compliance/summary", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) NonCompliant(ctx context.Context) (*NonCompliantResponse, error) {
	var resp NonCompliantResponse
	if err := s.transport.Do(ctx, "GET", "/v1/compliance/non-compliant", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
