# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-11)

**Core value:** Complete, production-ready blockchain toolkit with comprehensive test coverage and operational tooling
**Current focus:** v1.55 Backup content-integrity checksums complete

## Current Position

Phase: v1.5+ Continuous Deep-Dive Loop
Plan: Incremental milestones tracked in the RIL graph and git history
Status: v1.24–v1.55 complete (web/API/CLI parity, observability, security hardening, self-containment, backup integrity)
Last activity: 2026-08-23 — v1.55 closed: backup now records and verifies
  per-database content SHA-256, so a truncated/corrupted .db is rejected before
  restore (TASK-068, ISS-060, CHG-066 / 31f8fd9). RIL graph at round 65.

Progress: continuous loop — every resolved milestone advanced the graph; deep-dive
  found a data-integrity gap (backup verified metadata, not DB content) and closed it.

## Milestone History (recent)

| Version | Focus | Result |
| ------- | ----- | ------ |
| v1.53 | Web self-containment: remove unused external htmx | ✅ done |
| v1.54 | Oracle SSRF TOCTOU hardening (fetch-time re-validation) | ✅ done |
| v1.55 | Backup content-integrity checksums | ✅ done |

## Session Continuity

Last session: 2026-08-23 — v1.55 Backup content-integrity checksums complete
Next: continue graph-engineering deep-dive for the next milestone
