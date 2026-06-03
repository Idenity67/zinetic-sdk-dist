package nhi

import (
	"context"
	"testing"
)

type noopHardwareProvider struct{}

func (n *noopHardwareProvider) Available(_ context.Context) bool {
	return false
}

func (n *noopHardwareProvider) GenerateKey(_ context.Context, _ KeyOptions) (*PublicKeyInfo, error) {
	return nil, ErrHardwareUnavailable
}

func (n *noopHardwareProvider) Sign(_ context.Context, _ []byte) ([]byte, error) {
	return nil, ErrHardwareUnavailable
}

func (n *noopHardwareProvider) Attest(_ context.Context) (*AttestationDocument, error) {
	return nil, ErrHardwareUnavailable
}

func TestHardwareKeyProvider_InterfaceCompliance(t *testing.T) {
	var _ HardwareKeyProvider = (*noopHardwareProvider)(nil)
}

func TestNoopProvider_NotAvailable(t *testing.T) {
	p := &noopHardwareProvider{}
	if p.Available(context.Background()) {
		t.Fatal("noop provider should not be available")
	}
}

func TestNoopProvider_GenerateKeyFails(t *testing.T) {
	p := &noopHardwareProvider{}
	_, err := p.GenerateKey(context.Background(), KeyOptions{Algorithm: KeyAlgECDSAP256})
	if err != ErrHardwareUnavailable {
		t.Fatalf("expected ErrHardwareUnavailable, got %v", err)
	}
}

func TestNoopProvider_SignFails(t *testing.T) {
	p := &noopHardwareProvider{}
	_, err := p.Sign(context.Background(), []byte("challenge"))
	if err != ErrHardwareUnavailable {
		t.Fatalf("expected ErrHardwareUnavailable, got %v", err)
	}
}

func TestNoopProvider_AttestFails(t *testing.T) {
	p := &noopHardwareProvider{}
	_, err := p.Attest(context.Background())
	if err != ErrHardwareUnavailable {
		t.Fatalf("expected ErrHardwareUnavailable, got %v", err)
	}
}
