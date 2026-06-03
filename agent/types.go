package agent

import "time"

type Agent struct {
	ID             string            `json:"id"`
	TenantID       string            `json:"tenant_id"`
	Name           string            `json:"name"`
	AgentType      string            `json:"agent_type"`
	OwnerID        string            `json:"owner_id"`
	State          string            `json:"state"`
	Capabilities   []Capability      `json:"capabilities,omitempty"`
	TrustScore     float64           `json:"trust_score"`
	DriftScore     float64           `json:"drift_score"`
	AnchorID       string            `json:"anchor_id,omitempty"`
	PublicKeyJWK   string            `json:"public_key_jwk,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	LastActivityAt time.Time         `json:"last_activity_at,omitempty"`
	PostureStatus  *PostureStatus    `json:"posture_status,omitempty"`
}

type Capability struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	AllowedTools     []string `json:"allowed_tools"`
	AllowedResources []string `json:"allowed_resources"`
	RateLimit        int      `json:"rate_limit"`
	TimeWindowStart  string   `json:"time_window_start,omitempty"`
	TimeWindowEnd    string   `json:"time_window_end,omitempty"`
}

type PostureStatus struct {
	ModelVersion         string    `json:"model_version"`
	ModelHash            string    `json:"model_hash"`
	KnowledgeBaseVersion string    `json:"knowledge_base_version"`
	KnowledgeBaseHash    string    `json:"knowledge_base_hash"`
	SafetyFilterState    string    `json:"safety_filter_state"`
	TEEStatus            string    `json:"tee_status"`
	VerifiedAt           time.Time `json:"verified_at"`
}

type RegisterRequest struct {
	TenantID     string            `json:"tenant_id"`
	Name         string            `json:"name"`
	AgentType    string            `json:"agent_type"`
	OwnerID      string            `json:"owner_id"`
	Capabilities []Capability      `json:"capabilities,omitempty"`
	PublicKeyJWK string            `json:"public_key_jwk,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type RegisterResponse struct {
	Agent    *Agent `json:"agent"`
	AnchorID string `json:"anchor_id"`
}

