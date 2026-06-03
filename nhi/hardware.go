package nhi

import (
	"context"
	"errors"
)

var ErrHardwareUnavailable = errors.New("nhi: hardware key provider is not available on this platform")

type KeyAlgorithm string

const (
	KeyAlgECDSAP256 KeyAlgorithm = "ECDSA-P256"
	KeyAlgECDSAP384 KeyAlgorithm = "ECDSA-P384"
	KeyAlgRSA2048   KeyAlgorithm = "RSA-2048"
	KeyAlgRSA4096   KeyAlgorithm = "RSA-4096"
)

type KeyOptions struct {
	Algorithm     KeyAlgorithm
	Label         string
	NonExportable bool
}

type PublicKeyInfo struct {
	Algorithm    KeyAlgorithm      `json:"algorithm"`
	PublicKeyPEM string            `json:"public_key_pem"`
	KeyID        string            `json:"key_id"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type AttestationDocument struct {
	Format    string            `json:"format"`
	Data      []byte            `json:"data"`
	CertChain [][]byte          `json:"cert_chain,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type HardwareKeyProvider interface {
	Available(ctx context.Context) bool
	GenerateKey(ctx context.Context, opts KeyOptions) (*PublicKeyInfo, error)
	Sign(ctx context.Context, challenge []byte) ([]byte, error)
	Attest(ctx context.Context) (*AttestationDocument, error)
}
