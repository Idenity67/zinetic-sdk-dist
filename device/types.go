package device

import "time"

type Device struct {
	ID                string     `json:"id"`
	UserID            string     `json:"user_id"`
	DeviceID          string     `json:"device_id"`
	Name              string     `json:"name"`
	DeviceType        string     `json:"device_type"`
	Platform          string     `json:"platform"`
	OSVersion         string     `json:"os_version"`
	AppVersion        string     `json:"app_version"`
	Model             string     `json:"model"`
	Manufacturer      string     `json:"manufacturer"`
	Trusted           bool       `json:"trusted"`
	RiskScore         int        `json:"risk_score"`
	ComplianceStatus  string     `json:"compliance_status"`
	EnrollmentStatus  string     `json:"enrollment_status"`
	LastSeenAt        time.Time  `json:"last_seen_at"`
	LastPostureCheck  time.Time  `json:"last_posture_check"`
	CertificateExpiry *time.Time `json:"certificate_expiry,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	TenantID          string     `json:"tenant_id"`
	Version           int        `json:"version"`
}

type ListResponse struct {
	Devices    []Device `json:"devices"`
	Count      int      `json:"count"`
	NextCursor string   `json:"next_cursor,omitempty"`
}

type PostureReport struct {
	DeviceID         string                 `json:"device_id"`
	RiskScore        int                    `json:"risk_score"`
	ComplianceStatus string                 `json:"compliance_status"`
	Checks           []PostureCheck         `json:"checks"`
	EvaluatedAt      time.Time              `json:"evaluated_at"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

type PostureCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Details string `json:"details,omitempty"`
}

type HistoryEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Event       string    `json:"event"`
	RiskScore   int       `json:"risk_score"`
	Description string    `json:"description,omitempty"`
}

type HistoryResponse struct {
	History []HistoryEntry `json:"history"`
	Count   int            `json:"count"`
}

type TrendResponse struct {
	DeviceID   string       `json:"device_id"`
	DataPoints []TrendPoint `json:"data_points"`
}

type TrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	RiskScore int       `json:"risk_score"`
}

type TrustResponse struct {
	DeviceID   string        `json:"device_id"`
	TrustScore float64       `json:"trust_score"`
	Factors    []TrustFactor `json:"factors"`
}

type TrustFactor struct {
	Name   string  `json:"name"`
	Score  float64 `json:"score"`
	Weight float64 `json:"weight"`
}

type VerifyResponse struct {
	DeviceID string `json:"device_id"`
	Verified bool   `json:"verified"`
	Reason   string `json:"reason,omitempty"`
}

type ComplianceSummary struct {
	TotalDevices   int     `json:"total_devices"`
	Compliant      int     `json:"compliant"`
	NonCompliant   int     `json:"non_compliant"`
	Unknown        int     `json:"unknown"`
	ComplianceRate float64 `json:"compliance_rate"`
}

type NonCompliantResponse struct {
	Devices []Device `json:"devices"`
	Count   int      `json:"count"`
}
