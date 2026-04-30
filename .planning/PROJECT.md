# Project: Aurora

## What This Is

Aurora is a CLI/TUI blockchain system implementing VRF-based lottery, Ed25519-signed voting, NFTs, fungible tokens, and data oracle functionality. Users interact via CLI, TUI, REST API, or Web UI.

## Core Value

Complete, production-ready blockchain toolkit with comprehensive test coverage and operational tooling.

## Milestones

### ✅ v1.0 MVP (2026-04-26)
**Test Coverage Foundation** — Achieved 80%+ coverage across all modules

| Module | Coverage |
|--------|----------|
| Lottery | 93.3% |
| Token | 89.9% |
| NFT | 93.8% |
| Oracle | 94.5% |

Security: Voting timing, transactions, rate limiting, headers
Performance: Pagination, interruptible mining, configurable timeouts

### ✅ v1.1 Production Hardening (2026-04-26)
**Infrastructure and User-facing Features**

| Component | Features |
|-----------|----------|
| Migrations | `aurora migrate status|up|down`, auto-run |
| REST API | Chi router, auth middleware, CORS, JSON responses |
| Oracle | Real data fetching, validation, error handling |
| Web UI | Dashboard, Lottery, Voting pages (HTMX + Alpine.js) |
| Backup | `aurora backup create|verify`, JSON export |

### 🔄 v1.2 Operational Readiness (current)
**Complete deferred items and improve production readiness**

| Item | Status |
|------|--------|
| BCK-04: Backup restore | Deferred from v1.1 |
| API handler tests | Coverage 3.6-9.5% |
| UI package tests | Coverage 0% |
| Health check endpoint | Not implemented |
| Security hardening | API key validation |

## Context

### What's Shipped (v1.0 + v1.1)
- CLI with 5 modules (lottery, voting, NFT, token, oracle)
- TUI interfaces for all modules
- REST API server with authentication
- Web UI (browser-based access)
- Database migrations
- Backup/verify functionality
- 80%+ test coverage across domain layer

### Tech Stack
- Go 1.26+
- Chi (HTTP router)
- SQLite (database)
- HTMX + Alpine.js (Web UI)
- Cobra (CLI)
- Bubbletea (TUI)

### Deferred from v1.1 → v1.2
- BCK-04: Backup restore not implemented
- API handler tests needed
- UI package tests needed

## Requirements

### Validated

- ✓ VRF-based lottery system — v1.0
- ✓ Ed25519-signed voting — v1.0
- ✓ NFT minting/transfer — v1.0
- ✓ Fungible token system — v1.0
- ✓ Data oracle — v1.0
- ✓ CLI with 5 modules — v1.0
- ✓ TUI interfaces — v1.0
- ✓ REST API with auth — v1.1
- ✓ Web UI — v1.1
- ✓ Database migrations — v1.1
- ✓ Backup/verify — v1.1

### Active

- [ ] BCK-04: Backup restore functionality
- [ ] API handler tests (80%+ coverage)
- [ ] UI package tests (meaningful coverage)
- [ ] Health check endpoint (`/health`, `/ready`)
- [ ] API key runtime validation

### Out of Scope

- Metrics/observability (Prometheus, OpenTelemetry) — defer to future
- Performance optimization beyond current requirements
- Mobile app

## Constraints

- **Tech stack**: Go, SQLite, no major framework changes
- **Quality bar**: All existing tests must pass; new tests required for new code
- **Backward compatibility**: Existing CLI/API behavior must not break

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Test coverage target 80% for API handlers | Prevent regression in critical paths | Pending |
| Health check endpoint | Required for k8s/load balancer integration | Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition:**
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone:**
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-30 after v1.2 initialization*