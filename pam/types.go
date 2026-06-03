package pam

import "time"

type ElevationRequest struct {
	PrincipalID    string `json:"principal_id"`
	TargetResource string `json:"target_resource"`
	AccessLevel    string `json:"access_level"`
	Justification  string `json:"justification"`
	Duration       int    `json:"duration_minutes"`
	ApprovalType   string `json:"approval_type"`
}

type ElevationResponse struct {
	SessionID    string    `json:"session_id"`
	Status       string    `json:"status"`
	GrantedAt    time.Time `json:"granted_at,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	AuditEventID string    `json:"audit_event_id"`
}

type ElevationSession struct {
	SessionID      string    `json:"session_id"`
	PrincipalID    string    `json:"principal_id"`
	TargetResource string    `json:"target_resource"`
	AccessLevel    string    `json:"access_level"`
	Status         string    `json:"status"`
	GrantedAt      time.Time `json:"granted_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	ApprovedBy     []string  `json:"approved_by,omitempty"`
	Justification  string    `json:"justification"`
}

type SessionListResponse struct {
	Data       []ElevationSession `json:"data"`
	NextCursor string             `json:"next_cursor,omitempty"`
	HasMore    bool               `json:"has_more"`
}

type EphemeralCredential struct {
	ID           string    `json:"id"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	Credential   string    `json:"credential"`
	ExpiresAt    time.Time `json:"expires_at"`
	SessionID    string    `json:"session_id"`
}

type SessionRecording struct {
	SessionID   string    `json:"session_id"`
	Commands    []Command `json:"commands"`
	DataTouched []string  `json:"data_touched"`
	StartedAt   time.Time `json:"started_at"`
	EndedAt     time.Time `json:"ended_at,omitempty"`
}

type Command struct {
	Timestamp time.Time `json:"timestamp"`
	Input     string    `json:"input"`
	Output    string    `json:"output,omitempty"`
	ExitCode  int       `json:"exit_code"`
}

type AnalyticsResponse struct {
	PrincipalID      string         `json:"principal_id"`
	TotalElevations  int            `json:"total_elevations"`
	AveragesDuration float64        `json:"average_duration_minutes"`
	ResourceUsage    map[string]int `json:"resource_usage"`
	FrequencyPerWeek float64        `json:"frequency_per_week"`
	DeviationScore   float64        `json:"deviation_score"`
}
