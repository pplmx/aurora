# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-11)

**Core value:** Complete, production-ready blockchain toolkit with comprehensive test coverage and operational tooling
**Current focus:** v1.66 NFT operation audit-trail collapse complete

## Current Position

Phase: v1.5+ Continuous Deep-Dive Loop
Plan: Incremental milestones tracked in the RIL graph and git history
Status: v1.24–v1.66 complete (web/API/CLI parity, security hardening, observability, integrity, collision + extraction hardening)
Last activity: 2026-08-23 — v1.66 closed: nft.NewOperation now assigns each
  operation a unique UUID id, so the SQLite audit history no longer collapses
  to a single ""-key row and on-chain operation records are meaningful again
  (TASK-080, ISS-072, CHG-078 / ba906c9).
  RIL graph at round 76.

Progress: continuous loop — every resolved milestone advanced the graph;
  recent deep-dives closed a CRITICAL CORS/key-exfiltration flaw (v1.64), a
  baseline test-suite regression (v1.64), a silent backup data-loss path
  (v1.65), and an NFT audit-history collapse (v1.66).

## Milestone History (recent)

| Version | Focus | Result |
| ------- | ----- | ------ |
| v1.64 | CORS cross-origin key-exfiltration hardening | ✅ done |
| v1.65 | Backup self-overwrite guard | ✅ done |
| v1.66 | NFT operation audit-trail collapse | ✅ done |

## Session Continuity

Last session: 2026-08-23 — v1.66 NFT operation audit-trail collapse complete
Next: continue graph-engineering deep-dive for the next milestone
