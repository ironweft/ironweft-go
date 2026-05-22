package ironweft

// RegisterAgentRequest is the request body for POST /agents.
type RegisterAgentRequest struct {
	AgentName    string                 `json:"agent_name"`
	SponsorID    string                 `json:"sponsor_id"`
	Description  string                 `json:"description,omitempty"`
	InitialRoles []string               `json:"initial_roles,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// RegisterAgentResponse is the response body from POST /agents.
type RegisterAgentResponse struct {
	AgentID    string `json:"agent_id"`
	PublicKey  string `json:"public_key"`
	Status     string `json:"status"`
	RiskTier   string `json:"risk_tier"`
	TierReason string `json:"tier_reason"`
	CreatedAt  string `json:"created_at"`
}

// AgentPermissions is the response body from GET /agents/{id}/permissions.
type AgentPermissions struct {
	AgentID  string                 `json:"agent_id"`
	Status   string                 `json:"status"`
	Roles    []string               `json:"roles"`
	Metadata map[string]interface{} `json:"metadata"`
}

// UpdateAgentStatusRequest is the request body for PATCH /agents/{id}.
type UpdateAgentStatusRequest struct {
	Status string `json:"status"`
}

// UpdateAgentStatusResponse is the response body from PATCH /agents/{id}.
type UpdateAgentStatusResponse struct {
	AgentID string `json:"agent_id"`
	Status  string `json:"status"`
}

// IssueCredentialRequest is the request body for POST /agents/credentials.
type IssueCredentialRequest struct {
	AgentID    string                 `json:"agent_id"`
	Scopes     []string               `json:"scopes"`
	TTLMinutes int                    `json:"ttl_minutes,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

// IssueCredentialResponse is the response body from POST /agents/credentials.
type IssueCredentialResponse struct {
	Credential string   `json:"credential"`
	ExpiresAt  string   `json:"expires_at"`
	Scopes     []string `json:"scopes"`
}

// DelegateAgentRequest is the request body for POST /agents/{id}/delegate.
type DelegateAgentRequest struct {
	AgentName    string                 `json:"agent_name"`
	Scopes       []string               `json:"scopes"`
	InitialRoles []string               `json:"initial_roles,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// DelegateAgentResponse is the response body from POST /agents/{id}/delegate.
type DelegateAgentResponse struct {
	AgentID         string   `json:"agent_id"`
	ParentAgentID   string   `json:"parent_agent_id"`
	DelegationChain []string `json:"delegation_chain"`
	Status          string   `json:"status"`
	CreatedAt       string   `json:"created_at"`
}

// AuthorizeRequest is the request body for POST /authorize.
// Credential is the raw JWT returned by IssueCredential.
// Initiator is an optional string identifying the human or system that triggered
// the action; pass nil to omit it from the request.
type AuthorizeRequest struct {
	Credential string                 `json:"credential"`
	Action     string                 `json:"action"`
	Resource   string                 `json:"resource,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
	Initiator  *string                `json:"initiator,omitempty"`
}

// AuthorizeResponse is the response body from POST /authorize.
type AuthorizeResponse struct {
	Decision      string   `json:"decision"`
	Reason        string   `json:"reason"`
	AllowedScopes []string `json:"allowed_scopes"`
	AuditEventID  string   `json:"audit_event_id"`
	// Cached is set by the SDK cache; never serialized.
	Cached bool `json:"-"`
}

// BatchAuthorizeItem is a single action within a batch authorize request.
type BatchAuthorizeItem struct {
	Action     string                 `json:"action"`
	Resource   string                 `json:"resource,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
	Initiator  *string                `json:"initiator,omitempty"`
	// Ref is an optional caller-supplied ID echoed back in the result.
	Ref *string `json:"ref,omitempty"`
}

// BatchAuthorizeRequest is the request body for POST /authorize/batch.
type BatchAuthorizeRequest struct {
	Credential string               `json:"credential"`
	Actions    []BatchAuthorizeItem `json:"actions"`
}

// BatchResultItem is a single result within a batch authorize response.
type BatchResultItem struct {
	Ref          *string `json:"ref"`
	Action       string  `json:"action"`
	Decision     string  `json:"decision"`
	Reason       string  `json:"reason"`
	AuditEventID *string `json:"audit_event_id"`
	Cached       bool    `json:"-"`
}

// BatchSummary summarizes the decisions in a batch response.
type BatchSummary struct {
	Total     int `json:"total"`
	Allow     int `json:"allow"`
	Deny      int `json:"deny"`
	Challenge int `json:"challenge"`
}

// BatchAuthorizeResponse is the response body from POST /authorize/batch.
type BatchAuthorizeResponse struct {
	Results []BatchResultItem `json:"results"`
	Summary BatchSummary      `json:"summary"`
}

// LogAuditRequest is the request body for POST /audit/log.
type LogAuditRequest struct {
	AgentID   string                 `json:"agent_id"`
	EventType string                 `json:"event_type"`
	Action    string                 `json:"action"`
	Outcome   string                 `json:"outcome"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// LogAuditResponse is the response body from POST /audit/log.
type LogAuditResponse struct {
	EventID   string `json:"event_id"`
	ChainHash string `json:"chain_hash"`
}

// AuditEvent represents a single entry in an agent's audit trail.
type AuditEvent struct {
	EventID   string                 `json:"event_id"`
	AgentID   string                 `json:"agent_id"`
	EventType string                 `json:"event_type"`
	Action    string                 `json:"action"`
	Outcome   string                 `json:"outcome"`
	ChainHash string                 `json:"chain_hash"`
	CreatedAt string                 `json:"created_at"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// AuditTrailResponse is the response body from GET /audit/trail.
type AuditTrailResponse struct {
	Events []AuditEvent `json:"events"`
	Total  int          `json:"total"`
}

// UpdateTenantRequest is the request body for PATCH /tenants/{id}.
type UpdateTenantRequest struct {
	WebhookURL  string   `json:"webhook_url,omitempty"`
	IPAllowlist []string `json:"ip_allowlist,omitempty"`
}

// UpdateTenantResponse is the response body from PATCH /tenants/{id}.
type UpdateTenantResponse struct {
	TenantID string `json:"tenant_id"`
	Updated  bool   `json:"updated"`
}

// RotateKeyResponse is the response body from POST /tenants/{id}/rotate-key.
type RotateKeyResponse struct {
	TenantID string `json:"tenant_id"`
	APIKey   string `json:"api_key"`
	Warning  string `json:"warning"`
}
