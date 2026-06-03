package nhi

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
)

type encryptionKey struct {
	private *ecdh.PrivateKey
	jwk     map[string]string
}

type EncryptedCredentials struct {
	Algorithm  string            `json:"alg"`
	Encryption string            `json:"enc"`
	Ephemeral  map[string]string `json:"epk"`
	Nonce      string            `json:"nonce"`
	Ciphertext string            `json:"ciphertext"`
}

func newEncryptionKey() (*encryptionKey, error) {
	key, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("nhi: generate credential encryption key: %w", err)
	}
	pub := key.PublicKey().Bytes()
	if len(pub) != 65 || pub[0] != 4 {
		return nil, fmt.Errorf("nhi: unexpected P-256 public key encoding")
	}
	return &encryptionKey{
		private: key,
		jwk: map[string]string{
			"kty": "EC",
			"crv": "P-256",
			"x":   base64.RawURLEncoding.EncodeToString(pub[1:33]),
			"y":   base64.RawURLEncoding.EncodeToString(pub[33:65]),
		},
	}, nil
}

func (k *encryptionKey) PublicJWK() map[string]string {
	out := make(map[string]string, len(k.jwk))
	for key, value := range k.jwk {
		out[key] = value
	}
	return out
}

func (k *encryptionKey) Decrypt(pkg *EncryptedCredentials) (map[string]string, error) {
	if k == nil || k.private == nil {
		return nil, fmt.Errorf("nhi: credential encryption key is missing")
	}
	if pkg == nil {
		return nil, fmt.Errorf("nhi: encrypted credentials are missing")
	}
	if pkg.Algorithm != "ECDH-ES+HKDF-SHA256" || pkg.Encryption != "A256GCM" {
		return nil, fmt.Errorf("nhi: unsupported encrypted credential package: %s/%s", pkg.Algorithm, pkg.Encryption)
	}
	peer, err := publicKeyFromJWK(pkg.Ephemeral)
	if err != nil {
		return nil, fmt.Errorf("nhi: parse server encryption key: %w", err)
	}
	secret, err := k.private.ECDH(peer)
	if err != nil {
		return nil, fmt.Errorf("nhi: derive credential encryption secret: %w", err)
	}
	nonce, err := base64.RawURLEncoding.DecodeString(pkg.Nonce)
	if err != nil {
		return nil, fmt.Errorf("nhi: decode credential nonce: %w", err)
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(pkg.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("nhi: decode credential ciphertext: %w", err)
	}
	derivedKey, err := hkdf.Key(sha256.New, secret, nonce, "zinetic-nhi-credential-package-v2", 32)
	if err != nil {
		return nil, fmt.Errorf("nhi: derive credential package key: %w", err)
	}
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, fmt.Errorf("nhi: initialize credential decryptor: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("nhi: initialize credential GCM: %w", err)
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("nhi: decrypt credential package: %w", err)
	}
	var creds map[string]string
	if err := json.Unmarshal(plaintext, &creds); err != nil {
		return nil, fmt.Errorf("nhi: decode credential package: %w", err)
	}
	if len(creds) == 0 {
		return nil, fmt.Errorf("nhi: credential package is empty")
	}
	return creds, nil
}

func publicKeyFromJWK(jwk map[string]string) (*ecdh.PublicKey, error) {
	if jwk["kty"] != "EC" || jwk["crv"] != "P-256" {
		return nil, fmt.Errorf("expected EC P-256 JWK")
	}
	x, err := base64.RawURLEncoding.DecodeString(jwk["x"])
	if err != nil {
		return nil, fmt.Errorf("decode x: %w", err)
	}
	y, err := base64.RawURLEncoding.DecodeString(jwk["y"])
	if err != nil {
		return nil, fmt.Errorf("decode y: %w", err)
	}
	if len(x) != 32 || len(y) != 32 {
		return nil, fmt.Errorf("invalid P-256 coordinate length")
	}
	raw := make([]byte, 65)
	raw[0] = 4
	copy(raw[1:33], x)
	copy(raw[33:], y)
	return ecdh.P256().NewPublicKey(raw)
}

type signatureEnvelope struct {
	CredentialType       string                `json:"credential_type"`
	Credentials          map[string]string     `json:"credentials,omitempty"`
	EncryptedCredentials *EncryptedCredentials `json:"encrypted_credentials,omitempty"`
	ExpiresAt            string                `json:"expires_at"`
	TTLSeconds           int                   `json:"ttl_seconds"`
	AuditID              string                `json:"audit_id,omitempty"`
	PolicyVersion        string                `json:"policy_version,omitempty"`
	PolicySigningKeyID   string                `json:"policy_signing_key_id,omitempty"`
	TransactionHash      string                `json:"transaction_hash,omitempty"`
	LedgerAnchorHash     string                `json:"ledger_anchor_hash,omitempty"`
}

func verifyPolicySignature(resp *exchangeResponse, publicKey string, required bool) error {
	publicKey = strings.TrimSpace(publicKey)
	if publicKey == "" {
		if required {
			return fmt.Errorf("nhi: policy signature verification requires PolicyPublicKey")
		}
		return nil
	}
	if resp.PolicySignature == "" {
		return fmt.Errorf("nhi: exchange response is missing policy signature")
	}
	pub, err := parseEd25519PublicKey(publicKey)
	if err != nil {
		return err
	}
	sig, err := base64.RawURLEncoding.DecodeString(resp.PolicySignature)
	if err != nil {
		return fmt.Errorf("nhi: decode policy signature: %w", err)
	}
	expiresAt := resp.rawExpiresAt
	if expiresAt == "" {
		expiresAt = resp.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	payload, err := json.Marshal(signatureEnvelope{
		CredentialType:       resp.CredentialType,
		Credentials:          resp.Credentials,
		EncryptedCredentials: resp.EncryptedCredentials,
		ExpiresAt:            expiresAt,
		TTLSeconds:           resp.TTLSeconds,
		AuditID:              resp.AuditID,
		PolicyVersion:        resp.PolicyVersion,
		PolicySigningKeyID:   resp.PolicySigningKeyID,
		TransactionHash:      resp.TransactionHash,
		LedgerAnchorHash:     resp.LedgerAnchorHash,
	})
	if err != nil {
		return fmt.Errorf("nhi: canonicalize policy envelope: %w", err)
	}
	if !ed25519.Verify(pub, payload, sig) {
		return errors.New("nhi: policy signature verification failed")
	}
	return nil
}

func parseEd25519PublicKey(raw string) (ed25519.PublicKey, error) {
	if block, _ := pem.Decode([]byte(raw)); block != nil {
		return parseEd25519PublicKeyBytes(block.Bytes)
	}
	decoded, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(raw))
	if err != nil {
		decoded, err = base64.StdEncoding.DecodeString(strings.TrimSpace(raw))
	}
	if err != nil {
		return nil, fmt.Errorf("nhi: decode Ed25519 policy public key: %w", err)
	}
	return parseEd25519PublicKeyBytes(decoded)
}

func parseEd25519PublicKeyBytes(decoded []byte) (ed25519.PublicKey, error) {
	if len(decoded) != ed25519.PublicKeySize {
		key, err := x509.ParsePKIXPublicKey(decoded)
		if err != nil {
			return nil, fmt.Errorf("nhi: Ed25519 policy public key must be %d raw bytes or PKIX DER", ed25519.PublicKeySize)
		}
		pub, ok := key.(ed25519.PublicKey)
		if !ok {
			return nil, fmt.Errorf("nhi: policy public key must be Ed25519")
		}
		return pub, nil
	}
	return ed25519.PublicKey(decoded), nil
}
