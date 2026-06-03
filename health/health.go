package health

import (
	"context"
	"time"
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

type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Details   map[string]string `json:"details,omitempty"`
}

type ReadyResponse struct {
	Status       string              `json:"status"`
	Dependencies map[string]DepCheck `json:"dependencies,omitempty"`
}

type DepCheck struct {
	Status  string `json:"status"`
	Latency string `json:"latency,omitempty"`
}

type VersionResponse struct {
	Version   string `json:"version"`
	CommitSHA string `json:"commit_sha"`
	BuildTime string `json:"build_time"`
}

func (s *Service) Health(ctx context.Context) (*HealthResponse, error) {
	var resp HealthResponse
	err := s.transport.Do(ctx, "GET", "/health", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Ready(ctx context.Context) (*ReadyResponse, error) {
	var resp ReadyResponse
	err := s.transport.Do(ctx, "GET", "/ready", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Version(ctx context.Context) (*VersionResponse, error) {
	var resp VersionResponse
	err := s.transport.Do(ctx, "GET", "/version", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Metrics(ctx context.Context) (string, error) {
	var resp string
	err := s.transport.Do(ctx, "GET", "/metrics", nil, &resp)
	if err != nil {
		return "", err
	}
	return resp, nil
}
