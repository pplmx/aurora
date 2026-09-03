# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-11)

**Core value:** Complete, production-ready blockchain toolkit with comprehensive test coverage and operational tooling
**Current focus:** v1.94 — round 144 (web destructive-op + form-feedback deep-dive): NFT/Token burn forms destroyed assets with no confirmation (single click, Enter on an auto-filled form) while sibling destructive ops were gated everywhere else — oracle Delete confirms in-page and the CLI refused both burns without `--confirm`; both burns now confirm() first and abort when declined (TASK-257, ISS-253). The dashboard Recent Activity list collapsed to a false "No recent activity" on a transient all-endpoint poll failure (cards kept their values under the same TASK-151 policy, the list did not); list loaders now signal a failed cycle with null vs []-success and refresh() keeps the prior rows when every source fails (TASK-258, ISS-254). Three form-consistency gaps: NFT History used a separate historyId that mint advanced but a manually-pasted shared id silently desynced (now bound to the shared id, the token page's pattern); oracle Query Data zero-row success was indistinguishable from a no-op (now renders "No rows — run Fetch first", distinct from failure); token create decimals >127 hit Go's int8 decode ("invalid request") with no hint of the bound (now clamped to 0..127 and reflected back, oracle clampLimit contract) (TASK-259, ISS-255). All three closed with Node regression tests executing the shipped app.js. RIL graph at round 144 (895 nodes, 1037 edges). Then round 145 (backend + docs deep-dive, from the parallel Go/docs agents): lottery on-chain records were always serialized verified:false — the create use case wrote the chain block from JSON built before verification ran, so an auditor replaying the chain saw every draw unverified while DB/REST/CLI said true; verification now runs before the chain JSON so the immutable copy carries the real state (TASK-260, ISS-256). Token history default page size diverged (REST 20 vs CLI/service 50) for the identical no-flag query; REST default now matches 50 (TASK-261, ISS-257). lottery reset/db-info closed the process-global blockchain DB singleton (a sync.Once handle; closing without resetting poisoned the once — any later InitDB in the same process got a closed DB, a latent data-loss path); both now let the handle live to process exit like every other CLI command (TASK-262, ISS-258). Docs truthfulness: sample aurora.toml pointed at nonexistent `aurora --ui oracle` (now `oracle tui`), voting voter list was the one real subcommand absent from both quickstarts (added), and the [api.rateLimit]/[api.cors]/[oracle.scheduler] keys the server reads had no shipped sample (added, verified loadable); stale PROJECT.md claims (voting web page "In progress", metrics "Out of scope") refreshed to shipped (TASK-263/264, ISS-259/260). RIL graph at round 145 (910 nodes, 1057 edges). Then round 146 (token boundary deep-dive): Transfer/TransferFrom each read balance/allowance once before the atomic transaction and refused if the read looked short — but the read ran against a different snapshot than the transaction, so a concurrent top-up between the read and the atomic debit returned a spurious INSUFFICIENT_BALANCE/ALLOWANCE 400 even though the atomic path would have succeeded (ledger was always safe; only the pre-check could lie). Both fast-fail reads removed; the atomic TrySubtractBalance/TryDeductApproval primitives classify their own errors (handler still maps them to 400); tests switched to the errors.Is convention their own comment promised (TASK-265/ISS-261). Also recorded a decision DEC-018: allowance `expires_at` is write-only (schema+event carry it, nothing sets a non-zero value, TransferFrom never consults it) — expiry is a documented-but-unimplemented knob, aligned with DEC-002 allowance design; neither enforcing nor dropping it is a polish-round action (feature scope or schema/migration risk), deferred with rationale. RIL graph at round 146 (914 nodes, 1063 edges). Then the earlier round 141 token read-path correctness (TASK-254, ISS-250, from the round-141 app/API bug hunt): `GET /token/balance` / `/allowance` / `/history` returned `200 {"amount":"0"}` / `200 []` for a nonexistent token_id while `/token/info` 404'd the same id — a client typo or pre-create existence probe was indistinguishable from "no activity". The read paths now share the same `requireToken` existence guard every mutator (Mint/Burn/Transfer/TransferFrom/Approve) already used, so unknown ids report `404 TOKEN_NOT_FOUND` across REST/CLI/web (service + REST smoke regression tests). The bug-hunt's other three findings were triaged as intended, non-defect behavior and recorded as decisions (DEC-013: voting draft/open-window acceptance + `session start` non-enforcement is the DOCUMENTED window-gated lifecycle, DEC-004/DEC-005 — explicitly do-not-re-litigate; DEC-014: the allowance family — Approve/Increase/DecreaseAllowance — uniformly writes no on-chain block while the balance family does, a coherent DEC-002 design split, not an oversight; DEC-015: the voting window end is inclusive, `now > EndTime` rejects strictly-after). The five targeted risk patterns (int overflow, amount math, swallowed errors, pagination, TUI untypeable fields) all verified clean. Then round 150 (web accessibility polish: label/id+for association across 81 controls, focus-visible ring, aria-live result regions — all regression-tested). Then round 147–148 (surface-parity + correctness: lottery TUI create now enforces the shared domain validators the CLI/API use — duplicate participants / short seed / over-cap winner count were previously accepted and stored, then failed `record.Validate()` on re-import; `runLottery` propagates `GenerateKeyPair`/`VRFProve` errors instead of drawing on nil outputs (a discarded crypto error recorded a deterministic, unverifiable draw). REST lottery create resolves an omitted `winner_count` to the configured `lottery.defaultCount` (config.LotteryDefaultCount injected into the handler) — a non-3 configured default now draws the same count via CLI and API instead of API-hardcoded 3. lottery verify (usecase + CLI) rejects a stored record with an empty winners slice instead of the vacuously-true valid report; REST/web inherit via the use case. `web/lottery` history keeps prior rows on a transient refresh failure (keep-prior policy, TASK-151) — previously wiped the rendered list. k8s.yml default Job command uses absolute `/aurora` (k8s `command` replaces the image ENTRYPOINT, so `./aurora` exec'd against WORKDIR /app and failed every run). Docs: README/AGENTS flag-gap fill (oracle -t, nft --image/-t, token -d, voting -m/-d, lottery stats/verify/export/import/db-info), orphaned scripts/ deleted, justfile clean removes ./aurora.) Then historical (v1.93 — rounds 132–137 config-truthfulness + surface-parity polish (inert config knobs removed: `db.type` and `lottery.defaultSeedPrefix` were set-but-never-read; sample `aurora.toml` no longer ships dead `[[oracle.sources]]` blocks and now explains `[http]`/`[http.rateLimit]` govern the outbound fetcher, not the API server; `cmd/api` releases DB/event-store handles on the fatal bind-failure path; `oracle source add` gained `--method`/`--path` and `source list` prints Method/Path/Interval; oracle TUI add form now sets method/path/interval with client-side interval validation). Round 138: oracle TUI add form no longer retains stale method/path/interval across re-entry (recreates all six inputs); oracle TUI source detail shows method/path/interval for CLI `source list` parity. Round 139 (oracle deep-dive): oracle TUI method/path/interval fields were UNTYPABLE (Update routing only reached the first three inputs — focus showed the cursor but typing was dropped, so the v1.93 parity feature was dead in the TUI); source `interval` gained a 30-day upper bound (values ≥ ~9.22e9 overflowed the scheduler's Duration math into a negative "always-due" interval = fetch storm); deleting an unknown oracle source now reports not-found (REST 404 / CLI non-zero) instead of a silent `200 {"status":"deleted"}`. Round 140 (web UX deep-dive): dashboard integrity/oracle cards keep the last good value across transient poll failures (TASK-151 keep-prior; was blanking "OK"/"N OK" to "?" on every 15s hiccup — new Node regression test); token `transfer`/`transferFrom` advance the shared owner (+ Transfer From) to the recipient and refresh balance (the mint/NFT advance contract); `voting.html` Enter in the Session ID field no longer implicitly fires "Start Session" (must be an explicit click per DEC-004). Round 141 (data/extraction deep-dive): oracle fetcher `extractByPath` now returns success separately from the value — a path resolving to a legitimate empty string is recorded (was aborting a healthy 200 as "invalid source" + scheduler backoff), and a non-scalar object/array/null leaf fails closed with ErrInvalidSource instead of persisting Go map/array garbage as the on-chain Value; blockchain genesis is seeded only into an empty blocks table (was re-inserting height 0 and logging a false "Failed to insert genesis block" on every boot) via extracted `seedGenesisIfEmpty` + regression test. Round 142 (REST fetch hardening): oracle source request headers reject CR/LF at apply time (`INVALID_SOURCE`; Go JSON `"\r\n"` decodes to literal bytes, and net/http refused to write such a request anyway — the old "JSON cannot carry CRLF" comment was false); a rate-limited oracle fetch now maps to `429 RATE_LIMITED` instead of a masked `500`; and the oracle Fetch handler shares ONE fetcher across requests (per-request construction gave each call a fresh RateLimiter, making `http.rateLimit` a silent no-op at the REST surface — the scheduler already shared its fetcher). Round 143 (lifecycle deep-dive): the oracle scheduler reconciles each pass against the repository's live source list, pruning `lastFetch`/`failStreak`/`nextAttempt`/`stats` for deleted sources — `/oracle/health` and `/metrics/oracle` no longer advertise deleted sources forever and the maps stay bounded over create/delete cycles (a recreated id no longer inherits stale failure/backoff state); `voting.html` gained an in-page `↻ Retry` that re-runs the candidates/sessions loaders, so a transient load failure no longer permanently disables Cast Vote (the page has no polling; previously only a full reload could recover it) — regression test executes the shipped JS in Node. Round 144–146 (shared-ledger correctness deep-dive): blockchain multi-process height collisions — the documented API-server + CLI lottery/oracle norm shares one blocks DB, each process reserves heights from its own in-memory chain, and the old `INSERT OR REPLACE` persist silently overwrote the other process's committed block (both booted genesis → both wrote "height 1" → one payload lost, chains diverged, next boot failed VerifyIntegrity). Heights are now DB-authoritative: persist is a plain INSERT whose height UNIQUE conflict maps to ErrHeightConflict (matched by extended error code, not message text); appendBlock re-syncs from the shared DB before reserving and, on a lost height race, drops its candidate and retries at the true next free height (bounded); a phantom-tail rebuild covers a prior non-conflict persist failure followed by a peer's own commit at that height. Fix committed as a32c085 + code-review follow-ups 439ce06 (seam-rebuild HIGH, error-code matching, exact 5-retry bound, no persist hooks on CREATE TABLE failure), 9 new multi-process tests. Round 145: AsyncEventBus.Publish had a check-then-act race with Close — a publish that passed the closed test just before close drained-and-exited could return nil for an event the consumer never delivered. The closed-check + send now serialize on a mutex, making "every successful publish is delivered" an exact invariant of the bus (ISS-243; regression test stresses a burst + a background streamer against a close).

## Current Position

Phase: v1.5+ Continuous Deep-Dive Loop
Plan: Incremental milestones tracked in the RIL graph and git history
Last activity: 2026-09-03 — round 150 (cmd/api flag-surface + deep-dive open).
  The single active backlog item from round 148 shipped: `aurora-api` now
  handles `--help`/`-h` (usage, exit 0), `-v`/`--version` (build identity,
  exit 0) and rejects unknown flags / stray positionals before config
  loading or any listener (TASK-267, ISS-263; CHG-264 4f9174d). The version
  vars are package-main link-time targets injected via a new `just api`
  recipe (`-X main.Version/BuildTime` — the root package of a binary links
  as `main`, not by import path), mirroring the CLI's ldflags; `--version`
  prints the same i18n-labeled shape as `aurora version`. A go-reviewer
  pass (no CRITICAL/HIGH) closed as a follow-up (CHG-265 5706ec0): NArg is
  checked before the `--version` branch so `--version foo` errors instead
  of silently dropping the arg, the version test asserts via the same i18n
  keys the function renders (passes under `LANG=zh_CN.UTF-8`, review-M2),
  the `-v` alias registers its default from `*showVersion`, and five stdlib
  edge cases are pinned (`--version=false`, bare `--`, `--version foo`,
  lone `-`, `-=x`). README documents the flag surface; AGENTS.md lists the
  `just api` recipe. RIL graph at round 150 close: 933 nodes, 1088 edges,
  no active tasks; only ISS-084 stays parked (DEC-016). Deep-dive continues
  in the same session (round 151).

Last activity: 2026-09-02 — round 147/148 web + backend deep-dive (web audit
  agent + backend audit agent + independent observation). Five closed
  findings:

  1. Repo hygiene: an 18 MB compiled `api` binary (a `go build ./cmd/api`
     artifact) was accidentally re-tracked by the round-133 RIL state commit
     `20ce60e` after `e02302e` removed it; `.gitignore` covered `/aurora` and
     `/aurora-*` but not `/api`. Untracked, working copy deleted, `/api`
     gitignore rule added; the orphaned `.commit-msg.txt` (nothing reads it)
     deleted alongside (TASK-266, ISS-262; CHG-260).
  2. Config test isolation: `internal/config` TestMain set `$HOME` to the
     shared `os.TempDir()`, and viper's `SetConfigName("aurora")` matches a
     bare `aurora` file anywhere in the HOME search path — a parallel
     process's `/tmp/aurora` ELF broke every config test ("invalid character
     at start of key"). The isolation home is now a unique per-run
     `os.MkdirTemp` subdir (TASK-268, ISS-264; CHG-261).
  3. Web transfers that MOVE value/ownership were single-click while the
     sibling burns on the same pages confirm(): token transfer, token
     transfer_from (spend allowance) and NFT transfer now confirm() their
     amount/recipient and abort when declined (TASK-269, ISS-265; CHG-262).
  4. The NFT transfer form's From field desynced after success: owner
     advanced but this.from stayed stale, so a second transfer posted the
     old key; now both advance to the recipient like the token page's
     app.js:898 (TASK-270, ISS-266; CHG-262).
  5. Backend write endpoints stored unbounded free-text while token/lottery
     capped theirs: voting (candidate party/program + session description),
     NFT mint (description/image_url/token_uri) and oracle source
     (name/type/method/path/headers) now carry domain-edge length caps
     (voting gained validator.go, NFT.Validate extended, oracle AddSource
     bounded); the two unbounded read endpoints (GET /nft/{id}/history →
     SQL LIMIT 1000, GET /lottery/history → ?limit/?offset default 50/max
     100) are bounded like their siblings (TASK-271, ISS-267; CHG-263).

  Three standalone tasks also recorded as backlog: TASK-267/ISS-263
  (cmd/api ignores --help/--version — `aurora-api --help` starts the server;
  priority 0.42, below the 3.0 execute threshold) and the pre-existing
  ISS-084 phantom-block tradeoff stays deferred (DEC-016). Full
  `go test -race ./...` green at close. RIL graph at round 148 (930 nodes,
  1085 edges).
Status: v1.24–v1.88 complete (key-bound VRF verification, truthful on-chain block_height, atomic token-create, all-or-nothing backups, rate-limit window seconds, voting missing-resource 4xx, NFT key-length + base64 keys, CLI token audit events, single CLI error line, lottery default count, consistent envelopes, committed-ops-never-reported-failed, restore same-file+WAL guards, dead app.Wire retired, numeric TOML durations as seconds, failed-audit-publish durable outbox, backup atomic metadata/restore, voting wrong-length-key 400, duplicate roster candidates rejected, typable TUI forms, web API-failure surfacing, truthful CLI version, scrollable viewport TUI views, --confirm gate on destructive CLI ops, localized --help, oracle confirm visible selection, "?" help screen, hardcoded CJK → i18n, vendored Alpine, web auto-refresh, cancellable scheduler fetches, backup traversal rejection, bounded rate-limiters, unknown-resource 404s, listener-mutating event-bus handlers, once-guarded metrics registry, web error-surfacing consistency, oracle/dashboard polling polish, burn-amount isolation, NFT mint context advance; round 139: oracle TUI 6-field typability, bounded oracle interval, delete-unknown-source 404/CLI error)
Last activity: 2026-09-01 — round 139 oracle input/UX deep-dive (TASK-231/232/233,
  ISS-229/230/231):

  1. The oracle TUI add-source form's method/path/interval fields accepted no
     typed input: the Update loop routed keys only through name/url/type, so
     `updateInputFocus` focus on fields 3..5 showed the cursor but keys were
     dropped and handleAddSource silently read defaults. All six fields now
     receive Update; regression test types POST / bitcoin.usd / 300 into the
     three previously-dead fields (TASK-231, ISS-229).
  2. A source `interval` above ~9.22e9 s overflowed the scheduler's
     `time.Duration(src.Interval) * time.Second` to a negative duration treated
     as "always due" (fetch storm). AddSource now rejects intervals over a
     documented 30-day `MaxSourceIntervalSeconds` at the shared domain edge so
     REST/CLI/web all inherit it (TASK-232, ISS-230).
  3. Deleting an unknown oracle source reported success everywhere: the use
     case passed delete straight to the repo (no rows-affected check), so REST
     returned `200 {"status":"deleted"}` and the CLI printed success while
     enable/disable/latest 404'd. DeleteSource (service + sqlite/inmem repos +
     use case) now reports not-found; handler 404s and the CLI exits non-zero
     with "source not found" (TASK-233, ISS-231).

Last activity: 2026-09-01 — v1.93 closed (rounds 132–137 config-truthfulness
  + surface-parity deep-dive: three config knobs proved set-but-never-read
  (`db.type`, `lottery.defaultSeedPrefix`) or dead-by-design (sample
  `[[oracle.sources]]` blocks the binary never reads) and were removed /
  re-documented; `[http]`/`[http.rateLimit]` scope clarified in the sample
  config; `cmd/api` gained handle release on the fatal bind-failure path;
  `oracle source add` + TUI add form reached API parity with `--method`/
  `--path` and interval; oracle scheduler backoff and web/CLI/TUI/API field
  parity verified):

  1. Inert config knobs removed — `db.type` (DBConfig.Type + default + toml
     line + two test assertions) and `lottery.defaultSeedPrefix` (viper
     default + toml line + config-doc claim) were written but never read;
     the lottery seed is always caller-supplied and SQLite is the only
     backend. `config/aurora.toml` no longer ships two `[[oracle.sources]]`
     blocks that nothing reads, with a truthful comment pointing at
     `aurora oracle source add` / REST / TUI (TASK-223, TASK-226; ISS-219,
     ISS-220, ISS-224).

  2. Oracle capabilities reached parity: CLI `source add` gained
     `-m/--method` and `-p/--path` (previously the API/TUI/web-only fields),
     `source list` prints Method/Path/Interval, and the oracle TUI add form
     now sets method/path/interval with client-side interval validation
     (6-field focus cycle; empty -> default 60) (TASK-225, TASK-227;
     ISS-221, ISS-222, ISS-225).

  3. Lifecycle + docs: `cmd/api` calls the idempotent `srv.Close()` before
     the fatal ListenAndServe exit so WAL/event-store handles are released;
     `[http]`/`[http.rateLimit]` documented as outbound-fetcher scope (DEC-008
     respected — no shipped keys renamed) (TASK-224, TASK-225; ISS-221,
     ISS-223).

  RIL graph at round 131 (778 nodes, 917 edges).

Last activity: 2026-09-01 — round 138 oracle TUI deep-dive (TASK-228/229,
  ISS-226/227):

  1. `initAddSource` only recreated name/url/type, so method/path/interval
     retained the previous attempt's text after a successful add — the
     token/nft/lottery TUIs clear every field on form entry. Now recreates all
     six inputs (TASK-228, ISS-226).
  2. The CLI `source list` prints Method/Path/Interval (TASK-225) but the TUI
     source detail hid them, and `loadSources` dropped the fields when
     copying. The detail view now shows method (default GET), JSON path and
     interval (default 60s), and the copy carries the fields (TASK-229,
     ISS-227).

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
| v1.88 | Web error-surfacing + interactivity polish (apiFetch banner consistency incl. blockchain page, oracle inline errors replace alert()/silent table, dead res.ok guards removed, dashboard stat isolation, oracle + dashboard 15s-poll flicker fixes, token burn-amount isolation, NFT mint context advance) | ✅ done |
| v1.89 | Web + TUI detail-polish sweep (web Create Token owner fix, shared-context advances, AGENTS.md flag sync + JS syntax gate, TUI q-key typability, decimals honored, NFT key lengths, cursor bounds) | ✅ done |
| v1.90 | Correctness + ergonomics sweep (X-API-Key on all web writes + Node-exec gate, two-sided token history, typable TUI forms, GetSession 404→500 split, hardened DSN, block timestamp reload, strict decodeJSON, NFT list wiring, i18n leak sweep) | ✅ done |
| v1.91 | Residual-backlog polish + deep-dive (oracle TUI bounded scrollable viewport, per-action web submit busy state, oracle TUI query limit clamp, help screen scroll-key docs + i18n parity gate, oracle TUI fetch on-chain recorder wiring) | ✅ done |

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
  CJK outside i18n. Round 103 (round 104 in graph) added a CHANGELOG v1.x
  milestone line for v1.64–v1.87 (was stale at 0.0.1) (TASK-143, CHG-137).
  TUI numeric-input validation and token/oracle web flows re-checked: all
  friendly-error guarded, no panics. Round 105 (round 106 in graph) added 10
  i18n keys that code referenced but both catalogs lacked — T(key) was
  rendering raw keys to users in the oracle TUI placeholders/labels, the NFT
  TUI block-height label, and the oracle error path (TASK-145, CHG-139,
  9ad9e62); catalog parity re-verified at 306/306 with zero missing keys.
  RIL graph at round 106 (519 nodes).
Rounds 107-108 (2026-08-29, this session, v1.88) — web error-surfacing +
  interactivity polish per the user directive (细节优化/bug修复/打磨/交互性/
  实用性/使用性/易用性/文档). No active tasks remained, so a fresh deep-dive
  over the web frontend seeded six fixes in two commits:
  - Round 107 (6f1cd34): error-surfacing consistency (TASK-146, ISS-132/133/
    134, CHG-140) — blockchainApp.verify now goes through apiFetch so a
    down/unauthorized API raises the shared red banner like every other page
    (was the only raw-fetch holdout); oracle setEnabled/deleteSource dropped
    blocking alert() dialogs for inline sourcesError text and query() gained
    an inline queryError line instead of silently clearing the results table;
    six dead if(!res.ok) guards in votingApp removed (apiFetch already throws).
    Dashboard stat isolation (TASK-147, ISS-135, CHG-141) — split loadVoting's
    shared Promise.all into independent loadCandidatesStats/loadSessionsStats
    so one endpoint failing blanks only its own card. Oracle 15s-poll flicker
    (TASK-148, ISS-136, CHG-142) — Loading placeholders now require an empty
    dataset, so the Data Sources + Feed Health tables stay visible during a
    background refresh. RIL graph at round 107 (530 nodes).
  - Round 108 (bfb525c): burn-amount isolation (TASK-149, ISS-137, CHG-143) —
    token Burn gets its own burnAmount field so a Transfer quantity can never
    leak into a destroy. NFT mint context advance (TASK-150, ISS-138, CHG-144)
    — mint() fills this.id/this.owner from the response so the
    inspect/transfer/burn next step needs no manual copy. Dashboard 15s-poll
    flicker (TASK-151, ISS-139, CHG-145) — refresh() no longer blank-resets
    the stats grid or clears Recent Activity each poll; loaders overwrite
    their own card and the activity list swaps in atomically. Milestone doc
    v1.88-ROADMAP.md added; RIL graph at round 108 (539 nodes).
Rounds 109-113 (2026-08-29, this session, v1.89) — continued the detail-
  polish directive. The web Create Token bug was real and user-breaking;
  the TUI q-key defect made the letter q untypeable in every form.
  Round 109 (09a4216): token create sent {name,symbol,total_supply} with no
  owner, so EVERY web create 400'd PUBLIC_KEY_REQUIRED (the CLI generates a
  keypair; the web form had no field at all). Added a required Owner (public
  key) input mirroring NFT mint's Creator field, and advanced shared context
  after token/voting/lottery creates (TASK-155/156, ISS-147/148, CHG-149/
  150). Round 110 (b080257): token mint's auto getBalance() ran against the
  shared owner (blank unless the Balance section was used), surfacing a
  confusing error right after a successful mint — now advances owner=to +
  token_id before the refresh; the create form's owner got its own createOwner
  field so a create key never leaks into the Balance/History context (TASK-
  157/158, ISS-150/151, CHG-151/152). Round 111 (a8b7271): AGENTS.md
  documented flags that don't exist (nft get --id, voting session start -i)
  — synced to real spellings; added TestWebUIJS_SyntaxValid (node --check,
  skip without node) so a broken web/js/app.js can no longer pass green
  (TASK-159/160, ISS-152/153, CHG-153/154). Rounds 112-113 swept the TUI
  surfaces with a fresh audit: q is now typable in every form (was consumed
  as quit before the textinput), ctrl+c is the hard quit, the token TUI
  honors --decimals (validated but never assigned -> every create was 8
  decimals), NFT mint rejects wrong-length owner keys + transfer success
  refreshes the result NFT (was "Not found"), oracle sources cursor bound
  (down-key off-by-one lost it on empty lists), lottery count parse failure
  is a visible error not a silent 3, and [T]/[D] hotkeys accept uppercase
  (TASK-161..166, ISS-154..159, CHG-155..160). RIL graph at round 113
  (589 nodes).
Round 114 closed v1.90 and recorded two below-threshold residuals explicitly
  (oracle viewport TASK-176, web in-flight busy TASK-177). Round 115 (this
  session, v1.91) drained exactly that backlog plus one deep-dive finding:
  oracle result views (queryResult/fetchResult) now scroll via a bounding
  viewport and the sources menu gets a cursor-following window sized from
  WindowSizeMsg (TASK-176, ISS-174); the five web write apps' submit buttons
  bind per-action :disabled via a shared withBusy wrapper so double-click can
  no longer create duplicate records, pinned by a Node harness test that
  executes the shipped app.js (TASK-177, ISS-175); and oracle TUI handleQuery
  now clamps the query limit to [1,100] via clampQueryLimitValue, closing the
  one path (CLI + API already capped) where an inflated limit could drive an
  unbounded DB scan (TASK-178). RIL graph at round 115 (631 nodes).
Next: the deferred ISS-084 phantom on-chain blocks on rolled-back
  transactions remains parked per DEC-002 (cross-DB atomicity redesign; token
  event trail is already post-commit; reconfirmed by EV-044); NFT zero-key
  transfer is a product-semantics question parked for operator intent
  (DEC-005). Use `ril.py tasks --top 10` to load any converted backlog.
