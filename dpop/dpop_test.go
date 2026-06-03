package dpop

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateKey(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate DPoP key: %v", err)
	}
	if key.Curve != elliptic.P256() {
		t.Fatal("expected P-256 curve")
	}
}

func TestProver_CreateProof(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	prover := NewProver(key)

	proof, err := prover.CreateProof("POST", "https://api.zinetic.io/v1/agents", "")
	if err != nil {
		t.Fatalf("failed to create proof: %v", err)
	}

	parts := strings.Split(proof, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 JWT parts, got %d", len(parts))
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("failed to decode header: %v", err)
	}

	var header map[string]interface{}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		t.Fatalf("failed to parse header: %v", err)
	}

	if header["typ"] != "dpop+jwt" {
		t.Fatalf("expected typ=dpop+jwt, got %v", header["typ"])
	}
	if header["alg"] != "ES256" {
		t.Fatalf("expected alg=ES256, got %v", header["alg"])
	}

	jwkMap, ok := header["jwk"].(map[string]interface{})
	if !ok {
		t.Fatal("expected jwk in header")
	}
	if jwkMap["kty"] != "EC" || jwkMap["crv"] != "P-256" {
		t.Fatalf("unexpected JWK values: %v", jwkMap)
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		t.Fatalf("failed to parse claims: %v", err)
	}

	if claims["htm"] != "POST" {
		t.Fatalf("expected htm=POST, got %v", claims["htm"])
	}
	if claims["htu"] != "https://api.zinetic.io/v1/agents" {
		t.Fatalf("expected correct htu, got %v", claims["htu"])
	}
	if claims["jti"] == nil || claims["jti"] == "" {
		t.Fatal("expected non-empty jti")
	}
	if claims["iat"] == nil {
		t.Fatal("expected iat claim")
	}
}

func TestProver_CreateProofWithATH(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	prover := NewProver(key)

	proof, err := prover.CreateProofWithATH("GET", "https://api.zinetic.io/v1/me", "access-token-value")
	if err != nil {
		t.Fatalf("failed to create proof with ATH: %v", err)
	}

	parts := strings.Split(proof, ".")
	payloadBytes, _ := base64.RawURLEncoding.DecodeString(parts[1])

	var claims map[string]interface{}
	json.Unmarshal(payloadBytes, &claims)

	if claims["ath"] == nil || claims["ath"] == "" {
		t.Fatal("expected ath claim when access token provided")
	}
}

func TestProver_ServerNonce(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	prover := NewProver(key)

	prover.SetServerNonce("server-nonce-123")

	if prover.GetServerNonce() != "server-nonce-123" {
		t.Fatal("expected server nonce to be stored")
	}

	proof, err := prover.CreateProof("POST", "https://api.zinetic.io/v1/token", "")
	if err != nil {
		t.Fatalf("failed to create proof: %v", err)
	}

	parts := strings.Split(proof, ".")
	payloadBytes, _ := base64.RawURLEncoding.DecodeString(parts[1])

	var claims map[string]interface{}
	json.Unmarshal(payloadBytes, &claims)

	if claims["nonce"] != "server-nonce-123" {
		t.Fatalf("expected nonce in claims, got %v", claims["nonce"])
	}
}

func TestComputeJKTThumbprint(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	thumbprint := ComputeJKTThumbprint(&key.PublicKey)
	if thumbprint == "" {
		t.Fatal("expected non-empty thumbprint")
	}

	thumbprint2 := ComputeJKTThumbprint(&key.PublicKey)
	if thumbprint != thumbprint2 {
		t.Fatal("expected deterministic thumbprint")
	}

	key2, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	thumbprint3 := ComputeJKTThumbprint(&key2.PublicKey)
	if thumbprint == thumbprint3 {
		t.Fatal("expected different thumbprints for different keys")
	}
}

func TestProof_VerifiableWithPublicKey(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	prover := NewProver(key)

	proof, _ := prover.CreateProof("POST", "https://api.zinetic.io/v1/test", "")

	token, err := jwt.Parse(proof, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return &key.PublicKey, nil
	})
	if err != nil {
		t.Fatalf("failed to verify DPoP proof signature: %v", err)
	}
	if !token.Valid {
		t.Fatal("proof should be valid")
	}
}

func TestPadCoordinate(t *testing.T) {
	small := PadCoordinate(big.NewInt(1), 32)
	if len(small) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(small))
	}
	if small[31] != 1 {
		t.Fatal("expected value at last byte")
	}
	for i := 0; i < 31; i++ {
		if small[i] != 0 {
			t.Fatalf("expected zero padding at byte %d", i)
		}
	}
}
