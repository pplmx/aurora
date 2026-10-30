# Changelog

All notable changes to this project will be documented in this file.

Sections are labelled with the git tag they were cut from (0.0.1-alpha.0 in 2024
through v1.99 on 2026-10-21). Binaries report the tag at build time
(`-ldflags -X …Version=$(git describe …)`), so a build without those flags prints
the source default 0.0.1; `[Unreleased]` collects the work since the last tag.
The milestone notes under `.planning/` are history, not current instructions.

## [Unreleased]

### Changed

- **Write payloads are plain text**: the NFT mint record appended the raw
  32-byte owner key and the token mint/transfer/transfer-from/burn records
  appended raw public keys, so the ledger and every JSON response quoting those
  payloads carried control characters; the keys are now base64 like the
  signature message and the NFT transfers, and tests reject any control rune.
  Existing records keep their bytes — the chain is append-only and the desktop
  renders binary fields as base64.
- **The database's disk usage is visible in both surfaces**: `aurora lottery
  db-info` printed the path and block count but not the size, and Settings
  showed the path alone; both now sum the database and its `-wal`/`-shm`
  siblings through one formatter, and Settings measures it as the card draws so
  the number cannot go stale.
- **An oversized import file is refused before it is read**: the CLI and the
  desktop read the dropped/selected JSON whole, so a wrong file (a video, a
  multi-gigabyte dump) grew the process by its size before the parser rejected
  it. Both now go through one reader capped at 64 MiB, name the limit in the
  active language instead of echoing a byte count, and the CLI reference states
  it.
- **Metrics carry an `aurora_` namespace**: the exporter emitted bare
  `http_requests_total` and `oracle_fetch_*`, which collide with other
  exporters on a shared scrape; every family is now `aurora_…` with the
  `_total` suffix its counter type requires, each declares HELP and TYPE, and
  `promtool check metrics` is clean on both endpoints. Dashboards that read the
  old names need the new ones; the reference lists them.
- **繁體中文 renders Traditional strings**: the desktop's 繁體中文 option and
  any zh-Hant/zh-TW/zh-HK locale fell back to the Simplified catalog, so MyGo's
  own menus were Traditional while every Aurora string was Simplified. Aurora
  now ships a Traditional catalog (generated from the Simplified one with
  OpenCC's s2t tables plus modern-form overrides) for the desktop and the
  CLI; tests keep both key sets equal.
- **The single-session API answers in snake_case**: `GET /voting/session/{id}`
  and the start/end endpoints encoded the domain struct, so they returned `ID`,
  `Title`, `StartTime` while every other response uses `id`/`title`/`start_time`;
  all three now go through the list's DTO, which also gained `created_at` so one
  struct parses either shape. The reference shows the real body.
- **A duration typo in the config file fails startup**: `[http] timeout = "1am"`
  used to fall back to 10s silently for the CLI and the API server; the three
  duration keys are now validated at load, and the command (or the server, before
  binding) stops naming the key. A string `"0s"`/`"-5s"` no longer bypasses the
  non-positive guard either: it was returned verbatim, and net/http treats a
  non-positive client timeout as no timeout at all.
- **A mistyped `[log] level` warns instead of falling back silently**:
  `level = "warning"` (the level is spelled `warn`) produced info-level output
  with no signal; the logger now names the rejected value and the fallback on
  stderr. The CLI reference also records that logs are appended without
  rotation, so long-running deployments rotate them externally.
- **`aurora-api` accepts `--config`**: the server only searched
  `$HOME/aurora.toml` and `./config/aurora.toml`, so a service unit or container
  could not point it at its own file, while the CLI had the flag all along. An
  explicit path must exist and parse; it never falls back to defaults.
- **A typo in `--http-timeout` now fails before the command runs**: the flag
  feeds the outbound oracle timeout, and an unparseable value silently fell
  back to 10s, so fetches failed as if the network were at fault. A bad value
  now exits 1 with the flag named, while `30s`, `1m` and bare seconds (`10`)
  still work.
- **A restore can be undone with a command**: `backup restore` keeps the
  previous database in `<archive>.pre_restore`, but that copy was a bare
  database with no metadata, so Verify and Restore refused it and an operator
  who restored the wrong archive had to copy files back by hand. The safety
  copy is now a complete backup with checksums, `backup verify` accepts it, and
  the command prints the undo line:
  `aurora backup restore <archive>.pre_restore --confirm`.
- **A stacked Oracle page leads with its stage**: below 1240px, or at a text
  scale that crosses it, the two columns stack and the tall add form came
  first, so Sources, Health and Readings sat two screens down and opening
  Oracle looked like a form with no navigation. The stage now leads and the
  setup follows, because this page's first column is setup rather than the list
  the stage acts on.
- **Guide and empty-state copy names the card, not the column**: the two-column
  workbench stacks below 1240px, and any text scale that crosses the threshold,
  so "left column" and "on the right" pointed nowhere on narrow or large-text
  windows. The steps now name the card ("Register Voter", "Add Source",
  "Allowance"), and two empty states that said "above" point at the create form
  that actually sits below the list.
- **Readouts tell an empty result from a missing first step**: NFT History, the
  voting tally and feed health each said "nothing loaded" while their id field
  was simply empty. They now name the missing step (choose the asset, session or
  feed, or paste its id) and keep the "not loaded" and "nothing recorded" lines
  for a real empty read; an empty Oracle Health with no feeds asks for a source
  instead of waiting for a poll.
- **An empty module opens on its first record instead of a dead-end page**:
  Token, Oracle and NFT show their create form beside a numbered path to what
  opens after the first write, and Voting's Setup stage leads with its two
  registration forms the same way. A filter, search, typed subject, open stage
  or failed read keeps the full workbench; the collapsible guide waits for the
  first record, and Ballot and Results name the missing session with a way back
  to Setup instead of showing forms keyed by an id that cannot exist yet.
- **The empty Ledger list waits for the first module record**: with only the
  genesis block there is nothing to search or page through, so its search field
  and page-size control stay away and the panel says no module records exist
  instead of claiming the chain is empty beside a Blocks 1 counter.
- **Search fields name the collection they narrow**: the placeholder is the
  collection's own translated label ("Search tokens", "搜索代币") rather than
  MyGo's untranslated "Search", and it follows the window's language.
- **The command palette ranks matches by where the query hits**: a title prefix
  outranks a title substring, which outranks a description, with the sidebar
  order kept inside each group. Typing "t" now puts Token above Lottery.
- **A disabled destructive action reads as disabled**: Burn, Burn NFT and
  Delete used to stay filled red while unusable, which made them look armed;
  they now take the same quiet surface as other disabled submits and turn red
  only when their inputs are ready. The confirmation dialog still stands in
  front of the write.

- **The desktop dashboard is a ledger story, not a grid of equal counters**:
  the integrity hero now gives only the chain verdict; a single Ledger
  composition panel owns every total and shows lottery, voting, oracle and
  chain scale as module rails with proportional tracks. Voting expands into
  ballots, candidates and sessions, and its rail uses the sum of all three
  measures in the shared denominator. Recent activity is now a vertical
  timeline whose rule follows each record's real height, so wrapped entries
  stay connected; every record still opens the module that wrote it. The
  dashboard drops the three competing ledger links, the old metric-card
  helpers and their fixed value slot, while keeping Quick start first on an
  empty store and last once records exist.
- **The desktop has two semantic appearances instead of one fixed dark palette**:
  Appearance now offers Follow System, Light and Dark and saves the choice in
  `desktop-settings.json`; an older file without a theme field still opens in
  the previous dark design. Light uses a warm paper surface and deep ink, Dark
  keeps the near-black ledger surface, and both retain the same status meanings.
  The window, MyGo native chrome and first frame follow the saved choice, and
  a failed preference write still applies the palette for this session while
  Settings reports that it will not survive a restart.
- **Every module now owns the accent instead of one global Aurora blue**: cards
  carry a short module-colour rail, the page header, tabs, selected records,
  focus ring and primary action all resolve to the open module's hue. The
  wordmark stays fixed as a mint-to-warm-gold identity, so product branding
  does not drift as pages change. Primary buttons use the module gradient with
  a contrast-checked label; disabled submit actions use a quiet bordered
  surface and state which required fields are missing.
- **Both appearances pass WCAG AA at the surfaces they actually paint**:
  body, muted text, module ink, success, danger and warning text are checked on
  the window and card backgrounds for every module; the primary label is
  checked on its module fill, selected-row text on its wash, and every verdict
  colour on its status tint. The light status wash is deliberately faint so
  readable verdict text does not disappear into a coloured strip.
- **Every collection is one list of records, not a table plus a fallback**: the
  nine collections that carried both a `ui.Table` rendering and a hand-written
  record loop at 1.75x text now share one renderer. The table was giving the
  window three shallow columns and a selection highlight — no collection set
  `Sortable`, `Width`/`MaxWidth` ordering or `RowHeader`, and `ListState`'s
  sorting and cell selection were never touched — while the two renderings could
  disagree about which facts a row has: the table showed a shortened id where the
  record showed the full one, and a field added to one was not necessarily added
  to the other. `collectionList` projects each row into title, facts, fields and
  an optional action, so a record reads the same at every text size and there is
  one place to add a fact. Titles and long field values wrap rather than
  ellipsising, which is what the large-text fallback existed to protect. Facts
  with no value are dropped instead of drawing an empty label.
- **Some facts moved onto the record**: a candidate's program and tally, and a
  token's exact supply, used to appear only in a readout after the row was
  chosen — a fact about a row that the operator had to select the row to
  discover. They are drawn on the record now, and the readouts that repeated
  them are gone.
- **`namedTable`, `tableText`, `tableTextFull`, `tableColumnWidth` and the
  per-collection `ListState` fields are removed**, along with the ten record-loop
  functions the shared list replaced. `stackedRecords` became `wideLayout`, since
  the only surfaces that still change shape with text size are a readout's value
  and the tally's comparison bars.
- **A paged collection leads with one toolbar instead of four stacked bands**:
  every collection card used to stack the search field, a muted sentence saying
  the search only narrows the current page, a full-width Previous/range/Next
  strip, and then the rows — 247px before a one-row collection could be read.
  The query now spans the card and the pager sits right-aligned directly under
  it, so the two read as one control group; the sentence became the search
  field's accessible description, and the pager buttons lost their resting
  outline and height so they stop competing with the card's real actions. The
  strip is one shared helper (`collectionToolbar`), and collections without a
  search field — NFT operation history, Oracle readings — use `collectionPager`
  so a page never mixes two pager heights. The pager itself stays, even on a
  single page: its range is the only statement that a collection has exactly
  these N rows.
- **The ledger reads as records instead of a payload dump**: the Blockchain page
  and the dashboard's Recent activity card now render every chain entry as
  "module · action · object" with its readable facts — a draw's winners,
  participant count and seed; a token write's token and accounts; a ballot's
  candidate, voter and session; an observation's value and source; an NFT's name
  and owner — plus the block's own timestamp and height. The raw JSON or
  `verb|field|field` payload stays one click away behind a Payload control, so
  the decoded sentence never becomes the only copy of what the chain holds. Each
  payload shape is decoded in `desktop/internal/deskapp/native_records.go`; the
  ledger's search matches the decoded text as well as the stored bytes and the
  height, so a winner's name or the word "transfer" finds its entry. Both
  surfaces draw the same record card, so the overview and the audit page tell one
  story about the same chain, and the module pill shows the module's own page
  name in the window's language rather than the chain's lowercase identifier.
- **`BlockEntry` carries its block's timestamp**: token and NFT payloads have no
  time of their own, so the block's creation time is the only "when" every
  recorded entry shares. The ledger previously could date a draw but not a mint.
- **NFT transfer and burn records decode as NFT writes**: they are stored as
  `<operation-uuid>|transfer` / `<operation-uuid>|burn`, which the old decoder
  did not recognise and printed as a bare UUID followed by a word.
- **Desktop chrome reads as a surface, not one long column**: the page header is
  now a fixed band tinted with the open module's own hue and closed by a
  hairline, with the scrolling body in its own padded column beneath it, so the
  title stays put while a form scrolls. Cards finally keep their own corners:
  the page body and both work columns leave a gap between cards instead of
  letting one card's radius meet the next one's.
- **"How this works" collapsed is one quiet line**: it used to be a bordered
  panel with a tile, as heavy as the page's first real card and 68px tall on
  every module page. Collapsed it is a single line of guidance behind an accent
  bar; opening it still lists the ordered steps.
- **Data grids sit in their own frame**: every table is painted with a recessed
  surface and a hairline border on its own element, so header rule and last row
  stay inside one object instead of floating on the card's gradient. The
  dashboard's section separator fades out to the right rather than running
  across the page like a table edge, and each counter card carries a hairline of
  its module's hue along the top rim.
- **New chrome regression tests** pin the page rhythm, the fixed header band,
  the collapsed guide's single line, the framed table, the fading section rule
  and the table header's row height, so a later layout change cannot quietly
  undo them.
- **Status, busy and destructive prompts share one visual language**: the
  outcome strip is now a full-width message with a lit left edge whose tint
  fades to the right, instead of a rounded badge inside a heavy outline that
  fought the header band. The running-operation strip uses the same shape in
  the accent, so "working" and "finished" read as the same kind of message in
  the same place. Burn, Delete and Forget now open with a warning mark in the
  danger hue and a hairline above their decision row, so an irreversible
  question is recognisable before its title is read; focus and accessible name
  are unchanged.
- **Tabs are one underline bar**: Token burns, NFT actions, Oracle sections and
  the Voting stages used to be pills inside a bordered, recessed box, with the
  current segment carrying a wash, a border and an underline at once. The row is
  now a single hairline with the active label in the accent over its own
  underline, so "current" is said once and the row stops drawing a second frame
  inside the card. Keyboard behaviour is unchanged.
- **The subject strip and the counters show what they belong to**: the "Acting
  on" strip puts the record's id on the accent and turns amber when no record is
  chosen, and each dashboard counter's number takes the hue of the module it
  opens instead of body ink, matching the glyph beside it.
- **The desktop window has a new visual system**: the palette moved to a deeper
  layered night tone with a per-module accent (lottery amber, voting violet,
  token emerald, oracle sky, NFT pink, blockchain indigo), and the shared
  spacing, radius, type and surface helpers now live in `design.go` instead of
  being repeated per view. Cards use a subtle gradient, softer elevation and a
  top hairline; primary actions are a lit gradient, secondary actions a glass
  outline, and the destructive action stays solid red behind its dialog.
- **The shell exposes navigation instead of hiding it**: the sidebar gained a
  gradient brand mark, a "Jump to" control that opens the same Ctrl/Cmd+K
  command palette, and a local-ledger/version footer when the shell supplies a
  version. The page header now uses the module tile, a 28pt title and a refresh
  control that prints its chord.
- **The dashboard leads with the ledger verdict**: integrity is a full-width
  hero panel with icon, word and route to the audit page, followed by six
  module counters (including the chain's block count) and the existing
  activity and quick-start cards. Settings is a two-column page, and the
  Blockchain summary uses stat chips plus a verdict pill instead of a
  full-width readout for two numbers.
- **Small details now follow the same system**: a fresh dashboard names its
  first move instead of explaining an empty chain, ledger module cells carry
  their module's hue, the command palette prints its arrow/Enter/Escape keys,
  and every page's filter is built by one helper so its accessible label stays
  translated. Token and NFT moved Inspect under the write tabs, so both
  columns of their workbench end at roughly the same height instead of leaving
  most of the action column empty.
- **Voting works as a two-column workbench on every stage**: Ballot puts Cast
  Vote beside Session Controls, and Results puts the session query beside a
  tally card that draws each candidate's votes as a leader-scaled bar while
  keeping the exact count and an accessible "candidate: N votes" label. The
  record layout still takes over at enlarged text.
- **Feed Health uses records at every width**: its five columns never fit the
  half-width workbench, so headers ellipsised and the last error disappeared
  first; the record layout keeps the source name, all three counters and the
  verbatim error. The Settings shortcut list now renders each chord as a key
  cap, and the Voting lists and empty ledger states carry their own marks.
- **Oracle source actions live in the workbench column**: Source Actions sits
  under Add Source with the selected endpoint, cadence and a 2x2 grid of
  Enable/Disable/Fetch/Delete, so it stays usable while reading Health or
  Readings. Its delete confirmation is mounted once for the page instead of
  only from the Sources tab.
- **Verify offers the latest draw**: the verification card gains a "Use latest
  draw" action whenever History has a row, so re-checking the draw just
  recorded no longer starts with copying its id out of the table.
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

### Added

- **The CLI can read and verify the ledger**: the REST API and the desktop
  could report integrity and list records, but an operator on a machine without
  either had only the block count in `db-info`. `aurora blockchain verify`
  prints the block count, the verdict and the first break and exits non-zero
  when the chain is invalid; `aurora blockchain blocks` pages the recorded
  payloads newest first (genesis excluded) so scripts can grep the audit trail.
  Both take `--json` and print the REST documents, so a script parses one shape
  whether it called the CLI or the API, and both are localized.
- **The command palette can open the data folder**: Ctrl/Cmd+K then "folder"
  reaches Show Data Folder, so the keyboard path covers every File-menu action
  except the Open Recent list.
- **`just desktop-shot` captures the real window**: it builds the shell with
  the `e2e` tag (which imports MyGo's automation plugin, never present in
  release builds), opens one window under Xvfb when no display exists, presses
  each page's sidebar key, asserts the accessibility tree names the page and
  writes `dist/e2e/<page>.png`. It complements the offscreen tester and CI's
  packaging smoke: those prove behaviour and that the binary starts, this one
  shows what the real backend drew.
- **Desktop navigation and feedback are legible to screen readers**: every
  page and card title now carries a semantic heading rank, switching pages
  announces the new title, and success/failure strips are status regions whose
  outcome is spoken once rather than silently repainted. The command palette
  now labels its dialog and focuses its search field immediately, while new
  windowless tests cover its typing path plus keyboard activation of whole-card
  shortcuts and wrapping tabs. The sidebar tree, each page's tab list, and the
  separate date and time inputs in the session form now carry explicit names,
  so assistive technology distinguishes Navigation, Token actions, Start time
  hours and similar regions instead of announcing generic unlabelled groups.
  Every data grid also passes through one named-table helper, so Recent
  activity, Draw history, Voting results, Feed Health and the other collections
  announce what they contain rather than only "table". Form cards expose named
  groups as well, keeping repeated fields such as Session ID or Owner attached
  to the Cast Vote, Transfer or other action they belong to. Validation errors
  now live inside the field that caused them, so the control is marked invalid
  and its error is exposed as the control's description instead of a detached
  red line. Session-window and token-decimal validation use the same field
  errors, and invalid decimals open the Advanced section that contains the
  control. Editing a field clears its previous validation immediately instead
  of leaving a corrected value marked invalid until the next submit, while a
  newly appeared error is announced once with its field name even if focus
  remains on the submit button. Help text below search boxes, number ranges,
  session times,
  checkboxes and data actions remains visible while also describing the control
  it explains.
  Settings presents the installed languages as a named radio group with the
  current choice selected, so arrows move between languages and the separate
  Follow System action says that it uses the operating system's choice. Every
  search field announces the number of rows its new query leaves, including
  zero and clearing the query. Collection and ledger pagers name their source
  and visible range; their Previous/Next descriptions say what moves, and the
  new range is announced after a successful move. A failed move leaves the old
  offset and stays quiet. The ledger search also describes that it matches
  across the whole chain. Failed database reads now expose the error and the
  stale-data note as one status message announced once, while background writes
  announce when they start and again if they still need attention after 20
  seconds. In the command palette, the focused search field now exposes the
  arrow-selected row as its active descendant, and each row's effect is part of
  its accessible description. Closing the palette with Escape or by choosing
  an entry hands keyboard focus to the named Navigation sidebar instead of
  letting the disposed search field leave the window without focus. Ctrl/Cmd+
  number page shortcuts do the same when they replace the focused page control.
  A page status or busy strip now appears above a keyed body host, so the
  control whose action produced the message keeps keyboard focus instead of
  losing it to the rebuilt sibling tree. Dismissing a status moves focus to
  Navigation, and dismissing a one-time generated key card returns to its
  Generate action. Generating that key moves focus straight to Copy so the
  secret is one Enter away while it is still on screen.

- **Desktop shortcuts describe their real scope**: Settings now marks Reload as
  a data-page action instead of claiming every shortcut works on every page.
  The page-switch reference and header comment also match the eight selectable
  pages, including Settings at Ctrl/Cmd+8.

- **Destructive desktop dialogs default to Cancel**: burning tokens or an NFT
  and deleting an Oracle source now share a red alert whose initial focus is
  Cancel, so Enter dismisses instead of performing an irreversible write. One
  Tab still reaches the named destructive action. Both cancelling and
  confirming return focus to the initiating control; a background write holds
  that request until it has re-enabled the page.

- **Session-key clearing asks first**: turning off session key memory or
  choosing Forget session keys no longer discards in-memory private keys on a
  single click. The shared confirmation starts on Cancel and explains that
  keys not saved elsewhere cannot be recovered. Cancel returns to the button
  that asked, while confirmation returns to the remaining opt-in even when the
  Forget button disappears.

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
- **Settings reports the desktop's actual accessibility state**: the
  Appearance card now shows the effective text percentage, contrast mode and
  motion mode instead of labelling every row "Follows system". The values
  update as the operating system changes, and both English and Chinese use
  explicit states such as `Standard`/`Enhanced` and `Enabled`/`Reduced`.
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

- **A failed feed fetch is named once**: the Oracle fetcher wrapped transport
  errors with "failed to fetch data" and the fetch use case wrapped the result
  again, so Feed Health read "failed to fetch data: failed to fetch data:
  request failed: ...". The use case now owns the operation phrase while the
  fetcher reports only why the request failed.
- **Candidate registration and both Burn actions follow form readiness**:
  Register Candidate, Burn, and Burn NFT could open a submit or confirmation
  path with required fields empty. Their actions now stay disabled until every
  required value is present, and the handlers still reject incomplete input
  defensively.
- **Every desktop write handler now defends its own readiness**: Token
  create/mint/transfer/approve/transfer-from, NFT mint/transfer, source
  creation, session creation, and ballot casting can no longer reach the
  service through a stale frame with a required field empty. The handlers
  return a localized explanation even though their buttons are already
  disabled.
- **Voting session controls require a session**: Start and End were clickable
  with an empty Session ID, sending a write the service could only reject. The
  two buttons now follow the field's readiness, the field explains why they
  are disabled, and the handler refuses an empty id defensively.
- **Oracle source actions require a source**: Enable, Disable, Fetch now and
  Delete were clickable before a source was chosen, so the window could send
  an empty id or open a Delete dialog with no target. The four controls now
  share the action field's readiness state, include a visible hint, and the
  handlers defensively refuse an empty id.
- **macOS keeps the desktop file shortcuts**: the native menu was reduced to
  platform roles, but Cmd+E/I/B still relied on it even though Aurora no longer
  installed those items there. The window now registers export, import and
  backup on every platform, Settings lists them in the selected language, and
  background writes still pause all three.
- **Desktop file work pauses while a write owns the database**: the sidebar
  File menu, Ctrl/Cmd+E/I/B shortcuts, Refresh and command-palette file
  commands are unavailable while a background write is in flight, while page
  navigation stays live. An export, import, backup or reload could previously
  start a second repository operation and race the write's result.
- **CLI and desktop honor the standard locale environment**: locale detection
  previously read only `LANG`, so a shell with `LANG=C` and
  `LC_ALL=zh_CN.UTF-8` (or a `LANGUAGE=zh_CN:en` preference) still showed
  English help and dialogs. Aurora now resolves `LC_ALL` → `LANGUAGE` →
  `LC_MESSAGES` → `LANG`, selects the first supported tag in a `LANGUAGE`
  list, and uses `[i18n] locale` only when none of those variables is set.
- **Cobra-generated CLI help is localized too**: Chinese `--help` output no
  longer mixes translated command descriptions with English `Usage:`,
  `Available Commands:`, `Flags:` or `Global Flags:` headings, and the
  built-in `help`/`completion` descriptions and root banner now translate as
  well.
- **CLI help honors the configured locale**: command metadata is initialized
  before Cobra loads the config file, so startup locale detection now reads
  the documented default config paths (and an explicit `--config` argument)
  before command declarations. A configured `locale = "zh"` now translates
  root help, subcommand summaries, flag descriptions and the built-in help
  commands even when no locale environment variable is set.
- **Configured locale tags and the config-file notice are localized**: values
  such as `zh-CN`, `zh-Hans`, `zh-Hant-TW`, `en-GB` and environment-style
  `zh_CN.UTF-8` now canonicalize to the message tables Aurora actually ships,
  instead of silently falling back to English. The startup line that reports
  which config file was loaded follows the same language.
- **CLI command results now follow the selected locale**: token balances,
  allowances, supply and transfer history; voting key cards, vote ids and
  session states; NFT details, list/history headings and operation names;
  Oracle source/data/template labels; lottery verification evidence and import
  diagnostics; and the final error/audit warning all resolve their labels at
  print time. English output keeps the existing wording and field order.
- **CLI failures localize the full error path**: command context and known
  domain causes now render in the active language instead of producing lines
  such as `❌ 错误：failed to get token info: token not found`. The same pass
  translates Cobra's required/unknown flag errors, positional-argument counts,
  unknown commands with suggestions, and unknown help topics. Wrapped errors
  retain `errors.Is`/`errors.As`; unrecognized filesystem, network and database
  details remain verbatim rather than being replaced by a vague generic error,
  including when a known domain cause carries a technical detail such as
  `disk full`. Application-layer wrappers around known causes are translated
  when they add meaning (`invalid creator encoding`) and omitted when they only
  repeat the command context (`failed to get token`); Aurora's own validator
  details such as `invalid headers json` are localized too. Unsupported
  locales now fall back to English instead of accidentally using Chinese Cobra
  strings.
- **Lottery verification evidence and malformed imports stay in the selected
  language**: Chinese `lottery verify` and the desktop verifier now translate
  every finding, including winner/address mismatches, instead of printing
  English check lines under a Chinese failure heading. A malformed or `null`
  import is classified as an invalid portable document, so `lottery import`
  reports the localized document failure while retaining the parser detail
  needed to fix the file. File > Import, a dropped export and a path opened by
  the desktop shell now show that same localized failure in the native dialog
  instead of bypassing the desktop's error translator.
- **Chinese desktop guidance no longer mixes in English UI chrome**: Lottery,
  Voting, Token, Oracle, NFT, Blockchain and Settings copy now names visible
  controls in Chinese (`中奖人数`, `历史`, `被授权方`, `显示全部`, `准备`,
  `数据源`, `健康`, `读数`, `显示条数`, `仪表盘`) while retaining protocol
  names such as NFT, VRF, HTTP, JSON, URL and base64. Appearance values
  (`Dark`, `Follows system`) also pass through the language switch instead of
  staying English inside a Chinese window.
- **Current interface copy no longer advertises the deleted Web UI**:
  `aurora --help` now lists the CLI, native desktop window and REST API, and
  `aurora-api --help` calls itself the JSON-only REST API server. The
  second-launch desktop dialog identifies the requested `--data-dir` against
  the running database instead of mentioning a web page, and current docs no
  longer advertise the deleted TUI or the removed web form. The container
  manifests now describe `cmd/api` as the JSON API it actually is; both
  translations of the dialog, the root/API help lines, and every CLI command
  documented in the current guides are pinned by tests.
- **Oracle fetchers release their idle HTTP connections**: the SSRF-guarding
  transport wrapper hid the standard `CloseIdleConnections` hook, so closing
  the client was a silent no-op. API, desktop and CLI shutdown paths now close
  the fetcher they own, preventing keep-alive sockets from outliving the
  process services that created them.
- **Database restores preserve a symlinked `db.path`**: restore planned its
  temporary file, promotion rename and sidecar cleanup against the configured
  symlink, so a successful restore replaced the link itself while SQLite kept
  using WAL/SHM/journal files beside the untouched target. Restore now follows
  the final link (including a not-yet-created target), atomically replaces the
  real database and clears the sidecars at that same location.
- **Atomic writes create the target of a dangling symlink**: lottery exports
  and desktop preference saves used to fail when the destination was a valid
  symlink whose target did not exist yet. They now follow the final link chain
  from its real parent directory, including relative `..` targets, and
  atomically create or replace the target while preserving the link just like
  a normal write would.
- **Desktop preferences repair a permissive mode**: an existing
  `desktop-settings.json` created or edited with mode `0644` kept that mode
  because atomic replacement preserves established permissions, and startup
  hardening did not include the file. Both startup and the next save now force
  `0600` on Unix, matching the documented privacy boundary.
- **Desktop file URLs preserve Windows UNC hosts and reject remote POSIX
  paths**: a launcher-provided `file://server/share/draws.json` used to lose
  the server segment and resolve against the local path, while Windows UNC
  exports were not importable through their file URL. The desktop intent
  parser now maps UNC authority names to `\\server\share\...` on Windows and
  refuses non-local authorities everywhere else.
