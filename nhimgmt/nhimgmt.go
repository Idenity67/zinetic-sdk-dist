package nhimgmt

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

func (s *Service) Create(ctx context.Context, req *CreateRequest) (*Identity, error) {
	var resp Identity
	if err := s.transport.Do(ctx, "POST", "/v1/nhi/identities", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Get(ctx context.Context, id string) (*Identity, error) {
	var resp Identity
	if err := s.transport.Do(ctx, "GET", fmt.Sprintf("/v1/nhi/identities/%s", id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) List(ctx context.Context, kind string, limit, offset int) (*ListResponse, error) {
	params := map[string]string{}
	if kind != "" {
		params["kind"] = kind
	}
	if limit > 0 {
		params["limit"] = fmt.Sprintf("%d", limit)
	}
	if offset > 0 {
		params["offset"] = fmt.Sprintf("%d", offset)
	}
	path := s.transport.BuildQueryURL("/v1/nhi/identities", params)
	var resp ListResponse
	if err := s.transport.Do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Update(ctx context.Context, id string, req *UpdateRequest) (*Identity, error) {
	var resp Identity
	if err := s.transport.Do(ctx, "PATCH", fmt.Sprintf("/v1/nhi/identities/%s", id), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.transport.Do(ctx, "DELETE", fmt.Sprintf("/v1/nhi/identities/%s", id), nil, nil)
}

func (s *Service) Rotate(ctx context.Context, id string) error {
	return s.transport.Do(ctx, "POST", fmt.Sprintf("/v1/nhi/identities/%s/rotate", id), nil, nil)
}

func (s *Service) RecordConnection(ctx context.Context, conn *Connection) error {
	return s.transport.Do(ctx, "POST", "/v1/nhi/connections", conn, nil)
}

func (s *Service) GetConnections(ctx context.Context, agentID string) (*ConnectionsResponse, error) {
	var resp ConnectionsResponse
	if err := s.transport.Do(ctx, "GET", fmt.Sprintf("/v1/nhi/connections/%s", agentID), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetGraph(ctx context.Context) (*ConnectivityGraph, error) {
	var resp ConnectivityGraph
	if err := s.transport.Do(ctx, "GET", "/v1/nhi/graph", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Evaluate(ctx context.Context) (*EvaluateResponse, error) {
	var resp EvaluateResponse
	if err := s.transport.Do(ctx, "POST", "/v1/nhi/evaluate", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
