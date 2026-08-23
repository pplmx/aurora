# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-11)

**Core value:** Complete, production-ready blockchain toolkit with comprehensive test coverage and operational tooling
**Current focus:** v1.65 backup self-overwrite guard complete

## Current Position

Phase: v1.5+ Continuous Deep-Dive Loop
Plan: Incremental milestones tracked in the RIL graph and git history
Status: v1.24–v1.65 complete (web/API/CLI parity, security hardening, observability, integrity, collision + extraction hardening)
Last activity: 2026-08-23 — v1.65 closed: BackupService.Create refuses to back
  a database up into itself, so `aurora backup create ./data` can no longer
  O_TRUNC the live DB and report success while silently destroying it
  (TASK-079, ISS-071, CHG-077).
  RIL graph at round 75.

Progress: continuous loop — every resolved milestone advanced the graph;
  recent deep-dives closed a CRITICAL CORS/key-exfiltration flaw (v1.64), a
  baseline test-suite regression, and a silent backup data-loss path (v1.65).

## Milestone History (recent)

| Version | Focus | Result |
| ------- | ----- | ------ |
| v1.63 | Oracle path-extraction fail-closed | ✅ done |
| v1.64 | CORS cross-origin key-exfiltration hardening | ✅ done |
| v1.65 | Backup self-overwrite guard | ✅ done |

## Session Continuity

Last session: 2026-08-23 — v1.65 backup self-overwrite guard complete
Next: continue graph-engineering deep-dive for the next milestone
