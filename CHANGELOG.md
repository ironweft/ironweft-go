# Changelog

All notable changes to the IronWeft Go SDK. Follows [Semantic Versioning](https://semver.org/).

---

## [v0.2.0] — 2026-05-17

### Added
- `AgentHandle` via `client.Agent(agentID)` — scoped client for per-agent operations
- `AgentHandle.Gate()` — returns a closure wrapping any `func() error` with per-call credential issuance + authorization
- `AgentHandle.Batch()` — evaluate up to 50 actions in one `/authorize/batch` call
- `AgentHandle.Suspend()`, `Reactivate()`, `Retire()` — lifecycle management
- `AgentHandle.Permissions()`, `AuditTrail()` — inspection helpers
- `*AgentSuspended` and `*AgentRetired` error types; use `errors.As` to inspect
- In-process authorization cache (TTL-bound to credential expiry); disable with `WithCache(false)`
- `client.InvalidateCache(credential)` — evict cached decisions; pass `""` to clear all
- `client.DelegateAgent()` — spawn a child agent inheriting a constrained scope subset
- `client.UpdateTenant()`, `client.RotateTenantKey()` — tenant management
- `WithBaseURL()`, `WithHTTPClient()`, `WithCache()` functional options on `New()`

### Changed
- `New(apiKey string, opts ...Option)` replaces the previous `NewClient(apiKey, baseURL string)` signature
- `client.Authorize()` now caches `allow` decisions in-process by default

### Fixed
- HTTP timeout now applies correctly when using a custom `*http.Client` via `WithHTTPClient()`

---

## [v0.1.0] — 2026-04-30

### Added
- Initial release
- `Client` with `RegisterAgent()`, `IssueCredential()`, `Authorize()`, `GetAuditTrail()`
- `*IronWeftError`, `*AuthorizationDenied` error types
