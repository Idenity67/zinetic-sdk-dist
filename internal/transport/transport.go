package transport

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"net/url"
	pathpkg "path"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"sdk.zinetic.net/apierr"
	"sdk.zinetic.net/dpop"
)

const (
	defaultAPIBasePath      = "/api/v1"
	defaultMaxResponseBytes = 10 * 1024 * 1024
)

var errResponseTooLarge = errors.New("response body exceeds maximum allowed size")

type Transport struct {
	config     *Config
	tracer     trace.Tracer
	dpop       *dpop.Prover
	httpClient *http.Client

	tokenMu        sync.RWMutex
	currentToken   string
	currentRefresh string
}

func New(cfg *Config) *Transport {
	normalized := normalizeConfig(cfg)
	t := &Transport{
		config:         normalized,
		currentToken:   normalized.AccessToken,
		currentRefresh: normalized.RefreshToken,
	}
	if normalized.HTTPClient != nil {
		t.httpClient = normalized.HTTPClient
	} else {
		t.httpClient = &http.Client{
			Timeout: normalized.RequestTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		}
	}
	if normalized.EnableTracing {
		t.tracer = otel.Tracer(normalized.ServiceName)
	}
	if normalized.DPoPPrivateKey != nil {
		t.dpop = dpop.NewProver(normalized.DPoPPrivateKey)
	}
	return t
}

func (t *Transport) Do(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	return t.doWithRetry(ctx, method, path, body, result, nil)
}

func (t *Transport) DoWithHeaders(ctx context.Context, method, path string, body interface{}, result interface{}, extraHeaders map[string]string) error {
	return t.doWithRetry(ctx, method, path, body, result, extraHeaders)
}

func (t *Transport) DoRaw(ctx context.Context, method, path string, body interface{}, extraHeaders map[string]string) ([]byte, error) {
	return t.doRawWithRetry(ctx, method, path, body, extraHeaders)
}

func (t *Transport) DoStream(ctx context.Context, method, path string, body interface{}, extraHeaders map[string]string) (*http.Response, error) {
	return t.doStreamWithRetry(ctx, method, path, body, extraHeaders)
}

func (t *Transport) doWithRetry(ctx context.Context, method, path string, body interface{}, result interface{}, extraHeaders map[string]string) error {
	var lastErr error
	retryHeaders := t.headersWithStableIdempotency(method, extraHeaders)
	for attempt := 0; attempt <= t.config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := t.BackoffDelay(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := t.doRequest(ctx, method, path, body, result, retryHeaders)
		if err == nil {
			return nil
		}

		lastErr = err

		apiErr, ok := err.(*apierr.APIError)
		if !ok || !apiErr.IsRetryable() {
			return err
		}

		if rateErr, ok := err.(*apierr.RateLimitError); ok {
			if rateErr.RetryAfter > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(rateErr.RetryAfter):
				}
			}
		}
	}
	return lastErr
}

func (t *Transport) doRawWithRetry(ctx context.Context, method, path string, body interface{}, extraHeaders map[string]string) ([]byte, error) {
	var lastErr error
	retryHeaders := t.headersWithStableIdempotency(method, extraHeaders)
	for attempt := 0; attempt <= t.config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := t.BackoffDelay(attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		_, raw, err := t.doHTTP(ctx, method, path, body, retryHeaders, true)
		if err == nil {
			return raw, nil
		}

		lastErr = err

		apiErr, ok := err.(*apierr.APIError)
		if !ok || !apiErr.IsRetryable() {
			return nil, err
		}

		if rateErr, ok := err.(*apierr.RateLimitError); ok {
			if rateErr.RetryAfter > 0 {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(rateErr.RetryAfter):
				}
			}
		}
	}
	return nil, lastErr
}

