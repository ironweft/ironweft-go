package ironweft

import "context"

// AgentHandle is a client scoped to a single agent_id. Obtain one via
// Client.Agent — do not construct directly.
//
// Example:
//
//	agent := client.Agent("agt_4ae283ac")
//	cred, err := agent.Credential(ctx, []string{"payments:write"}, 15)
type AgentHandle struct {
	client  *Client
	agentID string
}

// AgentID returns the agent identifier this handle is scoped to.
func (a *AgentHandle) AgentID() string {
	return a.agentID
}

// ── Credential ────────────────────────────────────────────────────────────────

// Credential issues a short-lived JWT scoped to scopes with a lifetime of
// ttlMinutes. The raw credential string is returned; pass it directly to Check.
//
// POST /agents/credentials
func (a *AgentHandle) Credential(ctx context.Context, scopes []string, ttlMinutes int) (string, error) {
	resp, err := a.client.IssueCredential(ctx, IssueCredentialRequest{
		AgentID:    a.agentID,
		Scopes:     scopes,
		TTLMinutes: ttlMinutes,
	})
	if err != nil {
		return "", err
	}
	return resp.Credential, nil
}

// ── Authorization ─────────────────────────────────────────────────────────────

// Check calls POST /authorize with credential and returns the full response on
// an "allow" decision.
//
// On a non-allow decision Check returns a typed error:
//   - *AuthorizationDenied — policy denied the action
//   - *AgentSuspended      — the agent (or a node in its delegation chain) is suspended
//   - *AgentRetired        — the agent is hard-locked and cannot act
func (a *AgentHandle) Check(ctx context.Context, action, credential, resource string) (*AuthorizeResponse, error) {
	resp, err := a.client.Authorize(ctx, AuthorizeRequest{
		Credential: credential,
		Action:     action,
		Resource:   resource,
	})
	if err != nil {
		return nil, err
	}

	switch resp.Decision {
	case "allow":
		return resp, nil
	case "retired":
		return nil, &AgentRetired{AgentID: a.agentID}
	case "suspended":
		return nil, &AgentSuspended{
			Action:       action,
			AgentID:      a.agentID,
			Reason:       resp.Reason,
			AuditEventID: resp.AuditEventID,
		}
	default:
		return nil, &AuthorizationDenied{
			Action:       action,
			AgentID:      a.agentID,
			Reason:       resp.Reason,
			AuditEventID: resp.AuditEventID,
		}
	}
}

// Batch evaluates multiple actions in a single request.
// Cached allow decisions are served locally; uncached actions are bundled into
// one POST /authorize/batch call. Returns the full batch response.
//
// POST /authorize/batch
func (a *AgentHandle) Batch(ctx context.Context, credential string, actions []BatchAuthorizeItem) (*BatchAuthorizeResponse, error) {
	return a.client.AuthorizeBatch(ctx, BatchAuthorizeRequest{
		Credential: credential,
		Actions:    actions,
	})
}

// Gate returns a closure that, when invoked, issues a fresh credential, calls
// /authorize, and — only on an "allow" decision — calls fn. Any error from
// credential issuance, the authorization check, or fn itself is returned.
//
// Gate is the Go equivalent of the Python @agent.gate(...) decorator. Use it
// to wrap a function with per-call credential issuance and authorization:
//
//	send := agent.Gate(ctx, "payment.send", []string{"payments:write"}, "account_7721", 15)
//	if err := send(ctx, func() error {
//	    return processFunds(amount)
//	}); err != nil {
//	    // handle *ironweft.AuthorizationDenied, *ironweft.AgentSuspended, etc.
//	}
func (a *AgentHandle) Gate(
	action string,
	scopes []string,
	resource string,
	ttlMinutes int,
) func(ctx context.Context, fn func() error) error {
	return func(ctx context.Context, fn func() error) error {
		cred, err := a.Credential(ctx, scopes, ttlMinutes)
		if err != nil {
			return err
		}
		if _, err := a.Check(ctx, action, cred, resource); err != nil {
			return err
		}
		return fn()
	}
}

// ── Lifecycle ─────────────────────────────────────────────────────────────────

// Suspend manually suspends the agent.
//
// PATCH /agents/{id} with status "suspended"
func (a *AgentHandle) Suspend(ctx context.Context) (*UpdateAgentStatusResponse, error) {
	return a.client.UpdateAgentStatus(ctx, a.agentID, "suspended")
}

// Reactivate sets the agent status back to "active" after a suspension.
//
// PATCH /agents/{id} with status "active"
func (a *AgentHandle) Reactivate(ctx context.Context) (*UpdateAgentStatusResponse, error) {
	return a.client.UpdateAgentStatus(ctx, a.agentID, "active")
}

// Retire hard-locks the agent. This action is irreversible.
//
// PATCH /agents/{id} with status "retired"
func (a *AgentHandle) Retire(ctx context.Context) (*UpdateAgentStatusResponse, error) {
	return a.client.UpdateAgentStatus(ctx, a.agentID, "retired")
}

// ── Inspection ────────────────────────────────────────────────────────────────

// Permissions returns the current status, roles, and metadata for this agent.
//
// GET /agents/{id}/permissions
func (a *AgentHandle) Permissions(ctx context.Context) (*AgentPermissions, error) {
	return a.client.GetAgentPermissions(ctx, a.agentID)
}

// AuditTrail returns up to limit tamper-evident audit events for this agent.
// Pass offset to paginate.
//
// GET /audit/trail?agent_id=...
func (a *AgentHandle) AuditTrail(ctx context.Context, limit, offset int) (*AuditTrailResponse, error) {
	return a.client.GetAuditTrail(ctx, a.agentID, limit, offset)
}
