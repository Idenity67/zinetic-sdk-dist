package nhi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestDetectEnvironment_GitHub(t *testing.T) {
	clearCIEnv(t)
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", "https://token.actions.githubusercontent.com/request")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "test-token-123")

	env := DetectEnvironment(t.Context())
	if env != EnvGitHubActions {
		t.Fatalf("expected github-actions, got %s", env)
	}
}

func TestDetectEnvironment_GitLabCI(t *testing.T) {
	clearCIEnv(t)
	t.Setenv("GITLAB_CI", "true")
	t.Setenv("CI_PROJECT_PATH", "org/repo")

	env := DetectEnvironment(t.Context())
	if env != EnvGitLabCI {
		t.Fatalf("expected gitlab-ci, got %s", env)
	}
}

func TestDetectEnvironment_AWSExplicitRuntime(t *testing.T) {
	clearCIEnv(t)
	tokenFile := t.TempDir() + "/web-identity-token"
	if err := os.WriteFile(tokenFile, []byte("token"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", tokenFile)

	env := DetectEnvironment(t.Context())
	if env != EnvAWS {
		t.Fatalf("expected aws, got %s", env)
	}
}

func TestDetectEnvironment_Local(t *testing.T) {
	clearCIEnv(t)

	env := DetectEnvironment(t.Context())
	if env != EnvLocal {
		t.Fatalf("expected local, got %s", env)
	}
}

func clearCIEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"ACTIONS_ID_TOKEN_REQUEST_URL",
		"ACTIONS_ID_TOKEN_REQUEST_TOKEN",
		"GITLAB_CI",
		"CI_SERVER_URL",
		"CI_PROJECT_PATH",
		"AWS_WEB_IDENTITY_TOKEN_FILE",
		"AWS_CONTAINER_CREDENTIALS_FULL_URI",
		"AWS_CONTAINER_CREDENTIALS_RELATIVE_URI",
		"ECS_CONTAINER_METADATA_URI_V4",
		"ECS_CONTAINER_METADATA_URI",
		"AWS_EXECUTION_ENV",
	} {
		t.Setenv(name, "")
	}
}

func TestEnvironment_String(t *testing.T) {
	tests := []struct {
		env  Environment
		want string
	}{
		{EnvGitHubActions, "github-actions"},
		{EnvGitLabCI, "gitlab-ci"},
		{EnvKubernetes, "kubernetes"},
		{EnvAWS, "aws"},
		{EnvGCP, "gcp"},
		{EnvLocal, "local"},
		{EnvUnknown, "unknown"},
	}

	for _, tt := range tests {
		if got := tt.env.String(); got != tt.want {
			t.Errorf("Environment(%d).String() = %q, want %q", tt.env, got, tt.want)
		}
	}
}

func TestFetchAttestationToken_GitLabCI(t *testing.T) {
	t.Setenv("ZINETIC_ID_TOKEN", "gitlab-oidc-token")

	tok, err := FetchAttestationToken(t.Context(), EnvGitLabCI, "api://zinetic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "gitlab-oidc-token" {
		t.Fatalf("expected gitlab-oidc-token, got %q", tok)
	}
}

func TestFetchAttestationToken_GitLabCIMissingToken(t *testing.T) {
	for _, name := range []string{"ZINETIC_OIDC_TOKEN", "ZINETIC_ID_TOKEN", "ZINETIC_GITLAB_OIDC_TOKEN", "CI_JOB_JWT_V2", "CI_JOB_JWT"} {
		t.Setenv(name, "")
	}
	if _, err := FetchAttestationToken(t.Context(), EnvGitLabCI, "api://zinetic"); err == nil {
		t.Fatal("expected error when GitLab OIDC token is missing")
	}
}

