package nhi

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"sdk.zinetic.net/dpop"
)

type ProviderConfig struct {
	BackendURL             string
	Target                 string
	TenantID               string
	Audience               string
	Environment            Environment
	DPoPKey                *ecdsa.PrivateKey
	HTTPClient             *http.Client
	PolicyPublicKey        string
	RequirePolicySignature bool
	AllowPlaintextResponse bool
	HardwareMode           string
	HardwareProvider       HardwareKeyProvider
	RefreshThreshold       float64
	EventCallback          func(ProviderEvent)

	ExchangeTimeout time.Duration
	ProbeTimeout    time.Duration
}

const (
	HardwareModeAuto     = "auto"
	HardwareModeRequired = "required"
	HardwareModeOff      = "off"
)

type ProviderEventType string

const (
	EventExchangeSucceeded ProviderEventType = "exchange_succeeded"
	EventExchangeFailed    ProviderEventType = "exchange_failed"
	EventRenewalSucceeded  ProviderEventType = "renewal_succeeded"
	EventRenewalFailed     ProviderEventType = "renewal_failed"
)

type ProviderEvent struct {
	Type              ProviderEventType
	Target            string
	AuditID           string
	PolicyVersion     string
	TransactionHash   string
	LedgerAnchorHash  string
	ExpiresAt         time.Time
	Attempt           int
	Error             error
	HardwareMode      string
	HardwareAvailable bool
}

type Provider struct {
	cfg        ProviderConfig
	store      *store
	renewal    *renewal
	prover     *dpop.Prover
	env        Environment
	mu         sync.Mutex
	started    bool
	hardwareMu sync.Mutex
	hardware   *hardwareState
}

type hardwareState struct {
	key         *PublicKeyInfo
	attestation *AttestationDocument
	available   bool
}

func NewProvider(cfg ProviderConfig) (*Provider, error) {
	if cfg.BackendURL == "" {
		return nil, fmt.Errorf("nhi: backend URL is required")
	}
	if cfg.Target == "" {
		return nil, fmt.Errorf("nhi: target resource is required")
	}
	backendURL, err := normalizeBackendURL(cfg.BackendURL)
	if err != nil {
		return nil, fmt.Errorf("nhi: %w", err)
	}
	cfg.BackendURL = backendURL

	if cfg.DPoPKey == nil {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("nhi: generate ephemeral key: %w", err)
		}
		cfg.DPoPKey = key
	}

	if cfg.Audience == "" {
		cfg.Audience = cfg.BackendURL
	}
	if cfg.ExchangeTimeout == 0 {
		cfg.ExchangeTimeout = 30 * time.Second
	}
	if cfg.ProbeTimeout == 0 {
		cfg.ProbeTimeout = probeTimeout
	}
	if cfg.RequirePolicySignature && strings.TrimSpace(cfg.PolicyPublicKey) == "" {
		return nil, fmt.Errorf("nhi: policy signature verification requires PolicyPublicKey")
	}
	mode, err := normalizeHardwareMode(cfg.HardwareMode)
	if err != nil {
		return nil, fmt.Errorf("nhi: %w", err)
	}
	cfg.HardwareMode = mode
	if cfg.RefreshThreshold == 0 {
		cfg.RefreshThreshold = renewalThreshold
	}
	if cfg.RefreshThreshold <= 0 || cfg.RefreshThreshold >= 1 {
		return nil, fmt.Errorf("nhi: RefreshThreshold must be greater than 0 and less than 1")
	}

	p := &Provider{
		cfg:    cfg,
		store:  newStore(),
		prover: dpop.NewProver(cfg.DPoPKey),
	}

	p.renewal = newRenewal(p.store, p.doExchange, cfg.RefreshThreshold, p.emit)
	return p, nil
}

