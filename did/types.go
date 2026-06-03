package did

import "time"

type DID struct {
	ID         string      `json:"id"`
	Method     string      `json:"method"`
	Controller string      `json:"controller"`
	UserID     string      `json:"user_id"`
	Document   interface{} `json:"document"`
	Active     bool        `json:"active"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

type CreateRequest struct {
	Method             string `json:"method"`
	UserID             string `json:"user_id"`
	PublicKeyMultibase string `json:"public_key_multibase,omitempty"`
	KeyOwnership       string `json:"key_ownership,omitempty"`
}

type ListResponse struct {
	DIDs       []DID  `json:"dids"`
	Count      int    `json:"count"`
	NextCursor string `json:"next_cursor,omitempty"`
}

type RotateKeyRequest struct {
	DIDID  string `json:"did_id"`
	UserID string `json:"user_id"`
	Reason string `json:"reason,omitempty"`
	Async  bool   `json:"async,omitempty"`
}

type RotateKeyResponse struct {
	DIDID            string    `json:"did_id"`
	OldKeyID         string    `json:"old_key_id,omitempty"`
	NewKeyID         string    `json:"new_key_id,omitempty"`
	WorkflowID       string    `json:"workflow_id,omitempty"`
	RotatedAt        time.Time `json:"rotated_at"`
	BlockchainTxHash string    `json:"blockchain_tx_hash,omitempty"`
}

type RotationHistory struct {
	History []RotateKeyResponse `json:"history"`
	Count   int                 `json:"count"`
}

type ConsentReceipt struct {
	ID            string     `json:"id"`
	Subject       string     `json:"subject"`
	PolicyURI     string     `json:"policy_uri"`
	PolicyText    string     `json:"policy_text"`
	Locale        string     `json:"locale,omitempty"`
	UIVersion     string     `json:"ui_version,omitempty"`
	Withdrawable  bool       `json:"withdrawable"`
	WithdrawalURI string     `json:"withdrawal_uri,omitempty"`
	Purposes      []Purpose  `json:"purposes"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	WithdrawnAt   *time.Time `json:"withdrawn_at,omitempty"`
}

type Purpose struct {
	Name     string `json:"name"`
	Optional bool   `json:"optional"`
}

type IssueConsentRequest struct {
	Subject       string    `json:"subject"`
	PolicyURI     string    `json:"policy_uri"`
	PolicyText    string    `json:"policy_text"`
	Locale        string    `json:"locale,omitempty"`
	UIVersion     string    `json:"ui_version,omitempty"`
	Withdrawable  bool      `json:"withdrawable"`
	WithdrawalURI string    `json:"withdrawal_uri,omitempty"`
	Purposes      []Purpose `json:"purposes"`
}

type ConsentListResponse struct {
	Receipts []ConsentReceipt `json:"receipts"`
	Count    int              `json:"count"`
}

type DIDCommMessage struct {
	ID   string      `json:"id"`
	Type string      `json:"type"`
	From string      `json:"from"`
	To   string      `json:"to"`
	Body interface{} `json:"body"`
}

type DIDCommIdentity struct {
	DID string `json:"did"`
}
