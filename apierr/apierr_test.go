package apierr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseAPIError_ValidJSON(t *testing.T) {
	body := `{"code":"AGENT_NOT_FOUND","message":"agent not found","http_status":404,"request_id":"req-123"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-ID", "req-123")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(body))
	}))
	defer server.Close()

	resp, _ := http.Get(server.URL)
	apiErr := ParseAPIError(resp)

	if apiErr.Code != ErrCodeAgentNotFound {
		t.Fatalf("expected AGENT_NOT_FOUND, got %s", apiErr.Code)
	}
	if apiErr.Message != "agent not found" {
		t.Fatalf("expected 'agent not found', got %s", apiErr.Message)
	}
}

func TestParseAPIError_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-ID", "req-456")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	resp, _ := http.Get(server.URL)
	apiErr := ParseAPIError(resp)

	if apiErr.Code != ErrCodeInternal {
		t.Fatalf("expected INTERNAL_ERROR for unparseable response, got %s", apiErr.Code)
	}
	if apiErr.HTTPStatus != 500 {
		t.Fatalf("expected HTTP 500, got %d", apiErr.HTTPStatus)
	}
}

func TestAPIError_IsRetryable(t *testing.T) {
	cases := []struct {
		status    int
		retryable bool
	}{
		{http.StatusTooManyRequests, true},
		{http.StatusServiceUnavailable, true},
		{http.StatusGatewayTimeout, true},
		{http.StatusBadGateway, true},
		{http.StatusNotFound, false},
		{http.StatusBadRequest, false},
		{http.StatusUnauthorized, false},
		{http.StatusForbidden, false},
		{http.StatusInternalServerError, false},
	}

	for _, c := range cases {
		err := &APIError{HTTPStatus: c.status}
		if err.IsRetryable() != c.retryable {
			t.Errorf("HTTP %d: expected IsRetryable=%v, got %v", c.status, c.retryable, err.IsRetryable())
		}
	}
}

func TestAPIError_IsRateLimited(t *testing.T) {
	err := &APIError{HTTPStatus: http.StatusTooManyRequests}
	if !err.IsRateLimited() {
		t.Fatal("expected IsRateLimited=true for 429")
	}

	err2 := &APIError{HTTPStatus: http.StatusServiceUnavailable}
	if err2.IsRateLimited() {
		t.Fatal("expected IsRateLimited=false for 503")
	}
}

func TestAPIError_IsAuthError(t *testing.T) {
	err401 := &APIError{HTTPStatus: http.StatusUnauthorized}
	if !err401.IsAuthError() {
		t.Fatal("expected IsAuthError=true for 401")
	}

	err403 := &APIError{HTTPStatus: http.StatusForbidden}
	if !err403.IsAuthError() {
		t.Fatal("expected IsAuthError=true for 403")
	}

	err404 := &APIError{HTTPStatus: http.StatusNotFound}
	if err404.IsAuthError() {
		t.Fatal("expected IsAuthError=false for 404")
	}
}

func TestAPIError_ErrorString(t *testing.T) {
	err := &APIError{
		Code:       ErrCodePolicyDenied,
		HTTPStatus: 403,
		Message:    "access denied by policy",
		RequestID:  "req-789",
	}
	s := err.Error()
	if s == "" {
		t.Fatal("expected non-empty error string")
	}
	if !containsSubstr(s, "POLICY_DENIED") {
		t.Fatalf("expected error code in string, got: %s", s)
	}
}

func TestRateLimitError(t *testing.T) {
	body := `{"code":"AUTH_RATE_LIMITED","message":"too many requests","http_status":429}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "60")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("Retry-After", "10")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(body))
	}))
	defer server.Close()

	resp, _ := http.Get(server.URL)
	apiErr := ParseAPIError(resp)

	if apiErr.Code != ErrCodeAuthRateLimited {
		t.Fatalf("expected AUTH_RATE_LIMITED, got %s", apiErr.Code)
	}
}

func TestNewValidationError(t *testing.T) {
	apiErr := NewValidationError("email", "must be a valid email")
	if apiErr == nil {
		t.Fatal("expected non-nil error")
	}

	if apiErr.Code != ErrCodeValidation {
		t.Fatalf("expected VALIDATION_ERROR, got %s", apiErr.Code)
	}
	if len(apiErr.Details) != 1 {
		t.Fatalf("expected 1 field error, got %d", len(apiErr.Details))
	}
	if apiErr.Details[0].Field != "email" {
		t.Fatalf("expected field 'email', got '%s'", apiErr.Details[0].Field)
	}
}

func TestAPIError_JSON_Roundtrip(t *testing.T) {
	original := &APIError{
		Code:       ErrCodeAgentSuspended,
		HTTPStatus: 403,
		Message:    "agent is suspended",
		RequestID:  "req-abc",
		Details:    []FieldError{{Field: "status", Message: "agent must be active"}},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded APIError
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Code != original.Code {
		t.Fatalf("expected code %s, got %s", original.Code, decoded.Code)
	}
	if decoded.Message != original.Message {
		t.Fatalf("expected message %s, got %s", original.Message, decoded.Message)
	}
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
