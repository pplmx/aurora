# Roadmap: Aurora v1.2 Operational Readiness

## Milestones

- ✅ **v1.0 MVP** - Phases 1-4 (shipped 2026-04-26)
- ✅ **v1.1 Production Hardening** - Phases 5-6 (shipped 2026-04-26)
- 🚧 **v1.2 Operational Readiness** - Phases 7-10 (in progress)
- 📋 **v2.0 Future** - Phased roll-out (planned)

## Overview

Aurora v1.2 completes the operational readiness work deferred from v1.1. This release prioritizes security fixes (P0), health monitoring for Kubernetes deployment, complete backup/restore functionality, and comprehensive handler test coverage to establish regression safety.

## Phases

- [ ] **Phase 7: Security Hardening** - Fix API key security vulnerabilities
- [ ] **Phase 8: Operations & Health** - Kubernetes-ready health endpoints and graceful shutdown
- [ ] **Phase 9: Backup & Restore** - Complete backup/restore implementation with verification
- [ ] **Phase 10: Test Coverage** - Handler and TUI test coverage to 80%+

## Phase Details

### Phase 7: Security Hardening
**Goal**: Eliminate critical API key security vulnerabilities (timing attacks, hardcoded defaults, information leakage)
**Depends on**: Nothing (first v1.2 phase)
**Requirements**: SEC-01, SEC-02, SEC-03
**Success Criteria** (what must be TRUE):
  1. API key comparison uses `crypto/subtle.ConstantTimeCompare` (timing-attack resistant)
  2. Server refuses to start in production without explicit API key configuration
  3. Auth failure responses are generic: "authentication required" (no hints about key structure)
  4. Hardcoded default API key removed from source code
**Plans**: 3 plans

Plans:
- [x] 07-01: Implement constant-time API key comparison
- [x] 07-02: Remove hardcoded default, enforce env var requirement
- [x] 07-03: Sanitize auth error messages to prevent information leakage

### Phase 8: Operations & Health
**Goal**: Kubernetes-ready health endpoints with graceful shutdown behavior
**Depends on**: Phase 7
**Requirements**: OPS-01, OPS-02, OPS-03, OPS-04
**Success Criteria** (what must be TRUE):
  1. `GET /healthz` returns 200 if HTTP server responds (liveness probe)
  2. `GET /readyz` returns 503 when database is unreachable (readiness probe)
  3. Health endpoints bypass authentication (accessible without API key)
  4. Server shutdown waits for in-flight requests (graceful, not abrupt kill)
**Plans**: 4 plans

Plans:
- [ ] 08-01: Implement liveness endpoint `/healthz`
- [ ] 08-02: Implement readiness endpoint `/readyz` with database ping
- [ ] 08-03: Register health endpoints before auth middleware (bypass auth)
- [ ] 08-04: Implement graceful shutdown with `server.Shutdown(ctx)`

### Phase 9: Backup & Restore
**Goal**: Complete backup/restore functionality with verification and safety guarantees
**Depends on**: Phase 8
**Requirements**: BACK-01, BACK-02, BACK-03, BACK-04
**Success Criteria** (what must be TRUE):
  1. `aurora backup restore --file <path>` successfully restores database from backup
  2. Restore applies pending schema migrations if backup is from older version
  3. Restore automatically creates pre-restore backup before destructive operations
  4. `aurora backup verify` validates backup file integrity (schema, data, constraints)
**Plans**: 4 plans

Plans:
- [ ] 09-01: Implement `aurora backup restore` command (file-based restore)
- [ ] 09-02: Integrate golang-migrate for schema versioning during restore
- [ ] 09-03: Implement automatic pre-restore backup creation
- [ ] 09-04: Implement multi-level backup verification

### Phase 10: Test Coverage
**Goal**: Comprehensive test coverage for API handlers and TUI model logic
**Depends on**: Phase 9
**Requirements**: TEST-01, TEST-02, TEST-03, TEST-04, TEST-05, TEST-06
**Success Criteria** (what must be TRUE):
  1. Token API handlers achieve ≥ 80% test coverage
  2. NFT API handlers achieve ≥ 80% test coverage
  3. Lottery API handlers achieve ≥ 80% test coverage
  4. Voting API handlers achieve ≥ 80% test coverage
  5. Oracle API handlers achieve ≥ 80% test coverage
  6. TUI package model logic has unit tests covering core Update/View behavior
**Plans**: 6 plans

Plans:
- [ ] 10-01: Token API handler tests (≥ 80% coverage)
- [ ] 10-02: NFT API handler tests (≥ 80% coverage)
- [ ] 10-03: Lottery API handler tests (≥ 80% coverage)
- [ ] 10-04: Voting API handler tests (≥ 80% coverage)
- [ ] 10-05: Oracle API handler tests (≥ 80% coverage)
- [ ] 10-06: TUI package unit tests for model logic

## Progress

**Execution Order:**
Phases execute in numeric order: 7 → 8 → 9 → 10

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 7. Security Hardening | 3/3 | Plans created | - |
| 8. Operations & Health | 0/4 | Not started | - |
| 9. Backup & Restore | 0/4 | Not started | - |
| 10. Test Coverage | 0/6 | Not started | - |