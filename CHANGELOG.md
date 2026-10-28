# Changelog

All notable changes to this project will be documented in this file.

The v1.x line is milestone-tracked in `.planning/milestones/` and `.planning/STATE.md`
(release `Version` is injected at build time via `-ldflags -X cmd.Version=`). The
entries below summarise v1.64–v1.93; earlier v1.x milestones (v1.0–v1.63) are
documented in their per-milestone ROADMAP files.

## [Unreleased]

### Added

- **Registered voters stay visible after the one-time key card**: the desktop
  Voting page now lists stored voters with search, paging and ballot status;
  choosing one fills the ballot public-key field and offers the full key for
  copying. `GET /api/v1/voting/voters` exposes the same newest-first
  `limit`/`offset` summary. Both surfaces deliberately omit the registration
  private key, which exists only in the registration response.

- **Dashboard counters are aggregate reads**: the desktop's landing page now
  asks SQLite for exact lottery, candidate, vote, session and Oracle-source
  totals instead of decoding every stored collection on each live refresh.
  The visible counters are unchanged, but a large store no longer turns a
  background dashboard refresh into an unbounded materialisation of all rows.

- **Voter lists page from storage**: SQLite now returns voters newest-first
  with `LIMIT`/`OFFSET` and a `public_key` tie-breaker for same-second
  registrations. `aurora voting voter list` exposes `--limit`/`--offset`, keeps
  zero as the full-list form, and reports an empty later page without claiming
  the entire voter store is empty.

- **Lottery history pages in storage instead of after the read**: SQLite now
  applies the deterministic newest-first `LIMIT`/`OFFSET` used by
  `GET /api/v1/lottery/history`; the REST handler no longer materialises every
  stored draw before slicing it. `aurora lottery history` gained matching
  `--limit`/`--offset` flags, while its zero limit keeps the full-history form.
  The desktop History card pages 100 draws at a time, disables Next at an exact
  page boundary, and retains the last successful page when a move fails.

- **NFT operation history is paged instead of silently capped**: SQLite,
  `GET /api/v1/nft/{id}/history`, `aurora nft history`, and the desktop History
  card now share a newest-first `limit`/`offset` contract. The old fixed
  1000-row truncation could not reach older audit records, and a burned NFT's
  empty later page was indistinguishable from an unknown id; both cases are
  now explicit. Desktop paging keeps the last good page when a move fails and
  probes one row past each page for an exact end state.

- **Oracle sources use the same bounded collection contract**: SQLite,
  `GET /api/v1/oracle/sources`, `aurora oracle source list`, and the desktop
  Sources card now pass `limit`/`offset` instead of materialising every feed.
  The scheduler deliberately requests the unbounded form because it must check
  every enabled source. The desktop adds exact-end paging and keeps the last
  good rows/highlight when a source page read fails. Scheduler health counters
  now carry each source's name, so the Feed Health table stays readable for
  sources outside the page currently shown.

- **Voting collections are paged from storage to every surface**: candidate and
  session lists now use SQLite `LIMIT/OFFSET` with a stable newest-first order,
  `GET /api/v1/voting/{candidates,sessions}` accepts capped `limit`/`offset`
  query parameters, and `aurora voting candidate|session list` exposes the same
  controls while keeping zero as "the remainder from offset". The desktop
  Setup cards read 100 rows at a time with exact-end Previous/Next paging,
  keep the last good page and highlight when a move fails, and still expand an
  empty session roster to every registered candidate rather than only the
  visible page.

- **The token collection is paged from storage to every screen**: the desktop
  Tokens card now reads 100 records at a time with Previous/Next and an exact
  end state, `aurora token list` accepts `--limit`/`--offset` (zero still means
  every token), and `GET /api/v1/token/list` passes its bounded page through
  the use case and SQLite `LIMIT/OFFSET` instead of loading the whole table and
  slicing it in memory. The desktop clears its row highlight when the page
  changes, so an index from the old page cannot appear to select a different
  token on the new one.

- **Optional session-only key memory**: Settings → Session keys can keep a
  private key in process memory after a successful signature, so the next form
  with the same public key offers "Use saved key". It is off by default, never
  written to disk, and cleared by "Forget session keys" or when Aurora exits;
  failed or unknown operations store nothing.

- **The desktop can generate the owner/creator key it asks for**: the token and
  NFT forms required a base64 public key that nothing in the window produced,
  although `aurora token create` generates an owner pair when the flag is
  omitted. Both forms now have "Generate key pair": the public half fills the
  field, the private half is shown once with Copy and a Dismiss button, and the
  key-generation service is the descriptor's own `GenerateKeyPair`.
- **The desktop window has a visual design of its own**: `nativeui/theme.go` is
  the single palette — a polar-night background, one raised card surface,
  Aurora's accent and three status hues — applied on every frame instead of
  accepting the system appearance. The window used to set a dark background and
  then let MyGo draw its light theme over it, so every control was a white panel
  on a black page. The module list is a real `ui.Sidebar` (the toolkit tracks
  and tints the chosen item; arrows and first letters move between modules), the
  pages lay their work out as cards in two columns, and each page has one status
  line that states "Done" or "Failed" in words as well as colour.
- **Tokens and NFTs can be listed**: `token.Repository`/`nft.Repository` gained
  a list operation, exposed as `aurora token list`, `aurora nft list` (no longer
  requiring `-o`), `GET /api/v1/token/list` and
  `GET /api/v1/nft/list` without `?owner`. Both pages lead with the collection
  and choosing a row makes that asset the subject of the write cards. Before
  this, an id was reachable only from the output of the process that created it.
- **A voting session carries its ballot window**: `SessionResponse` includes
  `start_time`/`end_time`, and the desktop session table shows the window — the
  fact that explains a refused ballot.

- **The five-platform desktop matrix has a manual validation entry point**:
  `desktop-build.yml` now accepts `workflow_dispatch` in addition to
  `workflow_call`. A manual run defaults to the selected branch, packages
  Linux amd64/arm64, macOS amd64/arm64 and Windows amd64, uploads the
  artifacts, and does not create a release; signing remains optional unless
  the `require_signed_*` inputs are set.
- **Code-signing and privacy boundaries are now public**: the README and
  desktop guide link a signing policy and privacy statement. The policy names
  the maintainer approval roles, applies the SignPath Foundation phrase only
  after acceptance, and records that SignPath cannot replace the Apple
  Developer Program identity required for macOS notarization.
- **Release assets now receive GitHub build provenance**: after checksums are
  generated for the final signed-or-unsigned artifacts, the release workflow
  attests every checksum subject to the repository workflow run. Users can
  verify downloads with `gh attestation verify`; the attestation supplements
  but does not replace Authenticode or Apple Developer ID signatures.

### Fixed

- **Malformed transfer audit rows no longer abort owner history**: the SQLite
  owner-scoped transfer page called `json_extract` on every matching payload,
  so one malformed JSON row made the whole history fail with
  `SQL logic error: malformed JSON` and hid every readable transfer. The query
  now validates JSON before extracting `from`/`to`, so corrupt rows are skipped
  while readable pages and offsets keep their existing semantics.
- **Atomic file writes flush their directory entry before reporting success**:
  lottery exports and desktop preferences synced the temporary file, then
  replaced the destination without syncing the containing directory. A power
  loss immediately after a successful save could therefore lose the rename
  even though the new file contents were durable. Atomic writes now fsync the
  parent directory after promotion and return a durability error if that flush
  fails.
- **Backup safety copies are durable before promotion**: newly vacuumed
  database snapshots, the hidden create-rollback set, and the pre-restore
  safety snapshot were closed and renamed without explicitly flushing their
  bytes and directory entries. A successful desktop backup also created a new
  timestamp directory without syncing that directory into its parent, so a
  power loss could remove the entire backup. Create and Restore now flush each
  staged file and every new directory level before replacing visible data.
- **Read-only backup files remain replaceable**: flushing a hard-linked old
  snapshot no longer opens it for writing. A directory that permits replacing
  an archive now works even when the previous database and metadata were
  hardened to mode `0400`.
- **Restore promotes only bytes covered by its verification**: Restore
  re-read metadata and copied archive files after Verify, so replacing the
  archive in between could promote different bytes successfully. Restore now
  carries the verified metadata forward, records a SHA-256 for every verified
  database (including legacy archives), hashes the bytes actually copied into
  staging, and aborts before promotion on a mismatch. Every database is also
  reopened during verification so self-consistent metadata cannot bless
  arbitrary non-SQLite bytes.
- **Backup verification applies Restore's database-set contract**: Verify
  checked only the files listed inside metadata, so an archive for a different
  installation — or one listing the same database twice — could report success
  and then be rejected by Restore. Both paths now require a unique, exact match
  with the configured database names before an archive is called usable.
- **Backup promotions flush the containing directory before returning**:
  syncing the copied database and metadata files made their bytes durable, but
  the subsequent renames were not durable themselves. A power loss after
  `backup create` or `backup restore` reported success could resurrect the old
  directory entry while the previous file contents were gone. Successful and
  rolled-back promotions now fsync the destination directory; a directory
  flush failure makes the operation fail and rolls the visible set back.
- **An interrupted backup create recovers its previous verified set**: the
  two-phase create retained old snapshots in a hidden rollback directory, but
  only an in-process error could use it. A SIGKILL or power loss between the
  database rename and the metadata commit left new database bytes under old
  metadata, so `backup verify` reported checksum corruption indefinitely and
  the next create could not repair it. Verify, Restore and the next Create now
  detect that rollback directory, keep the set when its metadata hashes match,
  or restore the previous verified files when they do not; stale rollback
  copies are removed only after the visible set verifies again.
- **Backup crash recovery follows the archive, not the caller**: a valid
  archive belonging to another installation could be mistaken for an
  interrupted create, replacing its metadata or snapshots with a stale
  rollback set. Recovery now checks the visible files and unique names against
  their own metadata, restores the exact database names recorded in the
  rollback directory, and clears a partial first archive so the next Create
  can retry.
- **Backup recovery rejects unsafe configured names before cleanup**:
  first-create recovery joined configured database names with the output path
  before Create's normal validation ran, so a `../` name could remove a file
  outside the archive. Recovery now validates every configured name before it
  restores or removes any path.
- **Backup metadata is durable before it becomes the commit record**:
  `backup create` wrote `metadata.json` through a temporary file and renamed
  it without syncing the new contents. A crash immediately after the rename
  could leave the promoted database snapshots beside an empty or truncated
  metadata file, making the new backup unverifiable and replacing the previous
  commit record. The metadata temp is now fsynced and closed before promotion;
  a sync failure rolls the database set back to its previous verified state.
- **Token history pages stay full when an audit row is unreadable**: the
  SQLite reader applied `LIMIT`/`OFFSET` before decoding each transfer, then
  skipped corrupt JSON or amounts. One bad row could shorten a page and shift
  every later offset, so clients that stop on a short page silently lost
  readable history. Owner-filtered and token-wide reads now page over decoded
  transfers, scan past unreadable rows, and make `offset` count the rows a
  client actually receives.
- **Database restores sync copied bytes before replacing a live store**:
  `backup restore` and its rollback path copied a snapshot into a temporary
  file and renamed it after `Close` without flushing the file. A crash or
  power loss immediately after promotion could therefore leave the live path
  holding a truncated or empty database even though the restore reported
  success. The temporary copy is now fsynced before promotion, and a sync
  failure removes it without touching the destination.
- **A non-graceful API failure still drains its workers before storage
  closes**: `aurora-api` handles a listener error by calling `Server.Close`
  directly because `logger.Fatal` skips deferred stops. The server now owns
  the scheduler and outbox stop functions, stops them in reverse start order,
  and waits for in-flight repository work before closing SQLite. Concurrent
  and repeated `Close` calls wait for the same shutdown, and a worker registered
  after closure is stopped immediately.
- **An unusable database directory is no longer mistaken for temporary
  storage**: the shared path resolver used to create the configured parent
  directory and return `""` when that failed. SQLite treats an empty filename
  as a temporary database, so the CLI and API could report successful writes
  that vanished at exit. Path resolution is now side-effect free and preserves
  the configured name; repository/open errors surface to the caller, and token
  writes reuse one resolved path for the repository, event store and replay
  protection.
- **Background workers finish before their stores close**: stopping the API or
  desktop runtime used to cancel the Oracle scheduler and audit-outbox drainer
  without waiting for an in-flight pass, then immediately close SQLite. Their
  stop functions now cancel and join the worker, repeated stops are safe, and
  restarting the scheduler waits for the previous pass to exit. The outbox also
  attempts pending audit deliveries immediately at startup instead of waiting
  for the first one-second tick.
