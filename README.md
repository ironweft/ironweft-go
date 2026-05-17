# ironweft-go

Go SDK for the [IronWeft](https://ironweft.io) Agent IAM API.

- Zero external dependencies — standard library only
- Context-aware — every method accepts `context.Context`
- Typed errors — `AuthorizationDenied`, `AgentSuspended`, `AgentRetired`
- `Gate()` — wraps any function with per-call credential issuance + authorization

## Install

```bash
go get github.com/EmohSenoon/ironweft-go
```

Requires Go 1.21 or later.

## Quick start

```go
package main

import (
    "context"
    "errors"
    "fmt"

    ironweft "github.com/EmohSenoon/ironweft-go"
)

func main() {
    ctx    := context.Background()
    client := ironweft.New("iw_live_xxx")

    // Register an agent
    reg, err := client.RegisterAgent(ctx, ironweft.RegisterAgentRequest{
        AgentName:    "Grace",
        SponsorID:    "user_margaret_chen",
        InitialRoles: []string{"call_agent"},
    })
    if err != nil {
        panic(err)
    }

    // Scope a handle to that agent
    agent := client.Agent(reg.AgentID)

    // Issue a credential and authorize an action
    cred, err := agent.Credential(ctx, []string{"payments:write"}, 15)
    if err != nil {
        panic(err)
    }

    result, err := agent.Check(ctx, "payment.send", cred, "account_7721")
    if err != nil {
        var denied *ironweft.AuthorizationDenied
        if errors.As(err, &denied) {
            fmt.Println("Denied:", denied.Reason)
            return
        }
        panic(err)
    }
    fmt.Println("Allowed — audit event:", result.AuditEventID)

    // Gate: wraps a function with credential issuance + authorization
    send := agent.Gate("payment.send", []string{"payments:write"}, "account_7721", 15)
    _ = send(ctx, func() error {
        fmt.Println("payment sent")
        return nil
    })
}
```

Run the full quickstart example:

```bash
IW_API_KEY=iw_live_xxx go run ./examples/quickstart
```

## Configuration

```go
// Custom timeout / transport
client := ironweft.New("iw_live_xxx",
    ironweft.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
)

// Point at a local or staging instance
client := ironweft.New("iw_live_xxx",
    ironweft.WithBaseURL("http://localhost:8080"),
)
```

## API reference

### `ironweft.New(apiKey string, opts ...Option) *Client`

Create a new authenticated client. Options:

| Option | Description |
|--------|-------------|
| `WithBaseURL(u string)` | Override the API base URL (default: `https://ironweft.io`) |
| `WithHTTPClient(hc *http.Client)` | Replace the default HTTP client |

---

### `Client` methods

All methods accept `context.Context` as the first argument.

#### Agents

```go
// POST /agents
resp, err := client.RegisterAgent(ctx, ironweft.RegisterAgentRequest{
    AgentName:    "Grace",
    SponsorID:    "user_margaret_chen",
    Description:  "Outbound call agent",
    InitialRoles: []string{"call_agent"},
    Metadata:     map[string]interface{}{"env": "prod"},
})
// resp.AgentID, resp.PublicKey, resp.Status, resp.RiskTier, resp.TierReason, resp.CreatedAt

// GET /agents/{id}/permissions
perms, err := client.GetAgentPermissions(ctx, agentID)
// perms.AgentID, perms.Status, perms.Roles, perms.Metadata

// PATCH /agents/{id}  status: "active" | "suspended" | "retired"
upd, err := client.UpdateAgentStatus(ctx, agentID, "suspended")

// POST /agents/credentials
cred, err := client.IssueCredential(ctx, ironweft.IssueCredentialRequest{
    AgentID:    agentID,
    Scopes:     []string{"payments:write"},
    TTLMinutes: 15,
})
// cred.Credential, cred.ExpiresAt, cred.Scopes

// POST /agents/{id}/delegate
del, err := client.DelegateAgent(ctx, parentAgentID, ironweft.DelegateAgentRequest{
    AgentName: "Grace-sub",
    Scopes:    []string{"payments:read"},
})
// del.AgentID, del.ParentAgentID, del.DelegationChain
```

#### Authorize

```go
// POST /authorize  (no Authorization header — auth is in the JWT credential)
initiator := "user_margaret_chen"
result, err := client.Authorize(ctx, ironweft.AuthorizeRequest{
    Credential: rawJWT,
    Action:     "payment.send",
    Resource:   "account_7721",
    Initiator:  &initiator,
})
// result.Decision ("allow"|"deny"|"challenge"), result.Reason, result.AllowedScopes, result.AuditEventID
```

#### Audit

```go
// POST /audit/log
log, err := client.LogAuditEvent(ctx, ironweft.LogAuditRequest{
    AgentID:   agentID,
    EventType: "action",
    Action:    "payment.send",
    Outcome:   "success",
})
// log.EventID, log.ChainHash

// GET /audit/trail?agent_id=&limit=&offset=
trail, err := client.GetAuditTrail(ctx, agentID, 50, 0)
// trail.Events []AuditEvent, trail.Total
```

#### Tenants

```go
// PATCH /tenants/{id}
upd, err := client.UpdateTenant(ctx, tenantID, ironweft.UpdateTenantRequest{
    WebhookURL:  "https://example.com/hook",
    IPAllowlist: []string{"203.0.113.0/24"},
})

// POST /tenants/{id}/rotate-key
rot, err := client.RotateTenantKey(ctx, tenantID)
// rot.APIKey — update IW_API_KEY immediately
```

---

### `AgentHandle` methods

Obtain via `client.Agent(agentID)`.

```go
agent := client.Agent("agt_4ae283ac")

// Issue a credential
cred, err := agent.Credential(ctx, []string{"payments:write"}, 15)

// Authorize an action — returns typed errors on non-allow decisions
result, err := agent.Check(ctx, "payment.send", cred, "account_7721")

// Gate — wrap a function with per-call credential + authorization
send := agent.Gate("payment.send", []string{"payments:write"}, "account_7721", 15)
err = send(ctx, func() error {
    return processFunds(amount)
})

// Lifecycle
_, err = agent.Suspend(ctx)
_, err = agent.Reactivate(ctx)
_, err = agent.Retire(ctx)   // irreversible

// Inspection
perms, err := agent.Permissions(ctx)
trail, err  := agent.AuditTrail(ctx, 50, 0)
```

---

### Error types

```go
var iwErr *ironweft.IronWeftError
var denied *ironweft.AuthorizationDenied  // decision != "allow", not suspended/retired
var suspended *ironweft.AgentSuspended    // agent or chain node is suspended
var retired *ironweft.AgentRetired        // agent is hard-locked

// Use errors.As to inspect:
if errors.As(err, &denied) {
    fmt.Println(denied.Action, denied.Reason, denied.AuditEventID)
}
```

| Type | When |
|------|------|
| `*IronWeftError` | HTTP error or serialization failure |
| `*AuthorizationDenied` | Policy denied the action |
| `*AgentSuspended` | Agent (or delegation chain node) is suspended |
| `*AgentRetired` | Agent is hard-locked; cannot be reactivated |

All error types satisfy the standard `error` interface. Use `errors.As` for
type-safe inspection.
