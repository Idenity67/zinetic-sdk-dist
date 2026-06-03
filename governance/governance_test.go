package governance

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

func TestCreateCampaign_Success(t *testing.T) {
	mt := &mockTransport{
		result: &Campaign{
			ID:     "camp-001",
			Name:   "Q1 Access Review",
			Type:   "user_access",
			Status: "active",
		},
	}
	svc := NewService(mt)

	resp, err := svc.CreateCampaign(context.Background(), &CampaignCreateRequest{
		Name: "Q1 Access Review",
		Type: "user_access",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "camp-001" {
		t.Fatalf("expected camp-001, got %s", resp.ID)
	}
	if mt.method != "POST" {
		t.Fatalf("expected POST, got %s", mt.method)
	}
	if mt.path != "/v1/governance/certifications" {
		t.Fatalf("expected /v1/governance/certifications, got %s", mt.path)
	}
}

func TestCreateCampaign_MissingName(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.CreateCampaign(context.Background(), &CampaignCreateRequest{Type: "user_access"})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestCreateCampaign_MissingType(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.CreateCampaign(context.Background(), &CampaignCreateRequest{Name: "Review"})
	if err == nil {
		t.Fatal("expected error for missing type")
	}
}

func TestGetCampaign_Success(t *testing.T) {
	mt := &mockTransport{
		result: &Campaign{ID: "camp-001", Name: "Q1 Review"},
	}
	svc := NewService(mt)

	resp, err := svc.GetCampaign(context.Background(), "camp-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Name != "Q1 Review" {
		t.Fatalf("expected Q1 Review, got %s", resp.Name)
	}
	if mt.path != "/v1/governance/certifications/camp-001" {
		t.Fatalf("expected path /v1/governance/certifications/camp-001, got %s", mt.path)
	}
}

func TestGetCampaign_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.GetCampaign(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty campaign_id")
	}
}

func TestListCampaigns_Success(t *testing.T) {
	mt := &mockTransport{
		result: &CampaignListResponse{
			Data:    []Campaign{{ID: "c-1"}, {ID: "c-2"}},
			HasMore: true,
		},
	}
	svc := NewService(mt)

	resp, err := svc.ListCampaigns(context.Background(), "", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 campaigns, got %d", len(resp.Data))
	}
	if !resp.HasMore {
		t.Fatal("expected has_more=true")
	}
}

func TestGetCertificationItems_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, _, err := svc.GetCertificationItems(context.Background(), "", "", 0)
	if err == nil {
		t.Fatal("expected error for empty campaign_id")
	}
}

func TestGetCertificationItems_Unsupported(t *testing.T) {
	mt := &mockTransport{
		result: struct {
			Data       []CertificationItem `json:"data"`
			NextCursor string              `json:"next_cursor"`
		}{
			Data: []CertificationItem{
				{ID: "item-1", Decision: "approve"},
			},
			NextCursor: "cursor-abc",
		},
	}
	svc := NewService(mt)

	items, cursor, err := svc.GetCertificationItems(context.Background(), "camp-001", "", 50)
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if items != nil || cursor != "" {
		t.Fatalf("expected empty result, got items=%#v cursor=%q", items, cursor)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported certification item listing should not call transport")
	}
}

func TestDecideCertification_EmptyFields(t *testing.T) {
	svc := NewService(&mockTransport{})

	err := svc.DecideCertification(context.Background(), "", &CertificationDecision{ItemID: "i", Decision: "approve"})
	if err == nil {
		t.Fatal("expected error for empty campaign_id")
	}

	err = svc.DecideCertification(context.Background(), "camp-001", &CertificationDecision{Decision: "approve"})
	if err == nil {
		t.Fatal("expected error for empty item_id")
	}

	err = svc.DecideCertification(context.Background(), "camp-001", &CertificationDecision{ItemID: "i"})
	if err == nil {
		t.Fatal("expected error for empty decision")
	}
}

func TestDecideCertification_Success(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	err := svc.DecideCertification(context.Background(), "camp-001", &CertificationDecision{
		ItemID:   "item-1",
		Decision: "approve",
		Comment:  "looks good",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mt.method != "POST" {
		t.Fatalf("expected POST, got %s", mt.method)
	}
	if mt.path != "/v1/governance/certifications/items/item-1/decide" {
		t.Fatalf("unexpected decision path: %s", mt.path)
	}
}

func TestCreateAccessRequest_Success(t *testing.T) {
	mt := &mockTransport{
		result: &AccessRequest{
			ID:     "req-001",
			Status: "pending",
		},
	}
	svc := NewService(mt)

	resp, err := svc.CreateAccessRequest(context.Background(), &AccessRequestCreateRequest{
		ResourceType:  "database",
		ResourceID:    "db-prod",
		AccessLevel:   "read",
		Justification: "need for reporting",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "req-001" {
		t.Fatalf("expected req-001, got %s", resp.ID)
	}
	if mt.method != "POST" {
		t.Fatalf("expected POST, got %s", mt.method)
	}
}

func TestCreateAccessRequest_MissingFields(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.CreateAccessRequest(context.Background(), &AccessRequestCreateRequest{
		ResourceID: "db", AccessLevel: "read", Justification: "x",
	})
	if err == nil {
		t.Fatal("expected error for missing resource_type")
	}

	_, err = svc.CreateAccessRequest(context.Background(), &AccessRequestCreateRequest{
		ResourceType: "db", AccessLevel: "read", Justification: "x",
	})
	if err == nil {
		t.Fatal("expected error for missing resource_id")
	}

	_, err = svc.CreateAccessRequest(context.Background(), &AccessRequestCreateRequest{
		ResourceType: "db", ResourceID: "id", Justification: "x",
	})
	if err == nil {
		t.Fatal("expected error for missing access_level")
	}

	_, err = svc.CreateAccessRequest(context.Background(), &AccessRequestCreateRequest{
		ResourceType: "db", ResourceID: "id", AccessLevel: "read",
	})
	if err == nil {
		t.Fatal("expected error for missing justification")
	}
}

func TestGetAccessRequest_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.GetAccessRequest(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty request_id")
	}
}

func TestListAccessRequests_Success(t *testing.T) {
	mt := &mockTransport{
		result: &AccessRequestListResponse{
			Data: []AccessRequest{{ID: "r-1"}, {ID: "r-2"}},
		},
	}
	svc := NewService(mt)

	resp, err := svc.ListAccessRequests(context.Background(), "pending", "", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(resp.Data))
	}
}

func TestApproveAccessRequest_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})

	err := svc.ApproveAccessRequest(context.Background(), "", &ApproveRequest{Decision: "approve"})
	if err == nil {
		t.Fatal("expected error for empty request_id")
	}
}

