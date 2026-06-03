package audit

import "time"

type Event struct {
	EventID         string            `json:"event_id"`
	Timestamp       time.Time         `json:"timestamp"`
	TenantID        string            `json:"tenant_id"`
	Actor           *Actor            `json:"actor"`
	Action          string            `json:"action"`
	Resource        *Resource         `json:"resource"`
	Outcome         string            `json:"outcome"`
	CorrelationID   string            `json:"correlation_id"`
	PolicyVersion   string            `json:"policy_version,omitempty"`
	ClientContext   *ClientContext    `json:"client_context,omitempty"`
	DelegationChain []string          `json:"delegation_chain,omitempty"`
	Extra           map[string]string `json:"extra,omitempty"`
}

type Actor struct {
	Type         string `json:"type"`
	ActorID      string `json:"actor_id"`
	AuthStrength string `json:"auth_strength,omitempty"`
}

type Resource struct {
	Type       string `json:"type"`
	ResourceID string `json:"resource_id"`
}

type ClientContext struct {
	IP        string `json:"ip,omitempty"`
	Geo       string `json:"geo,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
	SPIFFEID  string `json:"spiffe_id,omitempty"`
}

type SearchRequest struct {
	TimeRangeStart *time.Time `json:"time_range_start,omitempty"`
	TimeRangeEnd   *time.Time `json:"time_range_end,omitempty"`
	ActorID        string     `json:"actor_id,omitempty"`
	ActorType      string     `json:"actor_type,omitempty"`
	Action         string     `json:"action,omitempty"`
	ResourceType   string     `json:"resource_type,omitempty"`
	ResourceID     string     `json:"resource_id,omitempty"`
	Outcome        string     `json:"outcome,omitempty"`
	CorrelationID  string     `json:"correlation_id,omitempty"`
	Cursor         string     `json:"cursor,omitempty"`
	Limit          int        `json:"limit,omitempty"`
}

type SearchResponse struct {
	Data       []Event `json:"data"`
	NextCursor string  `json:"next_cursor,omitempty"`
	HasMore    bool    `json:"has_more"`
}

type ExportFormat string

const (
	ExportFormatCEF  ExportFormat = "CEF"
	ExportFormatJSON ExportFormat = "JSON"
)

type SIEMConfigRequest struct {
	TargetType    string       `json:"target_type"`
	Endpoint      string       `json:"endpoint"`
	Format        ExportFormat `json:"format"`
	EventTypes    []string     `json:"event_types,omitempty"`
	AuthToken     string       `json:"auth_token,omitempty"`
	TransportType string       `json:"transport_type"`
}

type SIEMConfigResponse struct {
	ID         string       `json:"id"`
	TargetType string       `json:"target_type"`
	Endpoint   string       `json:"endpoint"`
	Format     ExportFormat `json:"format"`
	EventTypes []string     `json:"event_types,omitempty"`
	Status     string       `json:"status"`
	CreatedAt  time.Time    `json:"created_at"`
}
