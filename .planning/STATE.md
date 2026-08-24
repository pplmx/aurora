# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-11)

**Core value:** Complete, production-ready blockchain toolkit with comprehensive test coverage and operational tooling
**Current focus:** v1.67 ClaimNextNonce atomicity complete

## Current Position

Phase: v1.5+ Continuous Deep-Dive Loop
Plan: Incremental milestones tracked in the RIL graph and git history
Status: v1.24–v1.67 complete (web/API/CLI parity, security hardening, observability, integrity, collision + extraction hardening, concurrency atomicity)
Last activity: 2026-08-24 — v1.67 closed: SQLiteReplayProtection.ClaimNextNonce
  is now a single atomic UPSERT ... RETURNING nonce (with _busy_timeout=5000 on
  the replay DSN) instead of a deferred-tx read-modify-write, so concurrent
  claims over a real connection pool each get a unique monotonic nonce — no
  more SQLITE_BUSY_SNAPSHOT errors or duplicate nonces (TASK-081, ISS-075,
  CHG-079 / ebf21e3).
  RIL graph at round 77.

Progress: continuous loop — every resolved milestone advanced the graph;
  recent deep-dives closed a CRITICAL CORS/key-exfiltration flaw (v1.64), a
  baseline test-suite regression (v1.64), a silent backup data-loss path
  (v1.65), an NFT audit-history collapse (v1.66), and a non-atomic nonce
  claim that broke under a real SQLite pool (v1.67).

## Milestone History (recent)

| Version | Focus | Result |
| ------- | ----- | ------ |
| v1.64 | CORS cross-origin key-exfiltration hardening | ✅ done |
| v1.65 | Backup self-overwrite guard | ✅ done |
| v1.66 | NFT operation audit-trail collapse | ✅ done |
| v1.67 | Atomic ClaimNextNonce under a real connection pool | ✅ done |

## Session Continuity

Last session: 2026-08-24 — v1.67 ClaimNextNonce atomicity complete
Next: continue graph-engineering deep-dive for the next milestone (backlog:
  ISS-073 rate-limit spoof bypass, ISS-074 phantom events on tx rollback)