- **Windows UNC database paths no longer become invalid SQLite URI authorities**:
  the hardened DSN builder now emits the empty-authority four-slash form
  (`file:////server/share/aurora.db`) for UNC paths and keeps drive-qualified
  paths in `file:/C:/...` form. SQLite otherwise treats the first server
  segment as an unsupported authority and rejects the database before opening
  it.
- **SQLite `file:` fragments no longer disable hardened URI parameters**:
  appending `_foreign_keys`, `_txlock`, or a native `mode` after an existing
  `#fragment` made the driver consume the fragment as part of the final value,
  while SQLite ignored everything after it. Database DSN construction now
  drops the semantically ignored fragment before adding parameters, so
  encoded `#` characters remain part of the filename and native read-only mode
  is enforced.
- **Backups resolve disk-backed SQLite `file:` URIs before touching output**:
  `backup create` used to look for a literal path beginning with `file:` and
  fail on a valid URI-backed database, while `backup restore` planned its
  writes against that same non-path. Backup services now normalize every
  configured location before schema reads, source checks, restore planning, or
  archive validation; schema metadata remains correct, and memory-backed
  configurations are rejected before an output directory is created.
- **Disk-backed SQLite `file:` URIs receive the same owner-only boundary**:
  permission hardening previously skipped every `file:` DSN, leaving a
  percent-encoded or absolute disk database and its WAL, shared-memory, and
  rollback-journal files at the process umask. Database locations now resolve
  URI paths (including `localhost` and `mode=memory`) before directory
  creation, permission tightening, and lottery-export alias checks. Memory and
  empty-name temporary databases remain untouched.
