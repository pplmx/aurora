# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-11)

**Core value:** Complete, production-ready blockchain toolkit with comprehensive test coverage and operational tooling
**Current focus:** v1.71 bounded API request bodies complete

## Current Position

Phase: v1.5+ Continuous Deep-Dive Loop
Plan: Incremental milestones tracked in the RIL graph and git history
Status: v1.24–v1.71 complete (web/API/CLI parity, security hardening, observability, integrity, collision + extraction hardening, concurrency atomicity, event/state atomicity, rate-limit spoof hardening, sqlite writer serialization, bounded request bodies)
Last activity: 2026-08-25 — v1.71 closed: no JSON handler bounded its request
  body (no http.MaxBytesReader anywhere) and rate limiting is off by default,
  so an API-key holder could stream an arbitrarily large body (e.g. a fake
  lottery participants array) and the json.Decoder materialized every element.
  New BodyLimit middleware (4 MiB MaxBytesReader) sits outermost in the
  /api/v1 group (before the limiter and before APIKeyAuth); all 17 decode
  sites route through decodeJSON which maps a cap trip to 413 BODY_TOO_LARGE.
  Tests: middleware under/over-cap units + real-router 413 (TASK-085, ISS-077,
  CHG-083 / cad25cd).
  RIL graph at round 81.

Progress: continuous loop — every resolved milestone advanced the graph;
  recent deep-dives closed a CRITICAL CORS/key-exfiltration flaw (v1.64), a
  baseline test-suite regression (v1.64), a silent backup data-loss path
  (v1.65), an NFT audit-history collapse (v1.66), a non-atomic nonce claim
  that broke under a real SQLite pool (v1.67), a phantom-event leak on
  token transaction rollback (v1.68), a rate-limit spoof bypass via
  chi RealIP trusting client-supplied forwarded headers (v1.69),
  SQLITE_BUSY writer contention over the real pool killing concurrent
  transfers (v1.70), and unbounded JSON request bodies enabling a
  key-holder memory-exhaustion path (v1.71).

## Milestone History (recent)

| Version | Focus | Result |
| ------- | ----- | ------ |
| v1.64 | CORS cross-origin key-exfiltration hardening | ✅ done |
| v1.65 | Backup self-overwrite guard | ✅ done |
| v1.66 | NFT operation audit-trail collapse | ✅ done |
| v1.67 | Atomic ClaimNextNonce under a real connection pool | ✅ done |
| v1.68 | No phantom events on token tx rollback | ✅ done |
| v1.69 | Rate-limit spoof bypass via trusted-proxy allow-list | ✅ done |
| v1.70 | SQLite writer contention (SQLITE_BUSY) over the real pool | ✅ done |
| v1.71 | Unbounded JSON request bodies (4 MiB cap, 413) | ✅ done |

## Session Continuity

Last session: 2026-08-25 — v1.71 bounded API request bodies (4 MiB → 413)
Next: continue graph-engineering deep-dive for the next milestone (backlog
  empty after v1.71 — next candidate to surface via deep-dive)
