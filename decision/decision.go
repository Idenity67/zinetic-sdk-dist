package decision

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

func (s *Service) Evaluate(ctx context.Context, req *AuthRequest) (*AuthResponse, error) {
	if req.SubjectID == "" {
		return nil, fmt.Errorf("subject_id is required")
	}
	if req.Context == nil {
		return nil, fmt.Errorf("context is required")
	}

	var resp AuthResponse
	err := s.transport.Do(ctx, "POST", "/v1/decision/auth", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) BatchEvaluate(ctx context.Context, req *BatchAuthRequest) (*BatchAuthResponse, error) {
	if len(req.Requests) == 0 {
		return nil, fmt.Errorf("at least one request is required")
	}
	if len(req.Requests) > 100 {
		return nil, fmt.Errorf("batch size must not exceed 100, got %d", len(req.Requests))
	}

	return nil, fmt.Errorf("batch decision evaluation is not supported by the current backend OpenAPI contract")
}

func (s *Service) Simulate(ctx context.Context, req *SimulateRequest) (*SimulateResponse, error) {
	if req.Request == nil {
		return nil, fmt.Errorf("request is required")
	}
	if req.Request.SubjectID == "" {
		return nil, fmt.Errorf("subject_id is required")
	}

	return nil, fmt.Errorf("decision simulation is not supported by the current backend OpenAPI contract")
}
