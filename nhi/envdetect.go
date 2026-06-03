package nhi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Environment int

const (
	EnvUnknown Environment = iota
	EnvGitHubActions
	EnvGitLabCI
	EnvKubernetes
	EnvAWS
	EnvGCP
	EnvLocal
)

func (e Environment) String() string {
	switch e {
	case EnvGitHubActions:
		return "github-actions"
	case EnvGitLabCI:
		return "gitlab-ci"
	case EnvKubernetes:
		return "kubernetes"
	case EnvAWS:
		return "aws"
	case EnvGCP:
		return "gcp"
	case EnvLocal:
		return "local"
	default:
		return "unknown"
	}
}

const (
	k8sTokenPath    = "/var/run/secrets/kubernetes.io/serviceaccount/token" // #nosec G101 -- Kubernetes service-account token mount path, not a hardcoded credential value.
	awsIMDSTokenURL = "http://169.254.169.254/latest/api/token"             // #nosec G101 -- AWS IMDS token endpoint URL, not a hardcoded credential value.
	gcpMetadataURL  = "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/email"
	probeTimeout    = 2 * time.Second
	fetchTimeout    = 10 * time.Second
)

var envHTTPClient = &http.Client{
	Timeout: fetchTimeout,
	Transport: &http.Transport{
		DialContext:         (&net.Dialer{Timeout: 3 * time.Second}).DialContext,
		TLSHandshakeTimeout: 3 * time.Second,
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
	},
}

func DetectEnvironment(ctx context.Context) Environment {
	if isGitHubActions() {
		return EnvGitHubActions
	}
	if isGitLabCI() {
		return EnvGitLabCI
	}
	if isExplicitAWS() {
		return EnvAWS
	}
	if isKubernetes() {
		return EnvKubernetes
	}
	if isAWS(ctx) {
		return EnvAWS
	}
	if isGCP(ctx) {
		return EnvGCP
	}
	return EnvLocal
}

func FetchAttestationToken(ctx context.Context, env Environment, audience string) (string, error) {
	switch env {
	case EnvGitHubActions:
		return fetchGitHubActionsToken(ctx, audience)
	case EnvGitLabCI:
		return fetchGitLabCIToken()
	case EnvKubernetes:
		return fetchKubernetesToken()
	case EnvAWS:
		return fetchAWSIdentityDocument(ctx)
	case EnvGCP:
		return fetchGCPIdentityToken(ctx, audience)
	case EnvLocal:
		return fetchLocalToken()
	default:
		return "", fmt.Errorf("unsupported environment: %s", env)
	}
}

func isGitHubActions() bool {
	return os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL") != "" &&
		os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN") != ""
}

func isGitLabCI() bool {
	return strings.EqualFold(os.Getenv("GITLAB_CI"), "true") ||
		os.Getenv("CI_SERVER_URL") != "" ||
		os.Getenv("CI_PROJECT_PATH") != ""
}

func isKubernetes() bool {
	_, err := os.Stat(k8sTokenPath)
	return err == nil
}

func isExplicitAWS() bool {
	return os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE") != "" ||
		os.Getenv("AWS_CONTAINER_CREDENTIALS_FULL_URI") != "" ||
		os.Getenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI") != "" ||
		os.Getenv("ECS_CONTAINER_METADATA_URI_V4") != "" ||
		os.Getenv("ECS_CONTAINER_METADATA_URI") != "" ||
		os.Getenv("AWS_EXECUTION_ENV") != ""
}

func isAWS(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, awsIMDSTokenURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", "21600")

	resp, err := envHTTPClient.Do(req)
	if err != nil {
		return false
	}
	ok := resp.StatusCode == http.StatusOK
	if err := resp.Body.Close(); err != nil {
		return false
	}
	return ok
}

func isGCP(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, gcpMetadataURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := envHTTPClient.Do(req)
	if err != nil {
		return false
	}
	ok := resp.StatusCode == http.StatusOK
	if err := resp.Body.Close(); err != nil {
		return false
	}
	return ok
}

func fetchGitLabCIToken() (string, error) {
	for _, name := range []string{
		"ZINETIC_OIDC_TOKEN",
		"ZINETIC_ID_TOKEN",
		"ZINETIC_GITLAB_OIDC_TOKEN",
		"CI_JOB_JWT_V2",
		"CI_JOB_JWT",
	} {
		if token := strings.TrimSpace(os.Getenv(name)); token != "" {
			return token, nil
		}
	}
	return "", fmt.Errorf("GitLab CI OIDC token not found; configure id_tokens: ZINETIC_ID_TOKEN with aud set to the Zinetic backend URL")
}

