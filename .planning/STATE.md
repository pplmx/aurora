# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-11)

**Core value:** Complete, production-ready blockchain toolkit with comprehensive test coverage and operational tooling
**Current focus:** v1.70 SQLite writer-contention hardening complete

## Current Position

Phase: v1.5+ Continuous Deep-Dive Loop
Plan: Incremental milestones tracked in the RIL graph and git history
Status: v1.24–v1.70 complete (web/API/CLI parity, security hardening, observability, integrity, collision + extraction hardening, concurrency atomicity, event/state atomicity, rate-limit spoof hardening, sqlite writer serialization)
Last activity: 2026-08-25 — v1.70 closed: multi-statement write transactions
  (token mint/transfer/transferFrom/burn, voting, NFT) used a deferred BEGIN
  and grabbed the SQLite write lock mid-transaction after a read snapshot —
  over the API's real unlimited pool concurrent writers collided and ~60% of
  16 transfer-shaped transactions failed with SQLITE_BUSY (500s). Every
  write-path DSN is now hardened: `_txlock=immediate` (BEGIN IMMEDIATE
  serializes writers, removing the un-waitable SQLITE_BUSY_SNAPSHOT class)
  + `_busy_timeout=5000`, via a shared `dsn()` helper for token/nft/lottery/
  oracle, inline on the blockchain/voting InitDB singleton, and a
  `_busy_timeout` on the event store (single-INSERT only) (TASK-084, ISS-076,
  CHG-082 / 6e2697a). Regression: 24 concurrent transfers all commit with
  exact ledger accounting.
  RIL graph at round 80.

Progress: continuous loop — every resolved milestone advanced the graph;
  recent deep-dives closed a CRITICAL CORS/key-exfiltration flaw (v1.64), a
  baseline test-suite regression (v1.64), a silent backup data-loss path
  (v1.65), an NFT audit-history collapse (v1.66), a non-atomic nonce claim
  that broke under a real SQLite pool (v1.67), a phantom-event leak on
  token transaction rollback (v1.68), a rate-limit spoof bypass via
  chi RealIP trusting client-supplied forwarded headers (v1.69), and
  SQLITE_BUSY writer contention over the real pool killing concurrent
  transfers (v1.70).

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

## Session Continuity

Last session: 2026-08-25 — v1.70 serialized SQLite writers under the real pool
Next: continue graph-engineering deep-dive for the next milestone (backlog
  empty after v1.70 — next candidate to surface via deep-dive)
