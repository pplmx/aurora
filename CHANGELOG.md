# Changelog

All notable changes to this project will be documented in this file.

The v1.x line is milestone-tracked in `.planning/milestones/` and `.planning/STATE.md`
(release `Version` is injected at build time via `-ldflags -X cmd.Version=`). The
entries below summarise v1.64–v1.93; earlier v1.x milestones (v1.0–v1.63) are
documented in their per-milestone ROADMAP files.

## [v1.94] - 2026-09-02

### Fixed

- **The lottery TUI created draws the CLI/API reject**: its create form only
  enforced list-nonempty, count≥1, count≤len and a non-empty seed, so
  duplicate participant names, an over-short/over-long seed, a winner count
  above the 100 cap, or a malformed participant name slipped through — and
  the stored draw then failed `record.Validate()` on re-import, so the rest
  of the platform treated it as corrupt. `handleCreate` now runs the shared
  domain validators (`ValidateParticipants`/`ValidateSeed`/
  `ValidateWinnerCount`) exactly as the CLI/API do, with each sentinel mapped
  to a localized message (TASK-246, ISS-248). `runLottery` likewise
  propagates `GenerateKeyPair`/`VRFProve` failures instead of drawing on nil
  outputs — a discarded crypto error used to record a deterministic,
  unverifiable "successful" draw with empty VRF fields.
- **rest lottery create froze the default winner count at 3**: a config
  `lottery.defaultCount = 4` drew 4 winners through the CLI but 3 via the
  API, because the endpoint resolved an omitted `winner_count` to the
  hardcoded `DefaultWinnerCount`. It now resolves `config.LotteryDefaultCount()`
  (injected into the handler at server wiring), matching the CLI's `-c`
  absence exactly (TASK-247).
- **lottery verify reported a corrupt record as valid**: a stored draw with
  an empty winners slice (partial/corrupt write — a create always records ≥1
  winner) passed verification, because `SelectWinners(output, roster, 0)`
  returns an empty set that vacuously matches. Both the use case and the CLI
  verify path now reject "record has no winners" (TASK-249, ISS-249; the
  REST/web surfaces inherit the guard through the use case).
- **`web/lottery` history blanked already-rendered draws on a transient
  refresh failure**: `createLottery`'s follow-up `loadHistory` reload wiped
  `history=[]` on a 20s timeout / API blip, hiding a known-good list while
  the API recovered. It now keeps prior rows on failure (the dashboard/
  oracle keep-prior-rows policy, TASK-151), still flagging the failure via
  `historyFailed` so a first-load failure is never misread as an empty system
  (TASK-250, ISS-246).
- **The k8s default Job could not start**: the container `command` was
  `["./aurora", ...]`, but Kubernetes' `command` replaces the image
  ENTRYPOINT, so it exec'd the relative path against WORKDIR `/app` where no
  such file exists ("executable file not found" on every run — compose was
  unaffected because `docker compose run` appends to the ENTRYPOINT). The
  command now uses the absolute `/aurora` path.

### Added

- **The web UI forms are now keyboard/screen-reader accessible**: every
  `<label>` across the five form pages (`lottery`, `oracle`, `nft`,
  `voting`, `token` — 81 controls) gained an `id` + `for` pair. Previously
  the labels were bare siblings of their inputs, so no programmatic
  association existed: screen readers announced every field as
  "edit text, blank", and clicking a label did not focus its field
  (WCAG 2.1 1.3.1 / 4.1.2). The voting "Candidate Roster" group is now a
  `<fieldset>/<legend>` (its checkboxes already wrap their labels).
  `TestWebUIFormLabelsAssociated` reads the shipped pages and fails if any
  label lacks a resolving `for=` or a wrapped control, so a future page
  cannot silently reintroduce unlabeled inputs (TASK-251, ISS-247).
- **Keyboard focus is visible again on every web form**: `style.css` had
  cleared the browser's default focus outline on `input:focus` and replaced
  it with only a subtle border-color change, so a keyboard/AT user tabbing
  through a form saw no focus marker at all (WCAG 2.4.7). The border change
  now applies on `:focus` (mouse clicks stay ring-free) and `:focus-visible`
  draws a strong accent ring on keyboard focus, extended to `select` and
  `textarea`. `TestWebUIFocusVisible` guards that the shipped CSS never
  re-suppresses the outline (TASK-252, ISS-248).

### Docs

- README/AGENTS quickstart now spell the CLI flags that exist but were
  undocumented: `oracle source add -t/--type`, `nft mint --image` /
  `-t/--token-uri`, `token create -d/--decimals`, `voting candidate add
  -m/--program`, `voting session create -d/--description`, and the lottery
  `stats` / `verify` / `export` / `import` / `db-info` commands (TASK-248).
  Every added flag was cross-checked against `cmd/aurora/cmd` so the docs
  never invent a flag.
- Deleted the orphaned `scripts/go_fmt_check.sh` and `scripts/go_imports_check.sh`
  — nothing referenced them (the `justfile check` target and
  `.pre-commit-config.yaml` run gofmt/goimports inline).
