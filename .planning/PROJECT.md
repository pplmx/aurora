# Project: Aurora

## Overview
- **Project Name**: Aurora
- **Type**: CLI/TUI blockchain system
- **Status**: Active development

## Current Milestone: v1.1 Production Hardening

**Goal:** Add infrastructure and user-facing features to make Aurora production-ready.

**Target features:**
- REST API server for programmatic access
- Oracle real data fetching
- Web UI (browser interface)
- Database migration system
- Backup/restore functionality

## Context

Aurora is a CLI/TUI blockchain system implementing:
- VRF-based lottery
- Ed25519-signed voting
- NFT minting/transfer
- Fungible token system
- Data oracle

### v1.0 MVP (Completed)
| Module | Coverage |
|--------|----------|
| Lottery | 93.3% |
| Token | 89.9% |
| NFT | 93.8% |
| Oracle | 94.5% |

Security: Voting timing, transactions, rate limiting, headers
Performance: Pagination, interruptible mining, configurable timeouts

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
*Last updated: 2026-04-26 after v1.1 milestone started*