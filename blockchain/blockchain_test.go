package blockchain

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

func TestAnchorCredential(t *testing.T) {
	mt := &mockTransport{resp: &AnchorResponse{TxHash: "0xabc", Status: "confirmed", CredentialIndex: 1}}
	svc := NewService(mt)

	resp, err := svc.AnchorCredential(context.Background(), &AnchorRequest{
		CredentialID:   "cred-1",
		IssuerDID:      "did:web:issuer",
		CredentialHash: "0x" + strings.Repeat("ab", 32),
	})
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "POST" || mt.path != "/blockchain/credentials/anchor" {
		t.Fatalf("unexpected %s %s", mt.method, mt.path)
	}
	if resp.TxHash != "0xabc" {
		t.Fatalf("expected 0xabc, got %s", resp.TxHash)
	}
}

func TestRevokeCredential(t *testing.T) {
	mt := &mockTransport{resp: &TxResponse{TxHash: "0xdef", Status: "confirmed"}}
	svc := NewService(mt)

	resp, err := svc.RevokeCredential(context.Background(), &RevokeCredentialRequest{
		CredentialID:    "cred-1",
		IssuerDID:       "did:web:issuer",
		CredentialIndex: 1,
		Reason:          "OFFBOARDING",
	})
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/blockchain/credentials/revoke" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.TxHash != "0xdef" {
		t.Fatalf("expected 0xdef, got %s", resp.TxHash)
	}
}

func TestBatchRevoke(t *testing.T) {
	mt := &mockTransport{resp: &TxResponse{TxHash: "0x123"}}
	svc := NewService(mt)

	resp, err := svc.BatchRevoke(context.Background(), &BatchRevokeRequest{
		IssuerDID:         "did:web:issuer",
		CredentialIndices: []int{1, 2, 3},
		Reason:            "POLICY_VIOLATION",
	})
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/blockchain/credentials/batch-revoke" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.TxHash != "0x123" {
		t.Fatalf("expected 0x123, got %s", resp.TxHash)
	}
}

func TestGetCredentialStatus(t *testing.T) {
	mt := &mockTransport{resp: &CredentialStatusResponse{Status: "ACTIVE", StatusCode: 0}}
	svc := NewService(mt)

	resp, err := svc.GetCredentialStatus(context.Background(), "did:web:issuer", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mt.path, "issuer_did=did:web:issuer") {
		t.Fatalf("expected issuer_did param, got %s", mt.path)
	}
	if resp.Status != "ACTIVE" {
		t.Fatalf("expected ACTIVE, got %s", resp.Status)
	}
}

func TestSuspendCredential(t *testing.T) {
	mt := &mockTransport{resp: &TxResponse{TxHash: "0xsus"}}
	svc := NewService(mt)

	resp, err := svc.SuspendCredential(context.Background(), &SuspendRequest{
		IssuerDID:       "did:web:issuer",
		CredentialIndex: 5,
		Reason:          "INVESTIGATION",
	})
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/blockchain/credentials/suspend" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.TxHash != "0xsus" {
		t.Fatalf("expected 0xsus, got %s", resp.TxHash)
	}
}

func TestUnsuspendCredential(t *testing.T) {
	mt := &mockTransport{resp: &TxResponse{TxHash: "0xuns"}}
	svc := NewService(mt)

	resp, err := svc.UnsuspendCredential(context.Background(), &UnsuspendRequest{
		IssuerDID:       "did:web:issuer",
		CredentialIndex: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/blockchain/credentials/unsuspend" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.TxHash != "0xuns" {
		t.Fatalf("expected 0xuns, got %s", resp.TxHash)
	}
}

func TestRegisterDID(t *testing.T) {
	mt := &mockTransport{resp: &RegisterDIDResponse{TxHash: "0xreg", DID: "did:web:new"}}
	svc := NewService(mt)

	resp, err := svc.RegisterDID(context.Background(), &RegisterDIDRequest{
		DID:         "did:web:new",
		DIDDocument: "0x" + strings.Repeat("cd", 32),
	})
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/blockchain/dids/register" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.DID != "did:web:new" {
		t.Fatalf("expected did:web:new, got %s", resp.DID)
	}
}