- `justfile clean` now also removes `./aurora` (the `build-current`/`run`
  binary), not just the cross-build `aurora-<os>-<arch>` artifacts.

## [v1.93] - 2026-09-01

### Fixed

- **Config knobs that were set-but-never-read**: `db.type` was written
  (`viper.SetDefault` + `DBConfig.Type`) but no code reads it — SQLite is the
  only backend, so an operator setting `db.type=<other>` silently got SQLite.
  `lottery.defaultSeedPrefix` was likewise written but never consumed: the
  lottery seed is never auto-generated, it is always supplied by the caller
  (CLI `-s/--seed`, API, web form). Both were removed (field, default, sample
  toml key, and the test assertions / config-doc claims that only validated
  the dead plumbing) (TASK-223, TASK-226; ISS-219, ISS-224).
- **The sample config shipped a dead `[[oracle.sources]]` table**: the binary
  never reads one, so editing those two blocks looked like it configured the
  scheduler but had no effect. Replaced with a truthful comment: sources live
  in SQLite and are managed at runtime via `aurora oracle source add` / REST /
  TUI. The `[http]` / `[http.rateLimit]` sections were also unlabelled — they
  govern the outbound oracle fetcher (per-source pacing of reads), not the API
  server's inbound traffic (which uses `[api.rateLimit]`, off by default);
  documented in place, no shipped keys renamed (DEC-008) (TASK-225; ISS-220,
  ISS-223).
- **`cmd/api` tore down over an open DB on a fatal bind failure**:
  `logger.Fatal` in the `ListenAndServe` failure goroutine exits without main's
  deferred `srv.Close()`, leaving the outbox/scheduler goroutines ended over an
  open SQLite handle and the WAL uncleaned (port busy etc.). The idempotent
  `srv.Close()` now runs before the fatal log (TASK-224; ISS-221).
- **The oracle TUI add-source form retained stale values on re-entry**:
  re-opening the form recreated only the first three fields, so a previous
  attempt's `method`/`path`/`interval` values stayed on screen — the
  token/nft/lottery TUIs clear every field on form entry, so the oracle form
  now recreates all six inputs (TASK-228; ISS-226).
- **The oracle TUI add-source form's method/path/interval fields accepted no
  typed input**: the round-137/138 parity added the three fields with focus
  support, but the `Update` loop only forwarded keypresses to name/url/type,
  so focus on the new fields (3..5) showed the cursor yet typing did nothing
  and `handleAddSource` silently defaulted method→GET / path→empty /
  interval→60. All six fields now receive `Update` and a regression test types
  into each (TASK-231; ISS-229).
- **An oracle source `interval` had no upper bound**: a value ≥ ~9.22e9
  overflowed the scheduler's `time.Duration` arithmetic to a negative interval
  that was treated as "always due", turning that source into a fetch-storm on
  every check tick. `AddSource` now rejects intervals above a documented
  30-day cap (`MaxSourceIntervalSeconds`) at the shared domain boundary — the
  REST/CLI/web surfaces all inherit it (TASK-232; ISS-230).
- **Deleting a non-existent oracle source reported success everywhere**: the
  domain use case passed delete-through to the repo (`DELETE ... WHERE id = ?`
  with no rows-affected check), so `DELETE /oracle/sources/<unknown>` returned
  `200 {"status":"deleted"}` and the CLI printed success — while every sibling
  op (enable/disable/latest) returns 404. `DeleteSource` now reports not-found
  (service + repo + use case), the REST handler returns 404, and the CLI exits
  non-zero with "source not found" (TASK-233; ISS-231).
- **Dashboard integrity/oracle cards regressed to "?" on any transient poll
  failure**: `loadBlockchain`/`loadOracleHealth` blanked their cards
  unconditionally on every 15s poll hiccup, while every sibling loader
  (lotteries/candidates/sessions) keeps a previously-seen value per the TASK-151
  keep-prior policy. A brief API blip no longer hides a known-good "OK"/"N OK" —
  only a card that never loaded marks itself "?" (TASK-234; ISS-232, regression
  test runs the shipped JS in Node).
