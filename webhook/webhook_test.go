package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockTransport struct {
	method    string
	path      string
	body      interface{}
	result    interface{}
	err       error
	callCount int
}

func (m *mockTransport) Do(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	m.method = method
	m.path = path
	m.body = body
	m.callCount++
	if m.err != nil {
		return m.err
	}
	if m.result != nil && result != nil {
		data, _ := json.Marshal(m.result)
		json.Unmarshal(data, result)
	}
	return nil
}

func (m *mockTransport) DoWithHeaders(ctx context.Context, method, path string, body interface{}, result interface{}, headers map[string]string) error {
	return m.Do(ctx, method, path, body, result)
}

func (m *mockTransport) BuildQueryURL(path string, params map[string]string) string {
	if len(params) == 0 {
		return path
	}
	q := path + "?"
	for k, v := range params {
		q += k + "=" + v + "&"
	}
	return q[:len(q)-1]
}

func TestVerifySignature_ValidSignature(t *testing.T) {
	secret := "webhook-secret-key-abc123"
	body := `{"event":"agent.registered","agent_id":"agent-123"}`

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	signature := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(body))
	req.Header.Set("X-Signature", signature)

	valid, err := VerifySignature(secret, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Fatal("expected valid signature")
	}
}

func TestVerifySignature_InvalidSignature(t *testing.T) {
	secret := "webhook-secret-key-abc123"
	body := `{"event":"agent.registered","agent_id":"agent-123"}`

	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(body))
	req.Header.Set("X-Signature", "deadbeef00000000000000000000000000000000000000000000000000000000")

	valid, err := VerifySignature(secret, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Fatal("expected invalid signature")
	}
}

func TestVerifySignature_MissingHeader(t *testing.T) {
	req := httptest.NewRequest("POST", "/webhook", strings.NewReader("body"))

	_, err := VerifySignature("secret", req)
	if err == nil {
		t.Fatal("expected error for missing X-Signature header")
	}
}

func TestVerifySignature_TamperedBody(t *testing.T) {
	secret := "webhook-secret"
	originalBody := `{"event":"agent.registered"}`

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(originalBody))
	signature := hex.EncodeToString(mac.Sum(nil))

	tamperedBody := `{"event":"agent.decommissioned"}`
	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(tamperedBody))
	req.Header.Set("X-Signature", signature)

	valid, err := VerifySignature(secret, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Fatal("expected signature to fail for tampered body")
	}
}

func TestVerifySignature_EmptyBody(t *testing.T) {
	secret := "webhook-secret"
	body := ""

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	signature := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(body))
	req.Header.Set("X-Signature", signature)

	valid, err := VerifySignature(secret, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Fatal("expected valid signature for empty body")
	}
}

func TestVerifySignature_WrongSecret(t *testing.T) {
	body := `{"event":"agent.revoked"}`

	mac := hmac.New(sha256.New, []byte("correct-secret"))
	mac.Write([]byte(body))
	signature := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(body))
	req.Header.Set("X-Signature", signature)

	valid, err := VerifySignature("wrong-secret", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Fatal("expected signature to fail with wrong secret")
	}
}

func TestVerifySignature_ReadError(t *testing.T) {
	req := &http.Request{
		Header: http.Header{"X-Signature": []string{"abc"}},
		Body:   http.NoBody,
	}

	valid, err := VerifySignature("secret", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Fatal("expected invalid for empty body with wrong signature")
	}
}

func TestSubscribe_RequiresTargetURL(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.Subscribe(context.Background(), &SubscribeRequest{
		Events:       []string{"agent.registered"},
		SharedSecret: "secret",
	})
	if err == nil {
		t.Fatal("expected error for missing target_url")
	}
}

func TestSubscribe_RequiresEvents(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.Subscribe(context.Background(), &SubscribeRequest{
		TargetURL:    "https://example.com/webhook",
		SharedSecret: "secret",
	})
	if err == nil {
		t.Fatal("expected error for empty events")
	}
}

func TestSubscribe_RequiresSharedSecret(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.Subscribe(context.Background(), &SubscribeRequest{
		TargetURL: "https://example.com/webhook",
		Events:    []string{"agent.registered"},
	})
	if err == nil {
		t.Fatal("expected error for missing shared_secret")
	}
}

