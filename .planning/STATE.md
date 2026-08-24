# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-11)

**Core value:** Complete, production-ready blockchain toolkit with comprehensive test coverage and operational tooling
**Current focus:** v1.72 in-transaction nonce-deadlock fix complete

## Current Position

Phase: v1.5+ Continuous Deep-Dive Loop
Plan: Incremental milestones tracked in the RIL graph and git history
Status: v1.24–v1.72 complete (web/API/CLI parity, security hardening, observability, integrity, collision + extraction hardening, concurrency atomicity, event/state atomicity, rate-limit spoof hardening, sqlite writer serialization, bounded request bodies, in-tx deadlock fix)
Last activity: 2026-08-25 — v1.72 closed: the v1.70 BEGIN IMMEDIATE DSN
  change exposed a latent cross-connection write — Transfer/TransferFrom
  re-saved the claimed nonce INSIDE the transaction via the replay store's
  own connection to the same aurora.db, deadlocking against the token tx's
  write lock (5s busy timeout → spurious 500). The in-tx SaveNonce was
  redundant (ClaimNextNonce already persisted it) and is removed from both
  paths; every remaining transaction body writes only through txRepo
  (TASK-086, ISS-078, CHG-084 / 84c5e43). Regression:
  TestTokenTransfer_HappyPath + TestTokenNFT_Smoke… failed 3/3 pre-fix.
  RIL graph at round 82.

Progress: continuous loop — every resolved milestone advanced the graph;
  recent deep-dives closed a CRITICAL CORS/key-exfiltration flaw (v1.64), a
  baseline test-suite regression (v1.64), a silent backup data-loss path
  (v1.65), an NFT audit-history collapse (v1.66), a non-atomic nonce claim
  that broke under a real SQLite pool (v1.67), a phantom-event leak on
  token transaction rollback (v1.68), a rate-limit spoof bypass via
  chi RealIP trusting client-supplied forwarded headers (v1.69),
  SQLITE_BUSY writer contention over the real pool killing concurrent
  transfers (v1.70), unbounded JSON request bodies enabling a
  key-holder memory-exhaustion path (v1.71), and the in-transaction
  nonce deadlock the BEGIN IMMEDIATE change surfaced (v1.72).

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
| v1.72 | In-transaction nonce deadlock (v1.70 regression) | ✅ done |

## Session Continuity

Last session: 2026-08-25 — v1.72 fixed the in-tx nonce deadlock (v1.70 regression)
Next: continue graph-engineering deep-dive for the next milestone (backlog
  empty after v1.72 — next candidate to surface via deep-dive)
