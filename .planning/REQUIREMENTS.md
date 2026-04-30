# Requirements: Aurora v1.2

**Defined:** 2026-04-30
**Core Value:** Complete, production-ready blockchain toolkit with comprehensive test coverage and operational tooling

## v1 Requirements

Requirements for v1.2 Operational Readiness. Each maps to roadmap phases.

### Security

- [ ] **SEC-01**: API authentication uses constant-time comparison to prevent timing attacks
- [ ] **SEC-02**: Remove hardcoded default API key, fail in production if not configured
- [ ] **SEC-03**: API auth error responses are generic ("authentication required") to prevent information leakage

### Operations

- [ ] **OPS-01**: Liveness endpoint `GET /healthz` returns 200 if server is running
- [ ] **OPS-02**: Readiness endpoint `GET /readyz` returns 503 if database is unavailable
- [ ] **OPS-03**: API server graceful shutdown waits for in-flight requests
- [ ] **OPS-04**: Health endpoints bypass authentication middleware

### Backup/Restore

- [ ] **BACK-01**: `aurora backup restore --file <path>` restores from backup file
- [ ] **BACK-02**: Restore performs schema migration if needed
- [ ] **BACK-03**: Restore creates automatic pre-restore backup
- [ ] **BACK-04**: `aurora backup verify` validates backup file integrity

### Testing

- [ ] **TEST-01**: Token API handler coverage ≥ 80%
- [ ] **TEST-02**: NFT API handler coverage ≥ 80%
- [ ] **TEST-03**: Lottery API handler coverage ≥ 80%
- [ ] **TEST-04**: Voting API handler coverage ≥ 80%
- [ ] **TEST-05**: Oracle API handler coverage ≥ 80%
- [ ] **TEST-06**: Core TUI packages have unit tests for model logic

## v2 Requirements

Deferred to future release.

### Observability

- **OBS-01**: Prometheus metrics endpoint
- **OBS-02**: Structured logging with correlation IDs
- **OBS-03**: OpenTelemetry tracing

### Advanced Backup

- **BACK-05**: Point-in-time recovery by block height
- **BACK-06**: Incremental backup support

## Out of Scope

| Feature | Reason |
|---------|--------|
| Mobile app | Web-first, native later |
| Multi-node/clustering | Single instance for v1.x |
| OAuth/OIDC integration | API key sufficient for v1.x |
| Full E2E test suite | Unit/integration coverage priority |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| SEC-01 | Phase 7 | Pending |
| SEC-02 | Phase 7 | Pending |
| SEC-03 | Phase 7 | Pending |
| OPS-01 | Phase 8 | Pending |
| OPS-02 | Phase 8 | Pending |
| OPS-03 | Phase 8 | Pending |
| OPS-04 | Phase 8 | Pending |
| BACK-01 | Phase 9 | Pending |
| BACK-02 | Phase 9 | Pending |
| BACK-03 | Phase 9 | Pending |
| BACK-04 | Phase 9 | Pending |
| TEST-01 | Phase 10 | Pending |
| TEST-02 | Phase 10 | Pending |
| TEST-03 | Phase 10 | Pending |
| TEST-04 | Phase 10 | Pending |
| TEST-05 | Phase 10 | Pending |
| TEST-06 | Phase 10 | Pending |

**Coverage:**
- v1 requirements: 17 total
- Mapped to phases: 17
- Unmapped: 0 ✓

---
*Requirements defined: 2026-04-30*
*Last updated: 2026-04-30 after v1.2 initialization*