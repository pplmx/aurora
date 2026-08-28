# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-11)

**Core value:** Complete, production-ready blockchain toolkit with comprehensive test coverage and operational tooling
**Current focus:** v1.87 API-consistency + latent-concurrency sweep (consistent unknown-resource 404s, listener-mutating event-bus handlers, once-guarded metrics registry)

## Current Position

Phase: v1.5+ Continuous Deep-Dive Loop
Plan: Incremental milestones tracked in the RIL graph and git history
Status: v1.24–v1.87 complete (key-bound VRF verification, truthful on-chain block_height, atomic token-create, all-or-nothing backups, rate-limit window seconds, voting missing-resource 4xx, NFT key-length + base64 keys, CLI token audit events, single CLI error line, lottery default count, consistent envelopes, committed-ops-never-reported-failed, restore same-file+WAL guards, dead app.Wire retired, numeric TOML durations as seconds, failed-audit-publish durable outbox, backup atomic metadata/restore, voting wrong-length-key 400, duplicate roster candidates rejected, typable TUI forms, web API-failure surfacing, truthful CLI version, scrollable viewport TUI views, --confirm gate on destructive CLI ops, localized --help, oracle confirm visible selection, "?" help screen, hardcoded CJK → i18n, vendored Alpine, web auto-refresh, cancellable scheduler fetches, backup traversal rejection, bounded rate-limiters, unknown-resource 404s, listener-mutating event-bus handlers, once-guarded metrics registry)
Last activity: 2026-08-28 — v1.85 closed (round-97 UX deep-dive: three parallel
  audit agents over the TUI surfaces, CLI ergonomics and web frontend, then
  the top-three verified defects fixed; round-98 drained the backlog:
  viewport scroll bindings, destructive-op confirm gates, CLI --help
  localization; round-99 drained the rest — oracle confirm visible
  selection, "?" help screen, hardcoded CJK → i18n, vendored Alpine
  (DEC-007), web auto-refresh):

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
  Round-98 drained the rest of the backlog the audits seeded:
  4. Lottery/token/nft history and list views were backed by a viewport but
     never forwarded key events to it — long content was clipped with no way
     to scroll. Handed key and mouse-wheel events to viewport.Update so its
     native up/k, down/j, pgup/pgdn, space, b/f, u/d bindings work
     (TASK-127, ISS-119, CHG-121 / bf8f8df).
  5. Destructive CLI ops had no confirmation gate: token burn, nft burn,
     oracle source delete and migrate down destroyed data silently while
     `backup restore --confirm` and `lottery reset --yes` already gated
     similar actions. Added shared addConfirmFlag/requireConfirm helpers and
     a -y/--confirm gate on all four commands — refused non-zero without it
     (TASK-126, ISS-122, CHG-122 / 3a1efb7).
  6. Cobra resolves Short/Long/flag help at package var-init, before
     i18n.DetectAndInit(), and the lazy GetTranslator default was hard-locked
     to "en" — so every --help screen froze in English regardless of LANG.
     The lazy default now adopts DetectLocale(); the lottery command's Long
     uses a dedicated lottery.long i18n key (TASK-128, ISS-123, CHG-123 /
     8c70eeb). Verified: LANG=zh --help renders zh, unset renders en.
  Round-99:
  7. Oracle confirm dialogs rendered no visible selection while Enter
     committed off an invisible menuIndex and the advertised y/n keys were
     dead. Added confirmChoiceView (▶-highlighted Yes/No, locale-aware) and
     wired y/Y commit / n/N cancel (TASK-129, ISS-121, CHG-124 / a195d7d).
  8. token/nft/oracle menu footers advertised "? for help" but the key was
     dead (only lottery had a help screen). Added a shared localized
     components.HelpView() and showHelp handling: ? opens, esc/? closes,
     other keys swallowed (TASK-130, ISS-118, CHG-125 / e1add94).
  9. Hardcoded CJK leaked into en-locale token/lottery TUI views; all moved
     behind token.tui.*/lottery.tui.* keys, and lottery's inline help screen
     was replaced by the shared component. Regression tests pin en+zh paths
     (TASK-131, ISS-120, CHG-126 / 7923dab).
  10. All 7 web pages loaded Alpine from the unpkg CDN with no integrity
      hash — third-party supply-chain/interception surface, and the UI
      broke offline. Vendored the byte-verified build to
      web/vendor/alpine.min.js; every page loads it locally (DEC-007;
      TASK-132, ISS-124, CHG-127 / 4157ced).
  11. Dashboard stats and Oracle sources/feed-health loaded once and went
      stale though the scheduler keeps recording. Added shared startPolling,
      refresh() on dashboard/oracle, 15s auto-poll and a ↻ Refresh header
      button (TASK-133, ISS-125, CHG-128 / ada5683).
  Round-100 (v1.86 infra robustness sweep — fresh parallel audit over API
  error-path, infra concurrency/scheduler, CLI exit-code/output):
  12. Oracle scheduler in-flight fetches were uninterruptible: Run only saw
      ctx.Done() between passes and the fetch used a background-context
      client with a 10s timeout, so SIGTERM stalled shutdown up to N×10s and
      srv.Close() closed the sqlite pool under the still-live fetch
      goroutine. Threaded ctx through Fetcher / FetchDataUseCase / Scheduler
      so an in-flight fetch aborts on cancel (TASK-134, ISS-127, CHG-129 /
      68175ed).
  13. Backup Verify/Restore joined metadata.Databases names onto backupPath
      unsanitized — a crafted archive naming '../../victim' could stat/hash/
      open arbitrary host .db files. Added validateDatabaseName applied in
      Verify and re-applied in Restore (TASK-135, ISS-126, CHG-130 /
      0932e41).
  14. Both rate limiters (REST middleware FixedWindowLimiter, fetcher
      RateLimiter) grew a map key per client/source and never deleted it
      (Reset unwired). Allow now sweeps expired keys past a 1024 threshold
      so memory stays proportional to keys active within a window (TASK-136,
      ISS-129, CHG-131 / e978268).
  RIL graph at round 100.

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
| v1.85 | UX / interactivity / usability polish (typable TUI forms, web API-failure surfacing, truthful CLI version, scrollable viewport TUI views, --confirm gate on destructive CLI ops, localized --help, oracle confirm visible selection, "?" help screen, hardcoded CJK → i18n, vendored Alpine, web auto-refresh; round-97 backlog fully drained) | ✅ done |
| v1.86 | Infra robustness deep-dive (cancellable scheduler fetches, backup path-traversal rejection, bounded rate-limiters) | ✅ done |
| v1.87 | API-consistency + latent-concurrency sweep (unknown oracle source / NFT id → 404 not 200[]/500, /oracle/query missing param → 400, sqlite GetLatestData nil contract, SyncEventBus snapshot-outside-lock, once-guarded MetricsRegistry) | ✅ done |

