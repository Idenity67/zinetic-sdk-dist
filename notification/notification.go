package notification

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

func (s *Service) Send(ctx context.Context, req *SendRequest) (*SendResponse, error) {
	var resp SendResponse
	if err := s.transport.Do(ctx, "POST", "/v1/notifications/send", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Get(ctx context.Context, id string) (*Notification, error) {
	var resp Notification
	if err := s.transport.Do(ctx, "GET", fmt.Sprintf("/v1/notifications/%s", id), nil, &resp); err != nil {
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
	path := s.transport.BuildQueryURL("/v1/notifications", params)
	var resp ListResponse
	if err := s.transport.Do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) RegisterToken(ctx context.Context, req *RegisterTokenRequest) (*RegisterTokenResponse, error) {
	var resp RegisterTokenResponse
	if err := s.transport.Do(ctx, "POST", "/v1/push-tokens", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) ListTokens(ctx context.Context, userID string) (*ListTokensResponse, error) {
	path := fmt.Sprintf("/v1/push-tokens/users/%s", userID)
	var resp ListTokensResponse
	if err := s.transport.Do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) RevokeToken(ctx context.Context, platform, token string) error {
	path := fmt.Sprintf("/v1/push-tokens/%s/%s", platform, token)
	return s.transport.Do(ctx, "DELETE", path, nil, nil)
}
