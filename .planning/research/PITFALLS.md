# Pitfalls Research

**Date:** 2026-04-26
**Milestone:** v1.1 Production Hardening

## Common Mistakes When Adding These Features

### REST API Server

**Pitfalls:**
1. **Replicating logic** — Don't rewrite CLI handlers; reuse `internal/app/*`
2. **Auth complexity** — Start simple (API key), add OAuth later if needed
3. **Error responses** — Inconsistent JSON error formats
4. **Missing validation** — API should validate same as CLI

**Prevention:**
- Use existing use cases directly
- Create API-specific middleware, not new auth systems
- Use standard error response: `{ "error": "message", "code": "ERR_CODE" }`
- Share validation with CLI via domain layer

### Oracle Real Data

**Pitfalls:**
1. **Unvalidated sources** — Accepting data without validation
2. **No timeout handling** — Hanging on slow sources
3. **Rate limit blind** — Not respecting source APIs

**Prevention:**
- Validate response schema before storing
- Already have timeout (10s configurable)
- Respect source rate limits (already added)

### Web UI

**Pitfalls:**
1. **SPA overkill** — Using React/Vue for simple forms
2. **State sync** — UI and CLI state diverge
3. **Build complexity** — Frontend build pipeline

**Prevention:**
- HTMX + Alpine.js is sufficient
- UI reads same state as CLI (via API)
- Keep build simple: `go build` for backend, simple asset serving

### Database Migrations

**Pitfalls:**
1. **Data loss** — `ALTER TABLE` without data migration
2. **No rollback testing** — Migrations that can't go down
3. **Locking** — Long locks on production tables

**Prevention:**
- Always provide data migration for column changes
- Test `up` and `down` before committing
- Use `CONCURRENTLY` for Postgres (not needed for SQLite)

### Backup/Restore

**Pitfalls:**
1. **Corrupt backups** — Not verifying backup integrity
2. **No point-in-time** — Can't restore to specific time
3. **Backup during write** — Inconsistent snapshot

**Prevention:**
- Verify backup opens successfully
- Include timestamp in backup filename
- Implement application-level backup (pause writes)

## Watch Out For

### Integration Issues
- API server and CLI share config (ensure consistency)
- Web UI depends on API (can't work standalone initially)
- Migrations must run before API starts

### Testing Gaps
- No API integration tests yet
- No E2E tests for Web UI
- No migration tests

### Security Considerations
- API needs authentication (start with API key)
- Web UI needs CSRF protection
- Backup files should be encrypted in production

## Warning Signs
- ❌ Building new auth system (reuse existing)
- ❌ Frontend SPA without need (HTMX is enough)
- ❌ Migration without rollback (always test down)
- ❌ Backup without verification (test restore)