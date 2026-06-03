package tenant

import "time"

type Tenant struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Domain        string            `json:"domain,omitempty"`
	Status        string            `json:"status"`
	Region        string            `json:"region"`
	Tier          string            `json:"tier"`
	Configuration *TenantConfig     `json:"configuration,omitempty"`
	Branding      *Branding         `json:"branding,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

type TenantConfig struct {
	SessionIdleTimeout     int               `json:"session_idle_timeout"`
	SessionAbsoluteTimeout int               `json:"session_absolute_timeout"`
	PasskeyPolicy          string            `json:"passkey_policy,omitempty"`
	RateLimits             *RateLimits       `json:"rate_limits,omitempty"`
	LDAPGroupMappings      map[string]string `json:"ldap_group_mappings,omitempty"`
	AllowedRegions         []string          `json:"allowed_regions,omitempty"`
	CMKEnabled             bool              `json:"cmk_enabled"`
	CMKBackend             string            `json:"cmk_backend,omitempty"`
	DecisionCacheEnabled   bool              `json:"decision_cache_enabled"`
}

type RateLimits struct {
	DecisionAPI int `json:"decision_api"`
	TokenMint   int `json:"token_mint"`
	Management  int `json:"management"`
	AuditRead   int `json:"audit_read"`
}

type Branding struct {
	LogoURL        string `json:"logo_url,omitempty"`
	PrimaryColor   string `json:"primary_color,omitempty"`
	SecondaryColor string `json:"secondary_color,omitempty"`
	CustomDomain   string `json:"custom_domain,omitempty"`
	CompanyName    string `json:"company_name,omitempty"`
}

type CreateRequest struct {
	Name          string            `json:"name"`
	Domain        string            `json:"domain,omitempty"`
	Region        string            `json:"region"`
	Tier          string            `json:"tier"`
	Configuration *TenantConfig     `json:"configuration,omitempty"`
	Branding      *Branding         `json:"branding,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type UpdateRequest struct {
	Name          string            `json:"name,omitempty"`
	Domain        string            `json:"domain,omitempty"`
	Configuration *TenantConfig     `json:"configuration,omitempty"`
	Branding      *Branding         `json:"branding,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type ListResponse struct {
	Data       []Tenant `json:"data"`
	NextCursor string   `json:"next_cursor,omitempty"`
	HasMore    bool     `json:"has_more"`
}

type DataExportRequest struct {
	Format string `json:"format"`
}

type DataExportResponse struct {
	ExportID    string    `json:"export_id"`
	Status      string    `json:"status"`
	DownloadURL string    `json:"download_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
}
