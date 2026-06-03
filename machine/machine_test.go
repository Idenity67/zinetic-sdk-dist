package machine

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

func TestEnrollGitHub(t *testing.T) {
	mt := &mockTransport{resp: &GitHubIdentity{
		MachineIdentity: MachineIdentity{ID: "m-1", Type: "github"},
		RepoOwner:       "org",
		RepoName:        "repo",
	}}
	svc := NewService(mt)

	resp, err := svc.EnrollGitHub(context.Background(), &GitHubEnrollRequest{
		TenantID:   "t-1",
		Name:       "ci-identity",
		ApproverID: "admin-1",
		RepoOwner:  "org",
		RepoName:   "repo",
		RepoID:     "123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if mt.method != "POST" || mt.path != "/v1/machine/github/enroll" {
		t.Fatalf("unexpected %s %s", mt.method, mt.path)
	}
	if resp.ID != "m-1" {
		t.Fatalf("expected m-1, got %s", resp.ID)
	}
	if resp.RepoOwner != "org" {
		t.Fatalf("expected org, got %s", resp.RepoOwner)
	}
}

func TestMintGitHubToken(t *testing.T) {
	mt := &mockTransport{resp: &TokenResponse{AccessToken: "tok-gh", TokenType: "DPoP", ExpiresIn: 3600}}
	svc := NewService(mt)

	resp, err := svc.MintGitHubToken(context.Background(), &GitHubTokenRequest{
		RepoID:       "123",
		RepoOwner:    "org",
		RepoName:     "repo",
		WorkflowFile: "deploy.yml",
		WorkflowRef:  "refs/heads/main",
		RunnerName:   "ubuntu-latest",
		RunnerType:   "hosted",
		Branch:       "main",
		Commit:       "abc123",
		Actor:        "developer",
		EventName:    "push",
		OIDCToken:    "test-oidc-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/machine/github/tokens/mint" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.AccessToken != "tok-gh" {
		t.Fatalf("expected tok-gh, got %s", resp.AccessToken)
	}
}

func TestEnrollKubernetes(t *testing.T) {
	mt := &mockTransport{resp: &KubernetesIdentity{
		MachineIdentity: MachineIdentity{ID: "m-2", Type: "kubernetes"},
		ClusterID:       "cls-1",
		ClusterName:     "prod",
	}}
	svc := NewService(mt)

	resp, err := svc.EnrollKubernetes(context.Background(), &KubernetesEnrollRequest{
		TenantID:     "t-1",
		Name:         "prod-cluster",
		ApproverID:   "admin-1",
		ClusterID:    "cls-1",
		ClusterName:  "prod",
		APIServerURL: "https://k8s.internal:6443",
	})
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/machine/k8s/enroll" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.ClusterName != "prod" {
		t.Fatalf("expected prod, got %s", resp.ClusterName)
	}
}

func TestMintKubernetesToken(t *testing.T) {
	mt := &mockTransport{resp: &TokenResponse{AccessToken: "tok-k8s", TokenType: "DPoP", ExpiresIn: 900}}
	svc := NewService(mt)

	resp, err := svc.MintKubernetesToken(context.Background(), &KubernetesTokenRequest{
		ClusterID:         "cls-1",
		Namespace:         "default",
		ServiceAccount:    "api-svc",
		PodName:           "api-svc-7d4f5b-abc",
		PodUID:            "uid-123",
		ServiceAccountJWT: "test-service-account-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if mt.path != "/v1/machine/k8s/tokens/mint" {
		t.Fatalf("unexpected path %s", mt.path)
	}
	if resp.ExpiresIn != 900 {
		t.Fatalf("expected 900, got %d", resp.ExpiresIn)
	}
}

func TestEnrollGitHub_Error(t *testing.T) {
	mt := &mockTransport{err: fmt.Errorf("unauthorized")}
	svc := NewService(mt)

	_, err := svc.EnrollGitHub(context.Background(), &GitHubEnrollRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}