- **Height-stamped ledger appends fail closed when unwired**: the shared
  transaction bridge used by lottery draws and Oracle observations now rejects
  a missing writer, transaction manager or callback before dereferencing it.
  The SQLite path also has direct commit and rollback regressions proving that
  the on-chain JSON and business row receive the same height or disappear
  together.
- **Blockchain search stays bounded and can page every match**: searching the
  desktop ledger used to call `LedgerBlocksPage(0, 0)` and materialise the
  whole chain before filtering and truncating it in the view. Search now runs
  over the full chain in the desktop adapter, retains only the requested page,
  returns the exact match total, and lets Previous/Next walk every matching
  entry. Unfiltered paging keeps its bounded read, and surrounding whitespace no
  longer repeats the same successful query on every repaint.
- **Ledger payload search reaches past the visible cell**: the table still
  clips a long entry to 96 characters, but search now examines the complete
  decoded payload, so a value in a long JSON note or after two full token keys
  remains findable. Clipping also counts characters instead of bytes, so a
  Chinese or emoji payload can no longer end in an invalid UTF-8 fragment.
- **The ledger names every module's recorded payload**: vote blocks, Oracle
  observations and token `transferfrom` blocks used to appear under an em dash
  even though the other payload shapes were recognised. They now render as
  `voting`, `oracle` and `token`, which also makes their rows searchable by
  module and lets Recent activity route them to the owning page.
- **Token reads reject malformed public keys**: a base64 owner or spender that
  decoded to the wrong length used to read back as a zero balance, zero
  allowance or empty history for a real token. Balance, allowance and
  owner-filtered history now return `INVALID_PUBLIC_KEY_LENGTH` (and an empty
  required key returns `PUBLIC_KEY_REQUIRED`), while an omitted history owner
  still selects the whole token stream.
- **Token history can read the whole token, not only one owner**: the desktop
  History card promised that an empty Owner field would page the token's entire
  transfer stream, but the request reached an owner-scoped reader as a
  zero-byte key and returned an empty result. Token-wide reads now apply
  `LIMIT`/`OFFSET` to `token.transfer` events only, and the same optional-owner
  contract is available from `GET /api/v1/token/history` and
  `aurora token history`.
- **The command palette searches and renders its translated labels**: a Chinese
  window drew translated page names and purposes but filtered the list against
  the English source strings, so typing a name visible on screen ("设置") found
  nothing. Search now matches both source and display text, and the palette's
  refresh/backup/export/import commands have Chinese titles and descriptions
  instead of falling back to English.
- **NFT write cards keep their own subjects**: Transfer and Burn now remember
  which NFT each holder field was filled for. Typing or selecting an id in one
  card clears only that card's previous key and recipient, Inspect moves the
  matching cards to the owner it resolves, and a successful Transfer no longer
  retargets a Burn aimed at another asset. Inspect metadata and history also
  render only while their id inputs still match the loaded record.
- **A failed ledger query no longer repeats on every repaint**: changing the
  Blockchain search or Show value triggered a read from the view, but a failure
  left the requested and loaded keys different, so each frame issued the same
  failing query again. The page now records the attempted query, and its
  Previous/Next offset commits only after the rows arrive; a failed page move
  keeps the label over the rows it actually loaded.
- **Stored lifecycle labels follow the desktop language**: voting-session
  states (`draft`, `active`, `ended`, `closed`) and NFT history operations
  (`mint`, `transfer`, `burn`) were domain values drawn verbatim, so Chinese
  tables still showed lowercase English. Those values now use the same
  translation table as the labels around them.
- **Successful empty readouts remain open**: Token and NFT histories, an empty
  voting tally and an empty Oracle Query now remember the successful read even
  when it returned no rows. Refresh, selecting another subject, casting the
  first ballot or fetching the first value re-reads those cards instead of
  leaving "No … loaded yet" beside a result that actually loaded.
- **Typing another voter key clears the old ballot secret**: the Cast Vote card
  now tracks the public key its private key and result belong to. A direct edit
  to Voter Public Key drops the previous voter's secret and "Ballot recorded"
  sentence, just as choosing a different voter row already did.
- **Oracle presets do not inherit another source's credentials**: choosing a
  built-in preset now clears the Add Source form's request headers. The presets
  define no headers, so a private Authorization value from a previous form
  could otherwise be stored against the newly chosen public endpoint.
- **Empty reads say what was empty**: a successfully loaded token history, NFT
  history or session tally now reports "no transfers", "no operations" or "no
  votes" for that subject instead of claiming nothing had been loaded.
- **A desktop write moves the context to the record it changed**: adding an
  Oracle source now targets that source in the action and reading cards,
  minting or transferring an NFT moves the holder fields with the asset and
  clears the previous secret/recipient, and a selected token keeps its owner
  available even when the token row is outside the current paged list.
- **Voting session controls use the same guarded write path as every other
  mutation**: Start/End Session now run in the background with the busy banner
  and temporary form lock, instead of blocking the frame and leaving no sign
  that a write was in progress.
- **Delete-source confirmation resolves sources outside the visible page**: the
  Oracle action field accepts an id from the CLI or an earlier session, but the
  dialog named only the loaded page's rows. It now looks the id up through the
  shared service, so an older source is identified by name and id before its
  schedule entry is removed.
- **Lottery verification evidence follows the verifier input**: editing the
  draw id after a check left the previous verdict, or its lookup error, under
  the new value until Verify was pressed again. Both now render only while the
  field still contains the exact query that produced them; selecting or
  creating a draw clears the old attempt.
- **The Token page follows a manually typed token id**: direct edits to Token
  ID now clear the previous owner, private keys and recipients before filling
  the owner fields from the matching listed or resolved record. Inspect,
  Balance, History and Allowance answers and their warnings render only for
  the token/account/query currently shown, and Refresh re-reads an open
  Allowance card like the other reads.
- **A failed automatic tally read is no longer retried on every frame**: opening
  Results with a session whose result query fails used to leave the attempted
  id blank, so each repaint hammered the same database read. A failure now
  records the attempted session, keeps the error visible, and retries only
  through Show Results, Refresh, or a new subject.
- **The Chinese desktop no longer leaks two operational labels in English**:
  the banner shown after a write runs for 20 seconds and the ledger's
  “Entries X–Y of Z” count now use the desktop translation table like the rest
  of the window.

- **Session rosters resolve candidate names beyond the visible page**: the
  desktop candidate list pages at 100 rows, but a selected session's audit
  roster previously looked up names only in that page and fell back to opaque
  ids for every older candidate. The roster now resolves its own ids on demand,
  de-duplicates them, and keeps names from the same session across a failed
  refresh; switching sessions clears the previous roster before the new read.

- **Desktop collection pagers stop at an exact page boundary**: Token and NFT
  collections fetch one row beyond the visible page, so a collection whose size
  is exactly 100 no longer offers a Next button that opens an empty page and
  bounces back to the beginning. Searching for a row that is not on the current
  page also leaves the pager available instead of hiding the only way to
  continue.

- **Switching pages starts at the top**: the page body now tracks its scroll
  position explicitly and `setPage` resets it. The previous container relied on
  the old page's height clamping the offset; once the dashboard and a guide
  card changed the heights, a new page could open a few dozen pixels down.
- **Desktop detail cards show the fields the records actually carry**: the
  session adapter dropped `Description`, a selected candidate never showed its
  program, NFT Inspect hid the image URL, token URI and mint block, and a
  selected oracle source never showed its endpoint or polling interval. Each
  is now rendered from the stored record; the long URLs use the copy row, and
  tests select a row and assert the text/clipboard.
- **Voting card outcomes now say whether they succeeded**: the candidate,
  session, control and ballot results were plain strings drawn in the same
  muted colour for both outcomes, so a refused key or an invalid window looked
  like "Ballot recorded." They are `statusLine`s now and use the success/error
  colour, with tests asserting both the text and the polarity.
- **A failed desktop read can no longer leave another subject's answer on
  screen**: Token Inspect/Balance/History, NFT Inspect/History and Oracle
  Query/Latest now remember the token, account, asset, source, limit or offset
  they were loaded for. Changing the subject or query clears the readout
  before the new read, so a failure reports the error instead of showing the
  old answer under the new id; a same-subject refresh failure keeps the last
  successful rows and marks them stale. The tests fail each read after a
  selection, a query change and a Refresh and assert both the cleared/retained
  state and the warning.
- **The desktop Feed Health card reads the running scheduler**: the adapter
  returned an empty list unconditionally, so the Oracle page always said
  "(no feed activity yet)" while `GET /api/v1/oracle/health` had counters. The
  startup path now exposes the scheduler's per-source stats to the window,
  resolves each source name from the list the page already holds, and an
  integration test adds a source and waits for the scheduler attempt to reach
  `App.Services().OracleHealth()`.
- **Desktop token history shows the replay nonce**: `TransferResponse` omitted
  the nonce and the adapter would not have carried it anyway, so every row in
  the Token page's Nonce column read 0. The use case now includes
  `TransferEvent.Nonce`, the adapter maps it, and both halves are pinned by
  tests.
- **A failed tally read no longer shows another session's results**: selecting
  a session, refreshing, or re-reading after a ballot ignored the error, which
  left the previous session's rows beside the new id with no warning. The
  Results card now tracks which session its rows describe, clears them when
  the subject changes, keeps same-session rows and marks them stale on a
  failed refresh, and reports the failure instead of "No tally loaded yet."
- **A failed oracle preset read no longer hides its cause on the desktop**: the
  template catalog error was dropped, leaving a stale or missing preset picker
  with no explanation. The Add Source card now shows the stale warning above the
  previous presets (or none) and stays usable; the test fails the read and
  asserts the warning and the surviving form.
- **A failed feed-health read is visible on the desktop Oracle page**: the health
  counters were only assigned on success and the error was dropped, so a
  transient read failure left old counters on screen with no sign they were
  old (every other oracle list reported its failures). The card now draws the
  standard stale warning above the last rows; the test fails the read and
  asserts the warning, the reason and the retained row.
- **A failed desktop verification clears the previous verdict**: the Verify card
  kept the last draw's "Verified" panel while the field named a new id and the
  new lookup failed, so old evidence looked current. A new attempt clears the
  verdict before looking, leaving only the field error on failure; the test
  makes the second lookup fail and asserts the verdict is gone.
- **An import refreshes the page behind its result dialog**: importing a lottery
  export through the File menu, a drop or a forwarded second launch stored the
  rows but left the lottery history as it was, so dismissing "Imported N"
  revealed the pre-import list. Each entry point now reloads the current page
  (through `App.Reload`, which is the ordinary page refresh) and invalidates
  the window when rows were stored. `TestOpenFileReloadsAfterImport` observes
  the shell path end to end against a real database and export file.
- **A link forwarded to a running window now repaints it**: `Shell.Navigate`
  changed the app state but never asked the native window for a frame, so a
  second launch with `aurora://voting` left the old page on screen until the
  next click. Measured under Xvfb: before the fix the screenshot after the
  second launch was byte-identical to the token page; with `win.Invalidate()`
  it matches the voting page. The Go menu's page actions invalidate the same
  way, since a menu click is also outside the view's event loop.
- **A dashboard deep link now switches to the dashboard**: the parser spelled
  `aurora://` and `aurora://index` as an empty page, and ShowIntent treats an
  empty page as "leave the window alone", so the link did nothing when the
  window was on another page. Both spellings resolve to `dashboard`, which
  ShowPage selects (and still accepts `index` for callers using the old
  document name).
- **The desktop token history can page past its first window**: the card had a
  Limit but always sent offset 0, so a transfer stream longer than one page was
  unreachable from the window although the repository, CLI and REST all page.
  It now has an Offset number input, included in the post-write/refresh reload,
  and the test asserts the limit and offset the card sends.
- **The desktop ledger shows each block's real height**: the adapter numbered
  the filtered payloads by their index, so the first recorded block displayed
  height 0 and every row was one too low (genesis has no payload and is not in
  the slice). `BlockChain.BlockEntries` now pairs each payload with its block's
  own height and can stop after the newest N without materialising the chain;
  the desktop dashboard uses that limit. `TestBlockEntriesCarryTheirHeights`
  and `TestLedgerBlocksCarryBlockHeights` pin the numbers.
- **GET /api/v1/token/list is bounded like the other listings**: it returned
  every stored token in one response, so a key-holding caller could force an
  unbounded body while the NFT, lottery and token-history lists all paged. It
  now defaults to 20 rows, accepts `limit`/`offset`, caps the limit at 100 and
  the offset at 10000, and the reference documents the endpoint (which it had
  omitted). `TestTokenHandler_List_Paged` covers the page, the cap and the
  beyond-the-end case.
- **Choosing an NFT fills the holder the write cards need**: Transfer's From and
  Burn's Owner were left as typed while the row already knew the asset's
  current owner, so a stale or empty key could be signed with. Selection fills
  both from the chosen row; the selection test asserts the owner it sends.
