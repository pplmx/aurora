# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-11)

**Core value:** Complete, production-ready blockchain toolkit with comprehensive test coverage and operational tooling
**Current focus:** v1.80 phantom HOME data-dir cleanup complete

## Current Position

Phase: v1.5+ Continuous Deep-Dive Loop
Plan: Incremental milestones tracked in the RIL graph and git history
Status: v1.24–v1.80 complete (web/API/CLI parity, security hardening, observability, integrity, collision + extraction hardening, concurrency atomicity, event/state atomicity, rate-limit spoof hardening, sqlite writer serialization, bounded request bodies, in-tx deadlock fix, api secrets/audit wiring, bounded metrics labels, consistent online backups, CLI failure exit codes, NFT burn audit-trail retention, owner-scoped token-history paging + list-envelope consistency, cmd/api config-file loading, malformed-base64→400 classification, oracle scheduler on-chain recording, honored token create --decimals, lottery-reset declined/exit≠0, bounded NFT list paging, db.path honored everywhere, no phantom HOME data dir on CLI commands)
Last activity: 2026-08-27 — v1.80 closed (ISS-095):
  PersistentPreRunE ran app.Wire(dataDir) on every subcommand and stashed it
  in the never-read GlobalApp, so even `aurora version` created a phantom
  $HOME/.aurora/data with an unused tokens/events/nonces .db triple. Removed
  the dead wiring (root.go); the autoRun migrations target the same
  blockchain.DBPath() as the stores (TASK-103, CHG-099 / 3b369aa).
  RIL graph at round 90.

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
  silent-success exit codes (v1.76), the v1.77 audit-trail + API
  contract hardening, and the v1.78 client-error + oracle chain fixes.

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
| v1.78 | Base64→400 classification + oracle scheduler on-chain recording (voting reopen decided-not-a-bug, DEC-004) | ✅ done |
| v1.79 | Honored token create --decimals + lottery-reset declined→exit≠0 + bounded NFT list paging + db.path honored everywhere | ✅ done |
| v1.80 | Removed phantom HOME data-dir wiring (dead app.Wire/GlobalApp ran on every CLI command) | ✅ done |

## Session Continuity

Last session: 2026-08-27 — v1.80 phantom HOME data-dir cleanup (dead
  app.Wire/GlobalApp wiring removed; no CLI command creates $HOME/.aurora).
  RIL graph at round 90.
Next: the token/NFT TUI in-memory sandboxes were verified as INTENTIONAL
  (in-code documented) — not a bug, no change; the deferred ISS-084 phantom
  on-chain blocks on rolled-back transactions remains parked per DEC-002
  (invasive to fix). Run a fresh deep-dive round if continuing; possible
  angles: `internal/app.Wire`/`App` is now production-dead (still tested) —
  decide whether to retire it; audit remaining CLI/API surfaces for the
  silent-input / silent-success classes closed in v1.79–v1.80.