- **The desktop high-contrast setting now reaches the whole window**: the
  palette had a stronger border, text, muted-text, focus and selection set,
  but most custom-drawn cards, tables and controls still read the base colour
  constants directly. Those renderers now resolve every preference-sensitive
  colour from `c.Theme()`, and a source contract prevents the base palette
  from bypassing the toolkit theme again.
- **Lottery exports protect sidecars behind a database symlink**: a
  `DBPath()` that pointed at the live database through a symlink was checked
  only at the link location, while SQLite places WAL, shared-memory, and
  rollback-journal files beside the target. Exporting to the real sidecar
  could therefore replace live transaction state with JSON. The guard now
  includes both the configured path and every resolved target sidecar.
- **The desktop protects SQLite rollback journals at startup**: application
  data hardening covered the database, WAL, shared-memory file and log, but
  still constructed its own partial sidecar list. It now delegates to the
  shared SQLite boundary, so an `aurora.db-journal` left by a rollback is also
  tightened to `0600` and sidecars of a symlinked database are found at their
  real location.
- **Lottery exports refuse the live rollback journal too**: the shared CLI and
  desktop export guard covered the database and its WAL/SHM files, but a
  destination named `aurora.db-journal` could still replace a hot rollback
  journal with JSON. All SQLite safety boundaries now draw their sidecar list
  from one helper, so permission hardening, restore cleanup, and export
  protection cover the same three sidecars.
