package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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

func TestBreakGlass_RequiresRequester(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.RequestBreakGlass(context.Background(), &BreakGlassRequest{
		Approver1: "approver-1",
		Approver2: "approver-2",
		Reason:    "emergency",
	})
	if err == nil {
		t.Fatal("expected error for missing requester")
	}
}

func TestBreakGlass_RequiresApprover1(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.RequestBreakGlass(context.Background(), &BreakGlassRequest{
		Requester: "admin",
		Approver2: "approver-2",
		Reason:    "emergency",
	})
	if err == nil {
		t.Fatal("expected error for missing approver1")
	}
}

func TestBreakGlass_RequiresApprover2(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.RequestBreakGlass(context.Background(), &BreakGlassRequest{
		Requester: "admin",
		Approver1: "approver-1",
		Reason:    "emergency",
	})
	if err == nil {
		t.Fatal("expected error for missing approver2")
	}
}

func TestBreakGlass_RequiresReason(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.RequestBreakGlass(context.Background(), &BreakGlassRequest{
		Requester: "admin",
		Approver1: "approver-1",
		Approver2: "approver-2",
	})
	if err == nil {
		t.Fatal("expected error for missing reason")
	}
}

func TestBreakGlass_Unsupported(t *testing.T) {
	mt := &mockTransport{
		result: &BreakGlassResponse{
			SessionID:    "bg-session-1",
			AuditEventID: "audit-1",
		},
	}
	svc := NewService(mt)

	resp, err := svc.RequestBreakGlass(context.Background(), &BreakGlassRequest{
		Requester: "admin@zinetic.io",
		Approver1: "security-lead@zinetic.io",
		Approver2: "cto@zinetic.io",
		Reason:    "Production database corruption requiring manual intervention",
	})
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if resp != nil {
		t.Fatalf("expected nil response, got %#v", resp)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported break-glass request should not call transport")
	}
}

func TestRevokeBreakGlass_RequiresSessionID(t *testing.T) {
	svc := NewService(&mockTransport{})
	err := svc.RevokeBreakGlass(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty session_id")
	}
}

func TestReBAcCheck_RequiresUser(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.ReBAcCheck(context.Background(), &ReBAcCheckRequest{
		Relation: "viewer",
		Object:   "doc:123",
	})
	if err == nil {
		t.Fatal("expected error for missing user")
	}
}

func TestReBAcCheck_RequiresRelation(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.ReBAcCheck(context.Background(), &ReBAcCheckRequest{
		User:   "user:admin",
		Object: "doc:123",
	})
	if err == nil {
		t.Fatal("expected error for missing relation")
	}
}

func TestReBAcCheck_RequiresObject(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.ReBAcCheck(context.Background(), &ReBAcCheckRequest{
		User:     "user:admin",
		Relation: "viewer",
	})
	if err == nil {
		t.Fatal("expected error for missing object")
	}
}

func TestReBAcCheck_Unsupported(t *testing.T) {
	mt := &mockTransport{
		result: &ReBAcCheckResponse{Allowed: true},
	}
	svc := NewService(mt)

	resp, err := svc.ReBAcCheck(context.Background(), &ReBAcCheckRequest{
		User:     "user:agent-123",
		Relation: "executor",
		Object:   "tool:code-review",
	})
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if resp != nil {
		t.Fatalf("expected nil response, got %#v", resp)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported ReBAC check should not call transport")
	}
}

func TestReBAcWrite_RequiresAllFields(t *testing.T) {
	svc := NewService(&mockTransport{})

	cases := []struct {
		name string
		req  *ReBAcWriteRequest
	}{
		{"missing user", &ReBAcWriteRequest{Relation: "owner", Object: "doc:1"}},
		{"missing relation", &ReBAcWriteRequest{User: "user:1", Object: "doc:1"}},
		{"missing object", &ReBAcWriteRequest{User: "user:1", Relation: "owner"}},
	}

	for _, c := range cases {
		err := svc.ReBAcWrite(context.Background(), c.req)
		if err == nil {
			t.Fatalf("%s: expected error", c.name)
		}
	}
}

func TestReBAcDelete_RequiresAllFields(t *testing.T) {
	svc := NewService(&mockTransport{})

	cases := []struct {
		name string
		req  *ReBAcDeleteRequest
	}{
		{"missing user", &ReBAcDeleteRequest{Relation: "owner", Object: "doc:1"}},
		{"missing relation", &ReBAcDeleteRequest{User: "user:1", Object: "doc:1"}},
		{"missing object", &ReBAcDeleteRequest{User: "user:1", Relation: "owner"}},
	}

	for _, c := range cases {
		err := svc.ReBAcDelete(context.Background(), c.req)
		if err == nil {
			t.Fatalf("%s: expected error", c.name)
		}
	}
}

