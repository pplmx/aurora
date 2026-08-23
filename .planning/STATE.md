# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-11)

**Core value:** Complete, production-ready blockchain toolkit with comprehensive test coverage and operational tooling
**Current focus:** v1.54 Oracle SSRF TOCTOU hardening complete

## Current Position

Phase: v1.5+ Continuous Deep-Dive Loop
Plan: Incremental milestones tracked in the RIL graph and git history
Status: v1.24–v1.54 complete (web/API/CLI parity, observability, security hardening, self-containment)
Last activity: 2026-08-23 — v1.54 closed: Oracle fetch-time SSRF re-validation
  closes the add-time→dial TOCTOU window (TASK-067, ISS-059, CHG-065 / a4a166e).
  RIL graph at round 64.

Progress: continuous loop — every resolved milestone advanced the graph; deep-dive
  found a genuine TOCTOU/DNS-rebinding SSRF gap and closed it fail-closed.

## Milestone History (recent)

| Version | Focus | Result |
| ------- | ----- | ------ |
| v1.52 | Lottery web create payload fix + create/verify smoke | ✅ done |
| v1.53 | Web self-containment: remove unused external htmx | ✅ done |
| v1.54 | Oracle SSRF TOCTOU hardening (fetch-time re-validation) | ✅ done |

## Session Continuity

Last session: 2026-08-23 — v1.54 Oracle SSRF TOCTOU hardening complete
Next: continue graph-engineering deep-dive for the next milestone
