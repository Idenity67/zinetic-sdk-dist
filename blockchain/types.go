package blockchain

import "time"

type AnchorRequest struct {
	CredentialID   string `json:"credential_id"`
	IssuerDID      string `json:"issuer_did"`
	CredentialHash string `json:"credential_hash"`
}

type AnchorResponse struct {
	TxHash          string `json:"tx_hash"`
	Status          string `json:"status"`
	CredentialIndex int    `json:"credential_index"`
}

type RevokeCredentialRequest struct {
	CredentialID    string `json:"credential_id"`
	IssuerDID       string `json:"issuer_did"`
	CredentialIndex int    `json:"credential_index"`
	Reason          string `json:"reason"`
}

type BatchRevokeRequest struct {
	IssuerDID         string `json:"issuer_did"`
	CredentialIndices []int  `json:"credential_indices"`
	Reason            string `json:"reason"`
}

type CredentialStatusResponse struct {
	Status     string `json:"status"`
	StatusCode int    `json:"status_code"`
}

type CredentialHashResponse struct {
	CredentialHash string `json:"credential_hash"`
}

type RevocationStatusResponse struct {
	Revoked bool `json:"revoked"`
}

type RevocationsResponse struct {
	RevokedIndices []int `json:"revoked_indices"`
	Count          int   `json:"count"`
}

type SuspendRequest struct {
	IssuerDID       string `json:"issuer_did"`
	CredentialIndex int    `json:"credential_index"`
	Reason          string `json:"reason"`
}

type UnsuspendRequest struct {
	IssuerDID       string `json:"issuer_did"`
	CredentialIndex int    `json:"credential_index"`
}

type BatchSuspendRequest struct {
	IssuerDID         string `json:"issuer_did"`
	CredentialIndices []int  `json:"credential_indices"`
	Reason            string `json:"reason"`
}

type SuspensionStatusResponse struct {
	Suspended bool `json:"suspended"`
}

type SuspensionsResponse struct {
	SuspendedIndices []int `json:"suspended_indices"`
	Count            int   `json:"count"`
}

type RegisterDIDRequest struct {
	DID         string `json:"did"`
	DIDDocument string `json:"did_document"`
}

type RegisterDIDMetaRequest struct {
	DID         string `json:"did"`
	DIDDocument string `json:"did_document"`
	Deadline    uint64 `json:"deadline"`
	Signature   string `json:"signature"`
}

type RegisterDIDResponse struct {
	TxHash string `json:"tx_hash"`
	Status string `json:"status"`
	DID    string `json:"did"`
}

type UpdateDIDRequest struct {
	DIDDocument string `json:"did_document"`
}

type ResolvedDID struct {
	DID         string `json:"did"`
	DIDDocument string `json:"did_document"`
	Active      bool   `json:"active"`
}

type DIDStatusResponse struct {
	DID    string `json:"did"`
	Active bool   `json:"active"`
}

type TxResponse struct {
	TxHash string `json:"tx_hash"`
	Status string `json:"status"`
}

type TrustedIssuer struct {
	DID             string     `json:"did"`
	Name            string     `json:"name"`
	CredentialTypes []string   `json:"credential_types"`
	TrustLevel      int        `json:"trust_level"`
	TrustLevelLabel string     `json:"trust_level_label"`
	AccreditedBy    string     `json:"accredited_by,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	Revoked         bool       `json:"revoked"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type RegisterIssuerRequest struct {
	DID             string     `json:"did"`
	Name            string     `json:"name"`
	CredentialTypes []string   `json:"credential_types"`
	TrustLevel      int        `json:"trust_level"`
	AccreditedBy    string     `json:"accredited_by,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
}

type UpdateIssuerRequest struct {
	DID             string     `json:"did"`
	CredentialTypes []string   `json:"credential_types,omitempty"`
	TrustLevel      int        `json:"trust_level,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
}

type RevokeIssuerRequest struct {
	DID    string `json:"did"`
	Reason string `json:"reason"`
}

type IssuerTrustResponse struct {
	Trusted         bool   `json:"trusted"`
	TrustLevel      int    `json:"trust_level"`
	TrustLevelLabel string `json:"trust_level_label"`
}

type TrustedVerifier struct {
	DID           string    `json:"did"`
	Name          string    `json:"name"`
	AcceptedTypes []string  `json:"accepted_types"`
	TrustLevel    int       `json:"trust_level"`
	AccreditedBy  string    `json:"accredited_by,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type RegisterVerifierRequest struct {
	DID           string   `json:"did"`
	Name          string   `json:"name"`
	AcceptedTypes []string `json:"accepted_types"`
	TrustLevel    int      `json:"trust_level"`
	AccreditedBy  string   `json:"accredited_by,omitempty"`
}

type TrustedSchema struct {
	SchemaID      string    `json:"schema_id"`
	Name          string    `json:"name"`
	SchemaVersion string    `json:"schema_version"`
	TrustLevel    int       `json:"trust_level"`
	IssuedBy      []string  `json:"issued_by"`
	CreatedAt     time.Time `json:"created_at"`
}

type RegisterSchemaRequest struct {
	SchemaID      string   `json:"schema_id"`
	Name          string   `json:"name"`
	SchemaVersion string   `json:"schema_version"`
	TrustLevel    int      `json:"trust_level"`
	IssuedBy      []string `json:"issued_by"`
}

type EvaluateTrustResponse struct {
	Trusted    bool   `json:"trusted"`
	TrustLevel int    `json:"trust_level"`
	TrustLabel string `json:"trust_label"`
}

type TrustStats struct {
	IssuerCount int `json:"issuer_count"`
}

type BlockchainStatus struct {
	Connected        bool   `json:"connected"`
	Network          string `json:"network"`
	ChainID          int64  `json:"chain_id"`
	CurrentBlock     uint64 `json:"current_block"`
	WalletAddress    string `json:"wallet_address"`
	WalletBalance    string `json:"wallet_balance"`
	FailoverRPCCount int    `json:"failover_rpc_count"`
}

type BlockchainMetadata struct {
	CredentialStatuses []string `json:"credential_statuses"`
	RevocationReasons  []string `json:"revocation_reasons"`
	SuspensionReasons  []string `json:"suspension_reasons"`
	MaxBatchSize       int      `json:"max_batch_size"`
}

type IssuerInfo struct {
	CredentialCount int       `json:"credential_count"`
	LastUpdated     time.Time `json:"last_updated"`
	Registered      bool      `json:"registered"`
}
