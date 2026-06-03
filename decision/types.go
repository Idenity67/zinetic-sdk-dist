package decision

import "time"

type AuthRequest struct {
	SubjectID     string           `json:"subject_id"`
	Anchors       []string         `json:"anchors"`
	Context       *DecisionContext `json:"context"`
	PolicyVersion string           `json:"policy_version,omitempty"`
}

type DecisionContext struct {
	IP            string            `json:"ip,omitempty"`
	Geo           *GeoLocation      `json:"geo,omitempty"`
	UserAgent     string            `json:"user_agent,omitempty"`
	DevicePosture *DevicePosture    `json:"device_posture,omitempty"`
	TimeOfDay     string            `json:"time_of_day,omitempty"`
	RiskScore     float64           `json:"risk_score,omitempty"`
	GeoRegion     string            `json:"geo_region,omitempty"`
	SPIFFEID      string            `json:"spiffe_id,omitempty"`
	Extra         map[string]string `json:"extra,omitempty"`
}

type GeoLocation struct {
	Country   string  `json:"country"`
	Region    string  `json:"region,omitempty"`
	City      string  `json:"city,omitempty"`
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
}

type DevicePosture struct {
	DeviceID          string `json:"device_id"`
	Platform          string `json:"platform"`
	OSVersion         string `json:"os_version"`
	Compliant         bool   `json:"compliant"`
	EncryptionEnabled bool   `json:"encryption_enabled"`
	FirewallEnabled   bool   `json:"firewall_enabled"`
	AttestationToken  string `json:"attestation_token,omitempty"`
	TrustScore        int    `json:"trust_score"`
}

type AuthResponse struct {
	Decision      string    `json:"decision"`
	ReasonCode    string    `json:"reason_code"`
	PolicyID      string    `json:"policy_id"`
	PolicyVersion string    `json:"policy_version"`
	ContextHash   string    `json:"context_hash"`
	EvaluatedAt   time.Time `json:"evaluated_at"`
}

type BatchAuthRequest struct {
	Requests []AuthRequest `json:"requests"`
}

type BatchAuthResponse struct {
	Responses []AuthResponse `json:"responses"`
}

type SimulateRequest struct {
	Request       *AuthRequest `json:"request"`
	PolicyVersion string       `json:"policy_version"`
}

type SimulateResponse struct {
	Decision      string    `json:"decision"`
	ReasonCode    string    `json:"reason_code"`
	PolicyID      string    `json:"policy_id"`
	PolicyVersion string    `json:"policy_version"`
	ContextHash   string    `json:"context_hash"`
	EvaluatedAt   time.Time `json:"evaluated_at"`
	Simulated     bool      `json:"simulated"`
}
