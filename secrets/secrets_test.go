package secrets

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

func TestListMounts(t *testing.T) {
	mt := &mockTransport{resp: &ListMountsResponse{Mounts: []MountEngineStatus{{Path: "kv", EngineType: "kv"}}}}
	svc := NewService(mt)

	resp, err := svc.ListMounts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "GET" || mt.path != "/v1/secrets/sys/mounts" {
		t.Fatalf("unexpected %s %s", mt.method, mt.path)
	}
	if len(resp.Mounts) != 1 {
		t.Fatalf("expected 1 mount, got %d", len(resp.Mounts))
	}
}

func TestMount(t *testing.T) {
	mt := &mockTransport{resp: &MountEngineStatus{Path: "transit", EngineType: "transit"}}
	svc := NewService(mt)

	resp, err := svc.Mount(context.Background(), &MountEngineRequest{Path: "transit", EngineType: "transit"})
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "POST" || mt.path != "/v1/secrets/sys/mounts" {
		t.Fatalf("unexpected %s %s", mt.method, mt.path)
	}
	if resp.EngineType != "transit" {
		t.Fatalf("expected transit, got %s", resp.EngineType)
	}
}

func TestWrite(t *testing.T) {
	mt := &mockTransport{resp: &KVWriteResponse{Version: 1}}
	svc := NewService(mt)

	resp, err := svc.Write(context.Background(), "kv", "myapp/config", &KVWriteRequest{
		Data: map[string]string{"db_url": "postgres://localhost"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "PUT" || mt.path != "/v1/secrets/kv/data/myapp/config" {
		t.Fatalf("unexpected %s %s", mt.method, mt.path)
	}
	if resp.Version != 1 {
		t.Fatalf("expected version 1, got %d", resp.Version)
	}
}

func TestRead(t *testing.T) {
	mt := &mockTransport{resp: &KVReadResponse{Data: map[string]string{"key": "val"}, Version: 2}}
	svc := NewService(mt)

	resp, err := svc.Read(context.Background(), "kv", "myapp/config", 0)
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/secrets/kv/data/myapp/config" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.Version != 2 {
		t.Fatalf("expected version 2, got %d", resp.Version)
	}
}

func TestRead_WithVersion(t *testing.T) {
	mt := &mockTransport{resp: &KVReadResponse{Version: 3}}
	svc := NewService(mt)

	_, err := svc.Read(context.Background(), "kv", "myapp/config", 3)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mt.path, "version=3") {
		t.Fatalf("expected version param, got %s", mt.path)
	}
}

func TestDelete(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	err := svc.Delete(context.Background(), "kv", "myapp/old")
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "DELETE" || mt.path != "/v1/secrets/kv/data/myapp/old?reason=sdk requested secret deletion" {
		t.Fatalf("unexpected %s %s", mt.method, mt.path)
	}
}

func TestDeleteWithReason_SendsReason(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	err := svc.DeleteWithReason(context.Background(), "kv", "myapp/old", "cleanup")
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/secrets/kv/data/myapp/old?reason=cleanup" {
		t.Fatalf("unexpected path %s", mt.path)
	}
}

func TestWrite_EscapesSecretPathSegments(t *testing.T) {
	mt := &mockTransport{resp: &KVWriteResponse{Version: 1}}
	svc := NewService(mt)

	_, err := svc.Write(context.Background(), "kv prod", "app config/db#primary?rw", &KVWriteRequest{})
	if err != nil {
		t.Fatal(err)
	}
	want := "/v1/secrets/kv%20prod/data/app%20config/db%23primary%3Frw"
	if mt.path != want {
		t.Fatalf("expected %s, got %s", want, mt.path)
	}
}

func TestWrite_RejectsTraversalPath(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	if _, err := svc.Write(context.Background(), "kv", "app/../db", &KVWriteRequest{}); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
}

func TestEncrypt(t *testing.T) {
	mt := &mockTransport{resp: &TransitResponse{Result: "vault:v1:encrypted"}}
	svc := NewService(mt)

	resp, err := svc.Encrypt(context.Background(), "transit", &TransitEncryptRequest{
		Plaintext: "secret-data",
		KeyName:   "mykey",
	})
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/secrets/transit/encrypt" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.Result != "vault:v1:encrypted" {
		t.Fatalf("unexpected result %s", resp.Result)
	}
}

func TestDecrypt(t *testing.T) {
	mt := &mockTransport{resp: &TransitResponse{Result: "decrypted-data"}}
	svc := NewService(mt)

	resp, err := svc.Decrypt(context.Background(), "transit", &TransitDecryptRequest{
		Ciphertext: "vault:v1:encrypted",
		KeyName:    "mykey",
	})
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/secrets/transit/decrypt" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.Result != "decrypted-data" {
		t.Fatalf("unexpected result %s", resp.Result)
	}
}

func TestCreateLease(t *testing.T) {
	mt := &mockTransport{resp: &LeaseResponse{LeaseID: "lease-1", Renewable: true}}
	svc := NewService(mt)

	resp, err := svc.CreateLease(context.Background(), "db", "postgres/creds", &LeaseRequest{
		TTL:          "1h",
		IdentityID:   "svc-1",
		IdentityType: "service",
	})
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/secrets/db/lease/postgres/creds" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.LeaseID != "lease-1" {
		t.Fatalf("expected lease-1, got %s", resp.LeaseID)
	}
}

func TestRenewLease(t *testing.T) {
	mt := &mockTransport{resp: &LeaseResponse{LeaseID: "lease-1"}}
	svc := NewService(mt)

	resp, err := svc.RenewLease(context.Background(), &RenewLeaseRequest{LeaseID: "lease-1", Increment: "30m"})
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "PUT" || mt.path != "/v1/secrets/sys/leases/renew" {
		t.Fatalf("unexpected %s %s", mt.method, mt.path)
	}
	if resp.LeaseID != "lease-1" {
		t.Fatalf("expected lease-1, got %s", resp.LeaseID)
	}
}

func TestRevokeLease(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	err := svc.RevokeLease(context.Background(), &RevokeLeaseRequest{LeaseID: "lease-1"})
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "PUT" || mt.path != "/v1/secrets/sys/leases/revoke" {
		t.Fatalf("unexpected %s %s", mt.method, mt.path)
	}
}

func TestWrite_Error(t *testing.T) {
	mt := &mockTransport{err: fmt.Errorf("forbidden")}
	svc := NewService(mt)

	_, err := svc.Write(context.Background(), "kv", "x", &KVWriteRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}
