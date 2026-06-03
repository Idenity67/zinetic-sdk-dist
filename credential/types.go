package credential

import "time"

type AnchorRequest struct {
	TenantID     string            `json:"tenant_id"`
	SubjectID    string            `json:"subject_id"`
	CredentialID string            `json:"credential_id"`
	PublicKey    string            `json:"public_key"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type AnchorResponse struct {
	AnchorID    string       `json:"anchor_id"`
	Status      string       `json:"status"`
	AnchoredAt  time.Time    `json:"anchored_at"`
	MerkleProof *MerkleProof `json:"merkle_proof,omitempty"`
}

type MerkleProof struct {
	Root     string   `json:"root"`
	LeafHash string   `json:"leaf_hash"`
	TreeSize uint64   `json:"tree_size"`
	Hashes   []string `json:"hashes"`
	LogIndex uint64   `json:"log_index"`
}

type StatusResponse struct {
	AnchorID  string    `json:"anchor_id"`
	Status    string    `json:"status"`
	Reason    string    `json:"reason,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RevokeRequest struct {
	AnchorID string `json:"anchor_id"`
	Reason   string `json:"reason"`
	Actor    string `json:"actor"`
}

type RevokeResponse struct {
	RevokedAt    time.Time `json:"revoked_at"`
	AuditEventID string    `json:"audit_event_id"`
}

type VerifyRequest struct {
	AnchorID    string       `json:"anchor_id"`
	MerkleProof *MerkleProof `json:"merkle_proof"`
}

type VerifyResponse struct {
	Valid      bool      `json:"valid"`
	AnchorID   string    `json:"anchor_id"`
	Status     string    `json:"status"`
	VerifiedAt time.Time `json:"verified_at"`
}

type ListAnchorsResponse struct {
	Data       []AnchorResponse `json:"data"`
	NextCursor string           `json:"next_cursor,omitempty"`
	HasMore    bool             `json:"has_more"`
}