func TestCreate_RequiresName(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.Create(context.Background(), &CreateRequest{Rules: "allow all"})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestCreate_RequiresRules(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.Create(context.Background(), &CreateRequest{Name: "my-policy"})
	if err == nil {
		t.Fatal("expected error for missing rules")
	}
}

func TestDelete_RequiresPolicyID(t *testing.T) {
	svc := NewService(&mockTransport{})
	err := svc.Delete(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty policy_id")
	}
}

func TestRollbackBundle_RequiresVersion(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.RollbackBundle(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty version")
	}
}

func TestList_CapsLimitAt200(t *testing.T) {
	mt := &mockTransport{result: &ListResponse{}}
	svc := NewService(mt)
	_, _ = svc.List(context.Background(), "", 500)
	if !strings.Contains(mt.path, "limit=200") {
		t.Fatalf("expected limit capped at 200, got path: %s", mt.path)
	}
}

func TestEnableSimulation_RequiresPolicyID(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.EnableSimulation(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty policy_id")
	}
}

func TestDisableSimulation_RequiresPolicyID(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.DisableSimulation(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty policy_id")
	}
}

func TestCreateNamedLocation_RequiresName(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.CreateNamedLocation(context.Background(), &NamedLocationRequest{})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestCreate_TransportError(t *testing.T) {
	mt := &mockTransport{err: fmt.Errorf("connection refused")}
	svc := NewService(mt)
	_, err := svc.Create(context.Background(), &CreateRequest{Name: "test", Rules: "deny all"})
	if err == nil {
		t.Fatal("expected transport error")
	}
}

func TestGet_Success(t *testing.T) {
	mt := &mockTransport{result: &Policy{ID: "pol-1", Name: "my-policy"}}
	svc := NewService(mt)

	resp, err := svc.Get(context.Background(), "pol-1")
	if err != nil {
		t.Fatal(err)
	}
	if resp.ID != "pol-1" {
		t.Fatalf("expected pol-1, got %s", resp.ID)
	}
	if mt.method != "GET" || !strings.Contains(mt.path, "pol-1") {
		t.Fatalf("unexpected %s %s", mt.method, mt.path)
	}
}

func TestGet_RequiresPolicyID(t *testing.T) {
	svc := NewService(&mockTransport{})
	if _, err := svc.Get(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty policy_id")
	}
}

func TestUpdate_Unsupported(t *testing.T) {
	mt := &mockTransport{result: &Policy{ID: "pol-2", Name: "updated"}}
	svc := NewService(mt)

	resp, err := svc.Update(context.Background(), "pol-2", &UpdateRequest{Name: "updated"})
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if resp != nil {
		t.Fatalf("expected nil response, got %#v", resp)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported update should not call transport")
	}
}

func TestUpdate_RequiresPolicyID(t *testing.T) {
	svc := NewService(&mockTransport{})
	if _, err := svc.Update(context.Background(), "", &UpdateRequest{}); err == nil {
		t.Fatal("expected error for empty policy_id")
	}
}

func TestGetBundleInfo_Success(t *testing.T) {
	mt := &mockTransport{result: &BundleInfo{Version: "v3"}}
	svc := NewService(mt)

	resp, err := svc.GetBundleInfo(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if resp.Version != "v3" {
		t.Fatalf("expected v3, got %s", resp.Version)
	}
	if mt.path != "/v1/policies/bundles" {
		t.Fatalf("unexpected path %s", mt.path)
	}
}

func TestRollbackBundle_Unsupported(t *testing.T) {
	mt := &mockTransport{result: &BundleInfo{Version: "v2"}}
	svc := NewService(mt)

	resp, err := svc.RollbackBundle(context.Background(), "v2")
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if resp != nil {
		t.Fatalf("expected nil response, got %#v", resp)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported rollback should not call transport")
	}
}

func TestListNamedLocations_Unsupported(t *testing.T) {
	type listResp struct {
		Data []NamedLocation `json:"data"`
	}
	mt := &mockTransport{result: &listResp{Data: []NamedLocation{{ID: "loc-1", Name: "HQ"}}}}
	svc := NewService(mt)

	locs, err := svc.ListNamedLocations(context.Background())
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if locs != nil {
		t.Fatalf("expected nil locations, got %#v", locs)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported named-location listing should not call transport")
	}
}

func TestDeleteNamedLocation_Unsupported(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	if err := svc.DeleteNamedLocation(context.Background(), "loc-1"); err == nil {
		t.Fatal("expected unsupported error")
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported named-location deletion should not call transport")
	}
}

func TestDeleteNamedLocation_RequiresID(t *testing.T) {
	svc := NewService(&mockTransport{})
	if err := svc.DeleteNamedLocation(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty location_id")
	}
}

func TestEnableSimulation_Unsupported(t *testing.T) {
	mt := &mockTransport{result: &Policy{ID: "pol-1", SimulationMode: true}}
	svc := NewService(mt)

	resp, err := svc.EnableSimulation(context.Background(), "pol-1")
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if resp != nil {
		t.Fatalf("expected nil response, got %#v", resp)
	}
}

func TestDisableSimulation_Unsupported(t *testing.T) {
	mt := &mockTransport{result: &Policy{ID: "pol-1", SimulationMode: false}}
	svc := NewService(mt)

	resp, err := svc.DisableSimulation(context.Background(), "pol-1")
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if resp != nil {
		t.Fatalf("expected nil response, got %#v", resp)
	}
}

func TestRevokeBreakGlass_Unsupported(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	if err := svc.RevokeBreakGlass(context.Background(), "sess-1"); err == nil {
		t.Fatal("expected unsupported error")
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported break-glass revoke should not call transport")
	}
}
