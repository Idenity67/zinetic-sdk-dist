package apierr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ErrorCode string

const (
	ErrCodeAuthTokenExpired      ErrorCode = "AUTH_TOKEN_EXPIRED" // #nosec G101 -- symbolic API error code, not a credential.
	ErrCodeAuthTokenInvalid      ErrorCode = "AUTH_TOKEN_INVALID" // #nosec G101 -- symbolic API error code, not a credential.
	ErrCodeAuthDPoPRequired      ErrorCode = "AUTH_DPOP_REQUIRED"
	ErrCodeAuthDPoPInvalid       ErrorCode = "AUTH_DPOP_INVALID"
	ErrCodeAuthInsufficientScope ErrorCode = "AUTH_INSUFFICIENT_SCOPE"
	ErrCodeAuthStepUpRequired    ErrorCode = "AUTH_STEP_UP_REQUIRED"
	ErrCodeAuthRateLimited       ErrorCode = "AUTH_RATE_LIMITED"

	ErrCodeAgentNotFound       ErrorCode = "AGENT_NOT_FOUND"
	ErrCodeAgentSuspended      ErrorCode = "AGENT_SUSPENDED"
	ErrCodeAgentRevoked        ErrorCode = "AGENT_REVOKED"
	ErrCodeAgentCapabilityDeny ErrorCode = "AGENT_CAPABILITY_DENIED"
	ErrCodeAgentDriftDetected  ErrorCode = "AGENT_DRIFT_DETECTED"
	ErrCodeAgentInvalidState   ErrorCode = "AGENT_INVALID_STATE"

	ErrCodeAnchorNotFound      ErrorCode = "ANCHOR_NOT_FOUND"
	ErrCodeAnchorAlreadyExists ErrorCode = "ANCHOR_ALREADY_EXISTS"
	ErrCodeAnchorInvalidProof  ErrorCode = "ANCHOR_INVALID_PROOF"
	ErrCodeAnchorRevoked       ErrorCode = "ANCHOR_REVOKED"

	ErrCodePolicyNotFound  ErrorCode = "POLICY_NOT_FOUND"
	ErrCodePolicyEvalError ErrorCode = "POLICY_EVAL_ERROR"
	ErrCodePolicyTimeout   ErrorCode = "POLICY_TIMEOUT"
	ErrCodePolicyDenied    ErrorCode = "POLICY_DENIED"

	ErrCodeLegacyBindFailed  ErrorCode = "LEGACY_BIND_FAILED"
	ErrCodeLegacyUnsupported ErrorCode = "LEGACY_UNSUPPORTED"

	ErrCodeMCPToolUnauthorized ErrorCode = "MCP_TOOL_UNAUTHORIZED"
	ErrCodeMCPServerNotFound   ErrorCode = "MCP_SERVER_NOT_FOUND"
	ErrCodeMCPTokenInvalid     ErrorCode = "MCP_TOKEN_INVALID" // #nosec G101 -- symbolic API error code, not a credential.

	ErrCodeValidation       ErrorCode = "VALIDATION_ERROR"
	ErrCodeInternal         ErrorCode = "INTERNAL_ERROR"
	ErrCodeNotFound         ErrorCode = "NOT_FOUND"
	ErrCodeConflict         ErrorCode = "CONFLICT"
	ErrCodeTenantNotFound   ErrorCode = "TENANT_NOT_FOUND"
	ErrCodeForbidden        ErrorCode = "FORBIDDEN"
	ErrCodeServiceUnavail   ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeIdempotencyConfl ErrorCode = "IDEMPOTENCY_CONFLICT"
	ErrCodeResponseTooLarge ErrorCode = "RESPONSE_TOO_LARGE"
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type APIError struct {
	Code       ErrorCode    `json:"code"`
	HTTPStatus int          `json:"http_status"`
	Message    string       `json:"message"`
	Details    []FieldError `json:"details,omitempty"`
	RequestID  string       `json:"request_id"`
	Timestamp  time.Time    `json:"timestamp"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("zinetic: %s (HTTP %d): %s [request_id=%s]", e.Code, e.HTTPStatus, e.Message, e.RequestID)
}

func (e *APIError) IsRetryable() bool {
	if e.HTTPStatus == 0 && e.Code == ErrCodeServiceUnavail {
		return true
	}
	switch e.HTTPStatus {
	case http.StatusTooManyRequests,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
		http.StatusBadGateway:
		return true
	}
	return false
}

func (e *APIError) IsRateLimited() bool {
	return e.HTTPStatus == http.StatusTooManyRequests
}

func (e *APIError) IsAuthError() bool {
	return e.HTTPStatus == http.StatusUnauthorized || e.HTTPStatus == http.StatusForbidden
}

func ParseAPIError(resp *http.Response) *APIError {
	var apiErr APIError
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
		return &APIError{
			Code:       ErrCodeInternal,
			HTTPStatus: resp.StatusCode,
			Message:    fmt.Sprintf("unexpected error response (HTTP %d)", resp.StatusCode),
			RequestID:  resp.Header.Get("X-Request-ID"),
			Timestamp:  time.Now().UTC(),
		}
	}
	if apiErr.RequestID == "" {
		apiErr.RequestID = resp.Header.Get("X-Request-ID")
	}
	if apiErr.HTTPStatus == 0 {
		apiErr.HTTPStatus = resp.StatusCode
	}
	return &apiErr
}

type RateLimitError struct {
	*APIError
	Limit      int
	Remaining  int
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("zinetic: rate limited (limit=%d, remaining=%d, retry_after=%s) [request_id=%s]",
		e.Limit, e.Remaining, e.RetryAfter, e.RequestID)
}

func NewValidationError(field, message string) *APIError {
	return &APIError{
		Code:       ErrCodeValidation,
		HTTPStatus: http.StatusBadRequest,
		Message:    "validation failed",
		Details:    []FieldError{{Field: field, Message: message}},
		Timestamp:  time.Now().UTC(),
	}
}
