# Project: Aurora

## What This Is

Aurora is a CLI/TUI blockchain system implementing VRF-based lottery, Ed25519-signed voting, NFTs, fungible tokens, and data oracle functionality. Users interact via CLI, TUI, REST API, or Web UI.

## Core Value

Complete, production-ready blockchain toolkit with comprehensive test coverage and operational tooling.

## Milestones

### ✅ v1.0 MVP (2026-04-26)

**Test Coverage Foundation** — Achieved 80%+ coverage across all modules

| Module  | Coverage |
| ------- | -------- |
| Lottery | 93.3%    |
| Token   | 89.9%    |
| NFT     | 93.8%    |
| Oracle  | 94.5%    |

Security: Voting timing, transactions, rate limiting, headers
Performance: Pagination, interruptible mining, configurable timeouts

### ✅ v1.1 Production Hardening (2026-04-26)

**Infrastructure and User-facing Features**

| Component  | Features                                            |
| ---------- | --------------------------------------------------- |
| Migrations | `aurora migrate status|up|down`, auto-run           |
| REST API   | Chi router, auth middleware, CORS, JSON responses   |
| Oracle     | Real data fetching, validation, error handling      |
| Web UI     | Dashboard, Lottery, Voting pages (HTMX + Alpine.js) |
| Backup     | `aurora backup create|verify`, JSON export          |

### ✅ v1.2 Operational Readiness (2026-04-30)

**Complete deferred items and improve production readiness**

| Component              | Status                        |
| ---------------------- | ----------------------------- |
| BCK-04: Backup restore | ✅ Implemented                |
| API handler tests      | ✅ 39.3% (improved from 9.5%) |
| Health check endpoint  | ✅ /healthz, /readyz          |
| Security hardening     | ✅ Timing-safe comparison     |

### ✅ v1.3 Quality & Documentation (2026-04-30)

**Comprehensive test coverage and improved user experience**

| Focus            | Target                   |
| ---------------- | ------------------------ |
| UI package tests | Meaningful coverage      |
| Handler tests    | 80%+ coverage            |
| E2E tests        | Automated test suite     |
| Documentation    | Help system improvements |

### ✅ v1.4 CLI Command Test Coverage (2026-08-11)

**Every cobra subcommand tested — cmd/aurora/cmd 21.9% → 86.3%**

| Focus                                                                                  | Result                     |
| -------------------------------------------------------------------------------------- | -------------------------- |
| Lottery/NFT/Oracle/Token/Voting/root command trees                                     | All tested (happy + error) |
| Migration engine fixed so real checkout migrations apply                               | Previously never ran       |
| Latent product bugs surfaced: broken reset, verify disambiguation, broken voting flags | Fixed                      |

### ✅ v1.5 Fresh-Install Operations & Coverage Bar (2026-08-11)

**Make a fresh install usable + enforce the 80% coverage bar everywhere**

| Focus                               | Target                                          |
| ----------------------------------- | ----------------------------------------------- |
| `aurora migrate up/status/down` CLI | ✅ Shipped (4e3c9ba)                            |
| logger / i18n / backup coverage     | ✅ 96.6% / 97.0% / 80.5%                        |
| ui/nft / ui/token coverage          | ✅ 94.2% / 88.6% (fixed NFT TUI transfer guard) |

### 🚧 v1.6 Interactive Surface Parity (2026-08-22)

**Make every documented interactive surface real + VERIFY-gate hygiene**

| Focus                                            | Target                                   |
| ------------------------------------------------ | ---------------------------------------- |
| `token tui` CLI wiring (real TUI, not stub)      | ✅ Shipped (52f8f58)                     |
| Oracle SSRF: bracketed IPv6 literal bypass        | ✅ Shipped (0176524)                     |
| http loopback probe panic (skip instead of crash) | ✅ Shipped (20daab9)                     |

## Context

### What's Shipped (v1.0 + v1.1 + v1.2)

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
- ✓ Backup/restore/verify — v1.2
- ✓ Health check endpoints (/healthz, /readyz) — v1.2
- ✓ Security hardening — v1.2

### Active

- [ ] `aurora migrate up/status/down` CLI subcommand (restore v1.1 MIG-03)
- [ ] Coverage ≥ 80% in logger, i18n, backup, ui/nft, ui/token

### Out of Scope

- Metrics/observability (Prometheus, OpenTelemetry) — defer to future
- Performance optimization beyond current requirements
- Mobile app

## Constraints

- **Tech stack**: Go, SQLite, no major framework changes
- **Quality bar**: All existing tests must pass; new tests required for new code
- **Backward compatibility**: Existing CLI/API behavior must not break

## Key Decisions

| Decision                                  | Rationale                                  | Outcome |
| ----------------------------------------- | ------------------------------------------ | ------- |
| Test coverage target 80% for API handlers | Prevent regression in critical paths       | Pending |
| Health check endpoint                     | Required for k8s/load balancer integration | Pending |

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

*Last updated: 2026-08-11 after v1.4 completion / v1.5 kickoff*
