package did

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

type mockTransport struct {
	method  string
	path    string
	body    interface{}
	headers map[string]string
	resp    interface{}
	err     error
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

func (m *mockTransport) DoWithHeaders(_ context.Context, method, path string, body interface{}, result interface{}, headers map[string]string) error {
	m.headers = headers
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
	mt := &mockTransport{resp: &DID{ID: "did:web:example", Method: "web", Active: true}}
	svc := NewService(mt)

	resp, err := svc.Create(context.Background(), &CreateRequest{Method: "web", UserID: "u-1"})
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "POST" || mt.path != "/v1/dids/create" {
		t.Fatalf("unexpected %s %s", mt.method, mt.path)
	}
	if resp.ID != "did:web:example" {
		t.Fatalf("expected did:web:example, got %s", resp.ID)
	}
}

func TestResolve(t *testing.T) {
	mt := &mockTransport{resp: &DID{ID: "did:key:z6Mk", Active: true}}
	svc := NewService(mt)

	resp, err := svc.Resolve(context.Background(), "did:key:z6Mk")
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/dids/did:key:z6Mk" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if !resp.Active {
		t.Fatal("expected active")
	}
}

func TestList(t *testing.T) {
	mt := &mockTransport{resp: &ListResponse{Count: 3}}
	svc := NewService(mt)

	resp, err := svc.List(context.Background(), "u-1", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mt.path, "user_id=u-1") {
		t.Fatalf("expected user_id param, got %s", mt.path)
	}
	if resp.Count != 3 {
		t.Fatalf("expected 3, got %d", resp.Count)
	}
}

func TestRevoke(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	err := svc.Revoke(context.Background(), "did:web:x")
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "DELETE" || mt.path != "/v1/dids/did:web:x" {
		t.Fatalf("unexpected %s %s", mt.method, mt.path)
	}
}

func TestRotateKey(t *testing.T) {
	mt := &mockTransport{resp: &RotateKeyResponse{DIDID: "did:web:x", NewKeyID: "k-2"}}
	svc := NewService(mt)

	resp, err := svc.RotateKey(context.Background(), "did:web:x", &RotateKeyRequest{DIDID: "did:web:x", UserID: "u-1"})
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/dids/did:web:x/rotate" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.NewKeyID != "k-2" {
		t.Fatalf("expected k-2, got %s", resp.NewKeyID)
	}
}

func TestGetRotations(t *testing.T) {
	mt := &mockTransport{resp: &RotationHistory{Count: 2}}
	svc := NewService(mt)

	resp, err := svc.GetRotations(context.Background(), "did:web:x")
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/dids/did:web:x/rotations" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.Count != 2 {
		t.Fatalf("expected 2, got %d", resp.Count)
	}
}

func TestIssueConsent(t *testing.T) {
	mt := &mockTransport{resp: &ConsentReceipt{ID: "cr-1", Subject: "u-1"}}
	svc := NewService(mt)

	resp, err := svc.IssueConsent(context.Background(), &IssueConsentRequest{
		Subject:    "u-1",
		PolicyURI:  "https://example.com/policy",
		PolicyText: "We collect data",
		Purposes:   []Purpose{{Name: "analytics", Optional: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/consent/receipts" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.ID != "cr-1" {
		t.Fatalf("expected cr-1, got %s", resp.ID)
	}
}

func TestGetConsent(t *testing.T) {
	mt := &mockTransport{resp: &ConsentReceipt{ID: "cr-1"}}
	svc := NewService(mt)

	resp, err := svc.GetConsent(context.Background(), "cr-1")
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/consent/receipts/cr-1" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.ID != "cr-1" {
		t.Fatalf("expected cr-1, got %s", resp.ID)
	}
}

func TestListConsents(t *testing.T) {
	mt := &mockTransport{resp: &ConsentListResponse{Count: 5}}
	svc := NewService(mt)

	resp, err := svc.ListConsents(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/consent/receipts" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.Count != 5 {
		t.Fatalf("expected 5, got %d", resp.Count)
	}
}

func TestWithdrawConsent(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	err := svc.WithdrawConsent(context.Background(), "cr-1")
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "POST" || mt.path != "/v1/consent/receipts/cr-1/withdraw" {
		t.Fatalf("unexpected %s %s", mt.method, mt.path)
	}
}

func TestGetDIDCommIdentity(t *testing.T) {
	mt := &mockTransport{resp: &DIDCommIdentity{DID: "did:web:agent"}}
	svc := NewService(mt)

	resp, err := svc.GetDIDCommIdentity(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/didcomm/identity" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.DID != "did:web:agent" {
		t.Fatalf("expected did:web:agent, got %s", resp.DID)
	}
}

func TestSendDIDCommMessage(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	err := svc.SendDIDCommMessage(context.Background(), &DIDCommMessage{
		ID:   "msg-1",
		Type: "https://didcomm.org/basicmessage/2.0/message",
		From: "did:web:alice",
		To:   "did:web:bob",
		Body: map[string]string{"content": "hello"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "POST" || mt.path != "/v1/didcomm/inbox" {
		t.Fatalf("unexpected %s %s", mt.method, mt.path)
	}
	if mt.headers["Content-Type"] != "application/didcomm-plain+json" {
		t.Fatalf("expected didcomm content type, got %s", mt.headers["Content-Type"])
	}
}

func TestCreate_Error(t *testing.T) {
	mt := &mockTransport{err: fmt.Errorf("fail")}
	svc := NewService(mt)

	_, err := svc.Create(context.Background(), &CreateRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}