func TestFetchAttestationToken_AWSWebIdentity(t *testing.T) {
	path := t.TempDir() + "/token"
	if err := os.WriteFile(path, []byte(" web-identity-token \n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", path)

	tok, err := FetchAttestationToken(t.Context(), EnvAWS, "api://zinetic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "web-identity-token" {
		t.Fatalf("expected web identity token, got %q", tok)
	}
}

func TestFetchAttestationToken_ECS(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/creds":
			if got := r.Header.Get("Authorization"); got != "Bearer task-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Write([]byte(`{"AccessKeyId":"TEST_ACCESS_KEY_ID","SecretAccessKey":"TEST_SECRET_ACCESS_KEY","Token":"task-token","RoleArn":"arn:aws:iam::123:role/task"}`))
		case "/task":
			w.Write([]byte(`{"Cluster":"default","TaskARN":"arn:aws:ecs:task/abc"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	t.Setenv("AWS_CONTAINER_CREDENTIALS_FULL_URI", srv.URL+"/creds")
	t.Setenv("AWS_CONTAINER_AUTHORIZATION_TOKEN", "Bearer task-token")
	t.Setenv("ECS_CONTAINER_METADATA_URI_V4", srv.URL)

	raw, err := FetchAttestationToken(t.Context(), EnvAWS, "api://zinetic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(raw, `"source":"aws_sts"`) {
		t.Fatalf("expected AWS STS envelope, got %s", raw)
	}
	if !strings.Contains(raw, `"runtime":"ecs"`) || !strings.Contains(raw, "X-Amz-Signature") {
		t.Fatalf("expected signed ECS STS attestation, got %s", raw)
	}
}

func TestExtractJSONValue_Success(t *testing.T) {
	data, _ := json.Marshal(map[string]interface{}{"value": "tok-abc"})
	got, err := extractJSONValue(data, "value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "tok-abc" {
		t.Fatalf("expected tok-abc, got %q", got)
	}
}

func TestExtractJSONValue_MissingKey(t *testing.T) {
	data, _ := json.Marshal(map[string]interface{}{"other": "val"})
	if _, err := extractJSONValue(data, "value"); err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestExtractJSONValue_NonStringValue(t *testing.T) {
	data, _ := json.Marshal(map[string]interface{}{"value": 42})
	if _, err := extractJSONValue(data, "value"); err == nil {
		t.Fatal("expected error for non-string value")
	}
}

func TestExtractJSONValue_EmptyString(t *testing.T) {
	data, _ := json.Marshal(map[string]interface{}{"value": ""})
	if _, err := extractJSONValue(data, "value"); err == nil {
		t.Fatal("expected error for empty string value")
	}
}

func TestExtractJSONValue_InvalidJSON(t *testing.T) {
	if _, err := extractJSONValue([]byte("not json"), "value"); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestFetchLocalToken_Set(t *testing.T) {
	t.Setenv("ZINETIC_ACCESS_TOKEN", "local-token-xyz")
	tok, err := fetchLocalToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "local-token-xyz" {
		t.Fatalf("expected local-token-xyz, got %q", tok)
	}
}

func TestFetchLocalToken_NotSet(t *testing.T) {
	t.Setenv("ZINETIC_ACCESS_TOKEN", "")
	if _, err := fetchLocalToken(); err == nil {
		t.Fatal("expected error when ZINETIC_ACCESS_TOKEN not set")
	}
}

func TestFetchAttestationToken_LocalWithToken(t *testing.T) {
	t.Setenv("ZINETIC_ACCESS_TOKEN", "local-attest-tok")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_URL", "")
	t.Setenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN", "")

	tok, err := FetchAttestationToken(t.Context(), EnvLocal, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "local-attest-tok" {
		t.Fatalf("expected local-attest-tok, got %q", tok)
	}
}

func TestFetchAttestationToken_UnsupportedEnv(t *testing.T) {
	if _, err := FetchAttestationToken(t.Context(), EnvUnknown, ""); err == nil {
		t.Fatal("expected error for unsupported environment")
	}
}
