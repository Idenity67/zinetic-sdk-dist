package dpop

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Prover struct {
	privateKey  *ecdsa.PrivateKey
	mu          sync.RWMutex
	serverNonce string
}

func NewProver(key *ecdsa.PrivateKey) *Prover {
	return &Prover{
		privateKey: key,
	}
}

func GenerateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

func (d *Prover) SetServerNonce(nonce string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.serverNonce = nonce
}

func (d *Prover) GetServerNonce() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.serverNonce
}

func (d *Prover) CreateProof(method, uri, accessToken string) (string, error) {
	jtiBytes := make([]byte, 16)
	if _, err := rand.Read(jtiBytes); err != nil {
		return "", fmt.Errorf("failed to generate jti: %w", err)
	}

	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"jti": base64.RawURLEncoding.EncodeToString(jtiBytes),
		"htm": method,
		"htu": uri,
		"iat": now.Unix(),
		"exp": now.Add(60 * time.Second).Unix(),
	}

	nonce := d.GetServerNonce()
	if nonce != "" {
		claims["nonce"] = nonce
	}

	if accessToken != "" {
		h := sha256.Sum256([]byte(accessToken))
		claims["ath"] = base64.RawURLEncoding.EncodeToString(h[:])
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)

	jwk := BuildJWKThumbprint(&d.privateKey.PublicKey)
	token.Header["typ"] = "dpop+jwt"
	token.Header["jwk"] = jwk

	signedToken, err := token.SignedString(d.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign DPoP proof: %w", err)
	}

	return signedToken, nil
}

func (d *Prover) CreateProofWithATH(method, uri, accessToken string) (string, error) {
	return d.CreateProof(method, uri, accessToken)
}

func BuildJWKThumbprint(pub *ecdsa.PublicKey) map[string]interface{} {
	return map[string]interface{}{
		"kty": "EC",
		"crv": "P-256",
		"x":   base64.RawURLEncoding.EncodeToString(PadCoordinate(pub.X, 32)),
		"y":   base64.RawURLEncoding.EncodeToString(PadCoordinate(pub.Y, 32)),
	}
}

func PadCoordinate(n *big.Int, size int) []byte {
	b := n.Bytes()
	if len(b) >= size {
		return b
	}
	padded := make([]byte, size)
	copy(padded[size-len(b):], b)
	return padded
}

func PublicKeyJWK(key *ecdsa.PublicKey) ([]byte, error) {
	jwk := BuildJWKThumbprint(key)
	return json.Marshal(jwk)
}

func ComputeJKTThumbprint(key *ecdsa.PublicKey) string {
	jwk := map[string]string{
		"crv": "P-256",
		"kty": "EC",
		"x":   base64.RawURLEncoding.EncodeToString(PadCoordinate(key.X, 32)),
		"y":   base64.RawURLEncoding.EncodeToString(PadCoordinate(key.Y, 32)),
	}

	canonical, err := json.Marshal(jwk)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(canonical)
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
