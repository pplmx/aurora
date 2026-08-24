# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-11)

**Core value:** Complete, production-ready blockchain toolkit with comprehensive test coverage and operational tooling
**Current focus:** v1.75 consistent online backups complete

## Current Position

Phase: v1.5+ Continuous Deep-Dive Loop
Plan: Incremental milestones tracked in the RIL graph and git history
Status: v1.24–v1.75 complete (web/API/CLI parity, security hardening, observability, integrity, collision + extraction hardening, concurrency atomicity, event/state atomicity, rate-limit spoof hardening, sqlite writer serialization, bounded request bodies, in-tx deadlock fix, api secrets/audit wiring, bounded metrics labels, consistent online backups)
Last activity: 2026-08-25 — v1.73 closed (fleet deep-dive finds at the cmd/api
  boundary): config.Load — only used by cmd/api — never read AURORA_API_KEY
  (no AutomaticEnv/BindEnv in the API binary), so production always failed
  ErrMissingAPIKey even with the key set and dev minted a fresh random key
  each boot; and the server's SyncEventBus was never subscribed, silently
  dropping every token audit event so /api/v1/token/history was permanently
  empty on the HTTP path. Both fixed: BindEnv in Load(), audit+stats handlers
  subscribed in server.go. Regressions: env-key config test +
  TestTokenAudit_TransferAppearsInHistoryOverHTTP (TASK-087/088, ISS-079/080,
  CHG-085 / cd5387b).
  RIL graph at round 85.

Progress: continuous loop — every resolved milestone advanced the graph;
  recent deep-dives closed a CRITICAL CORS/key-exfiltration flaw (v1.64), a
  baseline test-suite regression (v1.64), a silent backup data-loss path
  (v1.65), an NFT audit-history collapse (v1.66), a non-atomic nonce claim
  that broke under a real SQLite pool (v1.67), a phantom-event leak on
  token transaction rollback (v1.68), a rate-limit spoof bypass via
  chi RealIP trusting client-supplied forwarded headers (v1.69),
  SQLITE_BUSY writer contention over the real pool killing concurrent
  transfers (v1.70), unbounded JSON request bodies enabling a
  key-holder memory-exhaustion path (v1.71), the in-transaction
  nonce deadlock the BEGIN IMMEDIATE change surfaced (v1.72), and the
  cmd/api secrets + audit-trail wiring gaps (v1.73).

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
| v1.73 | cmd/api secrets + audit-trail wiring | ✅ done |
| v1.74 | Unbounded metrics label cardinality (whitelist + other bucket) | ✅ done |
| v1.75 | Backups stale under live WAL server (VACUUM INTO snapshot) | ✅ done |

## Session Continuity

Last session: 2026-08-25 — v1.75 consistent online backups
Next: continue graph-engineering loop on the fleet deep-dive backlog
  (verified remaining candidates: phantom on-chain blocks on rolled-back
  txs — record accepted-tradeoff decision; CLI tui exits 0 on failure;
  API-key-in-HTML documented tech debt)
