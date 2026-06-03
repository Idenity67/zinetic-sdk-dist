package user

import "context"

type Transport interface {
	Do(ctx context.Context, method, path string, body interface{}, result interface{}) error
}

type Service struct {
	transport Transport
}

func NewService(t Transport) *Service {
	return &Service{transport: t}
}

func (s *Service) Me(ctx context.Context) (*MeResponse, error) {
	var resp MeResponse
	if err := s.transport.Do(ctx, "GET", "/v1/users/me", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
