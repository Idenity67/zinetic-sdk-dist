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

	// Token refresh via OAuth 2.0 refresh_token or client_credentials grant.
	// When AccessToken expires and TokenEndpoint is non-empty, the transport
	// automatically fetches a new token before retrying the failed request.
	TokenEndpoint      string
	TokenRefreshFormat string
	ClientID           string
	ClientSecret       []byte

	// OnTokenRefreshed is called after a successful automatic token refresh so
	// the caller can persist the new tokens. May be nil.
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