- **The desktop balance and history cards no longer share one account field**:
  typing an owner in History silently changed the account the Balance card would
  read, while its readout still described the earlier account. The two inputs
  are independent and each refresh path reads its own value;
  `TestBalanceAndHistoryUseTheirOwnAccount` sends different accounts and checks
  what each card asked for.
- **The desktop oracle preset cannot get stuck on an emptied form**: adding a
  source cleared the fields but kept the preset selected, so the same preset
  could not be applied again (a combobox reports changes only when the choice
  changes), and a preset with no interval left the previous one's value behind.
  The choice is cleared with the fields and every preset sets the interval it
  defines, including 0.
- **Choosing a desktop row now moves the readouts with it**: the Token Inspect,
  balance and history cards, the NFT detail/history cards and the oracle
  Query/Latest cards and an open voting tally kept describing the previously
  selected record while the id fields already named the new one; each now
  re-reads the chosen id; creating a token or NFT moves the open readouts to the
  new record, and a draw selection or creation clears the previous verdict
  instead of leaving it under a new id. `selection_test.go` records the id every read was asked for.
- **Refresh reloads the readouts that are on screen**: Ctrl/Cmd+R (and the
  Refresh button) reloaded the lists and counters but left an open Inspect
  readout, a shown balance, a loaded token/NFT history, the oracle Query and
  Latest cards and a voting tally at their previous values. Each page now
  re-reads exactly what it displays, through the same helpers the writes use.
  `TestPageRefreshReReadsOpenReadouts` counts every one of those reads.
- **The desktop clears a signing key after it is used**: token mint, transfer,
  approve, transfer-from and burn, and NFT transfer and burn, left the private
  key in its input after a successful write, so a shared screen kept showing
  the secret. Each success clears its field (a failure keeps it for the retry),
  and the refresh tests assert the field is empty after the write.
- **The desktop outcome line no longer scrolls away**: each page drew its
  `statusLine` as the first child of the scrolling body (and the Token page
  drew it inside the left column), so an action near the bottom of a form could
  report "Failed" above the viewport while the form looked unchanged — easy to
  read as success. The line is now rendered once in the fixed chrome below the
  page header, chosen by `pageStatus()`; `design_test.go` scrolls the body and
  fails if the line moves.
- **Desktop pages re-read what a write changed**: a successful mint, transfer,
  approve, burn, NFT write or oracle fetch left the collection, the Inspect
  readout, the loaded history, an open tally and the Query/Latest cards at their
  pre-write values, so "Done" sat beside stale numbers and the write looked like
  it had done nothing. Each success now refreshes exactly what the page is
  showing (a closed readout is not fetched), deleting a source clears the cards
  that were showing its readings, and a failed refresh keeps the previous rows
  rather than replacing a successful write with an error. `refresh_test.go`
  counts the read calls through the fake services and fails when a refresh
  disappears.