func (t *Transport) doStreamWithRetry(ctx context.Context, method, path string, body interface{}, extraHeaders map[string]string) (*http.Response, error) {
	var lastErr error
	retryHeaders := t.headersWithStableIdempotency(method, extraHeaders)
	for attempt := 0; attempt <= t.config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := t.BackoffDelay(attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err := t.doStream(ctx, method, path, body, retryHeaders, true)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		apiErr, ok := err.(*apierr.APIError)
		if !ok || !apiErr.IsRetryable() {
			return nil, err
		}

		if rateErr, ok := err.(*apierr.RateLimitError); ok && rateErr.RetryAfter > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(rateErr.RetryAfter):
			}
		}
	}
	return nil, lastErr
}

func (t *Transport) doStream(ctx context.Context, method, requestPath string, body interface{}, extraHeaders map[string]string, allowNonceRetry bool) (*http.Response, error) {
	fullURL, err := t.buildURL(requestPath)
	if err != nil {
		return nil, err
	}

	req, err := t.newRequest(ctx, method, fullURL, body, extraHeaders)
	if err != nil {
		return nil, err
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, &apierr.APIError{
			Code:       apierr.ErrCodeServiceUnavail,
			HTTPStatus: 0,
			Message:    fmt.Sprintf("request failed: %v", err),
			Timestamp:  time.Now().UTC(),
		}
	}

	if dpopNonce := resp.Header.Get("DPoP-Nonce"); dpopNonce != "" && t.dpop != nil {
		t.dpop.SetServerNonce(dpopNonce)
	}

	if resp.StatusCode < 400 {
		return resp, nil
	}

	defer resp.Body.Close()
	raw, readErr := readLimitedResponse(resp.Body, t.config.MaxResponseBytes)
	if readErr != nil {
		code := apierr.ErrCodeServiceUnavail
		if readErr == errResponseTooLarge {
			code = apierr.ErrCodeResponseTooLarge
		}
		return nil, &apierr.APIError{
			Code:       code,
			HTTPStatus: resp.StatusCode,
			Message:    fmt.Sprintf("failed to read response: %v", readErr),
			RequestID:  resp.Header.Get("X-Request-ID"),
			Timestamp:  time.Now().UTC(),
		}
	}
	if allowNonceRetry && resp.StatusCode == http.StatusUnauthorized && t.dpop != nil && shouldRetryNonce(resp, raw) {
		return t.doStream(ctx, method, requestPath, body, extraHeaders, false)
	}
	if resp.StatusCode == http.StatusUnauthorized && t.canRefreshToken() && !shouldRetryNonce(resp, raw) {
		if refreshErr := t.attemptTokenRefresh(ctx); refreshErr == nil {
			return t.doStream(ctx, method, requestPath, body, extraHeaders, false)
		}
	}
	return nil, t.errorFromResponse(resp, raw)
}

func (t *Transport) doRequest(ctx context.Context, method, requestPath string, body interface{}, result interface{}, extraHeaders map[string]string) error {
	resp, raw, err := t.doHTTP(ctx, method, requestPath, body, extraHeaders, true)
	if err != nil {
		return err
	}

	if result != nil && resp.StatusCode != http.StatusNoContent && len(raw) > 0 {
		if err := json.NewDecoder(bytes.NewReader(raw)).Decode(result); err != nil {
			return &apierr.APIError{
				Code:       apierr.ErrCodeInternal,
				HTTPStatus: resp.StatusCode,
				Message:    fmt.Sprintf("failed to decode response: %v", err),
				RequestID:  resp.Header.Get("X-Request-ID"),
				Timestamp:  time.Now().UTC(),
			}
		}
	}

	return nil
}

