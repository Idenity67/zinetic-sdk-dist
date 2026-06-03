package policy

import "time"

type Policy struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Description    string            `json:"description,omitempty"`
	Version        string            `json:"version"`
	TenantID       string            `json:"tenant_id,omitempty"`
	Rules          string            `json:"rules"`
	Status         string            `json:"status"`
	SimulationMode bool              `json:"simulation_mode"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

type CreateRequest struct {
	Name           string            `json:"name"`
	Description    string            `json:"description,omitempty"`
	Rules          string            `json:"rules"`
	TenantID       string            `json:"tenant_id,omitempty"`
	SimulationMode bool              `json:"simulation_mode,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

type UpdateRequest struct {
	Name           string            `json:"name,omitempty"`
	Description    string            `json:"description,omitempty"`
	Rules          string            `json:"rules,omitempty"`
	SimulationMode *bool             `json:"simulation_mode,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

type ListResponse struct {
	Data       []Policy `json:"data"`
	NextCursor string   `json:"next_cursor,omitempty"`
	HasMore    bool     `json:"has_more"`
}

type BundleInfo struct {
	Version     string    `json:"version"`
	Fingerprint string    `json:"fingerprint"`
	PolicyCount int       `json:"policy_count"`
	BuiltAt     time.Time `json:"built_at"`
	Status      string    `json:"status"`
}

type NamedLocation struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	CIDRs     []string `json:"cidrs,omitempty"`
	Countries []string `json:"countries,omitempty"`
	Trusted   bool     `json:"trusted"`
}

type NamedLocationRequest struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	CIDRs     []string `json:"cidrs,omitempty"`
	Countries []string `json:"countries,omitempty"`
	Trusted   bool     `json:"trusted"`
}

type BreakGlassRequest struct {
	Requester      string `json:"requester"`
	Approver1      string `json:"approver1"`
	Approver2      string `json:"approver2"`
	Reason         string `json:"reason"`
	MaxDuration    int    `json:"max_duration"`
	TargetResource string `json:"target_resource"`
}

type BreakGlassResponse struct {
	SessionID    string    `json:"session_id"`
	ExpiresAt    time.Time `json:"expires_at"`
	AuditEventID string    `json:"audit_event_id"`
}

type ReBAcCheckRequest struct {
	User     string `json:"user"`
	Relation string `json:"relation"`
	Object   string `json:"object"`
}

type ReBAcCheckResponse struct {
	Allowed bool `json:"allowed"`
}

type ReBAcWriteRequest struct {
	User     string `json:"user"`
	Relation string `json:"relation"`
	Object   string `json:"object"`
}

type ReBAcDeleteRequest struct {
	User     string `json:"user"`
	Relation string `json:"relation"`
	Object   string `json:"object"`
}