- **Trailing backup-directory separators keep restores safe and repeatable**:
  shell completion commonly supplies `backups/daily/`, which made the hidden
  `.pre_restore` undo copy land inside the archive instead of beside it. That
  could fail against a read-only archive or contaminate a directory that had
  just passed verification. Restore now normalizes the archive path before
  deriving its sibling safety directory.
- **Symlinked SQLite databases keep every sidecar owner-only**: permission
  hardening followed the database symlink for the main file but looked for
  `-wal`/`-shm`/`-journal` beside the link, while SQLite creates them beside
  the resolved target. The store now resolves the database path before
  restricting all four files, closing the world-readable sidecar window for
  symlinked data directories.
- **SQLite accepts database paths with repeated leading slashes**: a POSIX
  path such as `//srv/aurora.db` was emitted as `file://srv/aurora.db?...`, so
  SQLite treated `srv` as a URI authority and refused to open the database.
  Leading slashes now collapse to the URI root while special-character
  escaping, relative paths, `:memory:`, and explicit `file:` DSNs keep their
  existing behavior.
- **Database restores also remove a stale rollback journal**: promotion
  cleared the old `-wal`/`-shm` sidecars but could leave the previous
  generation's `-journal` beside the newly restored database. Restore and its
  failure rollback now share one sidecar set covering all three files, so
  SQLite cannot replay stale rollback pages against the archive.