func (t *Transport) doHTTP(ctx context.Context, method, requestPath string, body interface{}, extraHeaders map[string]string, allowNonceRetry bool) (*http.Response, []byte, error) {
	fullURL, err := t.buildURL(requestPath)
	if err != nil {
		return nil, nil, err
	}

	var spanCtx context.Context
	var span trace.Span
	if t.tracer != nil {
		spanCtx, span = t.tracer.Start(ctx, fmt.Sprintf("zinetic.%s %s", method, requestPath),
			trace.WithAttributes(
				attribute.String("http.method", method),
				attribute.String("http.url", fullURL),
				attribute.String("zinetic.tenant_id", t.config.TenantID),
			),
		)
		defer span.End()
		ctx = spanCtx
	}

	req, err := t.newRequest(ctx, method, fullURL, body, extraHeaders)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "request creation failed")
		}
		return nil, nil, err
	}

	httpClient := t.httpClient

	resp, err := httpClient.Do(req)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "request failed")
		}
		return nil, nil, &apierr.APIError{
			Code:       apierr.ErrCodeServiceUnavail,
			HTTPStatus: 0,
			Message:    fmt.Sprintf("request failed: %v", err),
			Timestamp:  time.Now().UTC(),
		}
	}
	defer resp.Body.Close()

	raw, readErr := readLimitedResponse(resp.Body, t.config.MaxResponseBytes)
	if readErr != nil {
		code := apierr.ErrCodeServiceUnavail
		if readErr == errResponseTooLarge {
			code = apierr.ErrCodeResponseTooLarge
		}
		return nil, nil, &apierr.APIError{
			Code:       code,
			HTTPStatus: resp.StatusCode,
			Message:    fmt.Sprintf("failed to read response: %v", readErr),
			RequestID:  resp.Header.Get("X-Request-ID"),
			Timestamp:  time.Now().UTC(),
		}
	}

	if dpopNonce := resp.Header.Get("DPoP-Nonce"); dpopNonce != "" && t.dpop != nil {
		t.dpop.SetServerNonce(dpopNonce)
	}

	if span != nil {
		span.SetAttributes(
			attribute.Int("http.status_code", resp.StatusCode),
			attribute.String("zinetic.request_id", resp.Header.Get("X-Request-ID")),
		)
	}

	if resp.StatusCode >= 400 {
		if allowNonceRetry && resp.StatusCode == http.StatusUnauthorized && t.dpop != nil && shouldRetryNonce(resp, raw) {
			return t.doHTTP(ctx, method, requestPath, body, extraHeaders, false)
		}
		if resp.StatusCode == http.StatusUnauthorized && t.canRefreshToken() && !shouldRetryNonce(resp, raw) {
			if refreshErr := t.attemptTokenRefresh(ctx); refreshErr == nil {
				return t.doHTTP(ctx, method, requestPath, body, extraHeaders, false)
			}
		}
		apiErr := t.errorFromResponse(resp, raw)
		if span != nil {
			span.RecordError(apiErr)
			span.SetStatus(codes.Error, apiErr.Error())
		}
		return nil, nil, apiErr
	}

	if span != nil {
		span.SetStatus(codes.Ok, "")
	}

	return resp, raw, nil
}

func (t *Transport) newRequest(ctx context.Context, method, fullURL string, body interface{}, extraHeaders map[string]string) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, &apierr.APIError{
				Code:       apierr.ErrCodeInternal,
				HTTPStatus: 0,
				Message:    fmt.Sprintf("failed to marshal request body: %v", err),
				Timestamp:  time.Now().UTC(),
			}
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, &apierr.APIError{
			Code:       apierr.ErrCodeInternal,
			HTTPStatus: 0,
			Message:    fmt.Sprintf("failed to create request: %v", err),
			Timestamp:  time.Now().UTC(),
		}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", t.config.UserAgent)
	if t.config.TenantID != "" {
		req.Header.Set("X-Tenant-ID", t.config.TenantID)
	}
	req.Header.Set("X-Request-ID", GenerateRequestID())

	if method != http.MethodGet && method != http.MethodHead {
		req.Header.Set("Idempotency-Key", t.config.IdempotencyKeyPrefix+GenerateRequestID())
	}

	token := t.getToken()
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if t.config.AttestationToken != "" {
		req.Header.Set("X-Attestation-Token", t.config.AttestationToken)
	}

	if t.dpop != nil {
		dpopProof, dpopErr := t.dpop.CreateProof(method, fullURL, token)
		if dpopErr != nil {
			return nil, &apierr.APIError{
				Code:       apierr.ErrCodeAuthDPoPInvalid,
				HTTPStatus: 0,
				Message:    fmt.Sprintf("failed to generate DPoP proof: %v", dpopErr),
				Timestamp:  time.Now().UTC(),
			}
		}
		req.Header.Set("DPoP", dpopProof)
	}

	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	return req, nil
}

