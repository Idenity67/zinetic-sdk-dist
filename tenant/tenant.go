package tenant

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

func (s *Service) Create(ctx context.Context, req *CreateRequest) (*Tenant, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.Region == "" {
		return nil, fmt.Errorf("region is required")
	}

	var resp Tenant
	err := s.transport.Do(ctx, "POST", "/v1/tenants", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Get(ctx context.Context, tenantID string) (*Tenant, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}

	tenantID, err := pathutil.Segment("tenant_id", tenantID)
	if err != nil {
		return nil, err
	}

	var resp Tenant
	err = s.transport.Do(ctx, "GET", fmt.Sprintf("/v1/tenants/%s", tenantID), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Update(ctx context.Context, tenantID string, req *UpdateRequest) (*Tenant, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}

	tenantID, err := pathutil.Segment("tenant_id", tenantID)
	if err != nil {
		return nil, err
	}

	var resp Tenant
	err = s.transport.Do(ctx, "PUT", fmt.Sprintf("/v1/tenants/%s", tenantID), req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Delete(ctx context.Context, tenantID string) error {
	if tenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}
	tenantID, err := pathutil.Segment("tenant_id", tenantID)
	if err != nil {
		return err
	}
	return s.transport.Do(ctx, "DELETE", fmt.Sprintf("/v1/tenants/%s", tenantID), nil, nil)
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

	path := s.transport.BuildQueryURL("/v1/tenants", params)

	var resp ListResponse
	err := s.transport.Do(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) ExportData(ctx context.Context, tenantID string, format string) (*DataExportResponse, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}

	return nil, fmt.Errorf("tenant data export is not supported by the current backend OpenAPI contract")
}

func (s *Service) GetExportStatus(ctx context.Context, tenantID, exportID string) (*DataExportResponse, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if exportID == "" {
		return nil, fmt.Errorf("export_id is required")
	}

	return nil, fmt.Errorf("tenant export status is not supported by the current backend OpenAPI contract")
}

func (s *Service) UpdateConfiguration(ctx context.Context, tenantID string, config *TenantConfig) (*Tenant, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}

	return nil, fmt.Errorf("tenant configuration update is not supported by the current backend OpenAPI contract")
}

func (s *Service) UpdateBranding(ctx context.Context, tenantID string, branding *Branding) (*Tenant, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}

	return nil, fmt.Errorf("tenant branding update is not supported by the current backend OpenAPI contract")
}
