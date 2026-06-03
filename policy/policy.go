package policy

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

func (s *Service) Create(ctx context.Context, req *CreateRequest) (*Policy, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.Rules == "" {
		return nil, fmt.Errorf("rules is required")
	}

	var resp Policy
	err := s.transport.Do(ctx, "POST", "/v1/policies/bundles", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Get(ctx context.Context, policyID string) (*Policy, error) {
	if policyID == "" {
		return nil, fmt.Errorf("policy_id is required")
	}

	policyID, err := pathutil.Segment("policy_id", policyID)
	if err != nil {
		return nil, err
	}

	var resp Policy
	err = s.transport.Do(ctx, "GET", fmt.Sprintf("/v1/policies/bundles/%s", policyID), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Update(ctx context.Context, policyID string, req *UpdateRequest) (*Policy, error) {
	if policyID == "" {
		return nil, fmt.Errorf("policy_id is required")
	}

	return nil, fmt.Errorf("policy bundle update is not supported by the current backend OpenAPI contract")
}

func (s *Service) Delete(ctx context.Context, policyID string) error {
	if policyID == "" {
		return fmt.Errorf("policy_id is required")
	}
	return fmt.Errorf("policy bundle deletion is not supported by the current backend OpenAPI contract")
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

	path := s.transport.BuildQueryURL("/v1/policies/bundles", params)

	var resp ListResponse
	err := s.transport.Do(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetBundleInfo(ctx context.Context) (*BundleInfo, error) {
	var resp BundleInfo
	err := s.transport.Do(ctx, "GET", "/v1/policies/bundles", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) RollbackBundle(ctx context.Context, version string) (*BundleInfo, error) {
	if version == "" {
		return nil, fmt.Errorf("version is required")
	}

	return nil, fmt.Errorf("policy bundle rollback is not supported by the current backend OpenAPI contract")
}

func (s *Service) CreateNamedLocation(ctx context.Context, req *NamedLocationRequest) (*NamedLocation, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	return nil, fmt.Errorf("named location creation is not supported by the current backend OpenAPI contract")
}

func (s *Service) ListNamedLocations(ctx context.Context) ([]NamedLocation, error) {
	return nil, fmt.Errorf("named location listing is not supported by the current backend OpenAPI contract")
}

func (s *Service) DeleteNamedLocation(ctx context.Context, locationID string) error {
	if locationID == "" {
		return fmt.Errorf("location_id is required")
	}
	return fmt.Errorf("named location deletion is not supported by the current backend OpenAPI contract")
}

func (s *Service) RequestBreakGlass(ctx context.Context, req *BreakGlassRequest) (*BreakGlassResponse, error) {
	if req.Requester == "" {
		return nil, fmt.Errorf("requester is required")
	}
	if req.Approver1 == "" {
		return nil, fmt.Errorf("approver1 is required")
	}
	if req.Approver2 == "" {
		return nil, fmt.Errorf("approver2 is required")
	}
	if req.Reason == "" {
		return nil, fmt.Errorf("reason is required")
	}

	return nil, fmt.Errorf("break-glass policy request is not supported by the current backend OpenAPI contract")
}

func (s *Service) RevokeBreakGlass(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	return fmt.Errorf("break-glass revoke is not supported by the current backend OpenAPI contract")
}

func (s *Service) ReBAcCheck(ctx context.Context, req *ReBAcCheckRequest) (*ReBAcCheckResponse, error) {
	if req.User == "" {
		return nil, fmt.Errorf("user is required")
	}
	if req.Relation == "" {
		return nil, fmt.Errorf("relation is required")
	}
	if req.Object == "" {
		return nil, fmt.Errorf("object is required")
	}

	return nil, fmt.Errorf("ReBAC check is not supported by the current backend OpenAPI contract")
}

func (s *Service) ReBAcWrite(ctx context.Context, req *ReBAcWriteRequest) error {
	if req.User == "" {
		return fmt.Errorf("user is required")
	}
	if req.Relation == "" {
		return fmt.Errorf("relation is required")
	}
	if req.Object == "" {
		return fmt.Errorf("object is required")
	}

	return fmt.Errorf("ReBAC tuple write is not supported by the current backend OpenAPI contract")
}

func (s *Service) ReBAcDelete(ctx context.Context, req *ReBAcDeleteRequest) error {
	if req.User == "" {
		return fmt.Errorf("user is required")
	}
	if req.Relation == "" {
		return fmt.Errorf("relation is required")
	}
	if req.Object == "" {
		return fmt.Errorf("object is required")
	}

	return fmt.Errorf("ReBAC tuple deletion is not supported by the current backend OpenAPI contract")
}

func (s *Service) EnableSimulation(ctx context.Context, policyID string) (*Policy, error) {
	if policyID == "" {
		return nil, fmt.Errorf("policy_id is required")
	}

	enabled := true
	return s.Update(ctx, policyID, &UpdateRequest{SimulationMode: &enabled})
}

func (s *Service) DisableSimulation(ctx context.Context, policyID string) (*Policy, error) {
	if policyID == "" {
		return nil, fmt.Errorf("policy_id is required")
	}

	disabled := false
	return s.Update(ctx, policyID, &UpdateRequest{SimulationMode: &disabled})
}
