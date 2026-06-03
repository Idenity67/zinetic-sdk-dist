package webhook

import "time"

type Subscription struct {
	ID           string    `json:"id"`
	TargetURL    string    `json:"target_url"`
	Events       []string  `json:"events"`
	SharedSecret string    `json:"shared_secret,omitempty"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type SubscribeRequest struct {
	TargetURL    string   `json:"target_url"`
	Events       []string `json:"events"`
	SharedSecret string   `json:"shared_secret"`
}

type SubscribeResponse struct {
	Subscription *Subscription `json:"subscription"`
}

type ListResponse struct {
	Data       []Subscription `json:"data"`
	NextCursor string         `json:"next_cursor,omitempty"`
	HasMore    bool           `json:"has_more"`
}

type DeliveryAttempt struct {
	ID            string    `json:"id"`
	WebhookID     string    `json:"webhook_id"`
	EventID       string    `json:"event_id"`
	StatusCode    int       `json:"status_code"`
	ResponseBody  string    `json:"response_body,omitempty"`
	AttemptNumber int       `json:"attempt_number"`
	AttemptedAt   time.Time `json:"attempted_at"`
}

type DeadLetterEntry struct {
	ID        string    `json:"id"`
	WebhookID string    `json:"webhook_id"`
	EventID   string    `json:"event_id"`
	Payload   string    `json:"payload"`
	LastError string    `json:"last_error"`
	CreatedAt time.Time `json:"created_at"`
}
