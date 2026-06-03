package device

import (
	"context"
	"encoding/json"
	"fmt"
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

func (m *mockTransport) BuildQueryURL(path string, _ map[string]string) string {
	return path
}

func TestList(t *testing.T) {
	mt := &mockTransport{resp: &ListResponse{Count: 2, Devices: []Device{{ID: "d-1"}, {ID: "d-2"}}}}
	svc := NewService(mt)

	resp, err := svc.List(context.Background(), "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "GET" || mt.path != "/v1/devices" {
		t.Fatalf("unexpected %s %s", mt.method, mt.path)
	}
	if resp.Count != 2 {
		t.Fatalf("expected 2, got %d", resp.Count)
	}
}

func TestGet(t *testing.T) {
	mt := &mockTransport{resp: &Device{ID: "d-1", Platform: "macOS", Trusted: true}}
	svc := NewService(mt)

	resp, err := svc.Get(context.Background(), "d-1")
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/devices/d-1" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if !resp.Trusted {
		t.Fatal("expected trusted")
	}
}

func TestRevoke(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	err := svc.Revoke(context.Background(), "d-1")
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "POST" || mt.path != "/v1/devices/d-1/revoke" {
		t.Fatalf("unexpected %s %s", mt.method, mt.path)
	}
}

func TestGetPosture(t *testing.T) {
	mt := &mockTransport{resp: &PostureReport{DeviceID: "d-1", RiskScore: 15, ComplianceStatus: "compliant"}}
	svc := NewService(mt)

	resp, err := svc.GetPosture(context.Background(), "d-1")
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/devices/d-1/posture" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.RiskScore != 15 {
		t.Fatalf("expected 15, got %d", resp.RiskScore)
	}
}

func TestGetHistory(t *testing.T) {
	mt := &mockTransport{resp: &HistoryResponse{Count: 5}}
	svc := NewService(mt)

	resp, err := svc.GetHistory(context.Background(), "d-1")
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/devices/d-1/history" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.Count != 5 {
		t.Fatalf("expected 5, got %d", resp.Count)
	}
}

func TestGetTrend(t *testing.T) {
	mt := &mockTransport{resp: &TrendResponse{DeviceID: "d-1"}}
	svc := NewService(mt)

	resp, err := svc.GetTrend(context.Background(), "d-1")
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/devices/d-1/trend" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.DeviceID != "d-1" {
		t.Fatalf("expected d-1, got %s", resp.DeviceID)
	}
}

func TestGetTrust(t *testing.T) {
	mt := &mockTransport{resp: &TrustResponse{DeviceID: "d-1", TrustScore: 0.92}}
	svc := NewService(mt)

	resp, err := svc.GetTrust(context.Background(), "d-1")
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/devices/d-1/trust" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.TrustScore != 0.92 {
		t.Fatalf("expected 0.92, got %f", resp.TrustScore)
	}
}

func TestVerify(t *testing.T) {
	mt := &mockTransport{resp: &VerifyResponse{DeviceID: "d-1", Verified: true}}
	svc := NewService(mt)

	resp, err := svc.Verify(context.Background(), "d-1")
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/devices/d-1/verify" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if !resp.Verified {
		t.Fatal("expected verified")
	}
}

func TestComplianceSummary(t *testing.T) {
	mt := &mockTransport{resp: &ComplianceSummary{TotalDevices: 100, Compliant: 90, NonCompliant: 10}}
	svc := NewService(mt)

	resp, err := svc.ComplianceSummary(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/compliance/summary" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.TotalDevices != 100 {
		t.Fatalf("expected 100, got %d", resp.TotalDevices)
	}
}

func TestNonCompliant(t *testing.T) {
	mt := &mockTransport{resp: &NonCompliantResponse{Count: 3}}
	svc := NewService(mt)

	resp, err := svc.NonCompliant(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/compliance/non-compliant" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.Count != 3 {
		t.Fatalf("expected 3, got %d", resp.Count)
	}
}

func TestGet_Error(t *testing.T) {
	mt := &mockTransport{err: fmt.Errorf("timeout")}
	svc := NewService(mt)

	_, err := svc.Get(context.Background(), "d-1")
	if err == nil {
		t.Fatal("expected error")
	}
}