func fetchGitHubActionsToken(ctx context.Context, audience string) (string, error) {
	requestURL := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL")
	requestToken := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")
	if requestURL == "" || requestToken == "" {
		return "", fmt.Errorf("GitHub Actions OIDC env vars not set")
	}

	u, err := url.Parse(requestURL)
	if err != nil {
		return "", fmt.Errorf("parse OIDC request URL: %w", err)
	}
	if err := validateGitHubActionsOIDCURL(u); err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("audience", audience)
	u.RawQuery = q.Encode()

	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil) // #nosec G107 G704 -- URL is validated to GitHub Actions OIDC before use.
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "bearer "+requestToken)

	resp, err := envHTTPClient.Do(req) // #nosec G107 G704 -- request URL is validated to GitHub Actions OIDC before use.
	if err != nil {
		return "", fmt.Errorf("fetch GitHub OIDC token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("GitHub OIDC token request failed (HTTP %d): %s", resp.StatusCode, body)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if err != nil {
		return "", fmt.Errorf("read GitHub OIDC response: %w", err)
	}
	return extractJSONValue(body, "value")
}

func validateGitHubActionsOIDCURL(u *url.URL) error {
	host := strings.ToLower(u.Hostname())
	if u.Scheme != "https" {
		return fmt.Errorf("GitHub OIDC request URL must use https")
	}
	if u.User != nil {
		return fmt.Errorf("GitHub OIDC request URL must not include user info")
	}
	if host != "actions.githubusercontent.com" && !strings.HasSuffix(host, ".actions.githubusercontent.com") {
		return fmt.Errorf("GitHub OIDC request URL host is not trusted")
	}
	return nil
}

func fetchKubernetesToken() (string, error) {
	data, err := os.ReadFile(k8sTokenPath)
	if err != nil {
		return "", fmt.Errorf("read Kubernetes service account token: %w", err)
	}
	token := string(data)
	if token == "" {
		return "", fmt.Errorf("empty Kubernetes service account token")
	}
	return token, nil
}

func fetchAWSIdentityDocument(ctx context.Context) (string, error) {
	if path := strings.TrimSpace(os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE")); path != "" {
		return fetchAWSWebIdentityToken(path)
	}
	if hasECSCredentialsEndpoint() {
		return fetchECSIdentityEnvelope(ctx)
	}

	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	tokenReq, err := http.NewRequestWithContext(ctx, http.MethodPut, awsIMDSTokenURL, nil)
	if err != nil {
		return "", fmt.Errorf("create IMDS token request: %w", err)
	}
	tokenReq.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", "21600")

	tokenResp, err := envHTTPClient.Do(tokenReq)
	if err != nil {
		return "", fmt.Errorf("fetch IMDS session token: %w", err)
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("IMDS token request failed (HTTP %d)", tokenResp.StatusCode)
	}
	imdsToken, err := io.ReadAll(io.LimitReader(tokenResp.Body, 4096))
	if err != nil {
		return "", fmt.Errorf("read IMDS token: %w", err)
	}

	identityReq, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"http://169.254.169.254/latest/dynamic/instance-identity/document", nil)
	if err != nil {
		return "", fmt.Errorf("create identity document request: %w", err)
	}
	identityReq.Header.Set("X-aws-ec2-metadata-token", string(imdsToken))

	identityResp, err := envHTTPClient.Do(identityReq)
	if err != nil {
		return "", fmt.Errorf("fetch instance identity document: %w", err)
	}
	defer identityResp.Body.Close()

	if identityResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("identity document request failed (HTTP %d)", identityResp.StatusCode)
	}
	doc, err := io.ReadAll(io.LimitReader(identityResp.Body, 8192))
	if err != nil {
		return "", fmt.Errorf("read identity document: %w", err)
	}

	pkcs7Req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"http://169.254.169.254/latest/dynamic/instance-identity/pkcs7", nil)
	if err != nil {
		return "", fmt.Errorf("create PKCS7 request: %w", err)
	}
	pkcs7Req.Header.Set("X-aws-ec2-metadata-token", string(imdsToken))

	pkcs7Resp, err := envHTTPClient.Do(pkcs7Req)
	if err != nil {
		return "", fmt.Errorf("fetch PKCS7 signature: %w", err)
	}
	defer pkcs7Resp.Body.Close()

	if pkcs7Resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("PKCS7 request failed (HTTP %d)", pkcs7Resp.StatusCode)
	}
	sig, err := io.ReadAll(io.LimitReader(pkcs7Resp.Body, 8192))
	if err != nil {
		return "", fmt.Errorf("read PKCS7 signature: %w", err)
	}

	envelope := map[string]string{
		"document":  string(doc),
		"signature": string(sig),
	}
	envelopeBytes, err := json.Marshal(envelope)
	if err != nil {
		return "", fmt.Errorf("marshal AWS attestation envelope: %w", err)
	}
	return string(envelopeBytes), nil
}