func TestApproveAccessRequest_MissingDecision(t *testing.T) {
	svc := NewService(&mockTransport{})

	err := svc.ApproveAccessRequest(context.Background(), "req-001", &ApproveRequest{})
	if err == nil {
		t.Fatal("expected error for missing decision")
	}
}

func TestApproveAccessRequest_Success(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	err := svc.ApproveAccessRequest(context.Background(), "req-001", &ApproveRequest{
		Decision: "approve",
		Comment:  "authorized",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mt.method != "POST" {
		t.Fatalf("expected POST, got %s", mt.method)
	}
}

func TestCheckSoDViolations_Success(t *testing.T) {
	mt := &mockTransport{
		result: struct {
			Violations []SoDViolation `json:"violations"`
		}{
			Violations: []SoDViolation{
				{PolicyID: "sod-1", Severity: "high"},
			},
		},
	}
	svc := NewService(mt)

	violations, err := svc.CheckSoDViolations(context.Background(), "user-123", []string{"admin", "auditor"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Severity != "high" {
		t.Fatalf("expected high severity, got %s", violations[0].Severity)
	}
}

func TestCheckSoDViolations_MissingPrincipal(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.CheckSoDViolations(context.Background(), "", []string{"admin"})
	if err == nil {
		t.Fatal("expected error for empty principal_id")
	}
}

func TestCheckSoDViolations_EmptyRoles(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.CheckSoDViolations(context.Background(), "user-123", []string{})
	if err == nil {
		t.Fatal("expected error for empty roles")
	}
}

func TestGetPostureScore_Unsupported(t *testing.T) {
	mt := &mockTransport{
		result: &IdentityPostureScore{
			PrincipalID:  "user-123",
			OverallScore: 85.5,
		},
	}
	svc := NewService(mt)

	score, err := svc.GetPostureScore(context.Background(), "user-123")
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if score != nil {
		t.Fatalf("expected nil score, got %#v", score)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported posture score should not call transport")
	}
}

func TestGetPostureScore_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.GetPostureScore(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty principal_id")
	}
}

func TestSetResourceOwner_MissingFields(t *testing.T) {
	svc := NewService(&mockTransport{})

	err := svc.SetResourceOwner(context.Background(), &ResourceOwner{OwnerID: "o-1"})
	if err == nil {
		t.Fatal("expected error for missing resource_id")
	}

	err = svc.SetResourceOwner(context.Background(), &ResourceOwner{ResourceID: "r-1"})
	if err == nil {
		t.Fatal("expected error for missing owner_id")
	}
}

func TestSetResourceOwner_Unsupported(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	err := svc.SetResourceOwner(context.Background(), &ResourceOwner{
		ResourceID: "db-prod",
		OwnerID:    "team-platform",
	})
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported resource owner update should not call transport")
	}
}

func TestGetResourceOwner_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.GetResourceOwner(context.Background(), "database", "")
	if err == nil {
		t.Fatal("expected error for empty resource_id")
	}
}

func TestListEntitlements_Unsupported(t *testing.T) {
	mt := &mockTransport{
		result: struct {
			Data       []Entitlement `json:"data"`
			NextCursor string        `json:"next_cursor"`
		}{
			Data:       []Entitlement{{ID: "e-1", Name: "DB Read"}},
			NextCursor: "next",
		},
	}
	svc := NewService(mt)

	items, cursor, err := svc.ListEntitlements(context.Background(), "", 50)
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if items != nil || cursor != "" {
		t.Fatalf("expected empty result, got items=%#v cursor=%q", items, cursor)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported entitlement listing should not call transport")
	}
}

func TestListAccessBundles_Unsupported(t *testing.T) {
	mt := &mockTransport{
		result: struct {
			Data []AccessBundle `json:"data"`
		}{
			Data: []AccessBundle{{ID: "b-1", Name: "Engineering Bundle"}},
		},
	}
	svc := NewService(mt)

	bundles, err := svc.ListAccessBundles(context.Background())
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if bundles != nil {
		t.Fatalf("expected nil bundles, got %#v", bundles)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported access bundle listing should not call transport")
	}
}

func TestTransportError(t *testing.T) {
	mt := &mockTransport{err: fmt.Errorf("connection refused")}
	svc := NewService(mt)

	_, err := svc.GetCampaign(context.Background(), "camp-001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
