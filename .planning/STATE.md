# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-11)

**Core value:** Complete, production-ready blockchain toolkit with comprehensive test coverage and operational tooling
**Current focus:** v1.77 audit-trail integrity + API contract hardening complete

## Current Position

Phase: v1.5+ Continuous Deep-Dive Loop
Plan: Incremental milestones tracked in the RIL graph and git history
Status: v1.24–v1.77 complete (web/API/CLI parity, security hardening, observability, integrity, collision + extraction hardening, concurrency atomicity, event/state atomicity, rate-limit spoof hardening, sqlite writer serialization, bounded request bodies, in-tx deadlock fix, api secrets/audit wiring, bounded metrics labels, consistent online backups, CLI failure exit codes, NFT burn audit-trail retention, owner-scoped token-history paging + list-envelope consistency, cmd/api config-file loading)
Last activity: 2026-08-25 — v1.77 closed (round-87 fleet deep-dive bugs):
  (1) nft_operations declared ON DELETE CASCADE so Burn's NFT delete wiped the
  whole operation trail incl. the just-saved burn op — FK removed from schema,
  ensureNoCascadeFK() heals legacy DBs at boot, 000001 migration updated
  (TASK-092, ISS-085, CHG-091 / ecc40aa); (2) token transfer history paged SQL
  over ALL transfers then filtered owner in-memory, under-filling pages on
  multi-owner tokens, and returned {"transfers":[...]} the web UI could never
  read — owner pushed into SQL via GetByAggregateAndTypePayload(json_extract),
  handler emits the bare array (TASK-093, ISS-086, CHG-090 / 28f17f3);
  (3) config.Load never ReadInConfig, so cmd/api ignored config/aurora.toml —
  Load now reads $HOME then ./config with env still winning, unparseable file
  fails loudly (TASK-094, ISS-087, CHG-089 / ed36acc).
  Recorded backlog for later rounds: ISS-088 voting lifecycle guard, ISS-089
  malformed-base64→500, ISS-090 oracle scheduler block_height=0.
  RIL graph at round 87.

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
  nonce deadlock the BEGIN IMMEDIATE change surfaced (v1.72), the
  cmd/api secrets + audit-trail wiring gaps (v1.73), unbounded metrics
  label cardinality (v1.74), stale live-WAL backups (v1.75), CLI TUI
  silent-success exit codes (v1.76), and the v1.77 audit-trail + API
  contract hardening above.

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
| v1.76 | CLI TUI commands exit 0 on failure (RunE) | ✅ done |
| v1.77 | NFT burn audit-trail retention + owner-scoped token-history paging/envelope + cmd/api config-file loading | ✅ done |

## Session Continuity

Last session: 2026-08-25 — v1.77 audit-trail integrity + API contract
  hardening; backlog recorded (voting lifecycle guard, malformed-base64→500,
  oracle scheduler block_height=0).
Next: promote the strongest recorded backlog issue (ISS-088 voting lifecycle,
  ISS-089 base64→4xx, or ISS-090 scheduler chain) into the next milestone, or
  run a fresh deep-dive round.