func TestSubscribe_Unsupported(t *testing.T) {
	mt := &mockTransport{
		result: &SubscribeResponse{
			Subscription: &Subscription{
				ID:     "sub-123",
				Status: "active",
			},
		},
	}
	svc := NewService(mt)
	resp, err := svc.Subscribe(context.Background(), &SubscribeRequest{
		TargetURL:    "https://example.com/webhook",
		Events:       []string{"agent.registered", "agent.revoked"},
		SharedSecret: "my-secret",
	})
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if resp != nil {
		t.Fatalf("expected nil response, got %#v", resp)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported subscribe should not call transport")
	}
}

func TestUnsubscribe_RequiresID(t *testing.T) {
	svc := NewService(&mockTransport{})
	err := svc.Unsubscribe(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty subscription_id")
	}
}

func TestList_CapsLimitAt200(t *testing.T) {
	mt := &mockTransport{result: &ListResponse{}}
	svc := NewService(mt)
	resp, err := svc.List(context.Background(), "", 500)
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if resp != nil {
		t.Fatalf("expected nil response, got %#v", resp)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported list should not call transport")
	}
}

func TestRetryDeadLetter_RequiresEntryID(t *testing.T) {
	svc := NewService(&mockTransport{})
	err := svc.RetryDeadLetter(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty entry_id")
	}
}

func computeHMAC(secret, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature_LargePayload(t *testing.T) {
	secret := "test-secret"
	body := strings.Repeat("x", 65536)

	signature := computeHMAC(secret, body)

	req := httptest.NewRequest("POST", "/webhook", bytes.NewBufferString(body))
	req.Header.Set("X-Signature", signature)

	valid, err := VerifySignature(secret, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Fatal("expected valid signature for large payload")
	}
}

func TestVerifySignature_ConstantTimeComparison(t *testing.T) {
	secret := "constant-time-secret"
	body := `{"event":"test"}`

	correctSig := computeHMAC(secret, body)
	almostCorrectSig := correctSig[:len(correctSig)-1] + "0"

	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(body))
	req.Header.Set("X-Signature", almostCorrectSig)

	valid, err := VerifySignature(secret, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Fatal("expected invalid for nearly-correct signature")
	}
}

func TestListDeliveries_RequiresSubscriptionID(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, _, err := svc.ListDeliveries(context.Background(), "", "", 10)
	if err == nil {
		t.Fatal("expected error for empty subscription_id")
	}
}

func TestListDeliveries_CapsLimit(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)
	entries, cursor, err := svc.ListDeliveries(context.Background(), "sub-1", "", 999)
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if entries != nil || cursor != "" {
		t.Fatalf("expected empty result, got entries=%#v cursor=%q", entries, cursor)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported delivery list should not call transport")
	}
}

func TestGet_RequiresID(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.Get(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty subscription_id")
	}
}

func TestGet_Unsupported(t *testing.T) {
	type subResp struct {
		ID        string `json:"id"`
		TargetURL string `json:"target_url"`
	}
	mt := &mockTransport{result: &subResp{ID: "sub-1", TargetURL: "https://hooks.example.com"}}
	svc := NewService(mt)

	resp, err := svc.Get(context.Background(), "sub-1")
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if resp != nil {
		t.Fatalf("expected nil response, got %#v", resp)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported get should not call transport")
	}
}

func TestListDeadLetters_Unsupported(t *testing.T) {
	type dlResp struct {
		Data       []DeadLetterEntry `json:"data"`
		NextCursor string            `json:"next_cursor"`
	}
	mt := &mockTransport{result: &dlResp{Data: []DeadLetterEntry{{ID: "dl-1", WebhookID: "sub-1"}}}}
	svc := NewService(mt)

	entries, cursor, err := svc.ListDeadLetters(context.Background(), "", 10)
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if entries != nil || cursor != "" {
		t.Fatalf("expected empty result, got entries=%#v cursor=%q", entries, cursor)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported dead-letter list should not call transport")
	}
}

func TestListDeadLetters_CapsLimit(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)
	entries, cursor, err := svc.ListDeadLetters(context.Background(), "", 999)
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if entries != nil || cursor != "" {
		t.Fatalf("expected empty result, got entries=%#v cursor=%q", entries, cursor)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported dead-letter list should not call transport")
	}
}
