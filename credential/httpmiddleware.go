package credential

import (
	"fmt"
	"net/http"
)

type ZineticTransport struct {
	Wrapped       http.RoundTripper
	Store         *MemStore
	CredentialKey string
	TokenType     string
}

func NewHTTPTransport(wrapped http.RoundTripper, store *MemStore, credentialKey string) *ZineticTransport {
	if wrapped == nil {
		wrapped = http.DefaultTransport
	}
	return &ZineticTransport{
		Wrapped:       wrapped,
		Store:         store,
		CredentialKey: credentialKey,
		TokenType:     "Bearer",
	}
}

func (t *ZineticTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.Store == nil {
		return nil, fmt.Errorf("credential store is nil")
	}

	token, ok := t.Store.Retrieve(t.CredentialKey)
	if !ok {
		return nil, fmt.Errorf("credential %q not found in store", t.CredentialKey)
	}

	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", t.TokenType+" "+string(token))

	zeroize(token)

	return t.Wrapped.RoundTrip(clone)
}
