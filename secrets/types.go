package secrets

import "time"

type MountEngineRequest struct {
	Path       string            `json:"path"`
	EngineType string            `json:"engine_type"`
	Config     map[string]string `json:"config,omitempty"`
	SealWrap   bool              `json:"seal_wrap,omitempty"`
}

type MountEngineStatus struct {
	Path       string            `json:"path"`
	EngineType string            `json:"engine_type"`
	Config     map[string]string `json:"config,omitempty"`
	SealWrap   bool              `json:"seal_wrap"`
	CreatedAt  time.Time         `json:"created_at"`
}

type ListMountsResponse struct {
	Mounts []MountEngineStatus `json:"mounts"`
}

type KVWriteRequest struct {
	Data     map[string]string `json:"data"`
	Metadata map[string]string `json:"metadata,omitempty"`
	TTL      string            `json:"ttl,omitempty"`
}

type KVWriteResponse struct {
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}

type KVReadResponse struct {
	Data      map[string]string `json:"data"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Version   int               `json:"version"`
	CreatedAt time.Time         `json:"created_at"`
}

type TransitEncryptRequest struct {
	Plaintext string `json:"plaintext"`
	KeyName   string `json:"key_name"`
}

type TransitDecryptRequest struct {
	Ciphertext string `json:"ciphertext"`
	KeyName    string `json:"key_name"`
}

type TransitResponse struct {
	Result string `json:"result"`
}

type LeaseRequest struct {
	TTL          string            `json:"ttl"`
	IdentityID   string            `json:"identity_id"`
	IdentityType string            `json:"identity_type"`
	Attestation  map[string]string `json:"attestation,omitempty"`
}

type LeaseResponse struct {
	LeaseID       string            `json:"lease_id"`
	Data          map[string]string `json:"data"`
	Renewable     bool              `json:"renewable"`
	LeaseDuration int               `json:"lease_duration"`
	ExpiresAt     time.Time         `json:"expires_at"`
}

type RenewLeaseRequest struct {
	LeaseID   string `json:"lease_id"`
	Increment string `json:"increment,omitempty"`
}

type RevokeLeaseRequest struct {
	LeaseID string `json:"lease_id"`
}
