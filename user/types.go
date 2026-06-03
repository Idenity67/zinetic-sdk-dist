package user

type MeResponse struct {
	ID        string                 `json:"id,omitempty"`
	Subject   string                 `json:"sub,omitempty"`
	Email     string                 `json:"email,omitempty"`
	Name      string                 `json:"name,omitempty"`
	TenantID  string                 `json:"tenant_id,omitempty"`
	Roles     []string               `json:"roles,omitempty"`
	RawClaims map[string]interface{} `json:"-"`
}
