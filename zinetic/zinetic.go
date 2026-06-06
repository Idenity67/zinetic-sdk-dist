package zinetic

import (
	"sdk.zinetic.net/apierr"
	"sdk.zinetic.net/dpop"
	"sdk.zinetic.net/nhi"
	"sdk.zinetic.net/pagination"
)

const (
	Version   = "0.2.4"
	UserAgent = "zinetic-sdk-go/" + Version
)

type (
	APIError       = apierr.APIError
	ErrorCode      = apierr.ErrorCode
	FieldError     = apierr.FieldError
	RateLimitError = apierr.RateLimitError

	DPoPProver = dpop.Prover

	PageParams = pagination.PageParams

	NHIProvider        = nhi.Provider
	NHIProviderConfig  = nhi.ProviderConfig
	NHICredential      = nhi.Credential
	NHIConnectorConfig = nhi.ConnectorConfig
	NHIEnvironment     = nhi.Environment
)

type Page[T any] = pagination.Page[T]

var (
	ParseAPIError      = apierr.ParseAPIError
	NewValidationError = apierr.NewValidationError

	NewDPoPProver        = dpop.NewProver
	GenerateDPoPKey      = dpop.GenerateKey
	DPoPPublicKeyJWK     = dpop.PublicKeyJWK
	ComputeJKTThumbprint = dpop.ComputeJKTThumbprint

	DefaultPageParams = pagination.DefaultPageParams

	NewNHIProvider       = nhi.NewProvider
	NewNHIConnector      = nhi.NewConnector
	NewNHIPoolConnector  = nhi.NewPoolConnector
	NewNHIHTTPClient     = nhi.NewHTTPClient
	NewNHIHTTPTransport  = nhi.NewHTTPTransport
	NHIDetectEnvironment = nhi.DetectEnvironment
)

const (
	ErrCodeAuthTokenExpired      = apierr.ErrCodeAuthTokenExpired
	ErrCodeAuthTokenInvalid      = apierr.ErrCodeAuthTokenInvalid
	ErrCodeAuthDPoPRequired      = apierr.ErrCodeAuthDPoPRequired
	ErrCodeAuthDPoPInvalid       = apierr.ErrCodeAuthDPoPInvalid
	ErrCodeAuthInsufficientScope = apierr.ErrCodeAuthInsufficientScope
	ErrCodeAuthStepUpRequired    = apierr.ErrCodeAuthStepUpRequired
	ErrCodeAuthRateLimited       = apierr.ErrCodeAuthRateLimited

	ErrCodeAgentNotFound       = apierr.ErrCodeAgentNotFound
	ErrCodeAgentSuspended      = apierr.ErrCodeAgentSuspended
	ErrCodeAgentRevoked        = apierr.ErrCodeAgentRevoked
	ErrCodeAgentCapabilityDeny = apierr.ErrCodeAgentCapabilityDeny
	ErrCodeAgentDriftDetected  = apierr.ErrCodeAgentDriftDetected
	ErrCodeAgentInvalidState   = apierr.ErrCodeAgentInvalidState

	ErrCodeAnchorNotFound      = apierr.ErrCodeAnchorNotFound
	ErrCodeAnchorAlreadyExists = apierr.ErrCodeAnchorAlreadyExists
	ErrCodeAnchorInvalidProof  = apierr.ErrCodeAnchorInvalidProof
	ErrCodeAnchorRevoked       = apierr.ErrCodeAnchorRevoked

	ErrCodePolicyNotFound  = apierr.ErrCodePolicyNotFound
	ErrCodePolicyEvalError = apierr.ErrCodePolicyEvalError
	ErrCodePolicyTimeout   = apierr.ErrCodePolicyTimeout
	ErrCodePolicyDenied    = apierr.ErrCodePolicyDenied

	ErrCodeLegacyBindFailed  = apierr.ErrCodeLegacyBindFailed
	ErrCodeLegacyUnsupported = apierr.ErrCodeLegacyUnsupported

	ErrCodeMCPToolUnauthorized = apierr.ErrCodeMCPToolUnauthorized
	ErrCodeMCPServerNotFound   = apierr.ErrCodeMCPServerNotFound
	ErrCodeMCPTokenInvalid     = apierr.ErrCodeMCPTokenInvalid

	ErrCodeValidation       = apierr.ErrCodeValidation
	ErrCodeInternal         = apierr.ErrCodeInternal
	ErrCodeNotFound         = apierr.ErrCodeNotFound
	ErrCodeConflict         = apierr.ErrCodeConflict
	ErrCodeTenantNotFound   = apierr.ErrCodeTenantNotFound
	ErrCodeForbidden        = apierr.ErrCodeForbidden
	ErrCodeServiceUnavail   = apierr.ErrCodeServiceUnavail
	ErrCodeIdempotencyConfl = apierr.ErrCodeIdempotencyConfl
)

