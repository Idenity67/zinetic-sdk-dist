package zinetic

import (
	"context"

	"sdk.zinetic.net/agent"
	"sdk.zinetic.net/audit"
	"sdk.zinetic.net/blockchain"
	"sdk.zinetic.net/credential"
	"sdk.zinetic.net/decision"
	"sdk.zinetic.net/device"
	"sdk.zinetic.net/did"
	"sdk.zinetic.net/governance"
	"sdk.zinetic.net/health"
	itransport "sdk.zinetic.net/internal/transport"
	"sdk.zinetic.net/machine"
	"sdk.zinetic.net/nhimgmt"
	"sdk.zinetic.net/notification"
	"sdk.zinetic.net/pam"
	"sdk.zinetic.net/policy"
	"sdk.zinetic.net/scim"
	"sdk.zinetic.net/secrets"
	"sdk.zinetic.net/tenant"
	"sdk.zinetic.net/token"
	"sdk.zinetic.net/user"
	"sdk.zinetic.net/webhook"
)

type Client struct {
	config    *Config
	tcfg      *itransport.Config
	transport *itransport.Transport

	Credentials   *credential.Service
	Decisions     *decision.Service
	Tokens        *token.Service
	Agents        *agent.Service
	Audit         *audit.Service
	Webhooks      *webhook.Service
	SCIM          *scim.Service
	Tenants       *tenant.Service
	Policies      *policy.Service
	Governance    *governance.Service
	PAM           *pam.Service
	Health        *health.Service
	Notifications *notification.Service
	Blockchain    *blockchain.Service
	DIDs          *did.Service
	Devices       *device.Service
	Secrets       *secrets.Service
	NHIManagement *nhimgmt.Service
	Machine       *machine.Service
	Users         *user.Service
}

type TransportInterface interface {
	Do(ctx context.Context, method, path string, body interface{}, result interface{}) error
	DoWithHeaders(ctx context.Context, method, path string, body interface{}, result interface{}, headers map[string]string) error
	BuildQueryURL(path string, params map[string]string) string
}

func NewClient(opts ...Option) (*Client, error) {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	tcfg := &itransport.Config{
		BaseURL:              cfg.BaseURL,
		APIBasePath:          effectiveAPIBasePath(cfg),
		BasePath:             cfg.BasePath,
		TenantID:             cfg.TenantID,
		HTTPClient:           cfg.HTTPClient,
		DPoPPrivateKey:       cfg.DPoPPrivateKey,
		AccessToken:          cfg.AccessToken,
		RefreshToken:         cfg.RefreshToken,
		AttestationToken:     cfg.AttestationToken,
		TokenEndpoint:        cfg.TokenEndpoint,
		TokenRefreshFormat:   cfg.TokenRefreshFormat,
		ClientID:             cfg.ClientID,
		ClientSecret:         cfg.ClientSecret,
		OnTokenRefreshed:     cfg.OnTokenRefreshed,
		MaxRetries:           cfg.MaxRetries,
		RetryBaseDelay:       cfg.RetryBaseDelay,
		RetryMaxDelay:        cfg.RetryMaxDelay,
		RequestTimeout:       cfg.RequestTimeout,
		MaxResponseBytes:     cfg.MaxResponseBytes,
		EnableTracing:        cfg.EnableTracing,
		ServiceName:          cfg.ServiceName,
		IdempotencyKeyPrefix: cfg.IdempotencyKeyPrefix,
		UserAgent:            cfg.UserAgent,
	}

	t := itransport.New(tcfg)

	c := &Client{
		config:    cfg,
		tcfg:      tcfg,
		transport: t,
	}

	c.Credentials = credential.NewService(t)
	c.Decisions = decision.NewService(t)
	c.Tokens = token.NewService(t)
	c.Agents = agent.NewService(t)
	c.Audit = audit.NewService(t)
	c.Webhooks = webhook.NewService(t)
	c.SCIM = scim.NewService(t)
	c.Tenants = tenant.NewService(t)
	c.Policies = policy.NewService(t)
	c.Governance = governance.NewService(t)
	c.PAM = pam.NewService(t)
	c.Health = health.NewService(t)
	c.Notifications = notification.NewService(t)
	c.Blockchain = blockchain.NewService(t)
	c.DIDs = did.NewService(t)
	c.Devices = device.NewService(t)
	c.Secrets = secrets.NewService(t)
	c.NHIManagement = nhimgmt.NewService(t)
	c.Machine = machine.NewService(t)
	c.Users = user.NewService(t)

	return c, nil
}

func (c *Client) SetAccessToken(token string) {
	c.config.AccessToken = token
	c.tcfg.AccessToken = token
	c.transport.SetAccessToken(token)
}

func (c *Client) SetRefreshToken(token string) {
	c.config.RefreshToken = token
	c.tcfg.RefreshToken = token
	c.transport.SetRefreshToken(token)
}

func (c *Client) DoRaw(ctx context.Context, method, path string, body interface{}, headers map[string]string) ([]byte, error) {
	return c.transport.DoRaw(ctx, method, path, body, headers)
}

func (c *Client) GetConfig() *Config {
	return c.config
}

func effectiveAPIBasePath(cfg *Config) string {
	if cfg.APIBasePath != "" {
		return cfg.APIBasePath
	}
	if cfg.BasePath != "" {
		return cfg.BasePath
	}
	return "/api/v1"
}