- **Oracle source headers now follow net/http's request model**: a configured
  `Host` was stored in the ordinary header map, which `net/http` ignores when
  the request URL already has a host. It now populates `Request.Host`, survives
  same-origin redirects (including absolute locations), and is cleared with
  other source headers on cross-origin redirects. Transport-derived
  `Content-Length`, `Transfer-Encoding` and `Trailer` fields, plus header names
  that differ only by case, are rejected at source creation instead of being
  silently ignored or applied in nondeterministic map order. Proxy credentials
  and hop-by-hop fields are rejected too: `Proxy-Authorization` otherwise goes
  to the origin on a direct fetch and is not sent with the HTTPS proxy CONNECT,
  while fields such as `Connection`, `Keep-Alive`, `TE` and `Upgrade` belong to
  the transport connection rather than an individual feed.
- **Oracle source headers can select the upstream representation**: the
  transport overwrote a configured `User-Agent` and `Accept` with Aurora's
  defaults, so feeds that route `application/vnd.*` representations or require
  an allowlisted client identity received the wrong request. Defaults now apply
  only when the source has not supplied its own value, an explicitly empty
  `User-Agent` still suppresses that header, and a configured media
  `Content-Type` is preserved. `X-Request-ID` remains generated per hop for
  correlation.
- **Oracle fetch errors no longer echo credential-bearing URLs**: HTTP status
  errors included the complete source URL, and `net/http` transport errors
  retained its path, query and userinfo. Those strings reached scheduler logs
  and `/oracle/health.last_error`. Errors now keep the actionable status or
  underlying cause while dropping URL-bearing wrappers; the source id still
  identifies the feed.