func (p *Provider) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.started {
		return nil
	}

	if p.env == EnvUnknown {
		env := p.cfg.Environment
		if env == EnvUnknown {
			detectCtx, cancel := context.WithTimeout(ctx, p.cfg.ProbeTimeout)
			env = DetectEnvironment(detectCtx)
			cancel()
		}
		p.env = env
	}

	cred, err := p.doExchange(ctx)
	if err != nil {
		p.emit(ProviderEvent{Type: EventExchangeFailed, Target: p.cfg.Target, Error: err, HardwareMode: p.cfg.HardwareMode})
		return fmt.Errorf("nhi: initial credential exchange: %w", err)
	}
	p.store.Set(cred)
	p.emit(credentialEvent(EventExchangeSucceeded, p.cfg.Target, cred, 0, nil, p.cfg.HardwareMode, p.hardwareAvailable()))
	p.renewal.Start()
	p.started = true
	return nil
}

func (p *Provider) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.started {
		return
	}
	p.renewal.Stop()
	p.store.Clear()
	p.started = false
}

func (p *Provider) GetCredential(key string) (string, error) {
	cred := p.store.Get()
	if cred == nil {
		return "", fmt.Errorf("nhi: no valid credential available")
	}
	v, ok := cred.Values[key]
	if !ok {
		return "", fmt.Errorf("nhi: credential key %q not found", key)
	}
	return v, nil
}

func (p *Provider) GetCredentials() (map[string]string, error) {
	cred := p.store.Get()
	if cred == nil {
		return nil, fmt.Errorf("nhi: no valid credential available")
	}
	result := make(map[string]string, len(cred.Values))
	for k, v := range cred.Values {
		result[k] = v
	}
	return result, nil
}

func (p *Provider) Metadata() (CredentialMetadata, error) {
	cred := p.store.Get()
	if cred == nil {
		return CredentialMetadata{}, fmt.Errorf("nhi: no valid credential available")
	}
	return cred.Metadata(), nil
}

func (p *Provider) GetPassword() (string, error) {
	return p.GetCredential("password")
}

func (p *Provider) GetToken() (string, error) {
	return p.GetCredential("token")
}