type SubjectType string

const (
	SubjectTypeHuman   SubjectType = "human"
	SubjectTypeMachine SubjectType = "machine"
	SubjectTypeAgent   SubjectType = "agent"
	SubjectTypeSystem  SubjectType = "system"
)

type Decision string

const (
	DecisionAllow  Decision = "ALLOW"
	DecisionDeny   Decision = "DENY"
	DecisionStepUp Decision = "STEP_UP"
)

type CredentialStatus string

const (
	CredentialStatusActive    CredentialStatus = "ACTIVE"
	CredentialStatusSuspended CredentialStatus = "SUSPENDED"
	CredentialStatusRevoked   CredentialStatus = "REVOKED"
	CredentialStatusExpired   CredentialStatus = "EXPIRED"
)

type AgentType string

const (
	AgentTypeAutonomous     AgentType = "autonomous"
	AgentTypeSemiAutonomous AgentType = "semi-autonomous"
	AgentTypeToolCalling    AgentType = "tool-calling"
	AgentTypeOrchestrator   AgentType = "orchestrator"
)

type AgentLifecycleState string

const (
	AgentStatePending        AgentLifecycleState = "pending"
	AgentStateActive         AgentLifecycleState = "active"
	AgentStateSuspended      AgentLifecycleState = "suspended"
	AgentStateRevoked        AgentLifecycleState = "revoked"
	AgentStateDecommissioned AgentLifecycleState = "decommissioned"
)

type AuditOutcome string

const (
	AuditOutcomeSuccess        AuditOutcome = "success"
	AuditOutcomeFailure        AuditOutcome = "failure"
	AuditOutcomeStepUpRequired AuditOutcome = "step_up_required"
)

type AuditAction string

const (
	AuditActionEnroll       AuditAction = "enroll"
	AuditActionLogin        AuditAction = "login"
	AuditActionStepUp       AuditAction = "step_up"
	AuditActionDeny         AuditAction = "deny"
	AuditActionRevoke       AuditAction = "revoke"
	AuditActionDelegate     AuditAction = "delegate"
	AuditActionMintToken    AuditAction = "mint_token"
	AuditActionAnchor       AuditAction = "anchor"
	AuditActionToolCall     AuditAction = "tool_call"
	AuditActionBreakGlass   AuditAction = "break_glass"
	AuditActionPolicyUpdate AuditAction = "policy_update"
)

type DriftSeverity string

const (
	DriftSeverityLow      DriftSeverity = "LOW"
	DriftSeverityMedium   DriftSeverity = "MEDIUM"
	DriftSeverityHigh     DriftSeverity = "HIGH"
	DriftSeverityCritical DriftSeverity = "CRITICAL"
)

type IncidentSeverity string

const (
	IncidentSeveritySEV1 IncidentSeverity = "SEV1"
	IncidentSeveritySEV2 IncidentSeverity = "SEV2"
	IncidentSeveritySEV3 IncidentSeverity = "SEV3"
	IncidentSeveritySEV4 IncidentSeverity = "SEV4"
)

type CampaignType string

const (
	CampaignTypeUserCentric        CampaignType = "user-centric"
	CampaignTypeApplicationCentric CampaignType = "application-centric"
	CampaignTypeRoleCentric        CampaignType = "role-centric"
	CampaignTypeEntitlementCentric CampaignType = "entitlement-centric"
)

type ResourceType string

const (
	ResourceTypeCredential ResourceType = "credential"
	ResourceTypeAgent      ResourceType = "agent"
	ResourceTypeMCPServer  ResourceType = "mcp_server"
	ResourceTypePolicy     ResourceType = "policy"
	ResourceTypeLegacy     ResourceType = "legacy_identity"
)

const (
	NHIEnvUnknown       = nhi.EnvUnknown
	NHIEnvGitHubActions = nhi.EnvGitHubActions
	NHIEnvKubernetes    = nhi.EnvKubernetes
	NHIEnvAWS           = nhi.EnvAWS
	NHIEnvGCP           = nhi.EnvGCP
	NHIEnvLocal         = nhi.EnvLocal
)
