package nhimgmt

import "time"

type Identity struct {
	ID           string            `json:"id"`
	Kind         string            `json:"kind"`
	Name         string            `json:"name"`
	Environment  string            `json:"environment"`
	Source       string            `json:"source"`
	Status       string            `json:"status"`
	SpiffeID     string            `json:"spiffe_id,omitempty"`
	OwnerHumanID string            `json:"owner_human_id,omitempty"`
	ExpiresAt    *time.Time        `json:"expires_at,omitempty"`
	Tags         map[string]string `json:"tags,omitempty"`
	TenantID     string            `json:"tenant_id"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type CreateRequest struct {
	Kind         string            `json:"kind"`
	Name         string            `json:"name"`
	Environment  string            `json:"environment"`
	Source       string            `json:"source"`
	SpiffeID     string            `json:"spiffe_id,omitempty"`
	OwnerHumanID string            `json:"owner_human_id,omitempty"`
	ExpiresAt    *time.Time        `json:"expires_at,omitempty"`
	Tags         map[string]string `json:"tags,omitempty"`
}

type UpdateRequest struct {
	Status       string            `json:"status,omitempty"`
	OwnerHumanID string            `json:"owner_human_id,omitempty"`
	ExpiresAt    *time.Time        `json:"expires_at,omitempty"`
	Tags         map[string]string `json:"tags,omitempty"`
}

type ListResponse struct {
	Identities []Identity `json:"identities"`
	Count      int        `json:"count"`
}

type RotateRequest struct {
	IdentityID string `json:"identity_id"`
}

type Connection struct {
	AgentID        string    `json:"agent_id"`
	TargetType     string    `json:"target_type"`
	TargetID       string    `json:"target_id"`
	TargetName     string    `json:"target_name"`
	TargetEndpoint string    `json:"target_endpoint"`
	Protocol       string    `json:"protocol"`
	AuthMethod     string    `json:"auth_method"`
	FirstSeenAt    time.Time `json:"first_seen_at"`
	LastSeenAt     time.Time `json:"last_seen_at"`
	RequestCount   int64     `json:"request_count"`
	ErrorCount     int64     `json:"error_count"`
	AvgLatencyMs   float64   `json:"avg_latency_ms"`
	IsActive       bool      `json:"is_active"`
	RiskScore      int       `json:"risk_score"`
}

type ConnectionsResponse struct {
	Connections []Connection `json:"connections"`
}

type GraphNode struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

type GraphEdge struct {
	Source    string `json:"source"`
	Target    string `json:"target"`
	Protocol  string `json:"protocol"`
	RiskScore int    `json:"risk_score"`
}

type ConnectivityGraph struct {
	Nodes          []GraphNode `json:"nodes"`
	Edges          []GraphEdge `json:"edges"`
	TotalNHIs      int         `json:"total_nhis"`
	TotalResources int         `json:"total_resources"`
	TotalEdges     int         `json:"total_edges"`
	HighRiskPaths  []string    `json:"high_risk_paths"`
	GeneratedAt    time.Time   `json:"generated_at"`
}

type RemediationAction struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	IdentityID  string `json:"identity_id"`
}

type EvaluateResponse struct {
	Actions []RemediationAction `json:"actions"`
}
