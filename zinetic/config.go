package zinetic

import (
	"crypto/ecdsa"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"sdk.zinetic.net/apierr"
)

type Config struct {
	BaseURL     string
	APIBasePath string

	BasePath   string
	TenantID   string
	HTTPClient *http.Client

	DPoPPrivateKey *ecdsa.PrivateKey

	AccessToken      string
	RefreshToken     string
	AttestationToken string

	TokenEndpoint string

	TokenRefreshFormat string
	ClientID           string

	ClientSecret []byte

	OnTokenRefreshed func(newAccessToken, newRefreshToken string)

	MaxRetries     int
	RetryBaseDelay time.Duration
	RetryMaxDelay  time.Duration

	RequestTimeout   time.Duration
	MaxResponseBytes int64

	EnableTracing bool
	ServiceName   string

	IdempotencyKeyPrefix string

	UserAgent string
}

func DefaultConfig() *Config {
	return &Config{
		APIBasePath:        "/api/v1",
		MaxRetries:         3,
		RetryBaseDelay:     500 * time.Millisecond,
		RetryMaxDelay:      30 * time.Second,
		RequestTimeout:     30 * time.Second,
		MaxResponseBytes:   10 * 1024 * 1024,
		EnableTracing:      true,
		ServiceName:        "zinetic-sdk",
		UserAgent:          UserAgent,
		TokenRefreshFormat: "auto",
	}
}

func (c *Config) Validate() error {
	if err := c.ValidateRaw(); err != nil {
		return err
	}
	if c.TenantID == "" {
		return apierr.NewValidationError("tenant_id", "tenant ID is required")
	}
	return nil
}

func (c *Config) ValidateRaw() error {
	if c.BaseURL == "" {
		return apierr.NewValidationError("base_url", "base URL is required")
	}
	if !allowedBaseURL(c.BaseURL) {
		return apierr.NewValidationError("base_url", "base URL must use HTTPS except for localhost or loopback development URLs")
	}
	return nil
}

type Option func(*Config)

func WithBaseURL(url string) Option {
	return func(c *Config) {
		c.BaseURL = url
	}
}

func WithBasePath(path string) Option {
	return func(c *Config) {
		c.BasePath = path
		c.APIBasePath = path
	}
}

func WithAPIBasePath(path string) Option {
	return func(c *Config) {
		c.APIBasePath = path
	}
}

func WithTenantID(id string) Option {
	return func(c *Config) {
		c.TenantID = id
	}
}

func WithHTTPClient(hc *http.Client) Option {
	return func(c *Config) {
		c.HTTPClient = hc
	}
}

func WithDPoPKey(key *ecdsa.PrivateKey) Option {
	return func(c *Config) {
		c.DPoPPrivateKey = key
	}
}

func WithAccessToken(token string) Option {
	return func(c *Config) {
		c.AccessToken = token
	}
}

func WithRefreshToken(token string) Option {
	return func(c *Config) {
		c.RefreshToken = token
	}
}

func WithAttestationToken(token string) Option {
	return func(c *Config) {
		c.AttestationToken = token
	}
}

func WithClientCredentials(clientID string, clientSecret []byte) Option {
	return func(c *Config) {
		c.ClientID = clientID
		c.ClientSecret = clientSecret
	}
}

func WithTokenEndpoint(endpoint string) Option {
	return func(c *Config) {
		c.TokenEndpoint = endpoint
	}
}

func WithTokenRefreshFormat(format string) Option {
	return func(c *Config) {
		c.TokenRefreshFormat = format
	}
}

func WithTokenRefreshedCallback(fn func(newAccessToken, newRefreshToken string)) Option {
	return func(c *Config) {
		c.OnTokenRefreshed = fn
	}
}

func WithMaxRetries(n int) Option {
	return func(c *Config) {
		c.MaxRetries = n
	}
}

func WithRetryDelay(base, max time.Duration) Option {
	return func(c *Config) {
		c.RetryBaseDelay = base
		c.RetryMaxDelay = max
	}
}

func WithRequestTimeout(d time.Duration) Option {
	return func(c *Config) {
		c.RequestTimeout = d
	}
}

func WithMaxResponseBytes(n int64) Option {
	return func(c *Config) {
		c.MaxResponseBytes = n
	}
}

func WithTracing(enabled bool) Option {
	return func(c *Config) {
		c.EnableTracing = enabled
	}
}

func WithServiceName(name string) Option {
	return func(c *Config) {
		c.ServiceName = name
	}
}

func WithUserAgent(ua string) Option {
	return func(c *Config) {
		c.UserAgent = ua
	}
}

func allowedBaseURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Hostname() == "" {
		return false
	}
	if u.Scheme == "https" {
		return true
	}
	if u.Scheme != "http" {
		return false
	}
	host := strings.Trim(strings.ToLower(u.Hostname()), "[]")
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
