package transport

import (
	"crypto/ecdsa"
	"net/http"
	"time"
)

type Config struct {
	BaseURL     string
	APIBasePath string
	BasePath    string
	TenantID    string
	HTTPClient  *http.Client

	DPoPPrivateKey *ecdsa.PrivateKey

	AccessToken      string
	RefreshToken     string
	AttestationToken string

	TokenEndpoint      string
	TokenRefreshFormat string
	ClientID           string
	ClientSecret       []byte

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
