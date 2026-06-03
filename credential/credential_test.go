package credential

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
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

func TestAnchor_Success(t *testing.T) {
	mt := &mockTransport{
		result: &AnchorResponse{
			AnchorID:   "anchor-123",
			Status:     "active",
			AnchoredAt: time.Now(),
		},
	}
	svc := NewService(mt)

	resp, err := svc.Anchor(context.Background(), &AnchorRequest{
		TenantID:     "tenant-1",
		SubjectID:    "sub-1",
		CredentialID: "cred-1",
		PublicKey:    "pk-jwk-data",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AnchorID != "anchor-123" {
		t.Fatalf("expected anchor-123, got %s", resp.AnchorID)
	}
	if mt.method != "POST" {
		t.Fatalf("expected POST, got %s", mt.method)
	}
	if mt.path != "/v1/credentials/anchor" {
		t.Fatalf("expected /v1/credentials/anchor, got %s", mt.path)
	}
}

func TestAnchor_MissingTenantID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.Anchor(context.Background(), &AnchorRequest{
		SubjectID:    "sub-1",
		CredentialID: "cred-1",
		PublicKey:    "pk",
	})
	if err == nil {
		t.Fatal("expected error for missing tenant_id")
	}
}

func TestAnchor_MissingSubjectID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.Anchor(context.Background(), &AnchorRequest{
		TenantID:     "tenant-1",
		CredentialID: "cred-1",
		PublicKey:    "pk",
	})
	if err == nil {
		t.Fatal("expected error for missing subject_id")
	}
}

func TestAnchor_MissingCredentialID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.Anchor(context.Background(), &AnchorRequest{
		TenantID:  "tenant-1",
		SubjectID: "sub-1",
		PublicKey: "pk",
	})
	if err == nil {
		t.Fatal("expected error for missing credential_id")
	}
}

func TestAnchor_MissingPublicKey(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.Anchor(context.Background(), &AnchorRequest{
		TenantID:     "tenant-1",
		SubjectID:    "sub-1",
		CredentialID: "cred-1",
	})
	if err == nil {
		t.Fatal("expected error for missing public_key")
	}
}

func TestStatus_Success(t *testing.T) {
	mt := &mockTransport{
		result: &StatusResponse{
			AnchorID: "anchor-123",
			Status:   "active",
		},
	}
	svc := NewService(mt)

	resp, err := svc.Status(context.Background(), "anchor-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AnchorID != "anchor-123" {
		t.Fatalf("expected anchor-123, got %s", resp.AnchorID)
	}
	if mt.method != "GET" {
		t.Fatalf("expected GET, got %s", mt.method)
	}
}

func TestStatus_EmptyAnchorID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.Status(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty anchor_id")
	}
}

func TestRevoke_Success(t *testing.T) {
	mt := &mockTransport{
		result: &RevokeResponse{
			RevokedAt:    time.Now(),
			AuditEventID: "audit-789",
		},
	}
	svc := NewService(mt)

	resp, err := svc.Revoke(context.Background(), &RevokeRequest{
		AnchorID: "anchor-123",
		Reason:   "compromised key",
		Actor:    "admin@zinetic.io",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AuditEventID != "audit-789" {
		t.Fatalf("expected audit-789, got %s", resp.AuditEventID)
	}
}

func TestRevoke_MissingAnchorID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.Revoke(context.Background(), &RevokeRequest{
		Reason: "reason",
		Actor:  "actor",
	})
	if err == nil {
		t.Fatal("expected error for missing anchor_id")
	}
}

func TestRevoke_MissingReason(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.Revoke(context.Background(), &RevokeRequest{
		AnchorID: "anchor-123",
		Actor:    "actor",
	})
	if err == nil {
		t.Fatal("expected error for missing reason")
	}
}

func TestRevoke_MissingActor(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.Revoke(context.Background(), &RevokeRequest{
		AnchorID: "anchor-123",
		Reason:   "reason",
	})
	if err == nil {
		t.Fatal("expected error for missing actor")
	}
}

func TestAnchor_TransportError(t *testing.T) {
	mt := &mockTransport{
		err: fmt.Errorf("network timeout"),
	}
	svc := NewService(mt)

	_, err := svc.Anchor(context.Background(), &AnchorRequest{
		TenantID:     "tenant-1",
		SubjectID:    "sub-1",
		CredentialID: "cred-1",
		PublicKey:    "pk",
	})
	if err == nil {
		t.Fatal("expected transport error to propagate")
	}
}