type UpdateRequest struct {
	Name         string            `json:"name,omitempty"`
	State        string            `json:"state,omitempty"`
	Capabilities []Capability      `json:"capabilities,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type ListResponse struct {
	Data       []Agent `json:"data"`
	NextCursor string  `json:"next_cursor,omitempty"`
	HasMore    bool    `json:"has_more"`
}

type AgentCard struct {
	AgentID                string   `json:"agent_id"`
	Name                   string   `json:"name"`
	Description            string   `json:"description,omitempty"`
	Capabilities           []string `json:"capabilities"`
	AuthenticationRequired []string `json:"authentication_required"`
	SupportedProtocols     []string `json:"supported_protocols"`
	TrustAnchors           []string `json:"trust_anchors,omitempty"`
	Endpoint               string   `json:"endpoint"`
	Version                string   `json:"version"`
}

type DelegationRequest struct {
	FromAgentID  string   `json:"from_agent_id"`
	ToAgentID    string   `json:"to_agent_id"`
	Capabilities []string `json:"capabilities"`
	MaxDepth     int      `json:"max_depth,omitempty"`
	TTL          int      `json:"ttl,omitempty"`
}

type DelegationResponse struct {
	DelegationID string    `json:"delegation_id"`
	Chain        []string  `json:"chain"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type DelegationChain struct {
	DelegationID string           `json:"delegation_id"`
	Links        []DelegationLink `json:"links"`
	CreatedAt    time.Time        `json:"created_at"`
	ExpiresAt    time.Time        `json:"expires_at"`
}

type DelegationLink struct {
	FromAgentID  string   `json:"from_agent_id"`
	ToAgentID    string   `json:"to_agent_id"`
	Capabilities []string `json:"capabilities"`
	Depth        int      `json:"depth"`
}

type BehavioralBaseline struct {
	AgentID         string           `json:"agent_id"`
	APIPatterns     map[string]int   `json:"api_patterns"`
	ToolUsage       map[string]int   `json:"tool_usage"`
	DataVolumes     map[string]int64 `json:"data_volumes"`
	CalibratedAt    time.Time        `json:"calibrated_at"`
	OwnerApprovedAt time.Time        `json:"owner_approved_at,omitempty"`
}

type DriftEvent struct {
	AgentID    string    `json:"agent_id"`
	Severity   string    `json:"severity"`
	Metric     string    `json:"metric"`
	Expected   float64   `json:"expected"`
	Actual     float64   `json:"actual"`
	Deviation  float64   `json:"deviation"`
	DetectedAt time.Time `json:"detected_at"`
}

type ForensicSnapshot struct {
	AgentID      string        `json:"agent_id"`
	Actions      []AgentAction `json:"actions"`
	ToolCalls    []ToolCall    `json:"tool_calls"`
	DataAccessed []DataAccess  `json:"data_accessed"`
	CapturedAt   time.Time     `json:"captured_at"`
}

type AgentAction struct {
	ActionID  string    `json:"action_id"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Timestamp time.Time `json:"timestamp"`
}

type ToolCall struct {
	ToolName   string            `json:"tool_name"`
	MCPServer  string            `json:"mcp_server,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
	Result     string            `json:"result,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
}

type DataAccess struct {
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	AccessType   string    `json:"access_type"`
	BytesRead    int64     `json:"bytes_read,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

type IntentProof struct {
	UserID              string `json:"user_id"`
	OriginalRequestHash string `json:"original_request_hash"`
	DeclaredScope       string `json:"declared_scope"`
	SessionID           string `json:"session_id"`
	IssuedAt            int64  `json:"iat"`
	Proof               string `json:"proof"`
}

type KillSwitchRequest struct {
	TenantID string `json:"tenant_id"`
	Reason   string `json:"reason"`
	Actor    string `json:"actor"`
}

type KillSwitchResponse struct {
	AgentsRevoked int       `json:"agents_revoked"`
	TokensRevoked int       `json:"tokens_revoked"`
	ExecutedAt    time.Time `json:"executed_at"`
	AuditEventID  string    `json:"audit_event_id"`
}

type InventoryResponse struct {
	Agents         []Agent `json:"agents"`
	TotalActive    int     `json:"total_active"`
	TotalSuspended int     `json:"total_suspended"`
	TotalRevoked   int     `json:"total_revoked"`
}

type MCPServer struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Endpoint     string            `json:"endpoint"`
	Tools        []MCPTool         `json:"tools"`
	AuthRequired bool              `json:"auth_required"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	RegisteredAt time.Time         `json:"registered_at"`
}

type MCPTool struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Parameters  map[string]string `json:"parameters,omitempty"`
	Scopes      []string          `json:"scopes"`
}

type MCPServerListResponse struct {
	Data       []MCPServer `json:"data"`
	NextCursor string      `json:"next_cursor,omitempty"`
	HasMore    bool        `json:"has_more"`
}

type MCPToolAuthRequest struct {
	AgentID    string            `json:"agent_id"`
	MCPServer  string            `json:"mcp_server"`
	ToolName   string            `json:"tool_name"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

type MCPToolAuthResponse struct {
	Authorized bool   `json:"authorized"`
	ReasonCode string `json:"reason_code,omitempty"`
	PolicyID   string `json:"policy_id,omitempty"`
}

type MCPToolCallRequest struct {
	ServerID     string                 `json:"server_id"`
	AgentID      string                 `json:"agent_id"`
	Tool         string                 `json:"tool"`
	Scope        string                 `json:"scope,omitempty"`
	InvocationID string                 `json:"invocation_id,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
}

type MCPToolCallResponse struct {
	ID                 string      `json:"id"`
	AgentID            string      `json:"agent_id"`
	MCPServerID        string      `json:"mcp_server_id"`
	Tool               string      `json:"tool"`
	Result             interface{} `json:"result,omitempty"`
	AuthorizationToken string      `json:"authorization_token,omitempty"`
	AuthorizedAt       time.Time   `json:"authorized_at"`
	ExpiresAt          time.Time   `json:"expires_at"`
}