func (t *Transport) headersWithStableIdempotency(method string, extraHeaders map[string]string) map[string]string {
	method = strings.ToUpper(method)
	if method == http.MethodGet || method == http.MethodHead {
		return extraHeaders
	}
	for k := range extraHeaders {
		if strings.EqualFold(k, "Idempotency-Key") {
			return extraHeaders
		}
	}
	headers := make(map[string]string, len(extraHeaders)+1)
	for k, v := range extraHeaders {
		headers[k] = v
	}
	headers["Idempotency-Key"] = t.config.IdempotencyKeyPrefix + GenerateRequestID()
	return headers
}

func (t *Transport) errorFromResponse(resp *http.Response, raw []byte) error {
	resp.Body = io.NopCloser(bytes.NewReader(raw))
	if resp.StatusCode == http.StatusTooManyRequests {
		rateErr := &apierr.RateLimitError{
			APIError: apierr.ParseAPIError(resp),
		}
		if limitStr := resp.Header.Get("X-RateLimit-Limit"); limitStr != "" {
			rateErr.Limit, _ = strconv.Atoi(limitStr)
		}
		if remainStr := resp.Header.Get("X-RateLimit-Remaining"); remainStr != "" {
			rateErr.Remaining, _ = strconv.Atoi(remainStr)
		}
		if retryStr := resp.Header.Get("Retry-After"); retryStr != "" {
			if secs, parseErr := strconv.Atoi(retryStr); parseErr == nil {
				rateErr.RetryAfter = time.Duration(secs) * time.Second
			}
		}
		return rateErr
	}
	return apierr.ParseAPIError(resp)
}

func readLimitedResponse(body io.Reader, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		maxBytes = defaultMaxResponseBytes
	}
	limited := io.LimitReader(body, maxBytes+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > maxBytes {
		return nil, errResponseTooLarge
	}
	return raw, nil
}

func (t *Transport) buildURL(requestPath string) (string, error) {
	requestPath = strings.TrimSpace(requestPath)
	if requestPath == "" {
		requestPath = "/"
	}
	if strings.HasPrefix(requestPath, "http://") || strings.HasPrefix(requestPath, "https://") {
		return "", &apierr.APIError{
			Code:       apierr.ErrCodeValidation,
			HTTPStatus: 0,
			Message:    "absolute request URLs are not allowed; configure BaseURL instead",
			Timestamp:  time.Now().UTC(),
		}
	}
	if !strings.HasPrefix(requestPath, "/") {
		requestPath = "/" + requestPath
	}

	basePath := t.config.APIBasePath
	if basePath == "" {
		basePath = cleanBasePath(t.config.BasePath)
	}
	if basePath == "" {
		basePath = defaultAPIBasePath
	}

	if requestPath == basePath || strings.HasPrefix(requestPath, basePath+"/") {
		return t.config.BaseURL + requestPath, nil
	}
	if isRootRoute(requestPath) {
		return t.config.BaseURL + requestPath, nil
	}
	if requestPath == "/v1" {
		requestPath = "/"
	} else if strings.HasPrefix(requestPath, "/v1/") {
		requestPath = strings.TrimPrefix(requestPath, "/v1")
	}
	return t.config.BaseURL + pathpkg.Join(basePath, requestPath), nil
}

func isRootRoute(requestPath string) bool {
	for _, prefix := range []string{"/health", "/ready", "/version", "/metrics", "/docs", "/.well-known", "/scim", "/oid4vci", "/oauth", "/auth/token", "/cli/device", "/dpop", "/pam"} {
		if requestPath == prefix || strings.HasPrefix(requestPath, prefix+"/") {
			return true
		}
	}
	return false
}

func shouldRetryNonce(resp *http.Response, raw []byte) bool {
	if resp.Header.Get("DPoP-Nonce") == "" {
		return false
	}
	var payload map[string]interface{}
	if json.Unmarshal(raw, &payload) != nil {
		return resp.StatusCode == http.StatusUnauthorized
	}
	errCode, _ := payload["error"].(string)
	return errCode == "use_dpop_nonce" || errCode == "invalid_dpop_proof" || resp.StatusCode == http.StatusUnauthorized
}

