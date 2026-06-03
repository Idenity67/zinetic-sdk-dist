package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
)

type Transport interface {
	Do(ctx context.Context, method, path string, body interface{}, result interface{}) error
	DoWithHeaders(ctx context.Context, method, path string, body interface{}, result interface{}, headers map[string]string) error
	BuildQueryURL(path string, params map[string]string) string
}

type Service struct {
	transport Transport
}

func NewService(t Transport) *Service {
	return &Service{transport: t}
}

func (s *Service) Subscribe(ctx context.Context, req *SubscribeRequest) (*SubscribeResponse, error) {
	if req.TargetURL == "" {
		return nil, fmt.Errorf("target_url is required")
	}
	if len(req.Events) == 0 {
		return nil, fmt.Errorf("at least one event is required")
	}
	if req.SharedSecret == "" {
		return nil, fmt.Errorf("shared_secret is required")
	}

	return nil, fmt.Errorf("webhook subscription is not supported without tenant scope in the current backend OpenAPI contract")
}

func (s *Service) Unsubscribe(ctx context.Context, subscriptionID string) error {
	if subscriptionID == "" {
		return fmt.Errorf("subscription_id is required")
	}
	return fmt.Errorf("webhook unsubscribe is not supported without tenant scope in the current backend OpenAPI contract")
}

func (s *Service) Get(ctx context.Context, subscriptionID string) (*Subscription, error) {
	if subscriptionID == "" {
		return nil, fmt.Errorf("subscription_id is required")
	}

	return nil, fmt.Errorf("webhook lookup is not supported without tenant scope in the current backend OpenAPI contract")
}

func (s *Service) List(ctx context.Context, cursor string, limit int) (*ListResponse, error) {
	return nil, fmt.Errorf("webhook listing is not supported without tenant scope in the current backend OpenAPI contract")
}

func (s *Service) ListDeliveries(ctx context.Context, subscriptionID string, cursor string, limit int) ([]DeliveryAttempt, string, error) {
	if subscriptionID == "" {
		return nil, "", fmt.Errorf("subscription_id is required")
	}

	return nil, "", fmt.Errorf("webhook delivery listing is not supported without tenant scope in the current backend OpenAPI contract")
}

func (s *Service) ListDeadLetters(ctx context.Context, cursor string, limit int) ([]DeadLetterEntry, string, error) {
	return nil, "", fmt.Errorf("webhook dead-letter listing is not supported by the current backend OpenAPI contract")
}

func (s *Service) RetryDeadLetter(ctx context.Context, entryID string) error {
	if entryID == "" {
		return fmt.Errorf("entry_id is required")
	}
	return fmt.Errorf("webhook dead-letter retry is not supported by the current backend OpenAPI contract")
}

func VerifySignature(secret string, r *http.Request) (bool, error) {
	signature := r.Header.Get("X-Signature")
	if signature == "" {
		return false, fmt.Errorf("missing X-Signature header")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read request body: %w", err)
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature)), nil
}
