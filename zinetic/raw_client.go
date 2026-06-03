package zinetic

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"sdk.zinetic.net/apierr"
	itransport "sdk.zinetic.net/internal/transport"
)

type RawClient struct {
	config    *Config
	tcfg      *itransport.Config
	transport *itransport.Transport
}

func NewRawClient(opts ...Option) (*RawClient, error) {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	if err := cfg.ValidateRaw(); err != nil {
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
	return &RawClient{config: cfg, tcfg: tcfg, transport: t}, nil
}

func (c *RawClient) Do(ctx context.Context, method, path string, body interface{}, result interface{}, headers map[string]string) error {
	if err := c.requireTenantForScopedPath(path); err != nil {
		return err
	}
	return c.transport.DoWithHeaders(ctx, method, path, body, result, headers)
}

func (c *RawClient) DoRaw(ctx context.Context, method, path string, body interface{}, headers map[string]string) ([]byte, error) {
	if err := c.requireTenantForScopedPath(path); err != nil {
		return nil, err
	}
	return c.transport.DoRaw(ctx, method, path, body, headers)
}

func (c *RawClient) DoStream(ctx context.Context, method, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	if err := c.requireTenantForScopedPath(path); err != nil {
		return nil, err
	}
	return c.transport.DoStream(ctx, method, path, body, headers)
}

func (c *RawClient) SetAccessToken(token string) {
	c.config.AccessToken = token
	c.tcfg.AccessToken = token
	c.transport.SetAccessToken(token)
}

func (c *RawClient) SetRefreshToken(token string) {
	c.config.RefreshToken = token
	c.tcfg.RefreshToken = token
	c.transport.SetRefreshToken(token)
}

func (c *RawClient) GetConfig() *Config {
	return c.config
}

func (c *RawClient) requireTenantForScopedPath(requestPath string) error {
	if strings.TrimSpace(c.config.TenantID) != "" || IsTenantOptionalPath(requestPath) {
		return nil
	}
	return &apierr.APIError{
		Code:       apierr.ErrCodeValidation,
		HTTPStatus: http.StatusBadRequest,
		Message:    fmt.Sprintf("tenant ID is required for scoped route %s", normalizeRouteForTenantCheck(requestPath)),
	}
}

func IsTenantOptionalPath(requestPath string) bool {
	path := normalizeRouteForTenantCheck(requestPath)
	if path == "" || path == "/" {
		return true
	}
	for _, prefix := range []string{
		"/health",
		"/ready",
		"/version",
		"/metrics",
		"/docs",
		"/.well-known",
		"/auth",
		"/oauth",
		"/cli/device",
		"/dpop",
		"/me/tenants",
		"/tenants",
		"/users/me",
		"/scim",
		"/oid4vci",
		"/pam",
	} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

func normalizeRouteForTenantCheck(requestPath string) string {
	path := strings.TrimSpace(requestPath)
	if path == "" {
		return "/"
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if path == "/api/v1" {
		return "/"
	}
	path = strings.TrimPrefix(path, "/api/v1")
	if path == "" {
		return "/"
	}
	if path == "/v1" {
		return "/"
	}
	path = strings.TrimPrefix(path, "/v1")
	if path == "" {
		return "/"
	}
	return path
}