- **Oracle retry and refresh clocks start when a fetch finishes**: both the
  next successful refresh interval and failed-fetch backoff were measured from
  the attempt's start time. A slow timeout could consume the entire backoff, so
  a persistently failing source retried almost continuously, and a slow
  success could become due early. Completion timestamps now drive
  `last_fetch`, `last_success_at`, and the next retry window.
- **Oracle sources reject unusable URL authorities at creation**: a URL with
  a port but no hostname, an empty port after `:`, or a port outside
  `1..65535` was previously persisted and failed only on the later fetch.
  Source validation now checks the parsed hostname and numeric port before
  saving, applying the same contract at add and fetch time across REST, CLI
  and desktop.
- **Oracle URL validation closes the remaining IPv6 private-space gaps**:
  deprecated IPv6 site-local (`fec0::/10`), IPv4-compatible (`::/96`) and
  IPv4-translated (`::ffff:0:0:0/96`) addresses, plus the IETF IPv6 protocol
  assignment block (`2001::/23`, including Teredo and ORCHID), previously
  passed add-time validation even though they are not public HTTP feed
  destinations. The shared policy now rejects them at source creation,
  redirect and dial time while continuing to allow IPv4-mapped public
  addresses.
- **Large-text record titles keep their complete identity**: Lottery history
  still rendered a shortened draw id inside its “full” record, and Oracle feed
  health forced the source name through a single-line ellipsis. Both titles now
  use the full stored identity and wrap within the record instead of depending
  on a tooltip or a later selection.
- **Guide step numbers survive enlarged text**: the numbered marker beside
  each “How this works” instruction kept an 18px slot while its glyphs scaled to
  2x, so the number was clipped even with the guide opened. The slot and
  non-shrinking width now follow the text scale, while the instruction takes
  the remaining width and wraps beneath the number.
- **Voting tallies keep the complete candidate beside the count**: the
  two-column results table left Votes readable at 2x but reduced a long
  candidate name to a single-line prefix. Results now become stacked records
  above the same 1.75x threshold, so the full identity and exact tally stay
  together under the total-vote readout.
- **Dashboard activity and the blockchain audit stay readable at large text**:
  at 2x the three-column tables left only a narrow strip for the audit payload,
  so the stored event was reduced to a prefix. Recent dashboard entries and
  Blockchain rows now become records with the complete summary beside the exact
  block and module; the dashboard record remains the control that opens its
  module.
- **Token and NFT histories keep every transfer party at large text**: the
  fixed account columns reduced base64 public keys to short prefixes at 2x,
  while the token amount/nonce and NFT block columns competed for the remaining
  width. Above the shared 1.75x threshold both histories now render one event
  per record, with full From/To keys and the exact amount, nonce or block height
  together.
- **Oracle sources and readings stay complete at large text**: the five-column
  source table pushed type, id, status and header state outside the card at 2x,
  while the shortened id made the row hard to identify before loading it into
  the actions. Sources now become selectable records with the full id, and the
  whole record still fills every action and refreshes open readings. Query
  results become records with a wrapping value, and the Latest Data readout
  also wraps instead of reducing nested JSON or long observations to a prefix.
- **Voting setup collections stay readable at large text**: at 2x text the
  voter table pushed the public-key column beyond the card, candidate names and
  parties were clipped, and the four-column session list could no longer show
  the id, title, status and acceptance window together. All three Setup lists
  now become selectable records above the same 1.75x threshold, preserving the
  complete public key or id and keeping the existing ballot, roster and results
  handoffs on the whole record.
- **Token and NFT collections stay identifiable at large text**: the token and
  NFT tables kept fixed id, name and symbol widths while their labels doubled
  at 2x text. In the minimum window the Supply header collapsed to 16px, a
  token name to 100px and an NFT name to 90px, and both ids were shortened
  before the operator could choose a row. At the same 1.75x threshold as the
  wide histories, both collections now become selectable records with the full
  name, exact supply or block height, and complete id visible; the whole record
  remains the control that loads the asset into every other card.
- **The Oracle SSRF policy now enforces every non-public class it claims**:
  the shared block list still accepted IPv4 multicast and the TEST-NET
  documentation ranges, and 6to4/benchmark IPv6 addresses could route around
  the IPv4 checks. Redirect validation also failed to recognize a bracketed
  IPv6 literal without an explicit port, rejecting public IPv6 feeds while
  relying on a resolver error for blocked ones. The shared policy now covers
  those ranges, and one host parser handles bracketed literals consistently.
- **Primary and secondary actions keep their breathing room at large text**:
  the shared Create/Refresh/Delete button style kept a 34px height and 14px
  horizontal inset while its 13pt label grew to 33px at 2x. All three action
  variants now scale the whole control with text size instead of fitting a
  scaled label into fixed chrome.
- **The pinned File menu scales with the sidebar**: the File control kept a
  34px height and 12px inset after large text widened the navigation column.
  It now grows with the system scale like the card actions and remains fully
  inside the fixed footer at 2x.
- **Multiline fields keep room for several lines at large text**: the NFT
  description and Oracle request-header editors stayed near 80px high while
  their line height doubled, leaving roughly one usable line. Text areas now
  scale with the system text size and continue to show the same working area.
- **Quiet actions keep their full label at large text**: compact Copy and
  Dismiss controls stayed 26px tall while their scaled 12pt label measured
  33px at 2x, clipping the text inside the hit target. Quiet buttons now grow
  with text scale and scale their horizontal padding too.
- **Wide histories keep every field at 2x text**: Lottery's Winners column
  began at x=901 and Oracle Health's Last error column at x=931 in the 900px
  minimum window. Above 1.75x text both tables now become compact record
  cards, preserving block counts, participants, winners, health counters, full
  errors and the Verify handoff without sideways scrolling.
- **Section tabs wrap instead of hiding later choices**: Token's five write
  tabs crossed the right edge at 2x text in the minimum window, leaving the
  Burn form without a visible entry point. Tab rows now keep MyGo's click and
  keyboard behavior but wrap onto additional lines when their labels do not
  fit.
- **One-time key previews survive large text**: in generated-key and other
  copy rows, the fixed label and full copy action consumed the whole line at
  2x text, collapsing the value itself to zero width. The value now keeps
  readable space and the copy action wraps below it when necessary, so an
  operator can see the secret before copying it.
- **Oracle actions wrap before they reach the window edge**: Enable, Disable,
  Fetch now and Delete shared one unwrapped row. At 2x text in the minimum
  window the destructive Delete button was pushed past the right edge. The
  action group now wraps onto a second line, keeping every verb and its full
  hit target visible.