## Session Continuity

Last session: 2026-08-28 — v1.85 UX polish fully drained then v1.86 infra
  robustness sweep opened and closed (round 100): cancellable scheduler
  fetches (TASK-134), backup path-traversal rejection (TASK-135), bounded
  rate-limiters (TASK-136). DEC-006 recorded that the user's polish
  directive overrides the low dx-category score in RIL priority ranking.
  RIL graph at round 100 (501 nodes).
Round 101 (2026-08-29, v1.87) drained the round-100 audit backlog:
  ISS-128 (SyncEventBus self-deadlock — Publish now snapshots handlers under
  the read lock and runs them outside it, so a handler may subscribe/
  unsubscribe during publish; TASK-138, CHG-132), ISS-130 (unknown-resource
  statuses consistent — /oracle/query missing param → 400 and unknown source
  → 404, /oracle/latest unknown source → 404 instead of an unclassified 500
  (sqlite GetLatestData now returns (nil, nil), the interface contract),
  /nft/{id}/history unknown id → 404; TASK-137, CHG-133), ISS-131
  (MetricsRegistry lazy-init now sync.Once-guarded so concurrent callers
  cannot split registries; TASK-139, CHG-134). Stale HYP-003 (TUI keypress
  forwarding) closed — the round-97 CHG-118 fix landed before it was
  recorded. RIL graph at round 102 (509 nodes).
Round 102 (2026-08-29, user polish directive) refreshed the usability/docs
  surface: the web API-error banner is now cleared on any successful fetch
  (sticky-banner fix, TASK-141, CHG-135), and README.md now documents the
  operational CLI (backup/migrate/version), extended token/oracle/lottery
  subcommands, the -y/--confirm destructive-op gates, and the previously
  undocumented cmd/api Web server (TASK-142, CHG-136). CLI exit-code audit
  re-run: already clean (centralized error line + os.Exit(1)); no hardcoded
  CJK outside i18n. RIL graph at round 103 (513 nodes).
Next: the deferred ISS-084 phantom on-chain blocks on rolled-back
  transactions remains parked per DEC-002 (cross-DB atomicity redesign; token
  event trail is already post-commit; reconfirmed by EV-044); NFT zero-key
  transfer is a product-semantics question parked for operator intent
  (DEC-005). Candidate next dimensions: TUI edge-case auditing, deeper web
  JS review, or a fresh API contract sweep. Use `ril.py tasks --top 10` to
  load any converted backlog.
