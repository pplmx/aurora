# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-11)

**Core value:** Complete, production-ready blockchain toolkit with comprehensive test coverage and operational tooling
**Current focus:** v1.83 config-duration & audit-durability sweep complete

## Current Position

Phase: v1.5+ Continuous Deep-Dive Loop
Plan: Incremental milestones tracked in the RIL graph and git history
Status: v1.24–v1.83 complete (key-bound VRF verification, truthful on-chain block_height, atomic token-create, all-or-nothing backups, rate-limit window seconds, voting missing-resource 4xx, NFT key-length + base64 keys, CLI token audit events, single CLI error line, lottery default count, consistent envelopes, committed-ops-never-reported-failed, restore same-file+WAL guards, dead app.Wire retired, numeric TOML durations as seconds, failed-audit-publish durable outbox, backup atomic metadata/restore)
Last activity: 2026-08-27 — v1.83 closed (deep-dive round after v1.82 left
  no active tasks):
  1. Bare numeric TOML durations now mean SECONDS across every config key —
     TASK-110 had fixed only api.rateLimit.window; http.timeout,
     http.rateLimit.window and oracle.scheduler.checkInterval were still read
     with raw viper.GetDuration, so `timeout = 60` silently produced a 60ns
     HTTP timeout (every oracle fetch failed), `window = 10` a 10ns limiter
     window (silently disabled), `checkInterval = 250` a 250ns scheduler
     ticker (busy-poll). One generalized config.DurationSeconds helper, all
     keys routed through it (TASK-118, ISS-110, CHG-114 / 69ec813).
  2. Failed post-commit audit publishes now heal instead of just reporting:
     v1.82 made a committed token op honest about audit-publish failure but
     SyncEventBus.Publish was fire-and-forget — a transient failure
     permanently dropped the audit record. Added the durable outbox
     (pending_events table + SaveIdempotent + AuditHandlerWithOutbox +
     OutboxDrainer with exponential backoff). API server drives it on a
     ticker, CLI on one DrainOnce before closing handles (TASK-119, ISS-111,
     CHG-115 / 7e88cba).
  3. Backup crash/failure-safety: Create's metadata.json is now written via
     .tmp + atomic rename (no truncated metadata over new .db files), and
     Restore copies the archive to a .tmp sibling before atomic-renaming over
     the live DB (a mid-copy disk-full can no longer leave a truncated .db at
     the live path) (TASK-120, ISS-112, CHG-116 / 9281277).
  RIL graph at round 95.

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
| v1.81 | Integrity & client-error sweep (key-bound VRF, on-chain block_height, atomic token create, all-or-nothing backups, rate-limit seconds, voting 4xx, NFT keys, CLI audit, single error line, envelopes) | ✅ done |
| v1.82 | Report-the-truth & dead-code sweep (committed token ops never reported failed, backup restore same-file + WAL-complete guards, retire dead app.Wire) | ✅ done |
| v1.83 | Config-duration & audit-durability sweep (numeric TOML durations as seconds everywhere, durable outbox heals failed audit publishes, backup atomic metadata/restore) | ✅ done |

## Session Continuity

Last session: 2026-08-27 — v1.83 config-duration & audit-durability sweep
  (numeric TOML durations as seconds everywhere, durable outbox heals failed
  audit publishes, backup atomic metadata/restore). RIL graph at round 95.
Next: the token/NFT TUI in-memory sandboxes were verified as INTENTIONAL
  (in-code documented) — not a bug, no change; the deferred ISS-084 phantom
  on-chain blocks on rolled-back transactions remains parked per DEC-002
  (invasive cross-DB atomicity redesign — token event trail is already
  post-commit, the park stands until a block-height reservation design
  lands); the same-second UUID event ordering was evaluated and parked by
  v1.82 decision (needs a schema/ID migration, ordering is approximate
  anyway since IDs are created pre-commit). Run a fresh deep-dive round if
  continuing; possible angles: a real-binary exerciser for `backup restore`
  under a live WAL-writing server, multi-DB restore partial-failure tests,
  DatabaseChecksums-through-restore coverage, symlink-alias restore guard,
  small rest of CLI/API surfaces for the silent-input / silent-success
  classes.