- **Scaled sidebar labels keep their full names**: at 1.5x and 2x text the
  English brand subtitle and module names were clipped inside the fixed-width
  navigation column. The sidebar now measures the localized brand, section and
  module text and grows only enough to show it, capped at 38% of the window;
  each row also exposes its full name as a tooltip when a longer translation
  reaches that cap.
- **Large text no longer overlaps or strands navigation**: the dashboard metric
  value slot now grows with its scaled type instead of letting a 2x number
  paint over the note below it. The sidebar takes the height left between the
  brand and the pinned File action, scrolls its module list internally when
  needed, and reveals the newly selected page whether it was chosen by pointer
  or keyboard.
- **Overlays remain usable in short windows and at large text sizes**: the
  command palette now scrolls within the minimum-height window and reveals the
  keyboard-selected row instead of highlighting commands below the viewport.
  The custom theme also scales MyGo's default font for alerts, inputs and tabs,
  while destructive confirmations bound long NFT/source names to a readable
  prefix plus the shortened unique ID so their actions stay on screen.
- **Numeric tables and detail panels stop clipping**: ledger heights, blocks,
  counters and vote totals now scale with text size, and every numeric cell,
  including nonces, exposes its full value on hover. Readouts for owner keys,
  descriptions, oracle values and other long details, plus abbreviated copy
  rows, similarly reveal the complete stored value instead of ending at the
  card edge.
- **Table values survive larger text and narrower columns**: desktop tables
  now mark clipped cells with an ellipsis and expose the full stored value on
  hover, including source names, payloads, keys and long error text. Session
  windows stack exact start/end timestamps, and the Supply and Block columns
  reserve enough width for the largest valid `int64` value as text scaling
  grows. Selecting a token also shows its exact supply below the collection.
- **Page switches preserve the destination layout**: the desktop's shared
  scroll container now carries a page identity instead of being reused by tree
  position. Opening NFT after Oracle previously reconciled the two distinct
  layouts and clipped the Mint button at 1.5x text.
- **Token operations keep their column**: the desktop Token page moved the
  read-only Allowance query from the sixth write tab to the left work column.
  The right column now contains the five mutating operations, so `Burn` remains
  inside the card at the default window width.
- **Two-column pages stack before their controls compress**: desktop pages now
  switch to one column below 1240px instead of waiting until 1000px, and the
  threshold scales with the user's text-size preference. Narrow windows no
  longer clip tab labels or squeeze base64 fields between two cards.
- **Readout labels stay intact**: desktop copy and readout rows now size their
  label column from the text and the user's text-size preference, keep labels
  on one line, and constrain long values to the panel. Settings entries such as
  "Remembered keys" and "Command palette" no longer split across two lines.
- **Voter history stays legible**: the desktop voter table now shows `Yes` /
  `No` under an `Ever voted` column and keeps each status cell to one line.
  The previous multi-word status wrapped inside the fixed-height row, clipping
  the next voter behind the explanatory note.
- **Committed ballots are immutable**: `SaveVote` used `INSERT OR REPLACE`
  for the vote row and its session mapping, so reusing a vote ID could
  silently rewrite the candidate, signature or message while the existing
  claim still pointed at that ID. Both writes are now append-only; only the
  exact `(session_id, voter_pk)` claim conflict is ignored and mapped to
  `ALREADY_VOTED`.
- **Voter eligibility is scoped to the voting session**: `voters.has_voted` was
  the only duplicate-ballot guard, so a voter who used their key once could
  never participate in a later election even though results and candidate
  rosters were already session-scoped. Session ballots now atomically claim
  `(session_id, voter_pk)` alongside the vote and tally; the profile flag is
  explicitly historical. Migration `000008` backfills claims from existing
  ballots, and both migration and self-bootstrap paths preserve duplicate
  historical audit rows while the earliest becomes the claim owner. Session
  tallies count only those claim-winning ballots, so preserved duplicate audit
  rows cannot double-count an election result.
- **Draft voting sessions cannot accept ballots**: a session created inside its
  start/end window accepted votes without the explicit start transition, making
  the "start session" step optional. Vote admission now checks the lifecycle
  state in the domain layer as well as the window: only `active` sessions
  accept ballots, while draft, ended and unknown states fail closed.
- **Session state is rechecked inside the ballot transaction**: ending a
  session or changing its candidate roster after the preflight read could no
  longer race an in-flight ballot into a committed vote. The transaction
  re-reads the session after acquiring SQLite's write lock, so the status,
  time window and roster are serialized with the vote, voter claim and tally.
- **The Oracle SSRF block list has one source of truth**: the domain URL
  validator and the HTTP fetch/dial guard each carried their own list of
  blocked address ranges, and the domain copy had drifted. Source creation
  still accepted benchmarking, reserved, multicast, documentation and NAT64
  addresses that the direct dialer already refused, leaving those decisions
  to a later fetch (or to an operator-configured proxy). Both layers now share
  `internal/netguard`, so add-time, redirect-time and dial-time checks enforce
  the same ranges.
- **Proxy client-IP parsing has a fixed work bound**: behind a trusted proxy,
  `X-Forwarded-For` is commonly appended to rather than replaced, so a client
  could still supply a header near the 64 KiB server limit. The middleware
  split that entire value into an allocation before authentication and walked
  every hop. It now parses right-to-left without materialising the full slice,
  and treats chains longer than 32 hops as unattributable rather than trusting
  the leftmost caller-supplied address.
- **Invalid server ports fail before startup work begins**: `server.port` from
  TOML or `AURORA_SERVER_PORT` was unmarshaled without a range check. A zero
  port silently requested an ephemeral listener while the logs still printed
  `:0`, and an out-of-range port failed only after the database, Oracle
  scheduler and audit outbox had started. Configuration loading now rejects
  anything outside `1..65535`, matching the existing command-line flag check.
- **Request metrics record the status clients actually received**: once a
  handler wrote a body, flushed a streaming response, or wrote its first header,
  `net/http` ignored any later status change, but the metrics wrapper still
  recorded the new code. A successful stream followed by an attempted error
  header could therefore appear as a failure in `/metrics`. The wrapper now
  locks the first committed status, matching the wire response.
- **A partial desktop store no longer claims to be empty**: the landing page's
  empty state only inspected lotteries, votes and Oracle sources. A store with
  registered candidates or sessions, or with token/NFT activity in its audit
  chain, could therefore show "Nothing recorded yet" and hide the quick-start
  guidance that belongs to a truly fresh install. The page and its quick-start
  note now share one check over every visible counter and recent ledger entry.
  A failed ledger read is treated as unknown rather than empty.
- **Production rejects short API keys**: the documented weak-key guard only
  compared against a handful of literal placeholders, so production accepted
  a one-character `AURORA_API_KEY`. Production keys must now be at least 32
  visible ASCII characters without whitespace, so a newline accidentally
  captured from a secret file cannot produce a server that rejects every
  request; development mode still permits short fixed keys for local-only
  workflows. The quick-start examples now use a placeholder that passes the
  production check.
