# Project: Aurora

## Overview
- **Project Name**: Aurora
- **Type**: CLI/TUI blockchain system with REST API and Web UI
- **Status**: Active development

## Context

Aurora is a CLI/TUI blockchain system implementing:
- VRF-based lottery
- Ed25519-signed voting
- NFT minting/transfer
- Fungible token system
- Data oracle

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
| Migrations | `aurora migrate status\|up\|down`, auto-run |
| REST API | Chi router, auth middleware, CORS, JSON responses |
| Oracle | Real data fetching, validation, error handling |
| Web UI | Dashboard, Lottery, Voting pages (HTMX + Alpine.js) |
| Backup | `aurora backup create\|verify`, JSON export |

**Deferred:** Backup restore (BCK-04) → v1.2

## Current State

### What's Shipped (v1.0 + v1.1)
- CLI with 5 modules (lottery, voting, NFT, token, oracle)
- TUI interfaces for all modules
- REST API server with authentication
- Web UI (browser-based access)
- Database migrations
- Backup/verify functionality
- 80%+ test coverage across modules

### Tech Stack
- Go 1.26+
- Chi (HTTP router)
- SQLite (database)
- HTMX + Alpine.js (Web UI)
- Cobra (CLI)
- Bubbletea (TUI)

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each milestone:**
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-26 after v1.1 Production Hardening milestone*