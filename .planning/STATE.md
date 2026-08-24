# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-11)

**Core value:** Complete, production-ready blockchain toolkit with comprehensive test coverage and operational tooling
**Current focus:** v1.68 post-commit token event publication complete

## Current Position

Phase: v1.5+ Continuous Deep-Dive Loop
Plan: Incremental milestones tracked in the RIL graph and git history
Status: v1.24–v1.68 complete (web/API/CLI parity, security hardening, observability, integrity, collision + extraction hardening, concurrency atomicity, event/state atomicity)
Last activity: 2026-08-24 — v1.68 closed: Mint/Transfer/TransferFrom/Burn
  now publish their audit event only AFTER the token transaction commits —
  the event store is a separate DB the token rollback cannot undo, so the old
  in-tx publish left a phantom event in GetTransferHistory whenever a later
  tx step failed (TASK-082, ISS-074, CHG-080 / ab49c03).
  RIL graph at round 78.

Progress: continuous loop — every resolved milestone advanced the graph;
  recent deep-dives closed a CRITICAL CORS/key-exfiltration flaw (v1.64), a
  baseline test-suite regression (v1.64), a silent backup data-loss path
  (v1.65), an NFT audit-history collapse (v1.66), a non-atomic nonce claim
  that broke under a real SQLite pool (v1.67), and a phantom-event leak on
  token transaction rollback (v1.68).

## Milestone History (recent)

| Version | Focus | Result |
| ------- | ----- | ------ |
| v1.64 | CORS cross-origin key-exfiltration hardening | ✅ done |
| v1.65 | Backup self-overwrite guard | ✅ done |
| v1.66 | NFT operation audit-trail collapse | ✅ done |
| v1.67 | Atomic ClaimNextNonce under a real connection pool | ✅ done |
| v1.68 | No phantom events on token tx rollback | ✅ done |

## Session Continuity

Last session: 2026-08-24 — v1.68 post-commit token event publication complete
Next: continue graph-engineering deep-dive for the next milestone (backlog:
  ISS-073 rate-limit spoof bypass via chi RealIP trusting client headers)