- **CORS preflight advertises the PATCH endpoint**: the API exposes
  `PATCH /api/v1/oracle/sources/{id}` for enable/disable, but
  `Access-Control-Allow-Methods` omitted `PATCH`. An operator-configured
  cross-origin client could pass the origin check and still have the browser
  block every enable/disable request. The shared CORS method policy now
  includes `PATCH`, with a preflight regression test for the oracle route.
- **Unauthenticated request headers are capped before routing**: `net/http`
  defaults to accepting headers up to 1 MiB, and parsing happens before any
  Aurora middleware, logging, rate limit, or authentication. The API server
  now caps the complete header block, including the request line, at 64 KiB
  and returns `431` when a client exceeds it. Normal API headers are only a
  few hundred bytes, so this adds no practical client constraint while
  reducing the memory an anonymous connection can force.
- **Access logs and request metrics bound unauthenticated request-field cost**:
  after request IDs were capped, the request line itself could still approach
  the server's 1 MiB header limit. The access logger copied the full method and
  path, and module metrics split the path into segments before authentication.
  Logged methods are now capped at 32 bytes and paths at 1024 bytes; module
  classification reads only the first bounded segment without allocating a
  slice for the rest of the URL.
- **Client request IDs are bounded and response-visible**: chi accepted any
  inbound `X-Request-ID` value and the access logger recorded it before
  authentication, so one header approaching the server's 1 MiB header limit
  could amplify into a near-1 MiB log line. Request IDs now accept at most 128
  visible ASCII characters, replace anything else with a random ID, propagate
  the sanitized value downstream, and echo it in the response. CORS allows and
  exposes the same correlation header to configured origins.
- **Panics after a partial write now abort the response**: catching the panic
  and returning normally let `net/http` finish a chunked response as if the
  handler had succeeded, so clients could accept truncated data. Recovery now
  re-raises the standard `http.ErrAbortHandler` after logging, while request
  metrics and access logs use deferred recording so the failed request remains
  observable. Handlers that deliberately raise `ErrAbortHandler` pass through
  unchanged instead of becoming a misleading `500`.
- **Recovery respects responses committed by streaming handlers**: an
  explicit `Flush` sends the successful response headers even before a handler
  writes a body. The recovery wrapper did not record that state, so a later
  panic appended an `INTERNAL_ERROR` envelope to a response clients had already
  received as `200`, and request metrics reported `500` instead of the actual
  status. Successful hijacks are now tracked the same way, preventing recovery
  from attempting HTTP writes on an upgraded connection.
- **Container shutdown is safe under concurrent `Close` calls**: repeated
  closes were only ordered, not synchronized. The unexpected-listener and
  signal-shutdown paths could therefore both close the same SQLite handles,
  race on the closer slice, and invoke cleanup more than once. `Close` now uses
  a single guarded operation, waits all callers for the same result, and is
  nil-safe.
- **Oracle redirects no longer forward custom credentials cross-origin**:
  `net/http` strips `Authorization` and `Cookie` when a redirect changes
  origin, but it does not recognise custom credentials such as the documented
  `X-API-Key` source header. A compromised feed could redirect to another
  origin and receive that secret. Cross-origin hops now start with an empty
  header set; the transport re-adds only its own User-Agent, Accept, request
  ID and Content-Type defaults.
- **Oracle fetches enforce the standard redirect-hop limit**: installing the
  custom SSRF-aware `CheckRedirect` callback also replaced `net/http`'s
  default ten-hop cap. A hostile feed could therefore keep redirecting until
  the HTTP timeout expired. The policy now rejects the tenth follow-up with
  `ErrTooManyRedirects`, and a transport-level regression pins the request
  count.
- **API responses stay out of shared caches**: authentication uses `X-API-Key`
  rather than `Authorization`, so caches do not apply their usual protection
  for authenticated requests. Successful responses can include one-time voter
  private keys or oracle request headers, but they carried no cache directive.
  Every response, including CORS preflight, now includes
  `Cache-Control: no-store`.
- **Rate-limit retries advertise the actual reset time**: a rejected request
  previously received the configured window as `Retry-After`, even when most
  of that window had elapsed, and sub-second or fractional windows were
  truncated (`500ms` became `0`, inviting an immediate retry). The response now
  reports the remaining bucket lifetime rounded up to whole seconds, with a
  minimum of one second.
- **Oversized trailing JSON is reported as 413**: `decodeJSON` classified the
  body cap only during the first decode. A complete JSON value followed by
  enough whitespace to cross `MaxBytesReader` during the strict trailing-data
  check therefore returned 400 `INVALID_REQUEST`. Both decode attempts now
  share the same classification and return 413 `BODY_TOO_LARGE`.
- **SQLite database paths keep characters that look like DSN syntax**: every
  repository, event store, replay protector, migration and explicit ledger
  handle appended `?options` to the configured filename directly. A path such
  as `ledger?copy.db` was therefore truncated to `ledger`, and the rest of the
  real name became connection parameters. All production DSNs now share one
  file-URI encoder that preserves `?`, `#`, `%` and spaces while retaining the
  existing hardened options and `:memory:` support.
- **Blockchain initialization no longer hides database failures**: when the
  configured SQLite database could not be opened or its directory could not be
  created, `InitWithError` logged a warning and returned a genesis-only
  in-memory chain with no error. Commands could therefore report a successful
  ledger write that disappeared at exit. Initialization now returns the open
  failure and publishes no singleton chain.
- **Height stamping no longer rounds audit-ledger numbers**: adding
  `block_height` decoded the whole JSON record through `map[string]any`, which
  turned every integer into a `float64`; values above 2^53 could change on
  their way into the immutable chain, and the valid JSON value `null` caused a
  nil-map panic. Stamping now edits only the `block_height` entry through
  `json.RawMessage`, preserving every unrelated number exactly and rejecting
  non-object JSON without panicking.
- **Ledger corruption discovered after startup now fails closed**: the
  append-time database sync rebuilt the in-memory chain from rows without
  verifying them, then logged any sync error and continued appending. A block
  tampered while the process was running could therefore become the parent of
  a successful new write even though the next restart would reject the ledger.
  Sync now builds and verifies a complete candidate before replacing memory,
  returns `ErrLedgerCorruption` before either direct or transactional appends,
  and leaves the last known-good chain untouched. Ordinary transient read
  failures keep the existing in-memory fallback.
- **Corrupt persisted blocks no longer start a partial chain**: the boot
  reload logged and skipped any block row the SQLite driver could not decode,
  then exposed the remaining rows as a usable chain. A missing height made
  later appends collide with the still-present row and could leave integrity
  checks blind when the corrupt row was the tail. Reloads are now atomic,
  binding verifies a matching persisted genesis, contiguous heights, links and
  proof-of-work before installing write hooks, and the API/desktop container
  plus every CLI write path surface the load failure instead of accepting
  writes against a partial ledger. `VerifyIntegrity` now rejects height
  discontinuities as well.
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
