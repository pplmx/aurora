# Phase 8: Operations & Health - Context

**Gathered:** 2026-04-30
**Status:** Ready for planning
**Mode:** Auto-generated (infrastructure phase)

<domain>
## Phase Boundary

Add Kubernetes-ready health endpoints and implement graceful shutdown:
1. `/healthz` - Liveness probe (always 200 if server responds)
2. `/readyz` - Readiness probe (503 if database unreachable)
3. Health endpoints bypass authentication (registered before auth middleware)
4. Graceful shutdown with `server.Shutdown(ctx)`

</domain>

<decisions>
## Implementation Decisions

### Agent's Discretion
All implementation choices are at the agent's discretion — pure infrastructure phase.

</decisions>

<codebase>
## Existing Code Insights

### Current Health Implementation
- `internal/api/router.go`: Has `/health` endpoint but no `/healthz` or `/readyz`
- `cmd/api/main.go`: Uses `server.Close()` which kills in-flight requests

### Chi Router Pattern
- Routes are set up in `newRouter()` function
- Auth middleware is applied via `r.Group()` after routes

### Required Changes
- Create `internal/api/health.go` with handlers
- Update `router.go` to add `/healthz`, `/readyz` BEFORE auth
- Update `cmd/api/main.go` to use `server.Shutdown(ctx)`

</codebase>

<specifics>
## Specific Ideas

Per research/HEALTH.md:
- Use 2s timeout for database ping
- Return 503 Service Unavailable when DB is down
- Include database check in readiness response

</specifics>

<deferred>
## Deferred Ideas

None — infrastructure phase, no scope for new features.

</deferred>
