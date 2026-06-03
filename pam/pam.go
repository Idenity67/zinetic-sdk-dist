package pam

import (
	"context"
	"fmt"

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

func (s *Service) RequestElevation(ctx context.Context, req *ElevationRequest) (*ElevationResponse, error) {
	if req.PrincipalID == "" {
		return nil, fmt.Errorf("principal_id is required")
	}
	if req.TargetResource == "" {
		return nil, fmt.Errorf("target_resource is required")
	}
	if req.AccessLevel == "" {
		return nil, fmt.Errorf("access_level is required")
	}
	if req.Justification == "" {
		return nil, fmt.Errorf("justification is required")
	}
	if req.Duration <= 0 {
		req.Duration = 30
	}
	if req.Duration > 240 {
		req.Duration = 240
	}

	var resp ElevationResponse
	err := s.transport.Do(ctx, "POST", "/pam/elevate", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) RevokeElevation(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	sessionID, err := pathutil.Segment("session_id", sessionID)
	if err != nil {
		return err
	}
	return s.transport.Do(ctx, "POST", fmt.Sprintf("/pam/elevate/%s/revoke", sessionID), nil, nil)
}

func (s *Service) GetSession(ctx context.Context, sessionID string) (*ElevationSession, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	sessionID, err := pathutil.Segment("session_id", sessionID)
	if err != nil {
		return nil, err
	}

	var resp ElevationSession
	err = s.transport.Do(ctx, "GET", fmt.Sprintf("/pam/elevations/%s", sessionID), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) ListSessions(ctx context.Context, status string, cursor string, limit int) (*SessionListResponse, error) {
	params := map[string]string{}
	if status != "" {
		params["status"] = status
	}
	if cursor != "" {
		params["cursor"] = cursor
	}
	if limit > 0 {
		if limit > 200 {
			limit = 200
		}
		params["limit"] = fmt.Sprintf("%d", limit)
	}

	path := s.transport.BuildQueryURL("/pam/elevations", params)

	var resp SessionListResponse
	err := s.transport.Do(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetEphemeralCredential(ctx context.Context, sessionID, resourceType, resourceID string) (*EphemeralCredential, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	if resourceType == "" {
		return nil, fmt.Errorf("resource_type is required")
	}
	if resourceID == "" {
		return nil, fmt.Errorf("resource_id is required")
	}

	return nil, fmt.Errorf("ephemeral PAM credential retrieval is not supported by the current backend OpenAPI contract")
}

func (s *Service) GetSessionRecording(ctx context.Context, sessionID string) (*SessionRecording, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	return nil, fmt.Errorf("PAM session recording retrieval is not supported by the current backend OpenAPI contract")
}

func (s *Service) GetAnalytics(ctx context.Context, principalID string) (*AnalyticsResponse, error) {
	if principalID == "" {
		return nil, fmt.Errorf("principal_id is required")
	}

	return nil, fmt.Errorf("PAM analytics retrieval is not supported by the current backend OpenAPI contract")
}

func (s *Service) ApproveElevation(ctx context.Context, sessionID string, approverID string, comment string) error {
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	if approverID == "" {
		return fmt.Errorf("approver_id is required")
	}

	body := map[string]string{
		"approver_id": approverID,
		"decision":    "approve",
		"comment":     comment,
	}
	sessionID, err := pathutil.Segment("session_id", sessionID)
	if err != nil {
		return err
	}
	return s.transport.Do(ctx, "POST", fmt.Sprintf("/pam/elevate/%s/approve", sessionID), body, nil)
}

func (s *Service) DenyElevation(ctx context.Context, sessionID string, approverID string, reason string) error {
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	if approverID == "" {
		return fmt.Errorf("approver_id is required")
	}

	body := map[string]string{
		"approver_id": approverID,
		"decision":    "deny",
		"comment":     reason,
	}
	sessionID, err := pathutil.Segment("session_id", sessionID)
	if err != nil {
		return err
	}
	return s.transport.Do(ctx, "POST", fmt.Sprintf("/pam/elevate/%s/approve", sessionID), body, nil)
}
