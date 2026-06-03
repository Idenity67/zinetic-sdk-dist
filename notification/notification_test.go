package notification

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

func TestSend(t *testing.T) {
	mt := &mockTransport{resp: &SendResponse{ID: "n-1", Status: "accepted"}}
	svc := NewService(mt)

	resp, err := svc.Send(context.Background(), &SendRequest{
		UserID:           "u-1",
		TenantID:         "t-1",
		NotificationType: "alert",
		Channel:          "push",
		Subject:          "Test",
		Body:             "Hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "POST" {
		t.Fatalf("expected POST, got %s", mt.method)
	}
	if mt.path != "/v1/notifications/send" {
		t.Fatalf("expected /v1/notifications/send, got %s", mt.path)
	}
	if resp.ID != "n-1" {
		t.Fatalf("expected id n-1, got %s", resp.ID)
	}
	if resp.Status != "accepted" {
		t.Fatalf("expected status accepted, got %s", resp.Status)
	}
}

func TestGet(t *testing.T) {
	mt := &mockTransport{resp: &Notification{ID: "n-1", Subject: "Hi"}}
	svc := NewService(mt)

	resp, err := svc.Get(context.Background(), "n-1")
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "GET" {
		t.Fatalf("expected GET, got %s", mt.method)
	}
	if mt.path != "/v1/notifications/n-1" {
		t.Fatalf("expected /v1/notifications/n-1, got %s", mt.path)
	}
	if resp.Subject != "Hi" {
		t.Fatalf("expected subject Hi, got %s", resp.Subject)
	}
}

func TestList(t *testing.T) {
	mt := &mockTransport{resp: &ListResponse{Count: 2}}
	svc := NewService(mt)

	resp, err := svc.List(context.Background(), "u-1", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "GET" {
		t.Fatalf("expected GET, got %s", mt.method)
	}
	if !strings.Contains(mt.path, "user_id=u-1") {
		t.Fatalf("expected user_id param, got %s", mt.path)
	}
	if resp.Count != 2 {
		t.Fatalf("expected count 2, got %d", resp.Count)
	}
}

func TestRegisterToken(t *testing.T) {
	mt := &mockTransport{resp: &RegisterTokenResponse{Status: "registered", Platform: "ios"}}
	svc := NewService(mt)

	resp, err := svc.RegisterToken(context.Background(), &RegisterTokenRequest{
		UserID:   "u-1",
		TenantID: "t-1",
		DeviceID: "d-1",
		Platform: "ios",
		Token:    "fcm-token-123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "POST" {
		t.Fatalf("expected POST, got %s", mt.method)
	}
	if mt.path != "/v1/push-tokens" {
		t.Fatalf("expected /v1/push-tokens, got %s", mt.path)
	}
	if resp.Status != "registered" {
		t.Fatalf("expected registered, got %s", resp.Status)
	}
}

func TestListTokens(t *testing.T) {
	mt := &mockTransport{resp: &ListTokensResponse{Tokens: []PushToken{{ID: "pt-1", Platform: "ios"}}}}
	svc := NewService(mt)

	resp, err := svc.ListTokens(context.Background(), "u-1")
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/push-tokens/users/u-1" {
		t.Fatalf("expected /v1/push-tokens/users/u-1, got %s", mt.path)
	}
	if len(resp.Tokens) != 1 {
		t.Fatalf("expected 1 token, got %d", len(resp.Tokens))
	}
}

func TestRevokeToken(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	err := svc.RevokeToken(context.Background(), "ios", "fcm-token-123")
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "DELETE" {
		t.Fatalf("expected DELETE, got %s", mt.method)
	}
	if mt.path != "/v1/push-tokens/ios/fcm-token-123" {
		t.Fatalf("expected /v1/push-tokens/ios/fcm-token-123, got %s", mt.path)
	}
}

func TestSend_Error(t *testing.T) {
	mt := &mockTransport{err: fmt.Errorf("network error")}
	svc := NewService(mt)

	_, err := svc.Send(context.Background(), &SendRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}
