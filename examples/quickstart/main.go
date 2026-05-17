// Package main demonstrates the IronWeft Go SDK with the same steps as the
// Python quickstart: register an agent, issue a credential, check authorization,
// use Gate, and pull the audit trail.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	ironweft "github.com/EmohSenoon/ironweft-go"
)

func main() {
	apiKey := os.Getenv("IW_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "IW_API_KEY environment variable is required")
		os.Exit(1)
	}

	ctx := context.Background()
	client := ironweft.New(apiKey)

	// ── 1. Register an agent (one-time setup) ────────────────────────────────
	regResp, err := client.RegisterAgent(ctx, ironweft.RegisterAgentRequest{
		AgentName:    "Grace",
		SponsorID:    "user_margaret_chen",
		InitialRoles: []string{"call_agent"},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "register agent: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Registered: %s (risk_tier=%s)\n", regResp.AgentID, regResp.RiskTier)

	// ── 2. Get a handle scoped to that agent ─────────────────────────────────
	agent := client.Agent(regResp.AgentID)

	// ── 3a. Explicit check — issue credential then authorize ─────────────────
	cred, err := agent.Credential(ctx, []string{"payments:write"}, 15)
	if err != nil {
		fmt.Fprintf(os.Stderr, "issue credential: %v\n", err)
		os.Exit(1)
	}

	result, err := agent.Check(ctx, "payment.send", cred, "account_7721")
	if err != nil {
		var denied *ironweft.AuthorizationDenied
		var suspended *ironweft.AgentSuspended
		var retired *ironweft.AgentRetired
		switch {
		case errors.As(err, &retired):
			fmt.Printf("Agent retired: %v\n", err)
		case errors.As(err, &suspended):
			fmt.Printf("Agent suspended: %v\n", err)
		case errors.As(err, &denied):
			fmt.Printf("Denied: %s\n", denied.Reason)
		default:
			fmt.Fprintf(os.Stderr, "check: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("Allowed — audit event: %s\n", result.AuditEventID)
		// → run the payment logic here
	}

	// ── 3b. Gate — wraps credential issuance + authorization automatically ───
	send := agent.Gate("payment.send", []string{"payments:write"}, "account_7721", 15)

	err = send(ctx, func() error {
		fmt.Println("Sending $2400.00 to account_7721")
		return nil
	})
	if err != nil {
		var denied *ironweft.AuthorizationDenied
		if errors.As(err, &denied) {
			fmt.Printf("Blocked: %v\n", denied)
		} else {
			fmt.Fprintf(os.Stderr, "gate: %v\n", err)
		}
	}

	// ── 4. Pull the audit trail ──────────────────────────────────────────────
	trail, err := agent.AuditTrail(ctx, 10, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "audit trail: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\nAudit trail (%d total):\n", trail.Total)
	for _, ev := range trail.Events {
		hash := ev.ChainHash
		if len(hash) > 12 {
			hash = hash[:12]
		}
		fmt.Printf("  %s  outcome=%-7s  chain=%s\n", ev.Action, ev.Outcome, hash)
	}
}