func (p *Provider) doExchange(ctx context.Context) (*Credential, error) {
	if _, ok := ctx.Deadline(); !ok && p.cfg.ExchangeTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.cfg.ExchangeTimeout)
		defer cancel()
	}

	hardware, err := p.prepareHardware(ctx)
	if err != nil {
		return nil, err
	}

	attestToken, err := FetchAttestationToken(ctx, p.env, p.cfg.Audience)
	if err != nil {
		return nil, fmt.Errorf("acquire attestation token: %w", err)
	}

	endpoint := p.exchangeEndpoint()
	encryptionKey, err := newEncryptionKey()
	if err != nil {
		return nil, err
	}

	body := exchangeRequest{
		Environment:             p.env.String(),
		AttestationToken:        attestToken,
		AttestationTokenType:    tokenTypeForEnv(p.env),
		Target:                  p.cfg.Target,
		TenantID:                p.cfg.TenantID,
		DPoPJWK:                 dpop.BuildJWKThumbprint(&p.cfg.DPoPKey.PublicKey),
		CredentialEncryptionJWK: encryptionKey.PublicJWK(),
		ExchangeVersion:         2,
		ClientMetadata:          collectClientMetadata(p.env),
		HardwareMode:            p.cfg.HardwareMode,
	}
	if hardware != nil && hardware.key != nil {
		body.HardwareKeyID = hardware.key.KeyID
		body.HardwareAttestation = hardwareAttestationPayload(hardware)
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal exchange request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create exchange request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	proof, err := p.prover.CreateProof(http.MethodPost, endpoint, "")
	if err != nil {
		return nil, fmt.Errorf("create DPoP proof: %w", err)
	}
	req.Header.Set("DPoP", proof)

	client := p.httpClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if err != nil {
		return nil, fmt.Errorf("read exchange response: %w", err)
	}

	if nonce := resp.Header.Get("DPoP-Nonce"); nonce != "" {
		p.prover.SetServerNonce(nonce)
		if resp.StatusCode == http.StatusUnauthorized {
			return p.retryWithNonce(ctx, payload, encryptionKey)
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("exchange failed (HTTP %d): %s", resp.StatusCode, respBody)
	}

	return p.decodeCredentialPackage(respBody, encryptionKey)
}

func (p *Provider) retryWithNonce(ctx context.Context, payload []byte, encryptionKey *encryptionKey) (*Credential, error) {
	endpoint := p.exchangeEndpoint()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create retry request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	proof, err := p.prover.CreateProof(http.MethodPost, endpoint, "")
	if err != nil {
		return nil, fmt.Errorf("create DPoP proof for retry: %w", err)
	}
	req.Header.Set("DPoP", proof)

	client := p.httpClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("retry request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if err != nil {
		return nil, fmt.Errorf("read retry response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("exchange retry failed (HTTP %d): %s", resp.StatusCode, respBody)
	}

	return p.decodeCredentialPackage(respBody, encryptionKey)
}

func (p *Provider) decodeCredentialPackage(respBody []byte, encryptionKey *encryptionKey) (*Credential, error) {
	var pkg exchangeResponse
	if err := json.Unmarshal(respBody, &pkg); err != nil {
		return nil, fmt.Errorf("decode exchange response: %w", err)
	}
	if pkg.EncryptedCredentials != nil {
		creds, err := encryptionKey.Decrypt(pkg.EncryptedCredentials)
		if err != nil {
			return nil, err
		}
		pkg.Credentials = creds
	} else if !p.cfg.AllowPlaintextResponse && p.env != EnvLocal {
		return nil, fmt.Errorf("exchange returned plaintext credentials; set AllowPlaintextResponse only for local development")
	}
	if len(pkg.Credentials) == 0 {
		return nil, fmt.Errorf("exchange returned empty credentials")
	}
	if err := verifyPolicySignature(&pkg, p.cfg.PolicyPublicKey, p.cfg.RequirePolicySignature); err != nil {
		return nil, err
	}
	return &Credential{
		Values:             pkg.Credentials,
		ExpiresAt:          pkg.ExpiresAt,
		TTLSeconds:         pkg.TTLSeconds,
		PolicySignature:    pkg.PolicySignature,
		PolicyVersion:      pkg.PolicyVersion,
		PolicySigningKeyID: pkg.PolicySigningKeyID,
		AuditID:            pkg.AuditID,
		TransactionHash:    pkg.TransactionHash,
		LedgerAnchorHash:   pkg.LedgerAnchorHash,
		Target:             p.cfg.Target,
	}, nil
}

func (p *Provider) httpClient() *http.Client {
	if p.cfg.HTTPClient != nil {
		return p.cfg.HTTPClient
	}
	return &http.Client{
		Timeout: p.cfg.ExchangeTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func (p *Provider) exchangeEndpoint() string {
	return p.cfg.BackendURL + "/api/v1/decision/exchange"
}

func normalizeBackendURL(rawBackendURL string) (string, error) {
	rawBackendURL = strings.TrimSpace(rawBackendURL)
	if rawBackendURL == "" {
		return "", fmt.Errorf("backend URL is required")
	}
	u, err := url.Parse(rawBackendURL)
	if err != nil || u.Scheme == "" || u.Hostname() == "" {
		return "", fmt.Errorf("backend URL must be an absolute URL")
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("backend URL must not include query or fragment")
	}
	if u.Scheme == "http" && !isLoopbackHost(u.Hostname()) {
		return "", fmt.Errorf("backend URL must use HTTPS except for localhost or loopback development URLs")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("backend URL must use HTTP or HTTPS")
	}

	path := strings.TrimRight(u.Path, "/")
	switch {
	case path == "/api/v1":
		path = ""
	case strings.HasSuffix(path, "/api/v1"):
		path = strings.TrimSuffix(path, "/api/v1")
	case path == "/v1":
		path = ""
	case strings.HasSuffix(path, "/v1"):
		path = strings.TrimSuffix(path, "/v1")
	}
	u.Path = strings.TrimRight(path, "/")
	u.RawPath = ""
	return strings.TrimRight(u.String(), "/"), nil
}

func isLoopbackHost(host string) bool {
	host = strings.Trim(strings.ToLower(host), "[]")
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

type exchangeRequest struct {
	Environment             string                 `json:"environment,omitempty"`
	AttestationToken        string                 `json:"attestation_token"`
	AttestationTokenType    string                 `json:"attestation_token_type,omitempty"`
	Target                  string                 `json:"target"`
	TenantID                string                 `json:"tenant_id,omitempty"`
	DPoPJWK                 map[string]interface{} `json:"dpop_jwk"`
	CredentialEncryptionJWK map[string]string      `json:"credential_encryption_jwk"`
	ExchangeVersion         int                    `json:"exchange_version"`
	ClientMetadata          *ClientMetadata        `json:"client_metadata,omitempty"`
	HardwareMode            string                 `json:"hardware_mode,omitempty"`
	HardwareKeyID           string                 `json:"hardware_key_id,omitempty"`
	ChallengeID             string                 `json:"challenge_id,omitempty"`
	ChallengeSignature      string                 `json:"challenge_signature,omitempty"`
	HardwareAttestation     map[string]interface{} `json:"hardware_attestation,omitempty"`
}

type ClientMetadata struct {
	Repository    string `json:"repo,omitempty"`
	Workflow      string `json:"workflow,omitempty"`
	RunID         string `json:"run_id,omitempty"`
	SHA           string `json:"sha,omitempty"`
	Pod           string `json:"pod,omitempty"`
	Namespace     string `json:"namespace,omitempty"`
	CloudProvider string `json:"cloud_provider,omitempty"`
	InstanceID    string `json:"instance_id,omitempty"`
	AccountID     string `json:"account_id,omitempty"`
	Region        string `json:"region,omitempty"`
}

type exchangeResponse struct {
	CredentialType       string                `json:"credential_type"`
	Credentials          map[string]string     `json:"credentials,omitempty"`
	EncryptedCredentials *EncryptedCredentials `json:"encrypted_credentials,omitempty"`
	ExpiresAt            time.Time             `json:"expires_at"`
	rawExpiresAt         string
	TTLSeconds           int    `json:"ttl_seconds"`
	PolicySignature      string `json:"policy_signature"`
	PolicyVersion        string `json:"policy_version,omitempty"`
	PolicySigningKeyID   string `json:"policy_signing_key_id,omitempty"`
	AuditID              string `json:"audit_id"`
	TransactionHash      string `json:"transaction_hash,omitempty"`
	LedgerAnchorHash     string `json:"ledger_anchor_hash,omitempty"`
}

func (r *exchangeResponse) UnmarshalJSON(data []byte) error {
	type wire exchangeResponse
	var decoded struct {
		*wire
		ExpiresAt json.RawMessage `json:"expires_at"`
	}
	decoded.wire = (*wire)(r)
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	if len(decoded.ExpiresAt) == 0 || string(decoded.ExpiresAt) == "null" {
		return nil
	}
	var raw string
	if err := json.Unmarshal(decoded.ExpiresAt, &raw); err != nil {
		return fmt.Errorf("decode expires_at: %w", err)
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return fmt.Errorf("parse expires_at: %w", err)
	}
	r.ExpiresAt = parsed
	r.rawExpiresAt = raw
	return nil
}

var jsonUnmarshal = json.Unmarshal

func tokenTypeForEnv(env Environment) string {
	switch env {
	case EnvGitHubActions, EnvGitLabCI:
		return "oidc"
	case EnvKubernetes:
		return "jwt"
	case EnvAWS:
		if strings.TrimSpace(os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE")) != "" {
			return "oidc"
		}
		if strings.TrimSpace(os.Getenv("AWS_CONTAINER_CREDENTIALS_FULL_URI")) != "" ||
			strings.TrimSpace(os.Getenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI")) != "" {
			return "aws_sts"
		}
		return "instance_identity"
	case EnvGCP:
		return "oidc"
	case EnvLocal:
		return "local_session"
	default:
		return "jwt"
	}
}

func collectClientMetadata(env Environment) *ClientMetadata {
	meta := &ClientMetadata{}
	switch env {
	case EnvGitHubActions:
		meta.Repository = os.Getenv("GITHUB_REPOSITORY")
		meta.Workflow = os.Getenv("GITHUB_WORKFLOW")
		meta.RunID = os.Getenv("GITHUB_RUN_ID")
		meta.SHA = os.Getenv("GITHUB_SHA")
	case EnvGitLabCI:
		meta.Repository = os.Getenv("CI_PROJECT_PATH")
		meta.Workflow = os.Getenv("CI_PIPELINE_SOURCE")
		meta.RunID = os.Getenv("CI_PIPELINE_ID")
		meta.SHA = os.Getenv("CI_COMMIT_SHA")
	case EnvKubernetes:
		meta.Pod = os.Getenv("HOSTNAME")
		if ns, err := os.ReadFile(k8sTokenPath[:len(k8sTokenPath)-len("/token")] + "/namespace"); err == nil {
			meta.Namespace = strings.TrimSpace(string(ns))
		}
	case EnvAWS:
		meta.CloudProvider = "aws"
		meta.Region = firstEnv("AWS_REGION", "AWS_DEFAULT_REGION")
	case EnvGCP:
		meta.CloudProvider = "gcp"
	}
	return meta
}

func (p *Provider) emit(event ProviderEvent) {
	if p.cfg.EventCallback != nil {
		if event.Target == "" {
			event.Target = p.cfg.Target
		}
		if event.HardwareMode == "" {
			event.HardwareMode = p.cfg.HardwareMode
		}
		if !event.HardwareAvailable {
			event.HardwareAvailable = p.hardwareAvailable()
		}
		p.cfg.EventCallback(event)
	}
}

func credentialEvent(eventType ProviderEventType, target string, cred *Credential, attempt int, err error, hardwareMode string, hardwareAvailable bool) ProviderEvent {
	event := ProviderEvent{
		Type:              eventType,
		Target:            target,
		Attempt:           attempt,
		Error:             err,
		HardwareMode:      hardwareMode,
		HardwareAvailable: hardwareAvailable,
	}
	if cred != nil {
		event.AuditID = cred.AuditID
		event.PolicyVersion = cred.PolicyVersion
		event.TransactionHash = cred.TransactionHash
		event.LedgerAnchorHash = cred.LedgerAnchorHash
		event.ExpiresAt = cred.ExpiresAt
	}
	return event
}

func normalizeHardwareMode(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", HardwareModeAuto:
		return HardwareModeAuto, nil
	case HardwareModeRequired:
		return HardwareModeRequired, nil
	case HardwareModeOff:
		return HardwareModeOff, nil
	default:
		return "", fmt.Errorf("HardwareMode must be one of auto, required, or off")
	}
}

func (p *Provider) prepareHardware(ctx context.Context) (*hardwareState, error) {
	if p.cfg.HardwareMode == HardwareModeOff {
		return nil, nil
	}
	if p.cfg.HardwareProvider == nil {
		if p.cfg.HardwareMode == HardwareModeRequired {
			return nil, ErrHardwareUnavailable
		}
		return nil, nil
	}
	if !p.cfg.HardwareProvider.Available(ctx) {
		if p.cfg.HardwareMode == HardwareModeRequired {
			return nil, ErrHardwareUnavailable
		}
		return nil, nil
	}

	p.hardwareMu.Lock()
	defer p.hardwareMu.Unlock()
	if p.hardware != nil && p.hardware.available {
		return p.hardware, nil
	}

	key, err := p.cfg.HardwareProvider.GenerateKey(ctx, KeyOptions{
		Algorithm:     KeyAlgECDSAP256,
		Label:         "zinetic-nhi-" + p.cfg.Target,
		NonExportable: true,
	})
	if err != nil {
		if p.cfg.HardwareMode == HardwareModeRequired {
			return nil, err
		}
		return nil, nil
	}
	attestation, err := p.cfg.HardwareProvider.Attest(ctx)
	if err != nil {
		if p.cfg.HardwareMode == HardwareModeRequired {
			return nil, err
		}
		return nil, nil
	}
	p.hardware = &hardwareState{key: key, attestation: attestation, available: true}
	return p.hardware, nil
}

func (p *Provider) hardwareAvailable() bool {
	p.hardwareMu.Lock()
	defer p.hardwareMu.Unlock()
	return p.hardware != nil && p.hardware.available
}

func hardwareAttestationPayload(state *hardwareState) map[string]interface{} {
	if state == nil {
		return nil
	}
	payload := map[string]interface{}{}
	if state.key != nil {
		payload["public_key"] = state.key
	}
	if state.attestation != nil {
		payload["attestation"] = state.attestation
	}
	return payload
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}
