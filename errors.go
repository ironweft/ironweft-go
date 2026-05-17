package ironweft

import "fmt"

// IronWeftError is the base error type for all errors returned by the IronWeft
// SDK. HTTP errors, validation errors, and policy errors all wrap this type.
type IronWeftError struct {
	// Message is a human-readable description of the error.
	Message string
	// StatusCode is the HTTP status code returned by the API, or 0 for
	// non-HTTP errors (e.g. serialization failures).
	StatusCode int
}

func (e *IronWeftError) Error() string {
	if e.StatusCode != 0 {
		return fmt.Sprintf("ironweft: HTTP %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("ironweft: %s", e.Message)
}

// AuthorizationDenied is returned by AgentHandle.Check when the /authorize
// endpoint returns a decision of "deny" or "challenge".
type AuthorizationDenied struct {
	Action       string
	AgentID      string
	Reason       string
	AuditEventID string
}

func (e *AuthorizationDenied) Error() string {
	return fmt.Sprintf("ironweft: action %q denied for agent %s: %s", e.Action, e.AgentID, e.Reason)
}

// AgentSuspended is returned by AgentHandle.Check when the /authorize endpoint
// indicates the agent (or a node in its delegation chain) has been suspended.
type AgentSuspended struct {
	Action       string
	AgentID      string
	Reason       string
	AuditEventID string
}

func (e *AgentSuspended) Error() string {
	return fmt.Sprintf("ironweft: agent %s is suspended — action %q blocked: %s", e.AgentID, e.Action, e.Reason)
}

// AgentRetired is returned by AgentHandle.Check when the /authorize endpoint
// indicates the agent has been hard-locked and cannot be reactivated.
type AgentRetired struct {
	AgentID string
}

func (e *AgentRetired) Error() string {
	return fmt.Sprintf("ironweft: agent %s is retired and cannot act", e.AgentID)
}