func TestResolveDID(t *testing.T) {
	mt := &mockTransport{resp: &ResolvedDID{DID: "did:web:x", Active: true}}
	svc := NewService(mt)

	resp, err := svc.ResolveDID(context.Background(), "did:web:x")
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/blockchain/dids/did:web:x" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if !resp.Active {
		t.Fatal("expected active")
	}
}

func TestDeactivateDID(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	err := svc.DeactivateDID(context.Background(), "did:web:x")
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "DELETE" || mt.path != "/blockchain/dids/did:web:x" {
		t.Fatalf("unexpected %s %s", mt.method, mt.path)
	}
}

func TestReactivateDID(t *testing.T) {
	mt := &mockTransport{resp: &TxResponse{TxHash: "0xre"}}
	svc := NewService(mt)

	resp, err := svc.ReactivateDID(context.Background(), "did:web:x")
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/blockchain/dids/did:web:x/reactivate" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.TxHash != "0xre" {
		t.Fatalf("expected 0xre, got %s", resp.TxHash)
	}
}

func TestRegisterIssuer(t *testing.T) {
	mt := &mockTransport{resp: &TrustedIssuer{DID: "did:web:issuer", TrustLevel: 2}}
	svc := NewService(mt)

	resp, err := svc.RegisterIssuer(context.Background(), &RegisterIssuerRequest{
		DID:             "did:web:issuer",
		Name:            "Corp Issuer",
		CredentialTypes: []string{"EmployeeCredential"},
		TrustLevel:      2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/blockchain/trust/issuers" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.TrustLevel != 2 {
		t.Fatalf("expected trust level 2, got %d", resp.TrustLevel)
	}
}

func TestIsIssuerTrusted(t *testing.T) {
	mt := &mockTransport{resp: &IssuerTrustResponse{Trusted: true, TrustLevel: 3}}
	svc := NewService(mt)

	resp, err := svc.IsIssuerTrusted(context.Background(), "did:web:issuer", "EmployeeCredential")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mt.path, "credential_type=EmployeeCredential") {
		t.Fatalf("expected credential_type param, got %s", mt.path)
	}
	if !resp.Trusted {
		t.Fatal("expected trusted")
	}
}

func TestEvaluateTrust(t *testing.T) {
	mt := &mockTransport{resp: &EvaluateTrustResponse{Trusted: true, TrustLabel: "HIGH"}}
	svc := NewService(mt)

	resp, err := svc.EvaluateTrust(context.Background(), "did:web:issuer", "EmployeeCredential")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mt.path, "/blockchain/trust/evaluate") {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.TrustLabel != "HIGH" {
		t.Fatalf("expected HIGH, got %s", resp.TrustLabel)
	}
}

func TestGetTrustStats(t *testing.T) {
	mt := &mockTransport{resp: &TrustStats{IssuerCount: 42}}
	svc := NewService(mt)

	resp, err := svc.GetTrustStats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/blockchain/trust/stats" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.IssuerCount != 42 {
		t.Fatalf("expected 42, got %d", resp.IssuerCount)
	}
}

func TestStatus(t *testing.T) {
	mt := &mockTransport{resp: &BlockchainStatus{Connected: true, Network: "mainnet"}}
	svc := NewService(mt)

	resp, err := svc.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/blockchain/status" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if !resp.Connected {
		t.Fatal("expected connected")
	}
}

