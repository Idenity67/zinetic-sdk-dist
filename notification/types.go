package notification

import "time"

type Notification struct {
	ID               string            `json:"id"`
	UserID           string            `json:"user_id"`
	TenantID         string            `json:"tenant_id"`
	NotificationType string            `json:"notification_type"`
	Channel          string            `json:"channel"`
	Subject          string            `json:"subject"`
	Body             string            `json:"body"`
	Priority         string            `json:"priority,omitempty"`
	Status           string            `json:"status"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	SentAt           *time.Time        `json:"sent_at,omitempty"`
	ReadAt           *time.Time        `json:"read_at,omitempty"`
}

type SendRequest struct {
	UserID           string            `json:"user_id"`
	TenantID         string            `json:"tenant_id"`
	NotificationType string            `json:"notification_type"`
	Channel          string            `json:"channel"`
	Subject          string            `json:"subject"`
	Body             string            `json:"body"`
	Priority         string            `json:"priority,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

type SendResponse struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type ListResponse struct {
	Notifications []Notification `json:"notifications"`
	Count         int            `json:"count"`
	NextCursor    string         `json:"next_cursor,omitempty"`
}

type PushToken struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	TenantID   string     `json:"tenant_id"`
	DeviceID   string     `json:"device_id"`
	Platform   string     `json:"platform"`
	Token      string     `json:"token"`
	AppBundle  string     `json:"app_bundle,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
}

type RegisterTokenRequest struct {
	UserID    string `json:"user_id"`
	TenantID  string `json:"tenant_id"`
	DeviceID  string `json:"device_id"`
	Platform  string `json:"platform"`
	Token     string `json:"token"`
	AppBundle string `json:"app_bundle,omitempty"`
}

type RegisterTokenResponse struct {
	Status   string `json:"status"`
	Platform string `json:"platform"`
}

type ListTokensResponse struct {
	Tokens []PushToken `json:"tokens"`
}
