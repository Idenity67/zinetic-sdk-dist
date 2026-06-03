package nhimgmt

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

type mockTransport struct {
	method string
	path   string
	body   interface{}
	resp   interface{}
	err    error
}

func (m *mockTransport) Do(_ context.Context, method, path string, body interface{}, result interface{}) error {
	m.method = method
	m.path = path
	m.body = body
	if m.err != nil {
		return m.err
	}
	if m.resp != nil && result != nil {
		b, _ := json.Marshal(m.resp)
		return json.Unmarshal(b, result)
	}
	return nil
}

func (m *mockTransport) DoWithHeaders(_ context.Context, method, path string, body interface{}, result interface{}, _ map[string]string) error {
	return m.Do(context.Background(), method, path, body, result)
}

func (m *mockTransport) BuildQueryURL(path string, params map[string]string) string {
	if len(params) == 0 {
		return path
	}
	parts := make([]string, 0, len(params))
	for k, v := range params {
		parts = append(parts, k+"="+v)
	}
	return path + "?" + strings.Join(parts, "&")
}

func TestCreate(t *testing.T) {
	mt := &mockTransport{resp: &Identity{ID: "nhi-1", Kind: "workload", Name: "api-svc"}}
	svc := NewService(mt)

	resp, err := svc.Create(context.Background(), &CreateRequest{
		Kind:        "workload",
		Name:        "api-svc",
		Environment: "production",
		Source:      "first_party",
	})
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "POST" || mt.path != "/v1/nhi/identities" {
		t.Fatalf("unexpected %s %s", mt.method, mt.path)
	}
	if resp.ID != "nhi-1" {
		t.Fatalf("expected nhi-1, got %s", resp.ID)
	}
}

func TestGet(t *testing.T) {
	mt := &mockTransport{resp: &Identity{ID: "nhi-1", Kind: "service"}}
	svc := NewService(mt)

	resp, err := svc.Get(context.Background(), "nhi-1")
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/nhi/identities/nhi-1" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.Kind != "service" {
		t.Fatalf("expected service, got %s", resp.Kind)
	}
}

func TestList(t *testing.T) {
	mt := &mockTransport{resp: &ListResponse{Count: 5}}
	svc := NewService(mt)

	resp, err := svc.List(context.Background(), "workload", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mt.path, "kind=workload") {
		t.Fatalf("expected kind param, got %s", mt.path)
	}
	if resp.Count != 5 {
		t.Fatalf("expected 5, got %d", resp.Count)
	}
}

func TestUpdate(t *testing.T) {
	mt := &mockTransport{resp: &Identity{ID: "nhi-1", Status: "suspended"}}
	svc := NewService(mt)

	resp, err := svc.Update(context.Background(), "nhi-1", &UpdateRequest{Status: "suspended"})
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "PATCH" || mt.path != "/v1/nhi/identities/nhi-1" {
		t.Fatalf("unexpected %s %s", mt.method, mt.path)
	}
	if resp.Status != "suspended" {
		t.Fatalf("expected suspended, got %s", resp.Status)
	}
}

func TestDelete(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	err := svc.Delete(context.Background(), "nhi-1")
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "DELETE" || mt.path != "/v1/nhi/identities/nhi-1" {
		t.Fatalf("unexpected %s %s", mt.method, mt.path)
	}
}

func TestRotate(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	err := svc.Rotate(context.Background(), "nhi-1")
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "POST" || mt.path != "/v1/nhi/identities/nhi-1/rotate" {
		t.Fatalf("unexpected %s %s", mt.method, mt.path)
	}
}

func TestRecordConnection(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	err := svc.RecordConnection(context.Background(), &Connection{
		AgentID:    "agent-1",
		TargetType: "database",
		TargetID:   "pg-prod",
		TargetName: "production-db",
	})
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "POST" || mt.path != "/v1/nhi/connections" {
		t.Fatalf("unexpected %s %s", mt.method, mt.path)
	}
}

func TestGetConnections(t *testing.T) {
	mt := &mockTransport{resp: &ConnectionsResponse{Connections: []Connection{{AgentID: "a-1", TargetType: "api"}}}}
	svc := NewService(mt)

	resp, err := svc.GetConnections(context.Background(), "a-1")
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/nhi/connections/a-1" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if len(resp.Connections) != 1 {
		t.Fatalf("expected 1, got %d", len(resp.Connections))
	}
}

func TestGetGraph(t *testing.T) {
	mt := &mockTransport{resp: &ConnectivityGraph{TotalNHIs: 10, TotalEdges: 25}}
	svc := NewService(mt)

	resp, err := svc.GetGraph(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/nhi/graph" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.TotalNHIs != 10 {
		t.Fatalf("expected 10, got %d", resp.TotalNHIs)
	}
}

func TestEvaluate(t *testing.T) {
	mt := &mockTransport{resp: &EvaluateResponse{Actions: []RemediationAction{{ID: "r-1", Type: "rotate"}}}}
	svc := NewService(mt)

	resp, err := svc.Evaluate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "POST" || mt.path != "/v1/nhi/evaluate" {
		t.Fatalf("unexpected %s %s", mt.method, mt.path)
	}
	if len(resp.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(resp.Actions))
	}
}

func TestCreate_Error(t *testing.T) {
	mt := &mockTransport{err: fmt.Errorf("conflict")}
	svc := NewService(mt)

	_, err := svc.Create(context.Background(), &CreateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}