func TestAnchorCredential_Error(t *testing.T) {
	mt := &mockTransport{err: fmt.Errorf("rpc error")}
	svc := NewService(mt)

	_, err := svc.AnchorCredential(context.Background(), &AnchorRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetCredentialHash(t *testing.T) {
	mt := &mockTransport{resp: &CredentialHashResponse{CredentialHash: "0xhash"}}
	svc := NewService(mt)

	resp, err := svc.GetCredentialHash(context.Background(), "did:web:issuer", 2)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mt.path, "credential_index=2") {
		t.Fatalf("expected credential_index in path, got %s", mt.path)
	}
	if resp.CredentialHash != "0xhash" {
		t.Fatalf("expected 0xhash, got %s", resp.CredentialHash)
	}
}

func TestGetRevocationStatus(t *testing.T) {
	mt := &mockTransport{resp: &RevocationStatusResponse{Revoked: true}}
	svc := NewService(mt)

	resp, err := svc.GetRevocationStatus(context.Background(), "did:web:issuer", 3)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Revoked {
		t.Fatal("expected revoked=true")
	}
}

func TestGetRevocations(t *testing.T) {
	mt := &mockTransport{resp: &RevocationsResponse{RevokedIndices: []int{1, 2}, Count: 2}}
	svc := NewService(mt)

	resp, err := svc.GetRevocations(context.Background(), "did:web:issuer")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Count != 2 {
		t.Fatalf("expected 2, got %d", resp.Count)
	}
}

func TestBatchSuspend(t *testing.T) {
	mt := &mockTransport{resp: &TxResponse{TxHash: "0xbatch"}}
	svc := NewService(mt)

	resp, err := svc.BatchSuspend(context.Background(), &BatchSuspendRequest{
		IssuerDID:         "did:web:issuer",
		CredentialIndices: []int{1, 2},
		Reason:            "POLICY",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.TxHash != "0xbatch" {
		t.Fatalf("expected 0xbatch, got %s", resp.TxHash)
	}
}

func TestGetSuspensionStatus(t *testing.T) {
	mt := &mockTransport{resp: &SuspensionStatusResponse{Suspended: true}}
	svc := NewService(mt)

	resp, err := svc.GetSuspensionStatus(context.Background(), "did:web:issuer", 4)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Suspended {
		t.Fatal("expected suspended=true")
	}
}

func TestGetSuspensions(t *testing.T) {
	mt := &mockTransport{resp: &SuspensionsResponse{SuspendedIndices: []int{3}, Count: 1}}
	svc := NewService(mt)

	resp, err := svc.GetSuspensions(context.Background(), "did:web:issuer")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Count != 1 {
		t.Fatalf("expected 1, got %d", resp.Count)
	}
}

func TestMetadata(t *testing.T) {
	mt := &mockTransport{resp: &BlockchainMetadata{MaxBatchSize: 50, RevocationReasons: []string{"OFFBOARDING"}}}
	svc := NewService(mt)

	resp, err := svc.Metadata(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if resp.MaxBatchSize != 50 {
		t.Fatalf("expected MaxBatchSize 50, got %d", resp.MaxBatchSize)
	}
}

func TestGetIssuerInfo(t *testing.T) {
	mt := &mockTransport{resp: &IssuerInfo{Registered: true, CredentialCount: 5}}
	svc := NewService(mt)

	resp, err := svc.GetIssuerInfo(context.Background(), "did:web:corp")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Registered {
		t.Fatal("expected registered=true")
	}
	if !strings.Contains(mt.path, "did:web:corp") {
		t.Fatalf("expected DID in path, got %s", mt.path)
	}
}

func TestRegisterDIDMeta(t *testing.T) {
	mt := &mockTransport{resp: &RegisterDIDResponse{TxHash: "0xmeta", DID: "did:web:m"}}
	svc := NewService(mt)

	resp, err := svc.RegisterDIDMeta(context.Background(), &RegisterDIDMetaRequest{
		DID:         "did:web:m",
		DIDDocument: "0x" + strings.Repeat("ef", 32),
		Deadline:    9999999,
		Signature:   "sig",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.TxHash != "0xmeta" {
		t.Fatalf("expected 0xmeta, got %s", resp.TxHash)
	}
}

func TestUpdateDID(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	err := svc.UpdateDID(context.Background(), "did:web:x", &UpdateDIDRequest{DIDDocument: "0xdoc"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mt.path, "did:web:x") {
		t.Fatalf("expected DID in path, got %s", mt.path)
	}
}

func TestGetDIDStatus(t *testing.T) {
	mt := &mockTransport{resp: &DIDStatusResponse{DID: "did:web:x", Active: true}}
	svc := NewService(mt)

	resp, err := svc.GetDIDStatus(context.Background(), "did:web:x")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Active {
		t.Fatal("expected active")
	}
}

func TestUpdateIssuer(t *testing.T) {
	mt := &mockTransport{resp: &TrustedIssuer{DID: "did:web:issuer", TrustLevel: 3}}
	svc := NewService(mt)

	resp, err := svc.UpdateIssuer(context.Background(), "did:web:issuer", &UpdateIssuerRequest{TrustLevel: 3})
	if err != nil {
		t.Fatal(err)
	}
	if resp.TrustLevel != 3 {
		t.Fatalf("expected 3, got %d", resp.TrustLevel)
	}
}

func TestRevokeIssuer(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	if err := svc.RevokeIssuer(context.Background(), "did:web:issuer", &RevokeIssuerRequest{DID: "did:web:issuer", Reason: "OFFBOARDING"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mt.path, "did:web:issuer") {
		t.Fatalf("expected DID in path, got %s", mt.path)
	}
}

func TestGetIssuer(t *testing.T) {
	mt := &mockTransport{resp: &TrustedIssuer{DID: "did:web:issuer", Name: "Corp"}}
	svc := NewService(mt)

	resp, err := svc.GetIssuer(context.Background(), "did:web:issuer")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Name != "Corp" {
		t.Fatalf("expected Corp, got %s", resp.Name)
	}
}

func TestRegisterVerifier(t *testing.T) {
	mt := &mockTransport{resp: &TrustedVerifier{DID: "did:web:verifier", Name: "Verifier Corp"}}
	svc := NewService(mt)

	resp, err := svc.RegisterVerifier(context.Background(), &RegisterVerifierRequest{
		DID:           "did:web:verifier",
		Name:          "Verifier Corp",
		AcceptedTypes: []string{"EmployeeCredential"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.DID != "did:web:verifier" {
		t.Fatalf("expected did:web:verifier, got %s", resp.DID)
	}
	if mt.path != "/blockchain/trust/verifiers" {
		t.Fatalf("unexpected path %s", mt.path)
	}
}

func TestGetVerifier(t *testing.T) {
	mt := &mockTransport{resp: &TrustedVerifier{DID: "did:web:verifier", Name: "V"}}
	svc := NewService(mt)

	resp, err := svc.GetVerifier(context.Background(), "did:web:verifier")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Name != "V" {
		t.Fatalf("expected V, got %s", resp.Name)
	}
}

func TestRegisterSchema(t *testing.T) {
	mt := &mockTransport{resp: &TrustedSchema{SchemaID: "schema-1"}}
	svc := NewService(mt)

	resp, err := svc.RegisterSchema(context.Background(), &RegisterSchemaRequest{
		SchemaID:      "schema-1",
		Name:          "Employee Schema",
		SchemaVersion: "1.0",
		TrustLevel:    2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.SchemaID != "schema-1" {
		t.Fatalf("expected schema-1, got %s", resp.SchemaID)
	}
}

func TestGetSchema(t *testing.T) {
	mt := &mockTransport{resp: &TrustedSchema{SchemaID: "schema-1", Name: "Employee"}}
	svc := NewService(mt)

	resp, err := svc.GetSchema(context.Background(), "schema-1")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Name != "Employee" {
		t.Fatalf("expected Employee, got %s", resp.Name)
	}
}
