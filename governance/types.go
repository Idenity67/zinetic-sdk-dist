package governance

import "time"

type Campaign struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	Status        string    `json:"status"`
	ReviewerType  string    `json:"reviewer_type"`
	DueDate       time.Time `json:"due_date"`
	CompletionSLA int       `json:"completion_sla_days"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CampaignCreateRequest struct {
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	ReviewerType  string    `json:"reviewer_type"`
	DueDate       time.Time `json:"due_date"`
	CompletionSLA int       `json:"completion_sla_days"`
}

type CampaignListResponse struct {
	Data       []Campaign `json:"data"`
	NextCursor string     `json:"next_cursor,omitempty"`
	HasMore    bool       `json:"has_more"`
}

type CertificationItem struct {
	ID            string    `json:"id"`
	CampaignID    string    `json:"campaign_id"`
	PrincipalID   string    `json:"principal_id"`
	ResourceID    string    `json:"resource_id"`
	ResourceType  string    `json:"resource_type"`
	CurrentAccess string    `json:"current_access"`
	ReviewerID    string    `json:"reviewer_id"`
	Decision      string    `json:"decision"`
	RiskLevel     string    `json:"risk_level"`
	DecidedAt     time.Time `json:"decided_at,omitempty"`
}

type CertificationDecision struct {
	ItemID   string `json:"item_id"`
	Decision string `json:"decision"`
	Comment  string `json:"comment,omitempty"`
}

type AccessRequest struct {
	ID            string         `json:"id"`
	RequesterID   string         `json:"requester_id"`
	ResourceType  string         `json:"resource_type"`
	ResourceID    string         `json:"resource_id"`
	AccessLevel   string         `json:"access_level"`
	Justification string         `json:"justification"`
	Duration      int            `json:"duration_hours,omitempty"`
	Status        string         `json:"status"`
	ApprovalChain []ApprovalStep `json:"approval_chain,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type ApprovalStep struct {
	ApproverID string    `json:"approver_id"`
	Role       string    `json:"role"`
	Decision   string    `json:"decision"`
	Comment    string    `json:"comment,omitempty"`
	DecidedAt  time.Time `json:"decided_at,omitempty"`
}

type AccessRequestCreateRequest struct {
	ResourceType  string `json:"resource_type"`
	ResourceID    string `json:"resource_id"`
	AccessLevel   string `json:"access_level"`
	Justification string `json:"justification"`
	Duration      int    `json:"duration_hours,omitempty"`
}

type AccessRequestListResponse struct {
	Data       []AccessRequest `json:"data"`
	NextCursor string          `json:"next_cursor,omitempty"`
	HasMore    bool            `json:"has_more"`
}

type ApproveRequest struct {
	Decision string `json:"decision"`
	Comment  string `json:"comment,omitempty"`
}

type Entitlement struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Permissions  []string `json:"permissions"`
	ResourceType string   `json:"resource_type"`
}

type AccessBundle struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description,omitempty"`
	Entitlements []Entitlement `json:"entitlements"`
}

type SoDPolicy struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	ConflictingRoles []string `json:"conflicting_roles"`
	Action           string   `json:"action"`
}

type SoDViolation struct {
	PolicyID    string   `json:"policy_id"`
	PrincipalID string   `json:"principal_id"`
	Roles       []string `json:"roles"`
	Severity    string   `json:"severity"`
}

type IdentityPostureScore struct {
	PrincipalID           string  `json:"principal_id"`
	PrincipalType         string  `json:"principal_type"`
	OverallScore          float64 `json:"overall_score"`
	AuthStrength          float64 `json:"auth_strength"`
	CredentialFreshness   float64 `json:"credential_freshness"`
	PermissionUtilization float64 `json:"permission_utilization"`
	BehavioralDrift       float64 `json:"behavioral_drift"`
	ComplianceViolations  int     `json:"compliance_violations"`
	BelowThreshold        bool    `json:"below_threshold"`
}

type ResourceOwner struct {
	ResourceID   string `json:"resource_id"`
	ResourceType string `json:"resource_type"`
	OwnerID      string `json:"owner_id"`
	OwnerType    string `json:"owner_type"`
}