func fetchAWSWebIdentityToken(path string) (string, error) {
	data, err := readRuntimeTokenFile(path)
	if err != nil {
		return "", fmt.Errorf("read AWS web identity token file: %w", err)
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", fmt.Errorf("AWS web identity token file is empty")
	}
	return token, nil
}

func readRuntimeTokenFile(path string) ([]byte, error) {
	cleaned := filepath.Clean(strings.TrimSpace(path))
	if !filepath.IsAbs(cleaned) {
		return nil, fmt.Errorf("runtime token file path must be absolute")
	}
	root, err := os.OpenRoot(string(os.PathSeparator))
	if err != nil {
		return nil, fmt.Errorf("open filesystem root: %w", err)
	}
	defer func() {
		_ = root.Close()
	}()
	return root.ReadFile(strings.TrimPrefix(cleaned, string(os.PathSeparator)))
}

func hasECSCredentialsEndpoint() bool {
	return strings.TrimSpace(os.Getenv("AWS_CONTAINER_CREDENTIALS_FULL_URI")) != "" ||
		strings.TrimSpace(os.Getenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI")) != ""
}

func fetchECSIdentityEnvelope(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	credentialsURL, err := ecsCredentialsURL()
	if err != nil {
		return "", err
	}
	credsRaw, err := fetchECSJSON(ctx, credentialsURL, 16384)
	if err != nil {
		return "", fmt.Errorf("fetch ECS task credentials: %w", err)
	}
	var creds awsCredentials
	if err := json.Unmarshal(credsRaw, &creds); err != nil {
		return "", fmt.Errorf("decode ECS task credentials: %w", err)
	}
	attestation, err := buildAWSSTSAttestation(creds, firstNonEmptyEnv("AWS_REGION", "AWS_DEFAULT_REGION"), "ecs", time.Now())
	if err != nil {
		return "", err
	}
	return attestation, nil
}

func ecsCredentialsURL() (string, error) {
	if raw := strings.TrimSpace(os.Getenv("AWS_CONTAINER_CREDENTIALS_FULL_URI")); raw != "" {
		u, err := url.Parse(raw)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return "", fmt.Errorf("AWS_CONTAINER_CREDENTIALS_FULL_URI must be an absolute URL")
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return "", fmt.Errorf("AWS_CONTAINER_CREDENTIALS_FULL_URI must use http or https")
		}
		return raw, nil
	}
	relative := strings.TrimSpace(os.Getenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI"))
	if relative == "" {
		return "", fmt.Errorf("ECS credentials endpoint is not configured")
	}
	if !strings.HasPrefix(relative, "/") {
		return "", fmt.Errorf("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI must start with /")
	}
	return "http://169.254.170.2" + relative, nil
}

func fetchECSJSON(ctx context.Context, rawURL string, limit int64) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil) // #nosec G107 -- ECS metadata URL is supplied by AWS runtime environment.
	if err != nil {
		return nil, fmt.Errorf("create ECS request: %w", err)
	}
	if token := strings.TrimSpace(os.Getenv("AWS_CONTAINER_AUTHORIZATION_TOKEN")); token != "" {
		req.Header.Set("Authorization", token)
	}
	resp, err := envHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ECS metadata request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ECS metadata returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	if err != nil {
		return nil, fmt.Errorf("read ECS metadata: %w", err)
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, fmt.Errorf("ECS metadata response is empty")
	}
	return json.RawMessage(body), nil
}

func firstNonEmptyEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}

func fetchGCPIdentityToken(ctx context.Context, audience string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	u := fmt.Sprintf(
		"http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity?audience=%s&format=full",
		url.QueryEscape(audience),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", fmt.Errorf("create GCP metadata request: %w", err)
	}
	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := envHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch GCP identity token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("GCP identity token request failed (HTTP %d): %s", resp.StatusCode, body)
	}

	token, err := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if err != nil {
		return "", fmt.Errorf("read GCP identity token: %w", err)
	}
	if len(token) == 0 {
		return "", fmt.Errorf("empty GCP identity token")
	}
	return string(token), nil
}

func extractJSONValue(data []byte, key string) (string, error) {
	var m map[string]interface{}
	if err := jsonUnmarshal(data, &m); err != nil {
		return "", fmt.Errorf("decode JSON response: %w", err)
	}
	v, ok := m[key]
	if !ok {
		return "", fmt.Errorf("key %q not found in response", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("key %q is not a string", key)
	}
	if s == "" {
		return "", fmt.Errorf("key %q is empty", key)
	}
	return s, nil
}

func fetchLocalToken() (string, error) {
	token := os.Getenv("ZINETIC_ACCESS_TOKEN")
	if token == "" {
		return "", fmt.Errorf("local environment requires ZINETIC_ACCESS_TOKEN env var; run 'zin auth login' first")
	}
	return token, nil
}