func (t *Transport) BuildQueryURL(path string, params map[string]string) string {
	if len(params) == 0 {
		return path
	}
	vals := url.Values{}
	for k, v := range params {
		vals.Set(k, v)
	}
	return path + "?" + vals.Encode()
}

func (t *Transport) BackoffDelay(attempt int) time.Duration {
	delay := t.config.RetryBaseDelay * time.Duration(math.Pow(2, float64(attempt-1)))
	if delay > t.config.RetryMaxDelay {
		delay = t.config.RetryMaxDelay
	}
	return delay/2 + cryptoJitter(delay/2)
}

func cryptoJitter(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return max / 2
	}
	return time.Duration(n.Int64())
}

func normalizeConfig(cfg *Config) *Config {
	if cfg == nil {
		cfg = &Config{}
	}
	basePath := cfg.APIBasePath
	if basePath == "" {
		basePath = cfg.BasePath
	}
	baseURL, apiBasePath := normalizeBase(cfg.BaseURL, basePath)
	cfg.BaseURL = baseURL
	cfg.APIBasePath = apiBasePath
	cfg.BasePath = apiBasePath
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = 30 * time.Second
	}
	if cfg.MaxResponseBytes <= 0 {
		cfg.MaxResponseBytes = defaultMaxResponseBytes
	}
	if strings.TrimSpace(cfg.TokenRefreshFormat) == "" {
		cfg.TokenRefreshFormat = "auto"
	}
	return cfg
}

func normalizeBase(rawBaseURL, basePath string) (string, string) {
	basePath = cleanBasePath(basePath)
	if strings.TrimSpace(rawBaseURL) == "" {
		return "", basePath
	}
	u, err := url.Parse(strings.TrimSpace(rawBaseURL))
	if err != nil {
		return strings.TrimRight(strings.TrimSpace(rawBaseURL), "/"), basePath
	}
	u.Path = strings.TrimRight(u.Path, "/")
	if u.Path == basePath || strings.HasSuffix(u.Path, basePath) {
		u.Path = strings.TrimSuffix(u.Path, basePath)
		u.Path = strings.TrimRight(u.Path, "/")
	}
	return strings.TrimRight(u.String(), "/"), basePath
}

func cleanBasePath(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return defaultAPIBasePath
	}
	if !strings.HasPrefix(v, "/") {
		v = "/" + v
	}
	return strings.TrimRight(v, "/")
}

func GenerateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func (t *Transport) getToken() string {
	t.tokenMu.RLock()
	defer t.tokenMu.RUnlock()
	return t.currentToken
}

func (t *Transport) SetAccessToken(token string) {
	t.tokenMu.Lock()
	defer t.tokenMu.Unlock()
	t.currentToken = token
	t.config.AccessToken = token
}

func (t *Transport) SetRefreshToken(token string) {
	t.tokenMu.Lock()
	defer t.tokenMu.Unlock()
	t.currentRefresh = token
	t.config.RefreshToken = token
}

func (t *Transport) canRefreshToken() bool {
	if t.config.TokenEndpoint == "" {
		return false
	}
	t.tokenMu.RLock()
	hasRefresh := t.currentRefresh != ""
	t.tokenMu.RUnlock()
	return hasRefresh || (t.config.ClientID != "" && len(t.config.ClientSecret) > 0)
}

