# Phase 7: Security Hardening - Context

**Gathered:** 2026-04-30
**Status:** Ready for planning
**Mode:** Auto-generated (infrastructure phase)

<domain>
## Phase Boundary

Eliminate critical API key security vulnerabilities:
1. Use `crypto/subtle.ConstantTimeCompare` for timing-safe key comparison
2. Remove hardcoded default API key
3. Fail in production if API key not configured
4. Use generic auth error messages

</domain>

<decisions>
## Implementation Decisions

### Agent's Discretion
All implementation choices are at the agent's discretion — pure infrastructure phase.

</decisions>

<codebase>
## Existing Code Insights

### Current Vulnerable Code
- `internal/config/config.go:40`: Hardcoded `viper.SetDefault("api.key", "aurora-api-key-default")`
- `internal/api/middleware/auth.go:12`: `if key != apiKey` — vulnerable to timing attacks
- `internal/api/middleware/auth.go:13`: `writeUnauthorized(w, "invalid api key")` — leaks information

### Existing Patterns
- Uses `viper` for configuration management
- Uses chi router middleware pattern
- JSON error responses via `json.NewEncoder`

</codebase>

<specifics>
## Specific Ideas

Per research/SECURITY.md:
- Generate secure random keys using `crypto/rand`
- Check for known insecure keys in production
- Log key length (not value) server-side for anomaly detection

</specifics>

<deferred>
## Deferred Ideas

None — infrastructure phase, no scope for new features.

</deferred>