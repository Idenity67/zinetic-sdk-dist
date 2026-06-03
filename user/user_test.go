package user

import (
	"context"
	"testing"
)

type mockTransport struct {
	method string
	path   string
	resp   *MeResponse
}

func (m *mockTransport) Do(_ context.Context, method, path string, _ interface{}, result interface{}) error {
	m.method = method
	m.path = path
	if out, ok := result.(*MeResponse); ok && m.resp != nil {
		*out = *m.resp
	}
	return nil
}

func TestMe(t *testing.T) {
	mt := &mockTransport{resp: &MeResponse{ID: "user-1", Email: "user@example.com"}}
	svc := NewService(mt)

	resp, err := svc.Me(context.Background())
	if err != nil {
		t.Fatalf("Me: %v", err)
	}
	if mt.method != "GET" || mt.path != "/v1/users/me" {
		t.Fatalf("request = %s %s, want GET /v1/users/me", mt.method, mt.path)
	}
	if resp.ID != "user-1" || resp.Email != "user@example.com" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}
