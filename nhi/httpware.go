package nhi

import (
	"net"
	"net/http"
	"time"
)

type Transport struct {
	provider *Provider
	base     http.RoundTripper
	tokenKey string
}

type TransportOption func(*Transport)

func WithBaseTransport(rt http.RoundTripper) TransportOption {
	return func(t *Transport) {
		t.base = rt
	}
}

func WithTokenKey(key string) TransportOption {
	return func(t *Transport) {
		t.tokenKey = key
	}
}

func NewHTTPTransport(provider *Provider, opts ...TransportOption) *Transport {
	t := &Transport{
		provider: provider,
		base:     http.DefaultTransport,
		tokenKey: "token",
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.provider.GetCredential(t.tokenKey)
	if err != nil {
		return nil, err
	}

	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", "Bearer "+token)

	proof, err := t.provider.prover.CreateProof(req.Method, req.URL.String(), token)
	if err == nil && proof != "" {
		clone.Header.Set("DPoP", proof)
	}

	return t.base.RoundTrip(clone)
}

func NewHTTPClient(provider *Provider, opts ...TransportOption) *http.Client {
	transport := NewHTTPTransport(provider, opts...)
	if transport.base == http.DefaultTransport {
		transport.base = &http.Transport{
			DialContext:         (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
			TLSHandshakeTimeout: 5 * time.Second,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		}
	}
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
}