func (t *Transport) attemptTokenRefresh(ctx context.Context) error {
	t.tokenMu.Lock()
	defer t.tokenMu.Unlock()

	refreshToken := t.currentRefresh
	var body io.Reader
	contentType := "application/x-www-form-urlencoded"
	if refreshToken != "" {
		if t.refreshAsJSON() {
			payload := map[string]string{
				"grant_type":    "refresh_token",
				"refresh_token": refreshToken,
			}
			if t.config.ClientID != "" {
				payload["client_id"] = t.config.ClientID
			}
			raw, err := json.Marshal(payload)
			if err != nil {
				return fmt.Errorf("encode token refresh request: %w", err)
			}
			body = bytes.NewReader(raw)
			contentType = "application/json"
		} else {
			formData := url.Values{
				"grant_type":    {"refresh_token"},
				"refresh_token": {refreshToken},
			}
			if t.config.ClientID != "" {
				formData.Set("client_id", t.config.ClientID)
			}
			body = strings.NewReader(formData.Encode())
		}
	} else {
		formData := url.Values{
			"grant_type":    {"client_credentials"},
			"client_id":     {t.config.ClientID},
			"client_secret": {string(t.config.ClientSecret)},
		}
		body = strings.NewReader(formData.Encode())
	}

	endpoint := t.config.TokenEndpoint
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		var buildErr error
		endpoint, buildErr = t.buildURL(endpoint)
		if buildErr != nil {
			return buildErr
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", t.config.UserAgent)
	if t.config.TenantID != "" {
		req.Header.Set("X-Tenant-ID", t.config.TenantID)
	}
	req.Header.Set("X-Request-ID", GenerateRequestID())
	if t.dpop != nil {
		dpopProof, dpopErr := t.dpop.CreateProof(http.MethodPost, endpoint, "")
		if dpopErr != nil {
			return &apierr.APIError{
				Code:       apierr.ErrCodeAuthDPoPInvalid,
				HTTPStatus: 0,
				Message:    fmt.Sprintf("failed to generate DPoP proof: %v", dpopErr),
				Timestamp:  time.Now().UTC(),
			}
		}
		req.Header.Set("DPoP", dpopProof)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := readLimitedResponse(resp.Body, t.config.MaxResponseBytes)
	if err != nil {
		code := apierr.ErrCodeServiceUnavail
		if err == errResponseTooLarge {
			code = apierr.ErrCodeResponseTooLarge
		}
		return &apierr.APIError{
			Code:       code,
			HTTPStatus: resp.StatusCode,
			Message:    fmt.Sprintf("failed to read token refresh response: %v", err),
			RequestID:  resp.Header.Get("X-Request-ID"),
			Timestamp:  time.Now().UTC(),
		}
	}
	if resp.StatusCode != http.StatusOK {
		return tokenRefreshError(resp, raw)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(raw, &tokenResp); err != nil {
		return fmt.Errorf("decode token refresh response: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return fmt.Errorf("token refresh response did not include an access_token")
	}

	t.currentToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		t.currentRefresh = tokenResp.RefreshToken
	}

	if t.config.OnTokenRefreshed != nil {
		t.config.OnTokenRefreshed(tokenResp.AccessToken, tokenResp.RefreshToken)
	}

	return nil
}

func tokenRefreshError(resp *http.Response, raw []byte) error {
	type oauthError struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
		Message          string `json:"message"`
		Code             string `json:"code"`
	}
	var parsed oauthError
	message := fmt.Sprintf("token refresh returned HTTP %d", resp.StatusCode)
	code := apierr.ErrCodeAuthTokenInvalid
	if err := json.Unmarshal(raw, &parsed); err == nil {
		if parsed.Error != "" && parsed.ErrorDescription != "" {
			message = parsed.Error + ": " + parsed.ErrorDescription
		} else if parsed.Error != "" {
			message = parsed.Error
		} else if parsed.Message != "" {
			message = parsed.Message
		}
		if parsed.Code != "" {
			code = apierr.ErrorCode(parsed.Code)
		}
	}
	return &apierr.APIError{
		Code:       code,
		HTTPStatus: resp.StatusCode,
		Message:    message,
		RequestID:  resp.Header.Get("X-Request-ID"),
		Timestamp:  time.Now().UTC(),
	}
}

func (t *Transport) refreshAsJSON() bool {
	format := strings.ToLower(strings.TrimSpace(t.config.TokenRefreshFormat))
	switch format {
	case "json":
		return true
	case "form":
		return false
	}
	u, err := url.Parse(t.config.TokenEndpoint)
	if err != nil {
		return strings.Contains(t.config.TokenEndpoint, "/auth/tokens/refresh")
	}
	return strings.Contains(u.Path, "/auth/tokens/refresh")
}
