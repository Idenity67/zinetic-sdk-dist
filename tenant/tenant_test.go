package tenant

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

type mockTransport struct {
	method    string
	path      string
	body      interface{}
	result    interface{}
	err       error
	callCount int
}

func (m *mockTransport) Do(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	m.method = method
	m.path = path
	m.body = body
	m.callCount++
	if m.err != nil {
		return m.err
	}
	if m.result != nil && result != nil {
		data, _ := json.Marshal(m.result)
		json.Unmarshal(data, result)
	}
	return nil
}

func (m *mockTransport) DoWithHeaders(ctx context.Context, method, path string, body interface{}, result interface{}, headers map[string]string) error {
	return m.Do(ctx, method, path, body, result)
}

func (m *mockTransport) BuildQueryURL(path string, params map[string]string) string {
	if len(params) == 0 {
		return path
	}
	q := path + "?"
	for k, v := range params {
		q += k + "=" + v + "&"
	}
	return q[:len(q)-1]
}

func TestCreate_Success(t *testing.T) {
	mt := &mockTransport{
		result: &Tenant{
			ID:     "tenant-001",
			Name:   "Acme Corp",
			Region: "us-east-1",
			Status: "active",
		},
	}
	svc := NewService(mt)

	resp, err := svc.Create(context.Background(), &CreateRequest{
		Name:   "Acme Corp",
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "tenant-001" {
		t.Fatalf("expected tenant-001, got %s", resp.ID)
	}
	if mt.method != "POST" {
		t.Fatalf("expected POST, got %s", mt.method)
	}
	if mt.path != "/v1/tenants" {
		t.Fatalf("expected /v1/tenants, got %s", mt.path)
	}
}

func TestCreate_MissingName(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.Create(context.Background(), &CreateRequest{Region: "us-east-1"})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestCreate_MissingRegion(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.Create(context.Background(), &CreateRequest{Name: "Acme"})
	if err == nil {
		t.Fatal("expected error for missing region")
	}
}

func TestGet_Success(t *testing.T) {
	mt := &mockTransport{
		result: &Tenant{
			ID:   "tenant-001",
			Name: "Acme Corp",
			Tier: "enterprise",
		},
	}
	svc := NewService(mt)

	resp, err := svc.Get(context.Background(), "tenant-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Name != "Acme Corp" {
		t.Fatalf("expected Acme Corp, got %s", resp.Name)
	}
	if mt.path != "/v1/tenants/tenant-001" {
		t.Fatalf("expected /v1/tenants/tenant-001, got %s", mt.path)
	}
}

func TestGet_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.Get(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty tenant_id")
	}
}

func TestUpdate_Success(t *testing.T) {
	mt := &mockTransport{
		result: &Tenant{ID: "tenant-001", Name: "Acme Updated"},
	}
	svc := NewService(mt)

	resp, err := svc.Update(context.Background(), "tenant-001", &UpdateRequest{Name: "Acme Updated"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Name != "Acme Updated" {
		t.Fatalf("expected Acme Updated, got %s", resp.Name)
	}
	if mt.method != "PUT" {
		t.Fatalf("expected PUT, got %s", mt.method)
	}
}

func TestUpdate_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.Update(context.Background(), "", &UpdateRequest{})
	if err == nil {
		t.Fatal("expected error for empty tenant_id")
	}
}

func TestDelete_Success(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	err := svc.Delete(context.Background(), "tenant-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mt.method != "DELETE" {
		t.Fatalf("expected DELETE, got %s", mt.method)
	}
	if mt.path != "/v1/tenants/tenant-001" {
		t.Fatalf("expected /v1/tenants/tenant-001, got %s", mt.path)
	}
}

func TestDelete_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})

	err := svc.Delete(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty tenant_id")
	}
}

func TestList_Success(t *testing.T) {
	mt := &mockTransport{
		result: &ListResponse{
			Data: []Tenant{
				{ID: "t-1", Name: "Org One"},
				{ID: "t-2", Name: "Org Two"},
			},
			HasMore: true,
		},
	}
	svc := NewService(mt)

	resp, err := svc.List(context.Background(), "", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 tenants, got %d", len(resp.Data))
	}
	if !resp.HasMore {
		t.Fatal("expected has_more=true")
	}
}

func TestList_LimitCapped(t *testing.T) {
	mt := &mockTransport{result: &ListResponse{}}
	svc := NewService(mt)

	_, err := svc.List(context.Background(), "", 500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExportData_Unsupported(t *testing.T) {
	mt := &mockTransport{
		result: &DataExportResponse{
			ExportID: "export-001",
			Status:   "processing",
		},
	}
	svc := NewService(mt)

	resp, err := svc.ExportData(context.Background(), "tenant-001", "json")
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if resp != nil {
		t.Fatalf("expected nil export response, got %#v", resp)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported export should not call transport")
	}
}

func TestExportData_EmptyTenantID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.ExportData(context.Background(), "", "json")
	if err == nil {
		t.Fatal("expected error for empty tenant_id")
	}
}

func TestGetExportStatus_Unsupported(t *testing.T) {
	mt := &mockTransport{
		result: &DataExportResponse{ExportID: "export-001", Status: "completed"},
	}
	svc := NewService(mt)

	resp, err := svc.GetExportStatus(context.Background(), "tenant-001", "export-001")
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if resp != nil {
		t.Fatalf("expected nil export status, got %#v", resp)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported export status should not call transport")
	}
}

func TestGetExportStatus_EmptyIDs(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.GetExportStatus(context.Background(), "", "export-001")
	if err == nil {
		t.Fatal("expected error for empty tenant_id")
	}

	_, err = svc.GetExportStatus(context.Background(), "tenant-001", "")
	if err == nil {
		t.Fatal("expected error for empty export_id")
	}
}

func TestUpdateConfiguration_Unsupported(t *testing.T) {
	mt := &mockTransport{
		result: &Tenant{ID: "tenant-001"},
	}
	svc := NewService(mt)

	resp, err := svc.UpdateConfiguration(context.Background(), "tenant-001", &TenantConfig{
		SessionIdleTimeout: 3600,
	})
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if resp != nil {
		t.Fatalf("expected nil tenant, got %#v", resp)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported configuration update should not call transport")
	}
}

func TestUpdateConfiguration_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.UpdateConfiguration(context.Background(), "", &TenantConfig{})
	if err == nil {
		t.Fatal("expected error for empty tenant_id")
	}
}

func TestUpdateBranding_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.UpdateBranding(context.Background(), "", &Branding{})
	if err == nil {
		t.Fatal("expected error for empty tenant_id")
	}
}

func TestTransportError(t *testing.T) {
	mt := &mockTransport{err: fmt.Errorf("connection timeout")}
	svc := NewService(mt)

	_, err := svc.Get(context.Background(), "tenant-001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
