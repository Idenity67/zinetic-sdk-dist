package governance

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

func (s *Service) CreateCampaign(ctx context.Context, req *CampaignCreateRequest) (*Campaign, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.Type == "" {
		return nil, fmt.Errorf("type is required")
	}

	var resp Campaign
	err := s.transport.Do(ctx, "POST", "/v1/governance/certifications", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetCampaign(ctx context.Context, campaignID string) (*Campaign, error) {
	if campaignID == "" {
		return nil, fmt.Errorf("campaign_id is required")
	}

	campaignID, err := pathutil.Segment("campaign_id", campaignID)
	if err != nil {
		return nil, err
	}

	var resp Campaign
	err = s.transport.Do(ctx, "GET", fmt.Sprintf("/v1/governance/certifications/%s", campaignID), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) ListCampaigns(ctx context.Context, cursor string, limit int) (*CampaignListResponse, error) {
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

	path := s.transport.BuildQueryURL("/v1/governance/certifications", params)

	var resp CampaignListResponse
	err := s.transport.Do(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetCertificationItems(ctx context.Context, campaignID string, cursor string, limit int) ([]CertificationItem, string, error) {
	if campaignID == "" {
		return nil, "", fmt.Errorf("campaign_id is required")
	}

	return nil, "", fmt.Errorf("certification item listing is not supported by the current backend OpenAPI contract")
}

func (s *Service) DecideCertification(ctx context.Context, campaignID string, decision *CertificationDecision) error {
	if campaignID == "" {
		return fmt.Errorf("campaign_id is required")
	}
	if decision.ItemID == "" {
		return fmt.Errorf("item_id is required")
	}
	if decision.Decision == "" {
		return fmt.Errorf("decision is required")
	}

	itemID, err := pathutil.Segment("item_id", decision.ItemID)
	if err != nil {
		return err
	}
	return s.transport.Do(ctx, "POST", fmt.Sprintf("/v1/governance/certifications/items/%s/decide", itemID), decision, nil)
}

func (s *Service) CreateAccessRequest(ctx context.Context, req *AccessRequestCreateRequest) (*AccessRequest, error) {
	if req.ResourceType == "" {
		return nil, fmt.Errorf("resource_type is required")
	}
	if req.ResourceID == "" {
		return nil, fmt.Errorf("resource_id is required")
	}
	if req.AccessLevel == "" {
		return nil, fmt.Errorf("access_level is required")
	}
	if req.Justification == "" {
		return nil, fmt.Errorf("justification is required")
	}

	var resp AccessRequest
	err := s.transport.Do(ctx, "POST", "/v1/governance/access-requests", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetAccessRequest(ctx context.Context, requestID string) (*AccessRequest, error) {
	if requestID == "" {
		return nil, fmt.Errorf("request_id is required")
	}

	return nil, fmt.Errorf("access request lookup by id is not supported by the current backend OpenAPI contract")
}

func (s *Service) ListAccessRequests(ctx context.Context, status string, cursor string, limit int) (*AccessRequestListResponse, error) {
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

	path := s.transport.BuildQueryURL("/v1/governance/access-requests", params)

	var resp AccessRequestListResponse
	err := s.transport.Do(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) ApproveAccessRequest(ctx context.Context, requestID string, req *ApproveRequest) error {
	if requestID == "" {
		return fmt.Errorf("request_id is required")
	}
	if req.Decision == "" {
		return fmt.Errorf("decision is required")
	}

	requestID, err := pathutil.Segment("request_id", requestID)
	if err != nil {
		return err
	}
	return s.transport.Do(ctx, "POST", fmt.Sprintf("/v1/governance/access-requests/%s/approve", requestID), req, nil)
}

func (s *Service) CheckSoDViolations(ctx context.Context, principalID string, roles []string) ([]SoDViolation, error) {
	if principalID == "" {
		return nil, fmt.Errorf("principal_id is required")
	}
	if len(roles) == 0 {
		return nil, fmt.Errorf("at least one role is required")
	}

	body := map[string]interface{}{
		"principal_id": principalID,
		"roles":        roles,
	}

	var resp struct {
		Violations []SoDViolation `json:"violations"`
	}
	err := s.transport.Do(ctx, "POST", "/v1/governance/entitlements/sod/check", body, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Violations, nil
}

func (s *Service) GetPostureScore(ctx context.Context, principalID string) (*IdentityPostureScore, error) {
	if principalID == "" {
		return nil, fmt.Errorf("principal_id is required")
	}

	return nil, fmt.Errorf("identity posture score is not supported by the current backend OpenAPI contract")
}

func (s *Service) ListEntitlements(ctx context.Context, cursor string, limit int) ([]Entitlement, string, error) {
	return nil, "", fmt.Errorf("entitlement listing is not supported by the current backend OpenAPI contract")
}

func (s *Service) ListAccessBundles(ctx context.Context) ([]AccessBundle, error) {
	return nil, fmt.Errorf("access bundle listing is not supported by the current backend OpenAPI contract")
}

func (s *Service) SetResourceOwner(ctx context.Context, owner *ResourceOwner) error {
	if owner.ResourceID == "" {
		return fmt.Errorf("resource_id is required")
	}
	if owner.OwnerID == "" {
		return fmt.Errorf("owner_id is required")
	}

	return fmt.Errorf("resource owner updates are not supported by the current backend OpenAPI contract")
}

func (s *Service) GetResourceOwner(ctx context.Context, resourceType, resourceID string) (*ResourceOwner, error) {
	if resourceID == "" {
		return nil, fmt.Errorf("resource_id is required")
	}

	return nil, fmt.Errorf("resource owner lookup is not supported by the current backend OpenAPI contract")
}