- **Token web transfers left the shared owner on the sender**: after a
  successful `transfer`/`transferFrom` the Balance/History/Allowance/Burn forms
  still keyed off the sender, so the next "Get Balance" showed the drained
  sender balance and read as a failed transfer. Both now advance the shared
  `owner` to the recipient (and Transfer's From field) and refresh the balance —
  the mint/NFT advance-the-context contract (TASK-235; ISS-233).
- **`voting.html` Session Controls: Enter in the Session ID field fired "Start
  Session"**: the form's submit handler is `startSession`, so an operator who
  typed a session ID and pressed Enter to finish the line could silently
  reactivate an ended election (the backend deliberately permits end→active
  reopen, but it must be an explicit click, not a typing accident). Enter in
  that input is now inert; Start/End are explicit button actions (TASK-236;
  ISS-234).
- **The oracle fetcher mishandled two extraction cases**: a path resolving to a
  legitimately empty string (upstream returns `{"price": ""}`) aborted a
  healthy 200 fetch as `ErrInvalidSource` — the scheduler backoff and the
  operator saw "invalid source" for a valid upstream. And a path resolving to a
  nested object/array stored Go representation garbage (`map[usd:64000]`) as
  the on-chain `Value`. `extractByPath` now returns success separately from the
  value: a real empty-string leaf is recorded as a data point, while a
  non-scalar leaf fails closed with `ErrInvalidSource` (misconfigured path)
  instead of persisting garbage (TASK-237; ISS-235).
- **Every boot after the first logged a false "Failed to insert genesis
  block"**: the reload skipped the persisted height-0 row into memory, so the
  chain was "genesis-only" again and the old `len(Blocks) <= 1` guard re-ran a
  plain `INSERT` of height 0 on a DB that already held it — PRIMARY KEY
  conflict on every healthy restart (until a block was eventually appended).
  Genesis is now seeded only into an empty blocks table (`seedGenesisIfEmpty`,
  keyed on any-persisted-row) with a regression test (TASK-238; ISS-236).
- **Oracle source request headers accepted newlines**: the header parser set
  names/values verbatim and claimed "JSON cannot carry a raw CRLF
  untransformed" — but Go JSON escape sequences (`"\r\n"`) decode to literal
  CR/LF bytes, an operator-held header-injection vector toward the upstream
  provider (modern net/http refuses to write such a request, surfacing as a
  confusing runtime error). Header CR/LF is now rejected at apply time with
  `INVALID_SOURCE` (TASK-239; ISS-237).
- **A rate-limited oracle fetch was classified as 500 INTERNAL_ERROR**: the
  fetcher's `ErrRateLimited` had no entry in `errorClassification`, so a
  throttled fetch fell through to 500 with the real cause masked. It now maps
  to `429 RATE_LIMITED` — a truthful client error (TASK-240; ISS-238).
- **`http.rateLimit` was a no-op at the REST surface**: the oracle Fetch
  handler built a fresh fetcher (and therefore a fresh per-source
  `RateLimiter`) on every request, so each call looked like the first of its
  budget and the documented limit never tripped across REST calls — only the
  long-lived scheduler fetcher throttled. The handler now shares one fetcher
  across requests (same model as the scheduler), so the configured budget is
  enforced at the endpoint that advertises it (TASK-241; ISS-239).
- **The oracle scheduler kept bookkeeping for deleted sources forever**:
  `lastFetch`/`failStreak`/`nextAttempt`/`stats` are keyed by source id and
  were never released, so a deleted source stayed in `GET /oracle/health` and
  `/metrics/oracle` indefinitely and the maps grew without bound across
  create/delete cycles (a recreated id would also inherit stale failure/
  backoff state). Each scheduler pass now reconciles against the repository's
  live source list and prunes ids that no longer exist (TASK-242; ISS-240).
- **`voting.html` permanently disabled Cast Vote after a transient load
  failure**: a candidates load failure at init left `candidatesFailed` set and
  the roster empty, gating both the candidate dropdown and Cast Vote — and the
  voting page has no polling, so nothing short of a full browser reload could
  un-stick it. The page now offers an in-page `↻ Retry` that re-runs the
  candidates/sessions loaders once the API recovers (TASK-243; ISS-241,
  regression test executes the shipped JS in Node).
- **Two processes sharing the chain DB silently overwrote committed blocks**:
  the API server and a CLI lottery/oracle command both boot a process-local
  `BlockChain`, so each computes the next height from its own `len(Blocks)` —
  when both booted genesis-only and both appended "height 1", the persist hook
  (`INSERT OR REPLACE` keyed on height) silently replaced the first process's
  row, losing its payload; in-memory chains then diverged and the next restart
  failed `VerifyIntegrity`. Heights are now DB-authoritative: `persist` uses a
  plain INSERT and maps the height UNIQUE conflict to `ErrHeightConflict` (with
  a code-level, not message-substring, match); `appendBlock` re-syncs the
  in-memory chain from the shared DB before reserving, and on a lost height
  race drops its candidate and retries at the true next free height (bounded).
  The reseam check also rebuilds from the ledger when a prior non-conflict
  persist failure left a phantom tail that a peer's commit invisibly diverged
  from (TASK-244/TASK-245-design; ISS-242, plus review-follow-up).
- **`AsyncEventBus.Publish` could report success for a silently dropped
  event**: `Publish` checked `closed` and then sent with no synchronization
  against `Close`, so a publish racing a close could pass the check, have the
  consumer drain-and-exit, then land its event in the buffer after the
  consumer was gone — returning nil while the event was never delivered (a
  silent loss the audit/outbox layers rely on never happening). The
  closed-check + send and the close now serialize on a mutex, making "every
  successful publish is delivered" an exact invariant (ISS-243).

### Added

- **Oracle source capabilities reach API/CLI/TUI parity**: `aurora oracle
  source add` gained `-m/--method` and `-p/--path` (a price source's JSON path
  could previously be set only via REST/web/TUI), `source list` now prints
  Method/Path/Interval, and the oracle TUI add-source form sets
  method/path/interval with client-side interval validation (6-field focus
  cycle; empty → default 60; non-numeric/negative → inline error before any
  write). Six new i18n keys (en/zh) (TASK-225, TASK-227; ISS-222, ISS-225).
- **Oracle TUI source detail reaches CLI `source list` parity**: the detail
  view now shows method (default GET), JSON path and interval (default 60s)
  — previously only name/url/type/status — and `loadSources` copies the
  fields into the TUI's source copy so they don't render empty (TASK-229;
  ISS-227).

## [v1.92] - 2026-08-31

### Fixed

- **Voting broke on a fresh database**: `NewVotingRepository` was the only
  repository that never self-created its tables (they existed only in the
  migrations), and neither the API server (`cmd/api` never runs migrations) nor
  the CLI (`migrate.autoRun` defaults to false) guaranteed they ran — so a
  fresh install returned `no such table` on every `/api/v1/voting/*` route and
  `voting` CLI op while lottery/token/nft/oracle worked out of the box. The
  constructor now bootstraps the schema with `IF NOT EXISTS` (idempotent on an
  already-migrated DB) and returns an error (TASK-197, ISS-193).
- **Web voting submitted a stale candidate after a session switch**: switching
  sessions left the previous selection bound, so a mismatched candidate was
  POSTed and rejected with 400 `CANDIDATE_NOT_IN_SESSION`. Switching sessions
  now resets the candidate, the submit guard also checks roster membership, and
  the voter private-key input is password-masked like the token/nft equivalents
  (TASK-198, ISS-194).
- **Web create forms were missing API/CLI capabilities**: token create had no
  way to set `decimals` (the API accepts it and the CLI has `--decimals`), NFT
  mint never advanced the History form's id, token create never advanced the
  Token Info form's id, and the oracle add-source interval silently defaulted
  an empty value to 60. All four now match the API/CLI contract (TASK-199,
  ISS-195).
- **Backend error-path inconsistencies**: `[i18n] locale` was read from a dead
  top-level `locale` key (the knob was inert; now a fallback for environments
  without `LANG`, which remains authoritative); the rate-limit `Retry-After`
  header was hardcoded 60s regardless of the configured window (now advertises
  the real window); token `info` returned error code `NOT_FOUND` where every
  other token 404 uses `TOKEN_NOT_FOUND`; and `lottery verify` masked genuine
  DB read failures as "lottery not found" (TASK-200, ISS-196).
- **Web lottery history disguised a load failure as an empty system**: a failed
  first load of `/api/v1/lottery/history` rendered "No lotteries yet"; it now
  distinguishes "nothing exists" from "couldn't load" like voting/oracle
  (TASK-201, ISS-197).
- **CLI/docs drift**: `aurora --help` and `aurora version` still described the
  project as a "voting system"/"VRF Lottery System" rather than the module
  suite; the `migrate down` example omitted the mandatory `--confirm`; and the
  v1.90 entry mislabelled the NFT TUI list view as the 5th menu item (it is the
  4th) (TASK-202, ISS-198).
- **Lottery TUI was invisible to the CLI surfaces**: draws created inside
  `lottery tui` only reached the chain's `blocks` table, so `lottery history/
  stats/export/verify` (which read the persistent `lottery_records`) never saw
  them — while `lottery import`ed draws never appeared in the TUI history,
  which additionally listed unrelated on-chain records (votes/transfers) as
  "lotteries". The TUI now dual-writes draws to `lottery_records` and reads
  history from that store, matching the CLI exactly (TASK-203, ISS-199).
- **Oracle TUI advertised editing a data source that cannot be edited**: the
  source detail view was titled "Edit Data Source" with a pencil icon and an
  "[Enter] Edit source" hint, but the view is read-only (no edit use case
  exists). Retitled to "Source Details" (TASK-204, ISS-200).
- **NFT/token TUIs silently dropped their work**: both ran on in-memory
  repositories (data lost on quit) while the lottery/oracle TUIs persist — a
  mismatch documented only in code comments. Both menu screens now show an
  explicit sandbox notice up front (TASK-205, ISS-201).

## [v1.91] - 2026-08-30

### Fixed

- **Oracle TUI has no scrollable viewport**: multi-row `query` results and the
  `fetch` result rendered past the terminal edge with no way to reach the tail
  (`sources` likewise spilled beyond the screen for long source lists). The
  read-only result views now render through a bounded bubbles viewport that
  handles `↑/↓/j/k/pgup/pgdn/space/f/b/u/d` scrolling, and the source menu gets
  a cursor-following window sized from the terminal (TASK-176, ISS-174).
- **Web submit buttons had no in-flight busy state**: a fast double-click on a
  create/transfer form fired two overlapping requests and recorded the resource
  twice (only the blockchain verify button guarded itself). Added a shared
  `withBusy` wrapper exposing per-action `busy.<name>` flags; every write form's
  submit button binds `:disabled` to its own flag, so one in-flight create
  disables only its own button, and re-entrant clicks are swallowed. A Node
  harness test executes the shipped `app.js` against a stubbed fetch to pin the
  guard (TASK-177, ISS-175).
- **Oracle TUI query limit was unbounded**: the CLI (`clampQueryLimit`) and the
  REST API (`maxQueryLimit=100`) both bound the query limit to `[1,100]`, but
  the TUI passed the raw parsed input straight to the use case, so a stray or
  inflated number (e.g. `999999999`) forced an unbounded DB scan. `handleQuery`
  now clamps through `clampQueryLimitValue` mirroring the other two surfaces
  (TASK-178).
- **Help screen never documented viewport scrolling**: every TUI's long
  result/list view is now a bounded viewport, but the shared help screen
  omitted the scroll keys (`↑/↓/j/k/pgup/pgdn/space/b/f/u/d`) an operator needs
  on a long list. Added a `tui.help.scroll` line (en + zh) to the shared
  `HelpView`, plus an en↔zh key-parity test gate so a missing translation can
  no longer silently fall back to English (TASK-179).
- **Oracle TUI fetches skipped on-chain recording**: the REST handler,
  scheduler, and CLI all call `FetchDataUseCase.SetChain`, but the TUI's
  `handleFetch` built the use case without a chain, so every TUI-fetched
  observation was saved at `block_height=0` with no ledger block — the exact
  TASK-097 scheduler regression, reintroduced by the TUI surface. The model
  now carries a chain (`SetChain`, wired to the same singleton the CLI uses)
  and `newFetchUseCase()` propagates it; the `FetchDataUseCase.Chain()` seam
  makes the wiring testable without a network fetch (TASK-180).

## [v1.90] - 2026-08-30

### Fixed

- **Every web write was authenticating with a dropped API key**: the round-97
  `apiFetch` refactor merged caller `headers` over the defaults with
  `Object.assign({headers: auroraHeaders()}, options)`, so any call that passed
  its own `Content-Type` **replaced** the whole headers object and silently
  dropped `X-API-Key` — every POST/PATCH returned 401 while reads kept working,
  leaving the UI looking alive but write-dead (verified live). `apiFetch` now
  merges the caller's headers on top of the key. A regression test **executes
  the shipped app.js in Node** against a captured fetch init (the `node
  --check` gate only parses, which is exactly why this slipped through several
  rounds).
- **Token history showed only transfers an account sent**: the reader paged
  `$.from == owner` and never surfaced transfers where the account was the
  recipient. Added a two-sided payload-pagination primitive
  (`GetByAggregateAndTypePayloadEither`), so received mints/transfers appear
  next to sent ones with correct paging. `/token/history` also now rejects
  missing/empty `token_id`/`owner` with 400, matching its sibling endpoints
  instead of silently returning `200 []`.
- **TUI forms still swallowed `j`/`k` and `?`** after the `q` fix: lottery
  could not type them at all, token/nft ate them positionally, and oracle moved
  focus AND typed the letter into the wrong field (a URL containing `k`
  corrupted a neighbour input); `?` always opened help, making query-string
  source URLs untypeable. `j`/`k`/`?` are now typable in every form; `↑`/`↓`
  and Tab are the form-navigation keys, and the menu/read-only bindings are
  unchanged.
- **Voting session start/end/Getsession**: real repository failures were
  flattened into `404 not found`, disguising an outage as a missing resource.
  Now routed through a `GetSessionUseCase`: missing session → 404
  `SESSION_NOT_FOUND`, genuine DB fault → generic 500 (session lifecycle
  transitions unchanged — DEC-004).
- **`aurora migrate` opened SQLite un-hardened** (no `_busy_timeout`/`_txlock`),
  so a migrate run against a DB another process held the write lock failed
  instantly with "database is locked". It now uses the same hardened DSN as
  every repository (`sqlite.DSN`, single source of truth).
- **Blocks reloaded from disk lost their timestamps**: the reload SELECT never
  scanned the persisted `timestamp`, so every in-memory block was `Timestamp==0`
  after restart (hash/VerifyIntegrity unaffected — PoW excludes timestamp).
- **Voting results could under-count silently**: `GetResults` did `continue`
  on ANY candidate-read error, treating a real DB failure as a deleted
  candidate (0 votes). Only `sqlite.ErrNotFound` counts as 0 now; genuine
  failures abort the report.
- **Oracle pollers wiped live tables** on a transient poll failure, violating
  the keep-existing-data invariant the dashboard honours; they now leave prior
  rows in place and let the shared API-error banner report the failure.
- **decodeJSON accepted trailing garbage** after the first JSON value
  (`{"a":1}{"b":2}` or `{...}non-json` passed as well-formed); it now requires
  the stream to end cleanly with `io.EOF`.
- **NFT TUI "List by Owner" was dead code**: the model, loader and viewport
  existed but the menu never reached the list view. The 4th menu item now opens
  an owner prompt feeding the scrollable list, mirroring the CLI/web surface.
- **i18n English leaks**: oracle/nft TUI labels and several CLI one-liners
  (token limit/offset flags, lottery "no records"/db-info, oracle "no data")
  stayed English in zh sessions; routed through the tables (both locales).
- **Dead oracle `data` view removed**; the voting web **Create Session** form
  now disables submit until ≥1 candidate is checked (was client-unvalidated 400).

## [v1.89] - 2026-08-29

### Fixed

- **Web Create Token actually works**: the form sent `{name, symbol,
  total_supply}` with no owner, and the API rejects an empty owner
  (`ErrPublicKeyRequired`), so every web create returned `400
  PUBLIC_KEY_REQUIRED` with no way to fix it. Added a required "Owner
  (public key)" input mirroring the NFT mint "Creator (public key)" field.
- **Web shared-context advances**: token create fills the shared
  tokenId/owner so Balance/Mint/Transfer/Approve/Burn/History pre-fill,
  voting session create fills start/end/vote/results ids, lottery create
  fills the Verify Draw id, token mint advances to the minted recipient
  (fixing a confusing balance error right after a successful mint), and
  oracle add-source fills the fetch/query/latest ids. The create owner field
  is isolated in its own `createOwner` state so a key typed for a create
  never leaks into the Balance/History context.
- **TUI `q` is typable in every form**: lottery/nft/token/oracle previously
  consumed `q` as a quit/bail key before the focused textinput saw it, so
  names/symbols/descriptions containing the letter ("quack", "Aq") could not
  be typed. `q` now quits only from the menu; `ctrl+c` is the hard quit; a
  read-only view still returns to the menu.
- **Token TUI create honors decimals**: the field was validated but never
  assigned, so a create with `18` silently produced an 8-decimal token. It
  is now range-checked (0–127) and stored.
- **NFT TUI key-length + transfer result**: mint no longer accepts a
  valid-base64 but wrong-length owner key (a permanently-untransferable
  NFT); transfer success re-fetches and shows the post-transfer owner
  instead of "⚠ Not found" on a fresh session.
- **Oracle sources cursor bound**: the down-key allowed one step past the
  last row (invisible cursor, dead Enter) — with zero sources a single
  `↓` lost the cursor on the one-row list.
- **Lottery TUI count parse is a visible error**: a cleared/non-numeric
  winners field previously ran a silent 3-winner draw; it now fails fast
  with a localized message.
- **Oracle `[T]`/`[D]` hotkeys accept uppercase**, matching the advertised
  footer and the `y/Y n/N` confirm-dialog pattern.
- **AGENTS.md CLI examples match reality**: `nft get --id`/`voting session
  start -i` documented flags that do not exist; synced to the real
  spellings (`--nft`, `--session/-s`).

### Added

- **JS syntax gate**: `TestWebUIJS_SyntaxValid` runs `node --check` over the
  shipped `web/js/app.js`, so a JS syntax regression fails `go test` instead
  of breaking every page at browser runtime (skips cleanly without node;
  CI's ubuntu-latest ships Node).

## [v1.88] - 2026-08-29

### Fixed

- **Web error surfaces are consistent**: the Blockchain page was the only
  surface bypassing the shared API-error banner (raw `fetch`); it now goes
  through `apiFetch` like every other page. Oracle enable/disable/delete no
  longer pop blocking `alert()` dialogs — failures render as inline text next
  to the source list — and the Query action no longer silently empties its
  results table. Six unreachable `if (!res.ok)` guards in the voting web app
  (dead after `apiFetch` throws) were removed.
- **Dashboard stats survive partial failures**: one voting endpoint failing no
  longer drops the sibling stat group; each card's loader is independent.
- **Live surfaces no longer flicker**: oracle Data Sources + Feed Health tables
  and the dashboard stats grid / Recent Activity list stay visible during the
  15s background polls instead of flashing Loading.../blanking to 0.
- **Token Burn uses its own amount field**: a Transfer quantity can no longer
  leak into a Burn (both shared the `xAmount` model field).

### Added

- **NFT mint advances the shared context**: a successful mint fills the
  Get/Transfer/Burn/History id + owner fields, so the next step needs no manual
  copy from the JSON result.

## [v1.87] - 2026-08-29

### Fixed

- **API resource-status consistency**: unknown oracle source / NFT id now return
  404 (not `200 []` or an unclassified 500); `/oracle/query` with a missing
  `source` param returns 400 (parity with `/latest`); the sqlite repos return
  `(nil, nil)` for a source with no data (the interface contract already used
  by the in-memory repo), so `/oracle/latest` no longer 500s on a known-but-empty
  source; `/nft/{id}/history` on a nonexistent id 404s while a real NFT with no
  operations keeps returning `200 []`.
- **SyncEventBus self-deadlock**: `Publish` snapshots handler lists under the
  read lock and runs handlers outside it, so a handler may subscribe/unsubscribe
  during publish without deadlocking the whole bus.
- **MetricsRegistry lazy-init race**: `Server.MetricsRegistry()` is
  `sync.Once`-guarded so concurrent callers cannot create two registries and
  split the request counters.

## [v1.86] - 2026-08-28

### Fixed

- **Cancellable oracle scheduler fetches**: in-flight HTTP fetches abort on
  shutdown instead of stalling SIGTERM for up to N sources × 10s.
- **Backup path-traversal rejection**: archive metadata naming
  `../../victim` can no longer make verify/restore touch arbitrary `.db` files.
- **Bounded rate-limiter buckets**: both limiters now evict fully-expired keys
  past a sweep threshold so bucket maps stay proportional to active clients.

## [v1.85] - 2026-08-28

### Added

- Typable TUI forms with Tab/↑/↓ focus cycling (lottery/token/nft); scrollable
  viewport history/list views; `?` opens a localized keyboard-shortcuts help;
  `--confirm`/`-y` gate on destructive CLI ops (token burn, nft burn, oracle
  source delete, migrate down); localized `--help`; oracle confirm dialogs with
  visible Yes/No selection; web auto-refresh + manual refresh button; Alpine
  vendored locally (no CDN).

### Fixed

- **Truthful `version`**: was fabricated (hardcoded 0.0.1 + fake Go version);
  now reads link-time `Version`/`BuildTime` and real `runtime.Version()`.
- **Web API failures surfaced**: list GETs no longer silently swallow the
  `{error, code}` envelope and render blank pages — a shared banner reports them.
- **Hardcoded CJK → i18n**: token/lottery TUI strings no longer leak Chinese
  into en-locale sessions.

## [v1.84] - 2026-08-28

### Fixed

- **Voting client-error classification**: wrong-length vote key → 400; duplicate
  roster candidates rejected; remaining flags triaged to documented design or
  parked decisions (DEC-004/005).

## [v1.83] - 2026-08-28

### Fixed

- **Config durations**: numeric TOML durations now treated as seconds everywhere
  (`30` = 30s, not 30ns).
- **Audit durability**: a durable outbox heals failed audit publishes.
- **Backup atomicity**: atomic metadata/restore.

## [v1.64]–[v1.82] - 2026-08 (earlier milestone sweep)

Documented in the v1.81–v1.84 sweep roadmaps and git history; highlights in this
range: CORS cross-origin key-exfiltration hardening (v1.64), backup
self-overwrite guard (v1.65), NFT operation audit-trail collapse (v1.66), atomic
`ClaimNextNonce` under a real connection pool (v1.67), no phantom events on token
tx rollback (v1.68), rate-limit spoof bypass via trusted-proxy allow-list (v1.69),
SQLite writer contention / `SQLITE_BUSY` and the in-transaction nonce deadlock it
regressed (v1.70/v1.72), unbounded JSON request bodies capped at 4 MiB (v1.71),
`cmd/api` secrets + audit-trail wiring (v1.73), bounded metrics label cardinality
(v1.74), backups stale under a live WAL server → `VACUUM INTO` snapshots (v1.75),
CLI TUI commands no longer exit 0 on failure (v1.76), NFT burn audit-trail
retention + owner-scoped token-history paging/envelope + `cmd/api` config-file
loading (v1.77), Base64→400 classification + oracle scheduler on-chain recording
(v1.78), honored `token create --decimals` + lottery-reset refused → exit≠0 +
bounded NFT list paging + `db.path` honored everywhere (v1.79), removed phantom
HOME data-dir wiring of the dead `app.Wire`/`GlobalApp` (v1.80), integrity &
client-error sweep (v1.81), report-the-truth & dead-code sweep — committed token
ops never reported failed, backup restore same-file + WAL-complete guards, dead
`app.Wire` retired (v1.82).

## [0.0.1] - 2026-04-07

### Added

- **SQLite Repository Tests**: Comprehensive test coverage for all repositories
    - `token_repo_test.go`: SaveToken, GetToken, SaveApproval, GetApproval
    - `blockchain_repo_test.go`: SaveBlock, GetBlock, GetAllBlocks, GetLotteryRecords
    - `nft_repo_test.go`: SaveNFT, GetNFT, GetNFTsByOwner, GetNFTsByCreator, SaveOperation
    - `oracle_repo_test.go`: SaveSource, GetSource, ListSources, SaveData, GetDataBySource
    - `voting_repo_test.go`: SaveVoter, GetVoter, SaveCandidate, ListCandidates, SaveVote

### Changed

- **Code Quality Improvements**:
    - Renamed `VoterPK` → `VoterPublicKey` for consistency
    - Extracted magic numbers to named constants
    - Added package comments to all domain modules
    - Added struct docstrings: NFT, Operation, Vote, Voter, Candidate, Session, Block, BlockChain
    - Standardized API error responses with helper functions

### Fixed

- **Critical Bug Fixes**:
    - Nonce not persisted in Transfer/TransferFrom (would cause duplicate nonce errors)
    - Mint not updating totalSupply (token supply not tracked)
- **Error Handling**:
    - Fixed error message leakage in HTTP handlers
    - Fixed ignored base64/json decode errors in SQLite repositories
    - Added nil check in BlockChain.AddBlock
- **Code Cleanup**:
    - Removed unused `pk` variable in lottery service
    - Removed no-op `Handle()` function in blockchain
    - Deleted debug `internal/main.go` that ran E2E on startup

### Test Coverage

| Module  | Domain | App   |
| ------- | ------ | ----- |
| Lottery | 73.8%  | 87.1% |
| Voting  | 87.5%  | 81.4% |
| NFT     | 72.7%  | 88.1% |
| Token   | 70.5%  | 91.9% |
| Oracle  | 76.1%  | 65.8% |
| SQLite  | -      | 45.5% |

## [0.0.1-alpha.3] - 2026-04-07

### Added

- **Token (FT) Module**: Complete Fungible Token system
    - `token/create` - Create new token with name, symbol, supply
    - `token/mint` - Mint tokens to address
    - `token/transfer` - Transfer tokens between addresses
    - `token/burn` - Burn tokens
    - `token/balance` - Query token balance
    - `token/history` - Query transfer history
    - `token/tui` - Interactive TUI interface

- **Complete TUI Functionality**:
    - NFT TUI: Mint, Transfer, Query, List by Owner
    - Oracle TUI: Source Management, Fetch, Query
    - Token TUI: Create, Mint, Transfer, Balance, History

- **UI Theme Library** (`internal/ui/components/theme.go`):
    - Unified styling across all TUI modules
    - Module-specific accent colors
    - Reusable components (MenuActiveStyle, InputStyle, ViewportStyle, etc.)

- **Comprehensive Test Coverage**:
    - Token E2E full flow test
    - Token App integration tests (15 use case tests)
    - Token Domain tests (12+ new tests)
    - NFT Domain tests (Transfer, Query, List)
    - Oracle Domain tests (AddSource, Toggle, Delete, Fetch, Query)

### Changed

- Refactored all TUI modules to use `components` package
- Added i18n support for error messages
- Fixed hardcoded Chinese strings → i18n.GetText()

### Fixed

- Token TUI import syntax error
- Unused variable warnings
- golangci-lint issues

### Test Coverage

| Module  | Domain | App   |
| ------- | ------ | ----- |
| Lottery | 74.0%  | 87.1% |
| Voting  | 87.5%  | 81.4% |
| NFT     | 72.7%  | 88.1% |
| Token   | 70.7%  | 40.7% |
| Oracle  | 76.1%  | 65.8% |

## [0.0.1-alpha.2] - 2026-04-06

### Refactored - DDD Architecture

Complete migration from monolithic modules to **Domain-Driven Design (DDD)** architecture:

- **Domain Layer** (`internal/domain/`): Entities, Services, Repository interfaces
    - `blockchain/` - Block, BlockChain, InitBlockChain()
    - `lottery/` - LotteryRecord, VRF Service, Repository
    - `voting/` - Vote, Voter, Candidate, Session
    - `nft/` - NFT, Operation, Service
    - `oracle/` - OracleData, DataSource

- **Infrastructure Layer** (`internal/infra/`): Storage implementations
    - `sqlite/` - SQLite repository implementations
    - `http/` - HTTP client for Oracle

- **Application Layer** (`internal/app/`): Use Cases
    - `lottery/` - CreateLotteryUseCase
    - `voting/` - CastVoteUseCase, RegisterVoterUseCase
    - `nft/` - MintNFTUseCase, TransferNFTUseCase
    - `oracle/` - FetchDataUseCase

- **Presentation Layer** (`internal/ui/`): TUI interfaces
    - `lottery/`, `nft/`, `oracle/`

### Dependencies Updated

- Bubble Tea v1 → v2 (`charm.land/bubbletea/v2`)
- Bubbles v1 → v2 (`charm.land/bubbles/v2`)
- Lipgloss v1 → v2 (`charm.land/lipgloss/v2`)
- GitHub Actions updated to latest versions

### Added

- **Domain Tests**: Unit tests for domain entities
- **SECURITY.md**: Security policy and vulnerability reporting
- **.goreleaser.yaml**: GoReleaser configuration
- **Pre-commit hooks**: Go format/lint hooks

### Fixed

- Docker build cache issues
- Go version in Dockerfile (1.26)
- golangci-lint configuration
- errcheck disabled in CI (matches local lint)

## [0.0.1-alpha.1] - 2024-04-06

### Added

- **VRF Lottery System**: Transparent lottery using Verifiable Random Function with blockchain storage
- **Voting System**: Ed25519 signed voting with real-time counting and blockchain persistence
- **Oracle Module**: HTTP API data fetching with predefined templates (BTC/ETH prices)
- **NFT System**: Mint, transfer, and burn NFTs with Ed25519 signatures
- **Blockchain Storage**: SQLite persistence with Write-through caching
- **CLI Commands**: Full command-line interface for all modules
- **TUI Interface**: Interactive terminal UI using Bubble Tea
- **E2E Tests**: Comprehensive end-to-end tests for all modules
- **CI/CD**: GitHub Actions workflows (test, build, docker, release)
- **Pre-commit Hooks**: Go format/lint hooks

### Changed

- Replaced Makefile with justfile
- Split CI workflows into separate files

### Features

- Cross-platform builds (darwin/linux/windows, amd64/arm64)
- Docker multi-platform images (amd64/arm64)
- Auto-release on tag
- Codecov integration

## [0.0.1-alpha.0] - 2024-04-05

### Initial Release

- Basic lottery system with VRF
- In-memory blockchain
- CLI and TUI interfaces
