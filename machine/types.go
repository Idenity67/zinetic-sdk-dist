package machine

import "time"

type MachineIdentity struct {
	ID          string            `json:"id"`
	TenantID    string            `json:"tenant_id"`
	Type        string            `json:"type"`
	Name        string            `json:"name"`
	Status      string            `json:"status"`
	ApproverID  string            `json:"approver_id"`
	Description string            `json:"description,omitempty"`
	Baseline    map[string]string `json:"baseline,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type GitHubIdentity struct {
	MachineIdentity
	RepoID           string   `json:"repo_id"`
	RepoOwner        string   `json:"repo_owner"`
	RepoName         string   `json:"repo_name"`
	WorkflowHash     string   `json:"workflow_hash,omitempty"`
	AllowedBranches  []string `json:"allowed_branches,omitempty"`
	AllowedRunners   []string `json:"allowed_runners,omitempty"`
	AllowedWorkflows []string `json:"allowed_workflows,omitempty"`
}

type KubernetesIdentity struct {
	MachineIdentity
	ClusterID       string   `json:"cluster_id"`
	ClusterName     string   `json:"cluster_name"`
	APIServerURL    string   `json:"api_server_url"`
	IssuerURL       string   `json:"issuer_url,omitempty"`
	JWKSURL         string   `json:"jwks_url,omitempty"`
	Audience        string   `json:"audience,omitempty"`
	Region          string   `json:"region,omitempty"`
	Namespaces      []string `json:"namespaces,omitempty"`
	ServiceAccounts []string `json:"service_accounts,omitempty"`
}

type GitHubEnrollRequest struct {
	TenantID     string   `json:"tenant_id"`
	Name         string   `json:"name"`
	ApproverID   string   `json:"approver_id"`
	Description  string   `json:"description,omitempty"`
	RepoID       string   `json:"repo_id"`
	RepoOwner    string   `json:"repo_owner"`
	RepoName     string   `json:"repo_name"`
	Workflow     string   `json:"workflow_file,omitempty"`
	WorkflowHash string   `json:"workflow_hash,omitempty"`
	Branches     []string `json:"branches,omitempty"`
	Runners      []string `json:"runners,omitempty"`
	Environment  string   `json:"environment,omitempty"`
}

type KubernetesEnrollRequest struct {
	TenantID        string   `json:"tenant_id"`
	Name            string   `json:"name"`
	ApproverID      string   `json:"approver_id"`
	Description     string   `json:"description,omitempty"`
	ClusterID       string   `json:"cluster_id"`
	ClusterName     string   `json:"cluster_name"`
	APIServerURL    string   `json:"api_server_url"`
	IssuerURL       string   `json:"issuer_url,omitempty"`
	JWKSURL         string   `json:"jwks_url,omitempty"`
	Audience        string   `json:"audience,omitempty"`
	Region          string   `json:"region,omitempty"`
	Namespaces      []string `json:"namespaces,omitempty"`
	ServiceAccounts []string `json:"service_accounts,omitempty"`
}

type GitHubTokenRequest struct {
	RepoID       string `json:"repo_id"`
	RepoOwner    string `json:"repo_owner"`
	RepoName     string `json:"repo_name"`
	WorkflowFile string `json:"workflow_file"`
	WorkflowRef  string `json:"workflow_ref"`
	RunnerName   string `json:"runner_name"`
	RunnerType   string `json:"runner_type"`
	Branch       string `json:"branch"`
	Commit       string `json:"commit"`
	Actor        string `json:"actor"`
	EventName    string `json:"event_name"`
	OIDCToken    string `json:"oidc_token"`
	Environment  string `json:"environment,omitempty"`
}

type KubernetesTokenRequest struct {
	ClusterID         string `json:"cluster_id"`
	Namespace         string `json:"namespace"`
	ServiceAccount    string `json:"service_account"`
	PodName           string `json:"pod_name"`
	PodUID            string `json:"pod_uid"`
	ServiceAccountJWT string `json:"service_account_jwt"`
	NodeName          string `json:"node_name,omitempty"`
}

type TokenResponse struct {
	AccessToken   string    `json:"access_token"`
	TokenType     string    `json:"token_type"`
	ExpiresIn     int       `json:"expires_in"`
	ExpiresAt     time.Time `json:"expires_at"`
	IdentityID    string    `json:"identity_id"`
	PolicyVersion string    `json:"policy_version"`
	Scope         string    `json:"scope,omitempty"`
}
