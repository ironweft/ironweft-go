// Package ironweft provides a Go client for the IronWeft Agent IAM API.
//
// Quick start:
//
//	client := ironweft.New("iw_live_xxx")
//	agent  := client.Agent("agt_4ae283ac")
//	cred, err := agent.Credential(ctx, []string{"payments:write"}, 15)
//	result, err := agent.Check(ctx, "payment.send", cred, "account_7721")
package ironweft

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const defaultBaseURL = "https://ironweft.io"

// Client is the IronWeft API client. Create one with New.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// Option is a functional option for configuring a Client.
type Option func(*Client)

// WithBaseURL overrides the default API base URL. Useful for testing against a
// local or staging instance of the IronWeft API.
func WithBaseURL(u string) Option {
	return func(c *Client) {
		c.baseURL = u
	}
}

// WithHTTPClient replaces the default HTTP client. The provided client's
// timeout setting is respected; pass a client with Timeout = 0 for no timeout.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// New creates a new Client authenticated with apiKey.
// Additional behaviour can be configured via Option values.
//
// Example:
//
//	client := ironweft.New("iw_live_xxx")
func New(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Agent returns an AgentHandle scoped to agentID.
// The handle exposes credential issuance, authorization checks, lifecycle
// management, and audit trail retrieval for that specific agent.
func (c *Client) Agent(agentID string) *AgentHandle {
	return &AgentHandle{client: c, agentID: agentID}
}

// ── internal HTTP layer ───────────────────────────────────────────────────────

// do executes an HTTP request and decodes the JSON response body into dst.
// If the server returns a 4xx or 5xx status code, do returns an *IronWeftError.
// For POST /authorize the caller must NOT send the Authorization header — this
// is handled by passing skipAuth = true.
func (c *Client) do(ctx context.Context, method, path string, body, dst interface{}, skipAuth bool) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return &IronWeftError{Message: fmt.Sprintf("marshal request: %v", err)}
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return &IronWeftError{Message: fmt.Sprintf("build request: %v", err)}
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if !skipAuth {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &IronWeftError{Message: fmt.Sprintf("http: %v", err)}
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return &IronWeftError{Message: fmt.Sprintf("read body: %v", err)}
	}

	if resp.StatusCode >= 400 {
		// Try to extract a detail field from the JSON error body.
		var apiErr struct {
			Detail string `json:"detail"`
		}
		msg := string(raw)
		if jsonErr := json.Unmarshal(raw, &apiErr); jsonErr == nil && apiErr.Detail != "" {
			msg = apiErr.Detail
		}
		return &IronWeftError{StatusCode: resp.StatusCode, Message: msg}
	}

	if dst != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, dst); err != nil {
			return &IronWeftError{Message: fmt.Sprintf("decode response: %v", err)}
		}
	}
	return nil
}

// ── Agents ────────────────────────────────────────────────────────────────────

// RegisterAgent registers a new agent and returns its identity and risk tier.
//
// POST /agents
func (c *Client) RegisterAgent(ctx context.Context, req RegisterAgentRequest) (*RegisterAgentResponse, error) {
	var resp RegisterAgentResponse
	if err := c.do(ctx, http.MethodPost, "/agents", req, &resp, false); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetAgentPermissions returns the current status, roles, and metadata for
// the agent identified by agentID.
//
// GET /agents/{id}/permissions
func (c *Client) GetAgentPermissions(ctx context.Context, agentID string) (*AgentPermissions, error) {
	var resp AgentPermissions
	if err := c.do(ctx, http.MethodGet, "/agents/"+agentID+"/permissions", nil, &resp, false); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateAgentStatus sets the lifecycle status of the agent identified by
// agentID. Valid values are "active", "suspended", and "retired".
//
// PATCH /agents/{id}
func (c *Client) UpdateAgentStatus(ctx context.Context, agentID, status string) (*UpdateAgentStatusResponse, error) {
	var resp UpdateAgentStatusResponse
	body := UpdateAgentStatusRequest{Status: status}
	if err := c.do(ctx, http.MethodPatch, "/agents/"+agentID, body, &resp, false); err != nil {
		return nil, err
	}
	return &resp, nil
}

// IssueCredential issues a short-lived scoped JWT for the given agent.
//
// POST /agents/credentials
func (c *Client) IssueCredential(ctx context.Context, req IssueCredentialRequest) (*IssueCredentialResponse, error) {
	var resp IssueCredentialResponse
	if err := c.do(ctx, http.MethodPost, "/agents/credentials", req, &resp, false); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DelegateAgent creates a child agent delegated from the parent identified by
// parentAgentID.
//
// POST /agents/{id}/delegate
func (c *Client) DelegateAgent(ctx context.Context, parentAgentID string, req DelegateAgentRequest) (*DelegateAgentResponse, error) {
	var resp DelegateAgentResponse
	if err := c.do(ctx, http.MethodPost, "/agents/"+parentAgentID+"/delegate", req, &resp, false); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ── Authorize ─────────────────────────────────────────────────────────────────

// Authorize evaluates an agent credential against a policy and returns the
// authorization decision.
//
// POST /authorize — does NOT send the Authorization header; authentication is
// carried by the JWT credential in the request body.
func (c *Client) Authorize(ctx context.Context, req AuthorizeRequest) (*AuthorizeResponse, error) {
	var resp AuthorizeResponse
	if err := c.do(ctx, http.MethodPost, "/authorize", req, &resp, true); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ── Audit ─────────────────────────────────────────────────────────────────────

// LogAuditEvent writes a tamper-evident audit entry and returns the event ID
// and its chain hash.
//
// POST /audit/log
func (c *Client) LogAuditEvent(ctx context.Context, req LogAuditRequest) (*LogAuditResponse, error) {
	var resp LogAuditResponse
	if err := c.do(ctx, http.MethodPost, "/audit/log", req, &resp, false); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetAuditTrail retrieves the tamper-evident audit trail, optionally filtered
// to a specific agent. Pass agentID = "" to retrieve trail across all agents.
//
// GET /audit/trail
func (c *Client) GetAuditTrail(ctx context.Context, agentID string, limit, offset int) (*AuditTrailResponse, error) {
	params := url.Values{}
	if agentID != "" {
		params.Set("agent_id", agentID)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	path := "/audit/trail"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	var resp AuditTrailResponse
	if err := c.do(ctx, http.MethodGet, path, nil, &resp, false); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ── Tenants ───────────────────────────────────────────────────────────────────

// UpdateTenant updates configuration for the tenant identified by tenantID.
//
// PATCH /tenants/{id}
func (c *Client) UpdateTenant(ctx context.Context, tenantID string, req UpdateTenantRequest) (*UpdateTenantResponse, error) {
	var resp UpdateTenantResponse
	if err := c.do(ctx, http.MethodPatch, "/tenants/"+tenantID, req, &resp, false); err != nil {
		return nil, err
	}
	return &resp, nil
}

// RotateTenantKey rotates the API key for the tenant identified by tenantID.
// The response contains the new key; update your environment immediately.
//
// POST /tenants/{id}/rotate-key
func (c *Client) RotateTenantKey(ctx context.Context, tenantID string) (*RotateKeyResponse, error) {
	var resp RotateKeyResponse
	if err := c.do(ctx, http.MethodPost, "/tenants/"+tenantID+"/rotate-key", nil, &resp, false); err != nil {
		return nil, err
	}
	return &resp, nil
}
