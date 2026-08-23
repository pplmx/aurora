# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-11)

**Core value:** Complete, production-ready blockchain toolkit with comprehensive test coverage and operational tooling
**Current focus:** v1.64 CORS cross-origin key-exfiltration hardening complete

## Current Position

Phase: v1.5+ Continuous Deep-Dive Loop
Plan: Incremental milestones tracked in the RIL graph and git history
Status: v1.24–v1.64 complete (web/API/CLI parity, security hardening, observability, integrity, collision + extraction hardening)
Last activity: 2026-08-23 — v1.64 closed: CORS no longer emits a wildcard, so a
  cross-origin page can no longer read the API key embedded in the served Web
  UI and drive /api/v1 as the operator; the stale pre-v1.63 fetcher validation
  test was also brought in line so the baseline suite is green again
  (TASK-077/078, ISS-069/070, CHG-075/076 / 7037712, 40bc48c).
  RIL graph at round 74.

Progress: continuous loop — every resolved milestone advanced the graph;
  deep-dive found a CRITICAL CORS/key-exfiltration composition flaw and closed
  it, plus a v1.63 regression that left the test suite red.

## Milestone History (recent)

| Version | Focus | Result |
| ------- | ----- | ------ |
| v1.62 | Token duplicate-symbol guard | ✅ done |
| v1.63 | Oracle path-extraction fail-closed | ✅ done |
| v1.64 | CORS cross-origin key-exfiltration hardening | ✅ done |

## Session Continuity

Last session: 2026-08-23 — v1.64 CORS cross-origin key-exfiltration hardening complete
Next: continue graph-engineering deep-dive for the next milestone
