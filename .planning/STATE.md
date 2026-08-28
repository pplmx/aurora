# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-11)

**Core value:** Complete, production-ready blockchain toolkit with comprehensive test coverage and operational tooling
**Current focus:** v1.85 UX / interactivity / usability polish complete

## Current Position

Phase: v1.5+ Continuous Deep-Dive Loop
Plan: Incremental milestones tracked in the RIL graph and git history
Status: v1.24–v1.85 complete (key-bound VRF verification, truthful on-chain block_height, atomic token-create, all-or-nothing backups, rate-limit window seconds, voting missing-resource 4xx, NFT key-length + base64 keys, CLI token audit events, single CLI error line, lottery default count, consistent envelopes, committed-ops-never-reported-failed, restore same-file+WAL guards, dead app.Wire retired, numeric TOML durations as seconds, failed-audit-publish durable outbox, backup atomic metadata/restore, voting wrong-length-key 400, duplicate roster candidates rejected, typable TUI forms, web API-failure surfacing, truthful CLI version)
Last activity: 2026-08-28 — v1.85 closed (round-97 UX deep-dive: three parallel
  audit agents over the TUI surfaces, CLI ergonomics and web frontend, then
  the top-three verified defects fixed):
  1. Lottery/token/nft TUI forms were UNTYPABLE — their Update loops never
     forwarded keypresses into the textinput models, so no participant, seed,
     address or amount could be typed; Enter only ever produced a
     required-field error. Wired per-view keypress forwarding + Tab/↑/↓ focus
     cycling (mirroring the oracle TUI), plus lottery count bounds validation
     (negative count panicked makeslice once typing worked; also participants-
     required / count>=1 / count<=participants), comma-separated participant
     parsing, and i18n for the create-form errors (TASK-123, ISS-115,
     CHG-118 / a12ff65).
  2. The web frontend silently swallowed API failures — with a missing/expired
     key, every list GET stored the {error, code} envelope into array state,
     rendering blank pages with zero feedback. Added shared apiFetch (checks
     res.ok, extracts the error envelope, shows a fixed top banner), kept list
     state as [], switched form errors to the real data.error field, and
     removed duplicate x-init="init()" on index/voting/oracle (Alpine auto-
     calls init(); it fired twice, duplicating dashboard activity) (TASK-124,
     ISS-116, CHG-119 / 42cf7a7).
  3. `aurora version` printed fabricated data — hardcoded 0.0.1 and a fake
     "1.26+" Go version (real toolchain: go1.27.0), ignoring -ldflags -X
     entirely. Moved to cmd/version.go with link-time-overridable
     cmd.Version/cmd.BuildTime and real runtime.Version(); startup log reads
     the same vars (TASK-125, ISS-117, CHG-120 / e99973a).
  The audits also recorded an 8-issue UX backlog (ISS-118..ISS-125) for
  future rounds: dead "?" help key, missing viewport scroll bindings,
  hardcoded CJK bypassing i18n, oracle confirm-selection invisible, no
  confirmation gate on destructive CLI ops, CLI help frozen English at
  package-init, unbundled Alpine from unpkg, no web refresh. RIL graph at
  round 97.

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
  contract hardening, the v1.78 client-error + oracle chain fixes,
  the v1.81 integrity/client-error sweep, the v1.82 report-the-truth
  sweep, the v1.83 config-duration/audit-durability sweep, the v1.84
  voting client-error sweep, and the v1.85 UX/interactivity/usability
  sweep above.

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
| v1.84 | Voting client-error classification sweep (wrong-length vote key → 400, duplicate roster candidates rejected; triage of has_voted/draft-window/NFT-zero-key to documented design or parked decisions) | ✅ done |
| v1.85 | UX / interactivity / usability polish (typable TUI forms, web API-failure surfacing, truthful CLI version; 8-issue UX backlog recorded) | ✅ done |

## Session Continuity

Last session: 2026-08-28 — v1.85 UX / interactivity / usability polish
  (typable TUI forms, web API-failure surfacing + banner, truthful CLI
  version; DEC-006 recorded that the user's polish directive overrides the
  low dx-category score in RIL priority ranking). RIL graph at round 97.
Next: convert the recorded UX backlog (ISS-118..ISS-125) into tasks across
  the next rounds; highest-priority candidates by severity are ISS-122 (no
  confirmation gate on destructive CLI ops — token/nft burn/transfer, oracle
  source delete, migrate down — while `backup restore --confirm` and
  `lottery reset --yes` already exist) and ISS-123 (CLI help resolved at
  package-init, so `--help` is frozen English regardless of locale). The
  deferred ISS-084 phantom on-chain blocks on rolled-back transactions
  remains parked per DEC-002 (cross-DB atomicity redesign; token event trail
  is already post-commit; reconfirmed by EV-044); NFT zero-key transfer is a
  product-semantics question parked for operator intent (DEC-005). Use
  `ril.py tasks --top 10` to load the converted backlog.