- **Desktop destructive confirmations name what they destroy, and copies are
  confirmed in place**: the token burn dialog said only "Burn tokens?", the NFT
  dialog "Burn this NFT?" and the oracle dialog "Delete this source?", leaving
  the operator to trust whichever values the shared id/amount fields held
  before an irreversible write. They now name the amount and account, the
  asset (its name when Inspect loaded it, its shortened id otherwise) and the
  source. Copy buttons also raise a `c.Toast` ("Private key is on the
  clipboard") because the page's status line sits above the cards and is often
  scrolled out of view; `confirmation_test.go` drives all four dialogs/toasts
  through `ui.Tester`.
- **The desktop session form edits local dates and times, not epoch seconds**:
  creating a voting session from the window required typing two Unix
  timestamps, which needed a clock and a converter outside the app. The card
  now uses the toolkit's `DateInput`/`TimeInput` for start and end, opens with
  now → +24h so it is submittable as it appears, rejects an end before the
  start before calling the service, and still sends Unix seconds so the CLI
  and REST surfaces see the same window.
- **The desktop Voting page can reach the ids it asks for, and one-time
  values can leave the window**: the page loaded every candidate but never
  rendered one, so a candidate id appeared only once in the "Registered …"
  line and was unrecoverable after a restart; the session table shortened
  UUIDs and had no row selection, so the controls, ballot and tally reader
  could not be filled from the window either. A Candidates card now lists
  ID/name/party and choosing a row loads its id into the ballot, choosing a
  session row loads the full id into all three session forms, and the
  registration cards offer Copy buttons for the new candidate/session ids.
  Voter registration likewise keeps the generated key pair on the card with
  Copy buttons: the store keeps only the public half and native text is not
  selectable, so the one copy of the signing key previously could not be
  kept. `voting_test.go` drives both tables and asserts the clipboard through
  `ui.Tester.Clipboard()`.
- **The CLI reference documents every command, and only real ones**:
  `docs/cli.md` never mentioned `aurora token list` even though the command
  existed, and its NFT section still presented `--owner` as required after
  `nft list` learned to list the whole collection (the help text still said
  "List NFTs by owner"). The reference now lists both spellings, and two
  repository checks walk the command tree parsed from `cmd/aurora/cmd`: one
  fails for a command with no documented line, the other for a documented
  `aurora …` line that names no real subcommand.
- **The CLI's global flags and the repair prompt are localized too**: a Chinese
  `--help` translated every module flag but left `--config` and
  `--http-timeout` in English (they were hardcoded in `root.go`), and
  `migrate repair` reused its long description as the `--confirm` action
  phrase, so the prompt read "此操作clears the dirty marker a failed migration
  left (it does not undo or re-run that migration)". Both flags now come from
  the catalogs, repair has the noun-phrase action key the other destructive
  commands use, and the superseded `migrate.repair_action` entry is gone. The
  translation-key check now also inspects the action keys passed to
  `requireConfirm`/`addConfirmFlag`, which are rendered dynamically at a
  destructive command's prompt.
- **Two `--help` screens no longer print raw translation keys**: the lottery
  module's `Short` pointed at `lottery.tui.title`, which no catalog carries, so
  `aurora --help` listed the module as `lottery.tui.title`; `migrate repair`
  printed `migrate.repair_confirm` as its description in every language. The
  lottery module now uses the `<module>.cmd` key the other modules use, and
  repair reuses its existing `migrate.repair_long` text. A new repository check
  (`internal/repocheck`) scans non-test sources for `GetText`/`T` calls with a
  literal key and fails when either catalog lacks it — the defect class is
  invisible at compile time and only shows up in a user's terminal.
- **Lottery history is newest-first even within one second**: `GetAll` ordered
  by `timestamp DESC` alone, and timestamps have one-second resolution. Draws
  created in the same second tied, so SQLite returned them by insertion order
  — an older draw imported after a newer one came first, and two paged requests
  could repeat or skip a row. The order is now
  `timestamp DESC, block_height DESC, id`: block height carries the chain
  append order and the id makes it total. Migration `000007` adds the matching
  `idx_lottery_recency` and drops the timestamp-only index (subsumed by the
  composite leading column); the repository bootstrap creates the same index,
  and the query-plan test fails if the sort reappears. The migration walk tests
  now derive their version expectations from `migrations/`, so adding a
  migration no longer turns unrelated command tests red.
- **A failed desktop refresh no longer hides the content it already loaded**:
  every loader keeps its previous rows when a read fails, but each view
  returned on the error line alone, so a transient database failure blanked
  the list the operator was reading. Voting was worse: it recorded the failure
  and never drew it, and with no stored sessions it claimed "No sessions yet"
  for a store it could not read. A failed refresh now reports the reason above
  the last successful content and labels those rows as stale; with nothing
  loaded yet it reports the failure instead of an empty-state note. The
  dashboard's failed first load also stops leaving its spinner running forever,
  because nothing would ever finish it. `load_failure_test.go` drives every
  page through a successful load, a failing refresh and a recovery.
- **Lottery verification no longer panics on a missing draw**: the resolution
  path assumed a repository either returns the record or `lottery.ErrNotFound`,
  but the `lottery.Repository` contract also permits a nil record with a nil
  error (in-memory fakes and the oracle module's documented convention). That
  spelling reached `VerifyRecord`, which dereferences the record, so
  `aurora lottery verify`, the desktop verifier and `GET /api/v1/lottery/{id}/verify`
  crashed instead of reporting "not found". A nil record now falls through to
  the block-height and substring steps, nil entries in those result slices are
  skipped, and a genuine read failure still propagates instead of being folded
  into "not found". `verify_resolution_test.go` pins each step directly — the
  path was previously covered only through CLI/desktop integration.
- **User docs no longer describe the deleted Web UI**: README, AGENTS and the
  API, CLI, desktop, ablation, privacy and lottery guides still advertised
  static pages, Alpine.js, a CSP nonce bootstrap and an embedded UI location.
  The gateway serves no static files, so the entry table, the REST section, the
  stack list and the About dialog now say what ships: a native window plus a
  JSON-only gateway. The About dialog's dead `Web UI: (embedded in the app)`
  line and its `desktop.dialog.about_embedded` key are gone, and a regression
  test fails if a UI location is reported again.
- **Per-source history reads no longer sort before limiting**: Oracle history
  (`GetDataBySource`) and NFT operation history
  (`GetOperations`) filter on the parent id and order by `timestamp DESC`.
  With only the single-column `source_id`/`nft_id` indexes, SQLite built a
  temporary B-tree over every matching row before the `LIMIT` could apply —
  measured with `EXPLAIN QUERY PLAN` as `USE TEMP B-TREE FOR ORDER BY` on all
  three queries. Migration `000005` adds `idx_oracle_data_source_ts` and
  `idx_nft_ops_nft_ts` (both `(parent, timestamp DESC)`); the same plan output
  now walks the index directly, so the `LIMIT` is a real early stop. The
  repository self-bootstrap and the `nft_operations` cascade-FK rebuild create
  the same indexes, so a fresh database, a migrated one and a rebuilt one
  agree. `internal/infra/sqlite/query_plan_test.go` pins both plans and fails
  if the sort reappears.

### Changed

- **The desktop is on MyGo v0.7.1**: the nested module upgraded from v0.6.3 and
  its Go directive to 1.27.3, and `just desktop-deps` / `just desktop-update-mygo`
  make the fast-moving dependency deliberate to check and refresh. Renovate
  now schedules MyGo updates at any time with a desktop label.
- **The ledger subsystem is no longer presented as a domain package**:
  `internal/domain/blockchain` moved out of domain and now lives at
  `internal/infra/blockchain`, where its SQLite
  bootstrap, singleton and file-permission work no longer blur the pure
  `domain` boundary. The package API and behavior are unchanged; a new
  `docs/architecture.md` records the dependency contract and the remaining
  refactor order.
- **Desktop UI copy now lives beside the views**: the desktop catalog moved
  from `internal/i18n` to `desktop/internal/nativeui/messages_zh.go`; the core
  translator keeps CLI/service keys only, and `tr`/`Translate` falls back to it
  for service errors and menu entries. `LoadLocaleFile` now writes to a cloned
  map instead of a package-level catalog.
- **The layer contract is executable**: `internal/repocheck/layers_test.go`
  parses production imports and fails if domain reaches infra/app/api, a use
  case imports `infra/sqlite`, or infra imports its callers. The architecture
  document is now backed by a test instead of review memory.
- **The storage composition root is now explicit**: `internal/app/container`
  opens the database, repositories, event bus, replay protection, token service
  and ledger singleton, and owns their close order. `api.Server` delegates
  `Services`/`Close` to it and only builds HTTP handlers and the oracle
  scheduler; the desktop continues to receive the same container instance.
- **The ledger API now reads as one concept**: `BlockChain` became `Chain`;
  `NewBlockChain`/`NewBlockChainWithDB`/`InitBlockChain`/`GetBlockChain`
  became `New`/`NewWithDB`/`Init`/`Get`; and `AddLotteryRecord` /
  `WithLotteryRecordTransaction` became `AddRecord` / `WithRecordTransaction`.
  The old names implied a lottery-only writer and a type that was really the
  shared ledger.
- **Desktop file import/export shares the runtime repository**: `Shell` takes
  the container's `lottery.Repository`, so File → Export/Import and a dropped
  export no longer open a second SQLite writer beside the app. An unwired shell
  reports "lottery repository is not ready"; the SQLite fallback is gone.
- **The API/desktop ledger no longer uses a process singleton**: `container.Open`
  calls `blockchain.OpenDB` and `NewWithDB` and owns the DB/Chain handles; only
  the CLI keeps the `InitDB`/`Init` convenience singleton. This lets API tests
  and desktop runtimes close their ledger without affecting another handle.
- **The ledger implementation now lives under infrastructure**: the
  SQLite-backed `blockchain` package moved to `internal/infra/blockchain`;
  `internal/ledger` holds the pure writer/transaction contracts and report
  types. The desktop adapter uses a narrow `ledgerChain` interface, so
  `native_*.go` no longer imports the implementation.
- **The token service is split by operation**: service.go now holds the
  contracts, constructor, transaction helpers and request types; queries, mint,
  transfer, approval and burn live in service_*.go files. The 811-line file is
  now a 223-line core plus focused units.
- **The oracle scheduler has an application runtime**: `oracleapp.Runtime` owns
  the fetch use case, scheduler, cancellation and stats/metrics view. The HTTP
  server only starts it and hands the same stats provider to the handler.
- **Desktop commands and long lists have real scope**: the Ctrl/Cmd+K palette now
  runs Refresh, Back up, Export and Import as well as pages. The ledger and NFT
  collection page through limit/offset (Previous/Next); a write disables only
  the page body, and a slow write shows a timeout reassurance.
- **Desktop forms gate their primary actions**: Create/Mint/Transfer/Approve/Cast
  buttons stay disabled until their required fields are present, and an
  in-flight write disables the frame so edits cannot race its result.
- **Desktop titles and outcome lines are explicit**: the native window title
  follows the visible page (and language), and the fixed status strip can be
  dismissed so a stale "Done" does not describe the next action.
- **Desktop writes have a busy state**: the real window moves service writes off
  the frame thread, paints a spinner/“working” banner in the fixed chrome, and
  applies the result on the frame thread. Tests keep the synchronous path.
- **Desktop has a command palette and richer empty states**: Ctrl/Cmd+K opens a
  keyboard-first page switcher (type to filter, arrows to move, Enter/Escape);
  Token/NFT/Oracle empty lists show the module icon, title and next move. The
  sidebar narrows from 216px to 184px below 1000px.
- **The dashboard counters reflow for the minimum window**: three cards per
  row on wide windows, two below 1100px, one below 760px, so the label/note
  pair does not wrap inside a 200px card.
- **The dashboard now leads with state and acts as shortcuts**: metric cards
  open their module, a recent-activity row opens the page that wrote it, and a
  non-empty dashboard shows counters/activity before the Quick start card (the
  empty dashboard keeps Quick start first). Live pages carry a `Live · 10s`
  badge, and Settings documents the window shortcuts.
- **Desktop backup uses the container port**: `Shell` takes a backup function
  from `container.Services` instead of constructing `infra/backup` itself;
  `deskapp` production code now has no `internal/infra` import.
- **HTTP handlers no longer import infrastructure**: lottery/NFT handlers take
  ledger writers, the blockchain handler takes an integrity verifier, the
  oracle handler takes an injected fetcher, and the rate-limit error moved to
  `oracle.ErrRateLimited`. The outbox drainer moved to the container, so
  `internal/api` production code has no `internal/infra` import.
- **Presentation layers see a narrow service port**: `container.Services`
  exposes domain repositories, token service, transaction managers, a ledger
  view and the oracle fetcher; `api.Server.Services()` no longer returns the
  concrete container or any `*sqlite.*`/DB handle.
- **Handler transaction managers come from the container**: the container now
  exposes `VotingTx` alongside `NFTTx`, so `api/handlers.go` no longer imports
  `infra/sqlite` or constructs transaction managers.
- **API handler assembly has its own seam**: `internal/api/handlers.go` builds
  the six handlers from a container; `Server` keeps lifecycle, metrics and the
  scheduler. Router tests may still construct a handler-less server without a
  nil dereference.
- **Transaction management is one shared port**: token, NFT and voting domain
  packages now alias `ledger.TransactionManager` instead of declaring three
  identical interfaces. Existing mocks and constructors stay source-compatible.
- **The ledger has a pure port package**: `internal/ledger` now owns
  `BlockWriter`, transaction writer interfaces and the transactional append
  helpers. Domain services, application use cases and handlers depend on that
  port; only `app/container` imports the SQLite-backed
  `internal/infra/blockchain`.
  The layer test forbids the old direction.
- **Application use cases no longer construct an HTTP client**: `oracleapp`'s
  default `http.NewFetcher` moved to the composition root; the API, CLI and
  desktop inject a fetcher, and a nil fetcher returns a clear error instead of
  panicking. The layer check now forbids every `internal/infra` import outside
  `app/container`.
- **The view package's widget grab bag is split by responsibility**:
  `components.go` became `controls.go`, `layout.go` and `values.go`, with the
  package comment in `doc.go`. No behavior changed.
- **The view package separates page identity and status from widgets**:
  `pages.go` owns the page enum/IDs/order, `status.go` owns the success/failure
  line and error localization. `components.go` now holds only controls and
  layout primitives.
- **The view package separates its data contract from its state machine**:
  `nativeui.Services` moved to `services.go` and the row DTOs to `rows.go`;
  `app.go` keeps the App state, construction and navigation. The contract now
  names the desktop adapter, not `api.Server`.
- **The desktop adapter now depends on domain ports, not SQLite types**:
  `nativeServices` fields/constructor use `lottery.Repository`,
  `voting.TransactableRepository`, `nft.TransactableRepository` and
  `oracle.Repository`; the container injects the NFT transaction manager.
  `ports_test.go` keeps `native_*.go` from importing infra/api again.
- **The desktop service adapter is split by module**: `native_services.go`
  keeps only the struct and constructor; lottery, ledger, voting, token, oracle,
  NFT and key-pair mappings live in their own files. The 779-line adapter is
  now an 80-line seam plus focused files, so a domain mapping change has one
  obvious home.
- **Application use cases no longer import SQLite for error classification**:
  the generic not-found sentinel now lives in `internal/domain/repoerr`, and
  SQLite keeps compatibility aliases for it and `voting.ErrAlreadyVoted`.
  `oracleapp`/`votingapp` depend on domain errors only, so the storage
  implementation can change without editing a use case.
- **The whole desktop follows the language switch, not only its menu**: page
  titles and purposes, navigation, buttons, field labels, hints, empty states,
  table headers, guides, dialogs and status lines go through `nativeui.tr`.
  English literals are the keys, so a missing entry falls back to readable
  English; Chinese translations live in
  `desktop/internal/nativeui/messages_zh.go`. Known service errors are translated by
  their message (a wrapped "seed: seed too short" keeps its prefix); unknown
  or dynamic diagnostics stay verbatim, and the backend log is unchanged.
- **Every module page has a "How this works" guide**: the steps state the
  order of operations in plain language, including which private key each
  write needs and that keys are never stored. The guide starts collapsed so
  the page stays task-first; it is one click away instead of leaving the
  operator to infer the workflow from card headings.
- **The desktop guides the first move and hides fields most uses do not
  need**: the dashboard now has a Quick start card ("Run a draw", "Create a
  token", "Register a voter", "Add a data source", "Mint an NFT") and each
  page title states a task instead of a module description. Token decimals,
  history paging, NFT optional metadata and oracle transport headers sit
  behind labelled disclosures; a preset opens the oracle disclosure so its
  filled fields are reviewable.
- **Selecting or creating a token fills the owner accounts and drops the
  previous secrets**: the record already knows its owner public key, so Mint
  To, Transfer From, Approve Owner, Burn From, balance and history owners start
  from it instead of asking for the same base64 key six times. Private keys,
  recipients and spenders from the previous token are cleared, and a token
  created with a generated key offers "Use generated owner key" in the owner-
  signed forms while that token is the subject; an NFT minted with a generated
  creator key offers the same shortcut in Transfer/Burn while the creator is
  still its holder.
- **Results and Readings load the object already chosen**: switching to
  Voting's Results stage loads the selected session's tally, and Oracle's
  Readings stage queries the selected source and its latest value. The
  explicit buttons remain the retry path; the loaded source/limit is tracked so
  a tab does not re-query every frame.
- **A successful create highlights its record**: token, NFT, lottery, candidate
  and session creation clear a search/filter that would hide the new row,
  select it, and move the page context to it. Before this, the status sentence
  said "Created" while the list still showed an unrelated or empty selection.
- **The desktop's dense pages became tabbed workbenches**: the Token page's
  six write forms and the NFT page's Transfer/Burn/History now show one
  operation at a time; Voting is split into Setup (register/create), Ballot
  (start/end, cast) and Results; Oracle's right column is Sources, Health and
  Readings. Scrolling past five hidden forms to reach Burn, or down four cards
  to find the tally, was work the operator did not need to do.
- **The desktop stacks its two columns on a narrow window**: below 1000px the
  second column moves under the first, so the 900px minimum no longer squeezes
  every card to about 300px. Tests pin the side-by-side layout at 1280 and the
  stacked one at 900.
- **Every id the window shows can be copied**: the active token, the inspected
  NFT, the draw in the Verify field, the selected source, session and candidate
  now carry copy rows; the tables truncate ids and native text is not
  selectable, so previously the only full id was a field that could be read but
  not copied.
- **The desktop no longer has a menu bar on Linux and Windows**: pages are
  navigated from the sidebar and refreshed from the header, so the
  File/Edit/Go/View/Window/Help bar was chrome the window did not need. The app
  installs no application menu there at all; `AutoHideMenuBar` is only a
  fallback if a host supplies one. Cmd/Ctrl+E/I/B are registered by the window;
  backup, export/import and recent exports live in a sidebar **File** menu, and
  language, data location, appearance and build facts live in a **Settings**
  page under the sidebar's Application section (Ctrl/Cmd+, also opens it).
- **Live updates are opt-in and persisted**: Settings can refresh the open
  Dashboard, Oracle health and open readings every 10 seconds; forms keep
  their text and Settings itself never starts a background refresh. The choice lives in
  `desktop-settings.json` beside the database and is loaded on the next launch;
  a failed write is shown in Settings instead of silently resetting.
- macOS
  cannot remove its global
  bar, so it keeps only the platform roles (app menu, File→Close, Edit, Window);
  Aurora's own entries appear in the window on all three platforms, so the menu
  contents no longer fork.
- **The desktop has a high-contrast variant of its palette**: when the OS asks
  for increased contrast, the same polar-night design uses brighter hairlines,
  primary/secondary text, focus ring and selection tint (`auroraTheme(true)`).
  A WCAG contrast-ratio test compares every pair against the normal theme and
  fails if the variant stops being stronger.
- **The desktop follows the OS text size**: every explicit FontSize in the view
  layer now goes through `textSize`, which multiplies the system TextScale
  preference. MyGo's theme only scaled text without an explicit size, which in
  this window was almost none of it; Reduce Motion already worked because MyGo
  sends animations straight to their target. `TestTextScaleIsHonoured` renders
  at 1.5x and fails if the header does not grow.
- **Choosing a desktop session shows who is on its ballot**: the row cannot
  carry the roster, so the selected session now prints its candidate names
  under the table (ids shortened when a candidate was deleted), which is what
  the operator needs before starting or ending it.
- **The desktop uses text areas for its long free-text fields**: the oracle
  Request Headers JSON and the NFT Description were single-line inputs; both
  are `ui.TextArea` now, so a header object can be read while it is edited.
- **The desktop numeric fields are real number inputs**: winner count, token decimals,
  oracle poll interval and the query/history page sizes are `ui.NumberInput` with
  the domain bounds and a step, instead of text parsed on submit; the interval
  ceiling is the domain's 30 days, and list sizes are capped at 1000 before they
  reach the repository. A value can no longer be rejected for its spelling.
- **The desktop menu bar has a Go menu for the seven pages**: the window's
  Ctrl/Cmd+1..7 shortcuts existed but nothing in the UI named them. A Go menu
  lists the pages in sidebar order (`nativeui.PageTitles`) with the same
  accelerators and calls the window's `ShowPage`; it is built after the app
  state exists, so `NewMenu` takes the page switcher and omits the menu when it
  is nil. `menu_test.go` pins the titles, accelerators and click target.
- **The desktop candidate list can build a session roster**: the Create Session
  card takes comma-separated candidate ids, so any roster other than "every
  candidate" meant copying UUIDs out of the shortened candidate table by hand.
  Choosing a candidate and pressing "Add to session roster" appends it (once)
  to the form's roster field, and the status line confirms which candidate was
  added; `TestCandidateCardBuildsACustomRoster` drives the clicks and creates a
  session from the built roster.
- **The desktop collections are searchable, and the tally states its total**:
  the Tokens and NFTs lists load every stored row and had no way to narrow them,
  so a long-lived store required scrolling for a name. Each list now has a
  `ui.SearchField` filtering by id/name (and symbol for tokens) as the user
  types, case-insensitively; the NFT search composes with the owner filter, a
  filtered row still fills the forms with its full id, and an empty result says
  "no match" rather than "nothing stored". The voting page's candidate and
  session lists gained the same field (id/name/party and id/title/status), as did
  the draw history (id/seed/participant/winner) and the oracle source list
  (name/id/type/URL), with selection resolved against the filtered rows so a
  search can never act on a different row than the one on screen, and a new
  query clearing the old row highlight rather than letting one index describe
  two different records. The ledger page filters by module/payload/height and
  bounds the table with a Show row limit (default 200, "Showing the newest N of
  M" when cut), while its counters still describe the whole chain. The voting
  Results card also prints the sum of the per-candidate counts, which is what a
  reader checks before trusting the rows.
- **The desktop dashboard lists recent chain activity**: the landing page showed
  six counters and a ledger verdict, which answer "how much is there" but not
  "what happened lately". A Recent activity card now lists the newest five
  chain entries (block, module, payload summary) from the application layer's
  `LedgerBlocks`, with its own failure line so a chain read error cannot blank
  the counters. The empty dashboard's onboarding line now has an "Open Lottery"
  button that selects the page, so the first move is one click rather than a
  sidebar hunt. `design_test.go` proves only the newest five are listed and the
  button switches pages.
- **The desktop window has one design system instead of ad-hoc widgets**: the
  sidebar now has a brand block, its own tone and a hairline against content,
  and every module row carries a Lucide glyph (`icons.go`) that takes the row's
  colour; the page header repeats that glyph beside the title and purpose.
  Cards are lifted with a soft shadow and a one-pixel top highlight rather than
  a border alone. Every action button comes from one of four styles —
  `primaryButton`, `secondaryButton`, `dangerButton` (destructive, still behind
  its dialog) and the small `quietButton` for copies — all `ui.ButtonBase` with
  one height, radius and weight plus hover/press colour and scale feedback,
  following GoRex and the MyGo NetEase client. Ctrl/Cmd+1…7 selects a page in
  sidebar order. The live window was captured under Xvfb to check the new
  chrome renders (sidebar and content tones differ; no panic in the log).
- **The desktop window draws native Go, and the web UI is gone**: the app used
  to host the same HTML the REST gateway served, which made a Go project depend
  on a browser engine and 3.6k lines of HTML/CSS/JavaScript — and it produced a
  defect only a browser could catch: a CSP without `'unsafe-eval'` silently
  disabled every Alpine directive, so no form in the HTTP UI could be
  submitted while the desktop masked it by stripping CSP. The window now builds
  seven pages from `desktop/internal/nativeui` with MyGo's own widgets over the
  same `api.Services` the REST handlers use. Removed with it: `web/`, the root
  embed package, the gateway's static-file branch and API-key HTML injection,
  the per-response CSP builder, the desktop CSP-stripping transport, and the
  `--web-root` flag. Irreversible actions (burn token/NFT, delete oracle
  source) ask through a native dialog, matching the CLI's `--confirm`. The view
  suite renders every page through `ui.Tester`, so the interface is testable
  without a window or an X server.
- **The five application packages took an `app` suffix**: `app/lottery` is now
  `app/lotteryapp` (and likewise voting, nft, token, oracle), so a domain
  package and its application package no longer share a name. That removed 41
  of 73 import aliases that existed only to tell them apart, and use-case
  constructors lost their stutter — `lottery.NewCreateLotteryUseCase` is
  `lotteryapp.NewCreate`.

### Removed

- **The RIL graph-engineering process and its stores**: removed
  `.agents/skills/graph-engineering`, `.planning/ril/` and the older
  `.planning/graph.json`, along with the contribution instructions that told
  agents to maintain them. The stale autonomous-loop `STATE.md` log went with
  them. Existing Git history is not rewritten; this removes the live tooling
  rather than pretending old commits never happened.
- **The four Bubble Tea TUIs are gone**: `aurora {lottery,voting,nft,token,oracle} tui`
  no longer exists and `internal/ui/` was deleted. The ablation criterion was
  not reachability but necessity: every operation those TUIs exposed already had
  a CLI command and a desktop surface, none had a capability of its own, and two
  of them (NFT/token) were in-memory sandboxes whose state vanished on exit —
  the opposite of the persistent semantics the rest of the product promises,
  which made them a trap rather than a convenience. Removed with them: the
  `charm.land/{bubbles,bubbletea,lipgloss}` dependencies, 234 TUI-only i18n
  keys, and the tests that existed only to pin TUI presence. The interactive
  surface is the desktop app, sharing the same app layer and SQLite database as
  the CLI. Net change: about 7.9k lines
  removed; `just test` drops from 34 packages to 29 and stays green.

- **Dead APIs and duplicate implementations found by a global ablation pass**:
  a symbol-level audit (every exported function/method with zero non-test
  callers) replaced the previous module-level reachability check. Removed:
  `http.NewFetcherWithConfig` (a third near-identical constructor),
  `Fetcher.FetchDataWithValidation`, `blockchain.MineBlockWithContext`
  (folded into `ProofOfWork.Run(ctx)`), `blockchain.ValidateProof`,
  `blockchain.Serialize`/`Deserialize`, `token.ValidateNonce` together with
  `ErrNonceTooLow` and its unreachable `NONCE_TOO_LOW` 409 mapping,
  `token.NewMintEventFromData`/`NewBurnEventFromData`, `lottery.VRFOutputToBytes`,
  `lottery.GetWinners`, and `ApproveEvent.SetExpiresAt`.
- **The event `StatsHandler` no longer runs**: `server.go` subscribed it to
  every event, but nothing ever read the counters — each publish paid a mutex
  lock and map increment for a value with no consumer. The subscription is
  gone; audit persistence is unchanged.
- **Deliberately kept, with reasons recorded in `docs/ablation.md`**:
  `Operation.IsTransfer`/`IsBurn`/`IsMint`, `RateLimiter.Remaining`/`Reset`,
  `LotteryRecord.FromJSON`, and `Fetcher.FetchData` all have real production
  or regression-test consumers.

- **The desktop shell now uses MyGo v0.6.3**: the patch release brings
  Windows WebView2 event-token and task-dialog lifetime fixes, WebKit content
  rules, and UI paste/rendering corrections. The Linux package and its real
  Xvfb startup smoke test pass with the upgraded framework.
- **The TUI product boundary is explicit**: all four reachable TUIs remain, but
  lottery/oracle are durable SQLite interfaces while NFT/token are intentionally
  sandboxed in-memory interfaces. The sandbox menus keep their localized warning,
  now pinned by regression tests, and the CLI/ablation docs direct durable
  NFT/token work to the CLI, REST/Web, or desktop surfaces.
- **SQLite storage is now pure Go**: Aurora uses `modernc.org/sqlite`
  instead of the cgo-only `mattn/go-sqlite3`. The hardened DSN (foreign keys,
  immediate write transactions, 5s busy timeout) is unchanged. CLI, API,
  Docker and desktop builds now run with `CGO_ENABLED=0`; cgo is only
  needed when the race detector is enabled during testing.

- **Cancelled Oracle REST fetches now stop their upstream request**: the
  protected `/oracle/fetch` handler called the background-context use case, so
  a disconnected client left the outbound request running until the fetcher
  timeout. The handler now passes `r.Context()` through to the context-aware
  fetch path, matching the scheduler's shutdown behavior.
- **Multi-database restores are now all-or-nothing**: Restore used to replace
  each configured database as it walked a map, so a later alias refusal or
  preparation error could leave earlier databases restored, and a stale-WAL
  cleanup failure could return an error after archive data was already live.
  Restore now requires the archive and configured database set to match,
  preflights every destination, stages every snapshot and replacement before
  promoting any of them, and rolls back all promoted files on a later failure.
- **CLI and API SQLite state is owner-only too**: the default `0644` umask
  made the live database and its WAL/SHM files readable by other local users
  even though they can contain Oracle bearer headers and operator state. Every
  live SQLite constructor now tightens the database and existing sidecars to
  `0600`; an existing world-readable database is corrected on its next open,
  while sidecars that SQLite has not created yet remain optional.
- **Oracle fetches now enforce the SSRF boundary at connect time**: source
  validation resolved a hostname before the request, but `net/http` resolved it
  again while dialing, leaving a DNS-rebinding window into loopback, private,
  link-local or cloud-metadata addresses. The fetcher now checks the concrete
  IP immediately before each direct connection, closing that application-side
  gap for both initial requests and redirects.
- **Failed in-memory transactions no longer leave phantom ledger rows**: the
  token TUI's nil-transaction fallback persisted its audit block before running
  a transfer that could still fail. A hash-qualified compensating delete now
  removes that exact row on failure, so a later successful transfer reuses the
  same height instead of re-syncing the failed block into memory.
- **Rolled-back domain writes no longer leave on-chain audit blocks**:
  token, NFT, voting, lottery and Oracle writes share `aurora.db`, but each
  appended its audit block before the business SQLite transaction committed.
  A losing concurrent write therefore left a permanent block for an operation
  that never happened (for example, an insufficient-balance transfer or a
  double vote). The block and domain row now commit or roll back together;
  the chain keeps its write lock through commit, appends to memory only after
  success, and still retries DB-authoritative height conflicts. Real SQLite
  regressions cover all five subsystems, and fault injection restores the
  pre-transaction ordering to prove each one detects the phantom block.
- **Desktop state is private to the current user**: the SQLite database can
  contain operator-supplied Oracle bearer headers, but the default app tree
  was created with mode `0755` and its database/log files with `0644`. The
  desktop now creates and tightens its default Unix data directory to `0700`
  before storage opens, including directories left by older releases, and
  keeps the SQLite/WAL/SHM and log files at `0600`. An explicit `--data-dir`
  keeps an existing directory's permissions.
- **Desktop database backups are private too**: File → Back Up Database used
  to create its timestamped backup directory as `0755`, the VACUUM INTO
  database snapshot as `0644`, and metadata as `0640`. The same Oracle bearer
  headers therefore remained readable from the backup even after the live
  data tree was protected. New backup sets are now `0700`; their database and
  metadata files are forced to `0600`, while an explicitly reused output
  directory keeps its existing permissions. Restore now applies the same
  owner-only contract to its hidden `.pre_restore` undo snapshot and to the
  promoted live database, including staging directories left by older
  releases.
- **Concurrent release runs are serialized by tag**: a tag push and one or
  more manual retries use the same non-cancelling concurrency group, so the
  publisher cannot interleave binaries, checksums, or provenance from different
  runs on one release page.
- **Manual release retries cannot mint a tag from an arbitrary ref**: the
  release workflow now validates `workflow_dispatch` input as an existing
  version-shaped tag before either build matrix starts. The build jobs receive
  read-only repository access, and publishing waits for the tag check, so a
  branch or commit cannot be paired with a newly created tag at another
  commit.
- **Pure-Go release gates now inspect every shipped executable**: release
  verification reads Go build metadata from the CLI and all desktop binaries,
  rejecting any artifact built with cgo, and restores the executable bit on
  the downloaded Linux CLI before running its `version` smoke test.
- **CLI release builds no longer occupy native macOS/Linux runners**: all four
  non-Windows CLI targets cross-compile with `CGO_ENABLED=0` on one Ubuntu
  runner. Windows keeps a native runner solely for Authenticode signing.
- **Sandbox checks no longer require cgo by default**: `just desktop-check`
  and the main desktop CI gate run their build, vet and tests with
  `CGO_ENABLED=0`; `just desktop-race` and a separate CI step explicitly opt
  in to the race detector's cgo runtime.
- **The migration engine is now SQLite-only**: the repository no longer
  depends on `golang-migrate` and its multi-database backend graph. A small
  in-tree runner preserves the `schema_migrations` table, ordered up/down
  steps, dirty-state refusal, hardened DSN and CLI output contracts. It
  rejects duplicate version files and missing down files instead of silently
  choosing one or skipping a rollback.

## [v1.99] - 2026-10-21

### Added

- **The desktop shell speaks Chinese end to end**: the menu, the file
  dialogs, every result and error box, the startup-failure dialog and
  `aurora-desktop --help`/`--version` now read from `internal/i18n` instead of
  hardcoded English, and follow the language MyGo resolved for the app
  (`App.Locale()` / `OnLocaleChanged`). MyGo already translates the labels of
  the items that carry a role — its own `文件`/`编辑`/`显示`/`窗口`/`帮助`,
  the native `好`/`取消` buttons — so a zh-CN session used to show those
  beside Aurora's English items and dialogs in one window. Both MyGo catalogs
  are linked (`locales/zh-Hans` and `locales/zh-Hant`), and `SetLanguages`
  runs before `Start` so the startup-failure dialog is translated too. A
  configured `[i18n] locale` (used when `$LANG` is unset) now reaches the
  window as well as the CLI.
- **The test suites no longer depend on the machine's language**: the CLI and
  TUI tests assert English output while `internal/i18n` follows `$LANG`, so a
  `go test ./...` run on a `LANG=zh_CN` machine failed `cmd/aurora/cmd`
  (backup, lottery, ...) and `internal/ui/components` on wording that had
  nothing to do with the code under test (reproduced on the previous commit).
  Each package pins its language in `TestMain`, and the lazy-init test sets
  the `LANG` it is about.
- **The reasons behind a failed launch are translated too**: the startup
  dialog's title had been localized but its body still carried English from
  `ResolveWebRoot` ("web root ... does not contain index.html"), and an
  unknown flag, a missing webview, an unusable data directory and the
  "no database path" behind the menu actions were English on stderr or in the
  result box. Those reasons now come from `desktop.error.*`, so the actionable
  half of the message is in the same language as its title.
- **A configured Traditional Chinese locale stayed Traditional**:
  `ApplyConfiguredLocale` sent every Chinese `[i18n] locale` to `zh-Hans`, so
  a config asking for `zh-TW` got Simplified MyGo roles. `zh-TW`, `zh-HK` and
  `zh-Hant-*` now map onto `zh-Hant`; Aurora's own strings keep using the one
  Chinese table it ships, the same split the system-locale path already uses.
- **The language can be switched from the window**: Help → Language offers
  Follow System / English / 简体中文 / 繁體中文. `App.SetLocale` persists the
  choice for the next launch (System Settings on macOS, a file in the user
  data directory elsewhere) and relabels MyGo's own roles; the same
  `OnLocaleChanged` listener rebuilds Aurora's labels and ticks the item in
  effect. Previously the only way to get another language was to change the
  system locale or the config file.
- **File → Open Recent lists the exports you used**: MyGo keeps the recent
  documents list (and feeds the system's own recent list), and Aurora now
  records every successful import or export, from the menu, a drop, or a
  launch argument. Choosing an entry goes through the same `OnOpenFile` import
  path, so the overwrite and duplicate guards still apply. Recording waits
  until the app is ready (`App.IsReady`) because `App.AddRecentDocument`
  reaches the main thread and would hang before `Run`; a recording failure
  never turns a completed import into an error.
- **The desktop module is covered by CI**: `desktop/` is a nested module, so
  the root `go test ./...` never looked inside it and a broken window layer
  passed every gate. `ci.yml` gains a `desktop` job (its own Go 1.27
  toolchain via `go-version-file`, then gofmt, build, vet and
  `go test -race`), and `just desktop-check` runs the same four commands
  locally. `deskapp/ci_test.go` fails if the job disappears or drifts from the
  recipe. That job compiles rather than packages: packaging needs cgo and the
  target platform (see Fixed below); the separate `desktop-package` job below
  covers the real Linux package path.
- **The release matrix now starts what it ships**: `desktop-build.yml` used
  to verify only that each artifact existed, so a Linux package whose
  WebKitGTK runtime was missing — or whose embedded UI never loaded — passed
  the gate and failed on the user's machine. The Linux legs now install the
  runtime plus Xvfb, check `--version` on the packaged binary (non-`0.0.1`,
  known build time), then launch it against an empty data directory and
  require `aurora.db` to appear while the process is alive. The same contract
  is pinned in `deskapp/ci_test.go`, so the smoke test cannot quietly drop
  out of the release matrix.
- **Desktop packages ship with releases**: `ci.yml` now runs the real
  `go tool mygo build -platform linux/amd64` packaging path and verifies the
  executable, desktop entry, `.deb`, tarball and `x-scheme-handler/aurora`
  registration. `desktop-build.yml` packages the native release matrix
  (Linux amd64/arm64, macOS arm64/amd64, Windows amd64), and `release.yml`
  waits for it before publishing, so the MyGo app is downloadable from the
  same release as the CLI. A tag push is the single release trigger; a
  manual `workflow_dispatch` retry can republish an existing tag without
  rebuilding after the release event. Uploads overwrite files from an
  interrupted run.
- **macOS desktop releases must be signed and notarized**: `desktop-build.yml`
  now imports a Developer ID certificate into a temporary keychain, stores
  notary credentials under `aurora-notary`, and passes
  `macos.signingIdentity`/`macos.notarize` to MyGo when the six repository
  secrets are present. `release.yml` marks signing as required, so a tag
  release fails before packaging if they are missing or partial instead of
  publishing a Gatekeeper-blocked ad-hoc DMG. Signed packages are checked
  with `codesign`, `spctl` and `stapler validate`; local
  `just desktop-package` remains ad-hoc when the environment is unset.
- **Windows desktop releases must be Authenticode-signed**: the native
  Windows package job now decodes `WINDOWS_CERTIFICATE` into a temporary
  `.pfx`, passes it to MyGo through `windows.certificate`, and reads its
  password from `WINDOWS_CERTIFICATE_PASSWORD`. `release.yml` requires both
  secrets, so a tag release fails before packaging when they are missing or
  partial instead of publishing an installer that triggers SmartScreen.
  `Aurora.exe` and every `*Setup*.exe` are then checked with
  `Get-AuthenticodeSignature`: the signature must be valid, use the imported
  certificate, and carry an RFC 3161 timestamp. Local packages remain
  unsigned when the signing environment is unset.
- **Release CLI binaries are native and verifiable too**: `build.yml` now
  builds each CLI binary on its target OS with cgo enabled, signs both Windows
  architectures with the same fail-closed Authenticode action as the desktop
  package, and `release.yml` checks the embedded `CGO_ENABLED`, `GOOS` and
  `GOARCH` before publishing. It also writes a `checksums.txt` covering every
  CLI binary and desktop distributable, verifies that manifest with
  `sha256sum -c`, and publishes it alongside the release files. Branch builds
  remain unsigned when the signing secrets are absent.
- **Docker images are published only from reviewed commits**: `docker.yml`
  used to authenticate to Docker Hub and GHCR and push on pull requests, where
  fork builds cannot read the required secrets. PRs now build both
  architectures without logging in or pushing. Main and tag pushes publish
  with an explicit short-SHA tag and semver tags for `v*`, and the smoke test
  runs a separately loaded native image instead of asking for a full-SHA tag
  that `metadata-action` never created. A tag push already builds the release
  image, so the duplicate `release: published` trigger is gone.
- **The desktop shell is a packaged MyGo app, not just a window**:
  `desktop/mygo.config.ts` declares the app identity (name, reverse-DNS
  identifier, version, Linux description) and `just desktop-package` runs
  `go tool mygo build` to produce a native executable, a desktop entry that
  registers `x-scheme-handler/aurora`, a Debian package and an installable
  tarball under `dist/desktop/<os>-<arch>/`. The `mygo` CLI is a pinned
  module tool in `desktop/go.mod`, and the version comes from `git describe`
  inside the config so the binary, the `.deb` and the tarball name always
  agree.
- **`aurora://` deep links**: `aurora://lottery?id=<draw>` (and the
  `aurora://localhost/lottery.html?id=<draw>` spelling the window itself
  uses) opens the app on that draw, fills the Verify field and verifies it
  immediately. Declared in `urlSchemes` for bundles and registered at run
  time on Linux/Windows; a second launch forwards its link to the running
  instance through `App.OnSecondInstance` instead of starting a second
  SQLite writer.
- **File drops and command-line imports**: dropping a lottery export on the
  window imports it through the same app-layer function as
  File → Import (TASK-069's overwrite guard included), and a `.json` path on
  the command line does the same. Unsupported files are reported rather than
  ignored.
- **The app has an icon**: `desktop/appicon/icon.png` (1024×1024) is picked
  up by `mygo build`, so the desktop entry names `Icon=aurora` instead of
  leaving it empty and the `.deb` installs 256/512 px copies under
  `usr/share/icons/hicolor`. The same PNG is embedded so an unpackaged
  `just desktop` window shows it in its title bar and taskbar too.

### Fixed

- **The README had turned into an engineering log**: 528 lines, half of them
  the desktop design rationale, pushed install and quick start below the fold
  and left 12 `rumdl` table errors behind. The front page is now a release
  landing page (three entry points, install, desktop menu, CLI, API, config,
  structure, coverage, doc index); the desktop rationale moved to
  `docs/desktop.md`, and the CLI/API references to `docs/cli.md` and
  `docs/api.md`. The CI/release secret contracts are still guarded, now
  against the page that documents them, and every new page passes `rumdl`.
- **`aurora lottery export` could overwrite the live database**: the command
  read the records and then wrote the chosen filename without checking what
  that filename was. Passing `./data/aurora.db` (or a symlink to it, or its
  `-wal`/`-shm` sidecar) replaced the SQLite file with the JSON export and
  still printed success; only the already-open handle made `history` appear to
  work until the process exited. Export now goes through one app-layer entry
  point that refuses a destination aliasing the live database before writing,
  for both the CLI and File → Export (the backup path already had this guard,
  ISS-071).
- **An empty lottery export was `null`, and importing `null` looked like a
  success**: `ExportRecords` marshalled the repository's nil slice, so
  `aurora lottery export` and File → Export wrote `null` for a database with
  no draws. `json.Unmarshal` accepts that into a slice, so
  `aurora lottery import null.json` (and a dropped `null` export) reported
  "imported 0 records" for a document that is not the portable array format.
  Empty exports are now `[]`, and a null document is refused before anything
  is written.
- **A failed lottery export destroyed the previous export**: both
  `aurora lottery export` and File → Export called `os.WriteFile`, which
  truncates the destination before writing. A disk-full or I/O failure
  therefore left an empty or partial file where the user's last good export
  had been. Exports now go through a temporary sibling and an atomic rename;
  an existing file keeps its permissions, a symlink keeps pointing at its
  target, and any failure leaves the previous document untouched.
- **Startup flags written after a deep link were rejected**: `flag.Parse`
  stops at the first positional argument, so
  `aurora-desktop aurora://voting --data-dir X` failed as an extra argument
  even though a second launch with the same ordering was handled. The shell
  now collects the one link or export and parses startup flags on either side
  of it, while preserving `--`, missing-value, and unknown-flag errors.
- **A packaged desktop build reported `Build Time: unknown`**: `mygo build`
  supplies its own `-ldflags`, so the justfile's `-X main.BuildTime` never
  reached a bundled app. The packaging command now stamps the build time
  into `resources/build-time.txt` before MyGo copies it next to the
  executable, and the shell reads that value back. The Linux packaging gate
  rejects a build that still reports `unknown`.
- **`aurora-api` could not listen on an IPv6 address**: `--host ::1`,
  `AURORA_SERVER_HOST=::1` and `[server].host = "::1"` were joined to the
  port with a plain `%s:%d`, producing `::1:9090`, which `net.Listen` rejects
  as having too many colons. The listen address now goes through
  `net.JoinHostPort`, so IPv6 literals are bracketed (`[::1]:9090`); an
  already-bracketed `[::1]` input is accepted as well.
- **A packaged build reported the wrong version**: `mygo build` supplies its
  own `-ldflags`, so the justfile's `-X main.Version` never landed in a
  bundle; `aurora-desktop --version` printed the `0.0.1` placeholder while the
  `.deb` it shipped in was versioned from git. The shell now prefers
  `App.Version()` (what the package links in) and only falls back to the
  build-time variable.
- **The data directory depended on call order**: `App.Path(PathUserData)`
  joins the application name, and the shell resolved it before
  `App.SetName`, so one run created `~/.config/MyGo` and the next
  `~/.config/Aurora`. The name is now set before anything asks MyGo about
  identity, which also keeps the single-instance lock (keyed by that name)
  consistent.
- **A deep link fetched the verdict out of sight**: the Verify panel sits
  below Create and History, so the `/verify` call returned 200 while the
  visible viewport still showed the create form. The panel is scrolled into
  view once the verdict renders (measured in a real window under Xvfb,
  before and after).
- **A command-line lottery export was never imported**: the launch argument
  was marked as already handled (correct for a page link, whose window URL
  covers it) and then nothing acted on it. The import now runs from
  `WhenReady`, where MyGo's dialogs are usable, and `Shell` guarantees it
  happens exactly once even though MyGo re-delivers the argument.
- **A second launch dropped its export and never raised the window**: MyGo
  forwards a second instance's raw arguments to `OnSecondInstance`, but it
  only converts them into `OnOpenURL`/`OnOpenFile` for schemes and file types
  it is configured to handle. Aurora deliberately claims no `.json`
  association (that would take over every JSON file on the desktop), so
  `aurora-desktop draws.json` against a running instance was silently
  discarded. The shell now parses the forwarded arguments itself (links and
  exports) and reveals the window, restoring it first when minimized.
- **A forwarded relative export looked for the wrong file**: `OnSecondInstance`
  also passes the launching process's working directory, which the shell
  discarded. `cd ~/Downloads && aurora-desktop draws.json` therefore resolved
  `draws.json` against the *running* instance's directory, found nothing and
  did nothing. Forwarded arguments are now resolved against the launcher's
  directory, and `file://` URLs are accepted like paths.
- **A forwarded relative export could still hit a same-named file in the
  running instance's directory**: the first fix tried the argument as written
  before joining the launcher's directory, so if the running app happened to
  contain `draws.json`, that decoy won over the export the user launched. The
  launcher directory is now the only base for a relative path, and the
  relative URL spellings `file:draws.json` / `file:./draws.json` (whose path
  Go keeps in `URL.Opaque`) resolve there too.
- **The same link stopped working after the first time**: the shell remembered
  every argument it had ever honored, so reopening `aurora://lottery?id=<draw>`
  later in the session was silently ignored, and so was re-importing an export
  (which MyGo never echoes anyway, since Aurora declares no file association).
  The bookkeeping now counts exactly the one echo MyGo delivers for a launch
  the shell has just handled, and forgets it once it arrives — bounded, and a
  genuine repeat is a real request again. A re-import surfaces the app layer's
  duplicate guard ("Import partially failed") instead of nothing.
- **The same import told the user two different stories**: File → Import built
  its own partial-failure dialog listing the 0-based rejected record indices
  and why a record is refused, while the path a dropped file or a command-line
  argument takes printed the bare index slice and nothing else. `importReport`
  now owns the wording (and `importReportType` the severity) for all three
  entry points, and the menu imports through `ImportLotteryFile` like the
  others instead of repeating the read/open/import sequence. A regression test
  pins the shell dialog to the menu dialog byte for byte.
- **A failed import from the shell looked like a notice**: `showMessage`, the
  constructor behind every desktop message, hard-coded `mygo.MessageInfo` and
  discarded the severity its callers passed. An unreadable export or an
  unsupported drop therefore appeared under a blue info icon while the same
  failure from File → Import was a red error, and the warning a partial import
  deserves was computed and then dropped. The type now flows from each call
  site to `mygo.Dialog.Message`, and a test replaces the dialog itself,
  because the shell tests replace `Shell.report` and would stay green while
  every real message came out as a notice.
- **A second launch that named another data directory was ignored**: MyGo
  forwards the raw command line, and the shell skipped every flag in it, so
  `aurora-desktop --data-dir X` against a running instance left the user
  looking at the first instance's database with no message. A launch that
  names a different `--data-dir` or `--web-root` is now reported and its links
  and exports are not applied at all (importing into the wrong database is
  worse than doing nothing); the flags are scanned wherever they appear,
  because `flag.Parse` stops at the first positional argument, and a
  matching value — absolute or relative to the launcher's directory — is still
  a silent no-op.
- **The same relative `--data-dir` was reported as a different database**:
  the first instance recorded `db/aurora.db` as spelled, while a second launch
  resolved `--data-dir db` against its launcher directory to an absolute
  path. The two names pointed at one database but compared unequal, so every
  normal relaunch was refused. The startup path now records an absolute
  `aurora.db` path, and a regression test starts from a relative directory
  before forwarding the same flag.
- **The same `--web-root` with a trailing slash looked different**: the
  first instance stored `/srv/web/` while a later launch forwarded `/srv/web`
  (or the reverse), and the conflict check rejected the launch even though
  both name the directory the window is serving. Both path comparisons now
  use one canonical spelling, with a regression test for either direction.
- **A symlink alias of the running data directory looked like another
  database**: the conflict check compared cleaned path strings, so a second
  launch that reached the same `aurora.db` or web root through a symlinked
  directory was refused. Existing paths are now compared by filesystem
  identity (`os.SameFile`), while missing paths still fall back to the
  lexical check and remain a conflict.
- **A deep link forwarded during startup was dropped**: a second launch can
  reach `OnSecondInstance` after the single-instance lock is taken but before
  `WhenReady` created the window, and navigation with no window simply
  returned. `aurora-desktop aurora://token` against an instance that was still
  booting opened the dashboard instead. The newest such page is now held and
  loaded as soon as the window attaches.
- **The scheme transport dropped Go's streaming interfaces**:
  `cspDroppingWriter` is the outermost writer around every desktop response
  and implemented only `Header`/`Write`/`WriteHeader`, so on `aurora://` a
  streamed response lost its flushes and `http.ResponseController` reported
  "feature not supported", even though the API's recovery and metrics
  middlewares document that they preserve `http.Flusher` and friends. `Flush`
  (stripping CSP first, since flushing commits the headers), `Hijack` and
  `Unwrap` are forwarded now.
- **`--web-root` was a hint rather than an answer**: an explicit path that did
  not contain `index.html` fell through to the executable-relative, working
  directory or configured copy, so a typo silently rendered a UI the developer
  was not editing. An explicit path is now the only candidate and startup
  fails with "web root ... does not contain index.html"; the fallback list
  remains for the no-flag case the embedded UI and `go run` use.
- **A failed startup was silent**: `Start` errors went to stderr and the
  process exited, which a packaged app launched from a desktop entry cannot
  show — a bad `--web-root` or an unusable data directory looked like a
  double-click that did nothing. The shell now keeps running long enough to
  put the reason in a native error dialog, then exits 1.
- **A failed backup folder picker looked like Cancel**: the action treated
  "the picker returned an error" and "no folder selected" as the same silent
  return, so a broken picker (no portal, permission problem) left the backup
  undone with nothing on screen. An error is reported now; only an empty
  selection stays quiet, and a shell with no database path says so too.
- **`logger.Init` raced with everything that logs**: it replaced the package
  logger — and zerolog's process-global time format, for the value zerolog
  already defaults to — while the oracle scheduler, the audit-outbox drainer
  and request handlers were logging. The race detector caught it through the
  desktop start, which re-initializes logging while those loops run. The
  logger is published under a lock and every accessor reads it under the read
  lock; `Init` and `Close` serialize on the same mutex.
- **Every desktop dialog floated free of its window**: MyGo passes the window a
  menu was chosen from and attaches a dialog to a `Parent` (a sheet on macOS, a
  modal child elsewhere), but the menu handlers discarded the window and no
  dialog set one, so file pickers and result boxes were stray windows the user
  could leave behind. The actions take the window now and `showMessage` is the
  single place that builds a dialog, so the shell's own messages are parented
  too.
- **On macOS, closing the window ended the app**: nothing handled
  `OnWindowAllClosed` or `OnActivate`, so MyGo's default applied — Cmd+W closed
  the window and quit, and clicking the Dock icon had no window to bring back.
  macOS now keeps running without windows and reopens one on activation (the
  documented pattern), every other platform still quits with its window, and
  both paths build the window through `openMainWindow` so a reopened window is
  never missing its icon or file-drop handler.
- **A launch after Cmd+W was applied without a window**: the macOS process
  outlives its last window, but a later launch only routed its link or export
  through the shell. Navigation was dropped for the destroyed window and an
  import could finish with no visible result. Every OS-delivered intent now
  recreates the main window first through the same `openMainWindow` factory,
  and a page requested while no window can show it is held until the
  replacement attaches.
- **A recreated window had no regression guard for its identity and bindings**:
  the shared `openMainWindow` factory looked correct, but the window options
  and per-window wiring were embedded in a function that needed a running
  native app, so nothing proved a replacement kept `StateKey: "main"`, the
  launch page, the icon, focus, or the drop handler. `mainWindowOptions` and
  `wireMainWindow` are now injectable seams covered across two consecutive
  creations; fault injection (removing the state key, drop handler, or focus)
  turns the test red. The application menu stays installed across recreation
  rather than being rebuilt or cleared.
- **File logging fell back to the console on every first run**: `logger.Init`
  validated `log.path` with `utils.NormalizePath` — which requires the
  directory to exist — before creating it, so the shipped default
  (`config/aurora.toml`, `path = "./logs/"`) never produced a log file. A
  desktop app has no console, so its entire log was lost. The directory is now
  created first, then normalized.
- **The desktop wrote its logs into the launch directory**: the "keep logs
  beside the database" default only matched the exact string `./logs`, while
  the shipped config spells it `./logs/`, so a packaged app created `./logs`
  relative to wherever it was started from. The comparison is now on the
  cleaned path.
- **The documented cross-build of the desktop app could not work**: `just
  desktop-package -platform windows/amd64` fails in `runtime/cgo` with
  `windows.h: No such file or directory`, because the core's `mattn/go-sqlite3`
  storage needs cgo and no Windows C toolchain is present. That is now stated
  where people will look (README's integration notes, the CI job's comment and
  `deskapp/ci_test.go`), so packaging is expected on each target OS instead of
  failing mysteriously in a release pipeline.

## [v1.98] - 2026-10-21

### Added

- **A native desktop shell for the same backend** (`desktop/`, a nested Go
  module): until now every non-terminal surface meant running `aurora-api`,
  i.e. binding a TCP port and pointing a browser at it, with the API key in
  the page. `aurora-desktop` hosts the exact `internal/api` router inside a
  [MyGo](https://github.com/egoist/mygo) window over the custom scheme
  `aurora://localhost/`, so there is no listener, no CORS allow-list and no
  network path for the key (TASK-274, ISS-270, COMP-013). The Web UI is
  embedded in the binary via the root package's `//go:embed all:web`
  (`embed.go`, which also keeps MyGo's Go 1.27 requirement out of the core
  module's build graph); `--web-root` serves it from disk for UI development.
  Data defaults to `os.UserConfigDir()/aurora/aurora.db` (override
  `--data-dir`) with logs beside it, the oracle scheduler and audit-outbox
  drainer start exactly as `cmd/api` starts them, and the File menu offers
  Export/Import Lotteries (shared app-layer serializer, same overwrite guard
  as the CLI), Back Up Database (same `BackupService`), and Show Data Folder.

### Changed

- **The lottery export/import/verify logic moved out of the Cobra command**
  into `internal/app/lottery` (`ExportRecords`, `ImportRecords`,
  `VerifyInput`/`VerifyRecord`), so the CLI and the desktop menu cannot drift
  apart: one implementation of the resolution order (exact ID → whole block
  height → substring), one set of integrity checks, one import contract
  (`ImportResult.Rejected` indexes, malformed document hard-fails before any
  write). CLI output and exit codes are unchanged, and the moved unit test
  now lives with the code it tests.
- **The gateway's router accepts a Web UI filesystem**
  (`newRouterWithWebFS` / `Server.RouterWithWebFS`), letting a host serve the
  same routes, API-key bootstrap and CSP from any `fs.FS`. `Router()`
  behavior is unchanged (it still uses `config.WebRoot()`).

### Notes

- The desktop transport strips the gateway's `Content-Security-Policy`
  header. This was measured, not assumed: on `aurora://localhost/` WebKitGTK
  refuses to run the page's own scripts under any policy — `'self' aurora:`,
  `'self' aurora://localhost` and even `*` each left the window inert with
  zero `/api/v1` requests, while removing the header made all six dashboard
  calls fire. The transport substitutes for the policy: the scheme handler
  serves only the embedded files and no listener or remote origin exists. The
  HTTP gateway keeps its strict nonce-based CSP untouched.

## [v1.97] - 2026-09-07

### Added

- **`aurora-api` now overrides its listen host/port from the command line**:
  before this, the round-150 flag surface covered only `--help`/`--version`,
  so an operator colliding on the busy default `0.0.0.0:8080` had to edit
  `config/aurora.toml` (or watch the server die with `bind: address already
  in use`). `-H`/`--host` and `-p`/`--port` (1-65535, range validated before
  any listener is created) now resolve the listen address with flag >
  environment > config file > default precedence; the same chain is
  completed for the environment with `AURORA_SERVER_HOST` /
  `AURORA_SERVER_PORT` bindings in `config.Load` (the help text's
  "environment variables" claim previously held only for `AURORA_API_KEY`).
  parseFlags returns the overrides in a `serverOptions` struct and a pure
  `resolveAddr(opts, cfg)` helper carries the precedence, both unit-tested
  alongside the config env-bindings tests (TASK-273, ISS-269).

## [v1.96] - 2026-09-03

### Security

- **The oracle dial guard also refuses NAT64, multicast and reserved address
  classes**: the SSRF block list covered loopback, RFC1918, link-local, CGNAT
  and ULA, but on an IPv6-only network `[64:ff9b::7f00:1]` translates to
  127.0.0.1 at the edge, so the IPv4 checks could be routed around. The
  well-known and local-use NAT64 prefixes, IPv6 multicast, IPv4 IETF/benchmark/
  reserved space and documentation ranges are refused now;
  `TestBlockedIPRanges` pins each class.
- **The served Web UI had no Content-Security-Policy**: the gateway embeds the
  API key as `window.AURORA_API_KEY` in every HTML document, and a knocked-in
  XSS payload could therefore load remote script or exfiltrate the key to an
  external origin (no CSP to stop it). Every resource the UI loads is
  same-origin (js/app.js, css/style.css, the vendored eval-free Alpine) and
  its only inline script is the key bootstrap, so a strict policy holds
  (ISS-268, TASK-272): `default-src 'self'`, `script-src 'self'
  'nonce-<n>'` (the bootstrap carries a per-response crypto/rand nonce the
  header echoes, so ONLY that script runs inline), `connect-src 'self'`,
  `object-src 'none'`, `base-uri 'none'`, `form-action 'self'`,
  `frame-ancestors 'none'` (+ `img`/`font`). `style-src` keeps
  `'unsafe-inline'` because the blockchain page's Alpine `:style` binding
  writes the style attribute at runtime. Regression tests assert the CSP on
  HTML, its absence on non-HTML assets, and header-nonce == script-nonce.

## [v1.95] - 2026-09-03

### Fixed

- **`aurora-api` ignored its command line**: `./aurora-api --help` (or any
  misspelled flag) ignored the argument and started the HTTP server, dying
  only when the bind failed, while the cobra CLI handled these surfaces
  everywhere else. The server now has a minimal flag surface: `--help`/`-h`
  prints usage and exits 0, `-v`/`--version` prints the build identity and
  exits 0, and unknown flags / stray positional arguments are rejected
  before config loading or any listener is created (exit 1 with a usage
  hint). `--version` mirrors the CLI's `aurora version` output shape (i18n
  labels, real Go toolchain) and reports the link-time-injected ref; a new
  `just api` recipe builds the binary with `-X main.Version/BuildTime`
  injected (the root package of a binary links as `main`, so the `-X`
  symbol differs from the CLI's `cmd/aurora/cmd` vars). README documents
  the surface (TASK-267, ISS-263; post-review tightening in a follow-up,
  e.g. `--version` no longer drops a stray positional and the version test
  asserts locale-aware labels so the suite passes under `LANG=zh`).

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
- **Token read paths masked a nonexistent token as legitimate data**: `GET
  /token/balance?token_id=MTK_TYPO` returned `200 {"amount":"0"}` and
  `/allowance` / `/history` returned `200` zero/`[]` for an id that was never
  created, while `/token/info` 404'd the same id — a client typo or a
  pre-create existence probe was indistinguishable from "no activity". The
  read paths now apply the same `requireToken` existence guard every mutator
  (Mint/Burn/Transfer/TransferFrom/Approve) already used, so unknown ids
  report `404 TOKEN_NOT_FOUND` across REST/CLI/web (TASK-254, ISS-250;
  service + REST regression tests). The round-141 triage also re-confirmed
  three flagged items as intended, non-defect behavior and recorded them as
  decisions (DEC-013 `voting draft/window acceptance` aligns with the
  documented window-gated lifecycle DEC-004/DEC-005; DEC-014 the allowance
  family uniformly writes no on-chain block by design per DEC-002; DEC-015
  the voting window end is inclusive).
- **The web NFT/Token burn forms destroyed assets with no confirmation**: one
  click on "Burn" permanently destroyed an NFT or burned tokens, while the
  sibling destructive ops were gated everywhere else (oracle Delete confirms
  in-page; the CLI refused both burns without `--confirm`). A mis-click or a
  stray Enter on an auto-filled form destroyed value with no undo. Both burn
  actions now `confirm()` first and abort when declined, mirroring the oracle
  `deleteSource` guard (TASK-257, ISS-253).
- **The web transfers that MOVE value/ownership were still single-click
  while the sibling burns on the same pages confirm()**: a wrong recipient
  key (or a stray Enter on an auto-filled transfer form after mint/get)
  permanently sent tokens or an NFT to the wrong account with no undo — the
  exact failure class the burn-confirm fix covered, but transfers were left
  out. Token `transfer`, token `transfer_from` (spend allowance) and NFT
  `transfer` now `confirm()` their amount/recipient first and abort when
  declined, mirroring the burn guards (TASK-269, ISS-265).
- **The NFT transfer form's From field desynced after a successful
  transfer**: `owner` advanced to the recipient but the Transfer form's own
  From field stayed stale, so a second transfer posted the pre-transfer key
  and failed "not the owner" (operator had to re-paste it). A successful
  transfer now advances both the shared owner and the form's From to the
  recipient, matching the token page's `app.js:898` pattern (TASK-270,
  ISS-266).
- **Backend write endpoints stored unbounded free-text while the token and
  lottery surfaces capped theirs**: voting candidate `party`/`program` and
  session `description` (only non-empty name/title were checked), NFT mint
  `description`/`image_url`/`token_uri`, and oracle source
  `name`/`type`/`method`/`path`/`headers` all flowed unvalidated into SQLite
  TEXT columns — a key-holding caller could grow rows and list/detail
  responses without bound, unlike token (name≤100/symbol≤10) and lottery.
  Length bounds now live at the shared domain edge so REST/CLI/TUI/web all
  inherit them: voting gained a `validator.go` mirroring the token validator
  (voter/candidate name ≤100, party ≤100, program ≤1000, session title ≤200,
  description ≤1000), NFT.Validate caps name ≤200 and description/image_url/
  token_uri ≤2000, and oracle AddSource caps name ≤100, type ≤50, method ≤10,
  path ≤500, headers ≤2000, url ≤2000. Over-length input maps to a 400
  across all surfaces (TASK-271, ISS-267).
- **`GET /nft/{id}/history` and `GET /lottery/history` returned every row
  with no LIMIT while their siblings (token history, NFT list, oracle query)
  page at parse time**: a key-holding caller could force an unbounded DB
  scan/response on these two. NFT operations are now capped at 1000 per query
  (SQL LIMIT); the lottery REST endpoint parses + clamps `?limit`/`?offset`
  (default 50, max 100, offset ≤10000) like the token handler — the CLI/TUI
  keep their full local `GetAll` semantics (TASK-271, ISS-267).
- **A stale 18 MB compiled `api` binary was tracked in the repo root**: a build
  artifact from `go build ./cmd/api` (README builds it as `./aurora-api`),
  removed in `e02302e` and accidentally re-added by the round-133 RIL state
  commit `20ce60e`. `.gitignore` covered `/aurora` and `/aurora-*` but not the
  bare `/api` name, so every future accidental `go build` could re-commit it.
  The binary is untracked, the working-tree copy deleted, and `/api` added to
  `.gitignore`; the orphaned `.commit-msg.txt` (nothing in the repo reads it)
  is deleted alongside (TASK-266, ISS-262).
- **The config test suite failed whenever a parallel process left a file named
  `aurora` in /tmp**: `TestMain` isolated `$HOME` to the shared
  `os.TempDir()` for a "clean" baseline, but viper's `SetConfigName("aurora")`
  search matches a bare `aurora` file anywhere in the `$HOME` search path — so
  a `/tmp/aurora` ELF from an unrelated build was parsed as TOML and every
  config test failed with "invalid character at start of key". The isolation
  home is now a fresh per-run `os.MkdirTemp` subdirectory (removed afterward),
  and the suite passes even with such an interfering file present (TASK-268,
  ISS-264).
- **The dashboard Recent Activity list cleared to a false "No recent
  activity" on a transient all-endpoint poll failure**: the stat cards kept
  their last-good values across an API blip (TASK-151/234) but the list
  rebuilt from per-loader arrays that collapse to `[]` on failure, so one
  all-fail 15s poll wiped a populated list and misled the operator into
  believing activity stopped. List loaders now return `null` for a failed
  cycle (vs `[]` = success-with-no-rows) and `refresh()` keeps the prior rows
  when every source fails (TASK-258, ISS-254).
- **Three web form-consistency gaps**: (1) NFT History used its own
  `historyId` that only mint advanced, so a manually pasted id in Get NFT
  showed one asset while Show History queried another — it now binds the same
  shared id as Get/Transfer/Burn (the token page's pattern); (2) oracle Query
  Data success with zero stored rows was indistinguishable from a no-op (the
  table hides for empty rows, the error sink was cleared), so it now renders
  "No rows for this source — run Fetch first" for a zero-row success, distinct
  from the failure path; (3) token create `decimals` min/max only constrained
  the spinner, so a keyboard-typed 200 hit Go's int8 JSON decode ("invalid
  request") with no hint of the 127 bound — the value is now clamped to
  0..127 and reflected back, matching the oracle `clampLimit` contract
  (TASK-259, ISS-255).
- **Lottery on-chain records always serialized `verified:false` while every
  other surface reported `verified:true`**: the create use case wrote the
  immutable chain block from JSON serialized *before* verification ran, so an
  auditor replaying the chain saw every draw as unverified. Verification now
  runs before the chain JSON is built, so the on-chain copy carries the same
  verified state the DB, REST create response, `GET /{id}` and CLI `verify`
  report (TASK-260, ISS-256; regression test decodes the chain block).
- **Token history page-size divergence**: the identical no-flag query returned
  at most 20 events via the REST API but up to 50 via the CLI and the service
  layer's fallback. The REST default now matches the CLI/service default of 50
  (TASK-261, ISS-257).
- **`lottery reset` / `lottery db-info` closed the process-global blockchain
  DB singleton they borrowed**: `InitDB` returns a `sync.Once`-guarded `*sql.DB`,
  and closing it without resetting the once poisoned the singleton — any later
  `InitDB()` in the same process (a composable command flow, a test, a TUI flow
  calling reset then create) got an already-closed handle. Both commands now let
  the handle live until process exit, exactly like every other CLI command and
  `cmd/api` (TASK-262, ISS-258).
- **Docs truthfulness (round-145)**: the sample `config/aurora.toml` pointed
  at the nonexistent `aurora --ui oracle` flag (now spelled `aurora oracle
  tui`); `voting voter list` — a real subcommand — was absent from both
  quickstarts (README/AGENTS now document it, preserving the round-142 "every
  subcommand documented" claim); and the `[api.rateLimit]`/`[api.cors]`/
  `[oracle.scheduler]` keys the server actually reads had no shipped sample
  (added, cross-checked against `config.go` and verified loadable). Stale
  `.planning/PROJECT.md` claims (voting web page "In progress", metrics "Out of
  scope") refreshed to shipped status (TASK-263, TASK-264; ISS-259, ISS-260).
- **Token `transfer` / `transfer_from` could return a spurious
  `INSUFFICIENT_BALANCE`/`INSUFFICIENT_ALLOWANCE` 400**: each read the balance /
  allowance once *before* the atomic transaction and refused if the read looked
  short — but that read ran against a different snapshot than the transaction,
  so a concurrent top-up (a mint/credit to the same account) between the read
  and the atomic debit produced a 400 even though the atomic path would have
  succeeded. The ledger was always safe (the atomic
  `TrySubtractBalance`/`TryDeductApproval` primitives are authoritative); only
  the pre-check could lie. Both fast-fail reads are gone; the atomic primitives
  classify their own errors and the handler maps them to the same codes
  (TASK-265, ISS-261).

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
- **Screen readers now announce form submit outcomes**: all 33 async
  `class="result"` regions across the six pages (dashboard/lottery/voting/
  token/oracle/nft/blockchain) carry `aria-live="polite"` — a blind user
  hitting Create Lottery / Mint / Cast Vote previously got no feedback
  because the container text swapped silently under `x-show`/`x-text`
  (WCAG 4.1.3). `TestWebUIResultLiveRegion` fails if a future result
  container loses the attribute (TASK-253, ISS-249).
- **Data tables expose their column names to screen readers**: all five
  shipped tables (lottery history, oracle sources/health/query, voting
  results) declare `scope="col"` on their headers, and the lottery Count
  field's hint is linked via `aria-describedby` (previously it floated
  unannounced below its input). `TestWebUITableHeaderScope` fails if any
  future `<th>` drops `scope` or a form hint appears unlinked (TASK-256,
  ISS-252).

### Docs

- **Docs/CLI truthfulness audit (round 142)**: cross-checked every cobra
  subcommand and flag against README/AGENTS in both directions — zero dead
  doc claims (every documented command/flag exists; `--start-time`/`--end-time`
  verified real, `--amend`/`--rm` are git/docker context). Two quickstart
  gaps closed: `lottery reset --confirm` (was only in the `--confirm` prose
  footnote) and the `token history --limit/--offset` paging flags (callable
  but never spelled) (TASK-255, ISS-251).

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
- **Voting results counted a candidate's whole history, not one election**:
  `votes` had no `session_id` and `GetResultsUseCase` summed the candidate's
  global `vote_count`, so reusing a candidate in a second session reported
  every earlier ballot there too. Votes now persist their session, results
  (REST and CLI) count `(session_id, candidate_id)` rows, and migration
  `000003` adds a `vote_sessions` mapping without rebuilding `votes`; legacy
  rows remain unattributed rather than being assigned to an arbitrary
  election. The global candidate tally remains the dashboard/lifetime count.
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
