# Changelog

All notable changes to this project will be documented in this file.

Sections are labelled with the git tag they were cut from (0.0.1-alpha.0 in 2024
through v1.99 on 2026-10-21). Binaries report the tag at build time
(`-ldflags -X …Version=$(git describe …)`), so a build without those flags prints
the source default 0.0.1; `[Unreleased]` collects the work since the last tag.
The milestone notes under `.planning/` are history, not current instructions.

## [Unreleased]

### Changed

- **Many copies are tested not to pile up toasts**: the copy buttons are six
  independent sites and a fast operator can reach three of them before the first
  expires. MyGo shows the newest three toasts and keys a toast by its message, so
  repeating one confirmation refreshes it rather than adding a second. Aurora relies on
  both properties; the check now runs them apart — five messages leave the newest
  three on screen, and a repeat costs one slot instead of two.

- **Every copy is tested to confirm itself with a toast**: the reference says a copy
  button writes the clipboard and also raises a toast, because the status line is
  pinned to the page header and can be a card away from the value just copied. Four of
  the five sites had a behavioural test for the toast, each written for its own card;
  one structural check now covers all five and reports a sixth that arrives without
  one.

- **A copy is tested to take the whole value, not the shortened cell**: the id
  copy test uses short identifiers, and a copy of a short value is also a copy of the
  cell that shows it. A long value is where the two differ. `actionSubject` is
  deliberately outside this — it draws the whole id and lets the layout ellipsise it,
  so its text node and its clipboard value are the same string, and asserting on its
  rendered text would say nothing about the copy.

- **The session card is tested to drop its choose hint once an id is typed**:
  three cards tell the operator to choose a record and then swap the sentence for a
  plain field once one is chosen. Readings had a test for both halves; Session
  Controls had one for the buttons starting to work and none for the sentence going
  away, so the card could keep contradicting a filled field while every button
  worked. The check now runs the field both ways, including after it is cleared.

- **Every action button is tested to state when it is ready**: the reference
  promises a primary action is disabled until its fields are filled, rather than
  submitting and reading back an error. The handler-side half was covered; the visible
  half was checked only where a failure had been seen. Each shape is now pinned where
  it is decided — `primaryButton`'s `ready` argument, every call site passing one, the
  Oracle card's single `sourceReady` for its four buttons, and Voting's
  `sessionReady` for its two.

- **Every page header is tested to stay inside the window at its tightest**: the
  existing header check exercises Token, whose purpose sentence is one of the
  shortest. Settings has no action at all and the longest purpose line, so it takes
  the other branch of `header()` and was never measured at any size. Each page is now
  checked at the smallest window and the largest text, where a header runs out of
  room first.

- **Every page is tested to scroll its last card into view**: one page had a scroll
  check, at the smallest window and the largest text, because that page's button was
  the one that got clipped. The container is shared, but each page builds its own tree
  under it, so a page whose content is taller than the viewport could stop short and
  nothing would notice. Each page now scrolls to the end and its last card's bottom
  edge must be inside the window.

- **The Shortcuts card is tested to list what the header actually handles**: the
  card is the only place a reader learns the chords without opening the File menu,
  and nothing tied it to the registrations. A chord the header handles but the card
  omits is a command most operators never find; a chord on the card that nothing
  listens for sends them to press a key that does nothing. The comparison now runs
  both ways, and the number-shortcut test's comment was corrected from 1..7 to the
  eight pages its loop already covered.

- **The warning triangle is documented as marking one thing, not two**: its
  comment claimed an irreversible confirmation and a load that kept stale rows; only
  the first was true. A failed load draws the same status strip as success and busy —
  a dot and the word "Failed" — so the three read as one family. A comment that lists
  a use the code does not have tells the next reader this shape already covers a case
  it does not.

- **The layout scale is spelled, not numbered**: eighteen call sites wrote a
  spacing or radius value that is exactly one of the scale's constants. The scale
  exists so a card, a button and a page header agree on rhythm; a literal that
  happens to equal a step reads as a local decision and drifts the moment the step
  changes. Values are unchanged, so nothing on screen moves. One existing check
  located the language options by matching `Gap(16)` and caught the change
  immediately; it now matches `Gap(space4)`.

- **The timeline spine is tested to run through its dots**: the dashboard paints
  it with a Painter call, which leaves no element for `Tester` to measure, so the
  geometry can only be pinned where the numbers are written — an 18-wide dot column
  whose centre is 9, and a 1px spine at 8.5 whose span centres on the same 9.
  Changing either alone puts the spine beside the dots, which reads as two vertical
  rules rather than one line through the events.

- **Only the destructive button is tested to fill with the danger colour**:
  `Danger` is the page's "this will not come back" ink. A primary action filled
  with it would make an ordinary write look destructive, and a danger fill anywhere
  else would make that button less of a warning. Text and borders may use it
  freely, because those report a failure; the check is about the fill.

- **High contrast is tested to be higher contrast**: the contrast test runs both
  preference states against the same 4.5 floor, which says "readable", not "more
  readable" — a high-contrast palette identical to the normal one would have passed
  it. The comparison is now per surface and per ink, and the same test pins the
  documented boundary from both directions: the module accent must not be repainted,
  because a module's identity may not change with a readability setting, and the
  border must change, because that is one of the three things the setting affects.

- **The accent is checked against the navigation layer too**: the contrast test
  covered the window background and the card surface. The navigation paints a third
  layer, darker than both in the light palette, and nothing measured it. The lottery
  accent that shipped before the previous commit was 4.32 there — below the 4.5
  floor — while both tested surfaces passed. It moves again to `#964B29`, which keeps
  the new distance from the warning colour and clears the sidebar at 5.06.

- **The lottery accent no longer reads as the warning colour**: `pageAccent`
  promises a module's identity "can never read as success or failure". The lottery
  accent was `#9A5C00` and the light warning `#9A5B00` — ΔE 0.6 in CIELAB, one
  channel value apart — and in the dark palette `#F2C14E` sat ΔE 6.5 from
  `#F4C15D`. Both inks move to the orange the module's identity actually is, and a
  new check measures CIELAB distance rather than channel distance. Its floor is 15,
  the palette's own tolerance: the closest two module accents are ΔE 19.7 apart.
  Token stays ΔE 10.1 from success and is deliberately not asserted — every teal
  that clears the floor collides with the oracle or dashboard accent instead, so
  that range does not exist within this palette.

- **All three catalogs run the large-text boundary check**: it measured what a
  Chinese window at double text does to the layout and ran the two Chinese
  catalogs only. English is not the risky locale, but the assertion is about the
  layout and a layout regression is not locale-specific; running the three together
  also keeps the margins comparable, since each one's widest string has to fit under
  the same check.

- **Larger text is tested to move the breakpoint up and widen the sidebar**: both
  layout decisions multiply by the text scale, and neither interaction was tested —
  every existing layout test runs at the default scale, so dropping either multiplier
  left them passing. The breakpoint is now measured at one fixed width across four
  scales, and the sidebar by where the content area starts, bounded by its stated
  fraction of the window.

- **The two-column breakpoint is tested at its stated width**: the layout tests
  compare 1280 against 900, so a breakpoint nudged by a few pixels — or compared
  with `>` instead of `>=` — would leave both passing while the documented width
  stopped matching the behaviour. The one-pixel pair is now pinned: at
  `twoColumnMinWidth` the oracle rail sits beside lottery, one pixel lower it stacks
  under it, and the first rail's own position is asserted unchanged so the
  difference is attributable to the rail that changes column.

- **A fresh dashboard is tested to lead with the first move**: the empty state
  changes one thing about the layout — a store with nothing recorded opens on Quick
  start, while a store with records reads verdict → composition → activity with the
  quick start at the end as a reminder. All three sections were tested for presence;
  the order was not, and the order is what a reader acts on. The assertion is on
  vertical position, since both layouts contain the same headings.

- **Every rail and every voting metric is tested to open its own page**: the
  composition card has four module rails and three voting metrics, each a button
  with its destination baked in at compile time, and only the first rail had a
  test. Each case now starts from the dashboard and asserts the page it lands on,
  so a rail wired to the wrong module — or to the dashboard itself, which would
  look like a click that did nothing — fails instead of passing.

- **Every module that writes a record is tested to have a page to open**: the
  dashboard timeline takes a reader to the module that wrote a record through
  `pageForBlockModule`. A module the mapping does not know leaves the row as a
  button that hovers, presses and then discards the click. Two checks, one per
  direction: the five known modules must map to their own page rather than the
  dashboard fallback, and every `record.module.*` word in the catalog must map too,
  so adding a module to the vocabulary without a destination fails here instead of
  producing dead rows.

- **The two session time controls are tested to be told apart by their field
  name**: MyGo names each `TimeInput` spin button after the field's `Label` plus the
  segment word, so the Label is what distinguishes Start from End. The first version
  of the test looked at each control on its own and could not fail — giving Start's
  input the End label left every single-control assertion satisfied. It now requires
  the two names to differ and requires the form to carry both.

- **A field error is tested to reach the control, not just the space below it**:
  MyGo's `Field.Error` draws the message, marks the control invalid so its border
  turns red and assistive technology reports `AccessInvalid`, and prepends the
  message to the control's description. Only the first is visible to `Tester`,
  which exposes text and not the invalid state, so the call that asks for all three
  is pinned where it is made — together with a refusal to draw the message twice.

- **Every choice group is tested to carry its name**: radio groups, tab rows and
  checkboxes each render a set of controls that are indistinguishable without the
  group's name — three radios reading "Follow System", "Light", "Dark" say nothing
  about what they choose. All six are named today, and each shape is checked where
  it is decided: the `Label` for the radio groups, the argument for the checkboxes,
  and the call site for the tab rows' group name.

- **The ledger's record list now carries the name its pager already uses**: the
  reference promises a table or record list says which records it holds before
  reading the first one. The ledger's pager was named "Blockchain ledger" and the
  list under it was a plain column, so a screen reader entered two records with no
  idea what they were. `collectionList` names its own list from the argument every
  caller passes — all twelve pass one — and the three hand-built lists set the role
  and label themselves, which is where this went missing. A check now covers both
  shapes.

- **The pager is tested to stop at both ends**: Previous on the first page and
  Next on the last change nothing and announce no new range. The offset clamp
  already keeps the store from being asked for a page before the first or past the
  end, so those clicks are the observable half; the disabled conditions themselves
  cannot be observed by clicking, since `Tester.Click` presses coordinates and never
  reports a disabled control. A structural check pins both, because a dead button
  that looks live is the failure worth preventing.

- **The search announcement is tested to say the count in the window's
  language**: the structural check counted `announceSearchResults` calls, and
  nothing checked the words. No match is still a result worth speaking, one result
  takes the singular, and the rest take the count — in both locales, since Chinese
  has no plural form and its two entries agree. An unchanged query stays silent, or
  every repaint repeats the sentence.

- **The palette is tested to match and rank what the window shows**: rows render
  translated titles, so a query copied from the screen has to match that text and be
  ranked by it — otherwise typing part of a visible label finds nothing in a Chinese
  window. Both halves already looked at the original spelling and its translation;
  neither had a test. Writing it corrected an assumption: "备份数据库…" contains 数据
  but does not start with it, so its rank is 1, not 0.

- **The four busy gates are documented where they sit, and checked**: the File
  menu, the shortcuts, Refresh and the palette's reload/file entries all check the
  same `busy` flag, and each does it differently — the menu button is disabled so
  its menu is never built, the shortcuts return before reading their keys, the
  palette omits the entries from its list, and Refresh is a disabled button. The
  reference said what was blocked but not where, which is what a maintainer needs
  before touching one of them. A structural check keeps all four plus the
  deliberate exception: page entries stay listed, because a page switch does not
  touch the store.

- **The File menu is tested to stay closed while a write runs**: the busy gate had
  two halves and only the shortcuts were covered. The menu button is disabled for
  the same reason — a document dialog raised mid-write would ask the operator to
  pick a path while the store is mid-transaction — but nothing asserted it. The
  gate sits on the button, because a disabled `MenuButton` never builds its menu.

- **An import is tested to reload the page only when it stored something**: the
  menu action re-reads the page behind its own report dialog when records were
  stored and leaves it alone when nothing was — the difference between the
  imported draws appearing once the dialog closes and the operator seeing the
  history from before the import. The fixture has to be a real draw: `ImportRecords`
  re-derives integrity with the verifier, which reads the VRF output and proof as
  hex, so neither a hand-written record nor a base64-encoded one is accepted.

- **Cancel and a failed picker are tested apart for export and import**: each
  action opens a native dialog and then either writes, reports a failure, or does
  nothing. Cancel is "no error, nothing chosen" and must raise nothing; a picker
  that failed must say so. Collapsing the two into one `err != nil || empty` test
  erases the difference — which is what had happened to backup before it was
  fixed, and the boundary was unpinned for the other two.

- **A failed settings save now has a documented contract and a check**: three
  Settings choices persist to `desktop-settings.json` — language, theme mode and
  live updates — and each applies in the window first, then reports a failed write.
  A silent failure looks like success until the next launch reverts it, and the
  operator cannot tell that from having misremembered. The reference described the
  file but not the failure; it does now, and a structural check keeps the three
  branches, since a refactor can drop them while the controls keep working.

- **Cancelling the forget-keys confirmation is tested to change nothing**:
  turning memory off while keys are held opens a confirmation, because those keys
  may exist only in this process. Cancel means "keep both" — the policy stays on,
  the keys stay, and the checkbox agrees with the policy again, since otherwise the
  box reads as unchecked while keys are still held.

- **The generated key is tested to be offered only where it can sign**: the two
  key offers — the token owner key and the NFT creator key — each depend on a second
  condition the tests never moved, so a version that offered them on every tab would
  have passed. Transfer and Approve now assert both directions; Mint keeps the case
  with no account field; Burn reads its own id and holder, so the creator-key test
  covers it separately, including another asset with a matching holder.

- **An incomplete write form is tested never to reach the service**: six writes
  had no call counter in the fake — create token, mint, approve, transfer from,
  mint NFT, transfer NFT — so a test could only see the message, not whether the
  call went out. Each now counts, and the cases fill every field but one, so a
  guard that checked only some of them still fails. The assertion is the counter:
  a message beside a call that went out anyway reads as "I was stopped" while the
  ledger has already changed.

- **An empty verifier id is tested never to reach the service**: the reference
  lists four inputs the window refuses before calling a service — empty
  participants, a short seed, an over-large winner count and an empty verifier id.
  The first three asserted the fake's recorded call; the fourth had no test, and
  the fake did not count `VerifyDraw`, so it could not have had one. The assertion
  is the counter rather than the message, since a message beside a call that went
  out anyway still reads as "I was stopped".

- **A result and a busy write are tested to share the frame**: a write usually
  starts right after a result, so both strips are on screen — the previous outcome
  above, the running operation below. Neither may displace the other, and they stay
  distinct in speech: the result carries Done or Failed while the banner carries the
  operation's own name, so the two are not heard as one event.

- **A cross-field error is tested to blame the field to change**: a rule spanning
  two inputs still has to put its message on the one the reader must edit. The
  session window is the case the reference names — a missing start blames Start, a
  missing end blames End, and an end before the start blames End, since Start holds
  the value typed correctly. Swapping two error fields type-checks and still
  renders a sentence, just next to the wrong input.

- **A field error is tested to be announced once, and again after a correction**:
  `attachFieldError` speaks the message when it appears and stays silent on the
  frames that rebuild it, so a repaint does not repeat the sentence — but it also
  has to reset when the field is corrected, or the same mistake made twice would
  be silent the second time, leaving a red border with nothing announced. The
  status strip had a test; field errors are a separate path, because the message
  belongs to the field the reader is sent back to.

- **The card's group name is tested where it can be observed**: a card is a named
  group, and the name comes from `Label` alone — MyGo's default role inference
  resolves a labelled container to `RoleGroup`, so `formCard` does not also set the
  role. `ui.Tester` renders frames but exposes no accessibility tree, so the check
  reads `layout.go` and asserts what the helper does, including that it does not add
  the redundant role. The reference records the mechanism, since a later reader
  could reasonably add the explicit role and think it was required.

- **The documented keyboard shortcuts are tested to be registered**: the
  reference promises Ctrl/Cmd+1…8 in sidebar order, Ctrl/Cmd+, for Settings and
  Ctrl/Cmd+K for the palette. They are registered on the window rather than on
  buttons, so a refactor that moves the header can drop one without any page
  looking wrong — the action stays, just no longer reachable by its chord. The
  check also reads `pageOrder` back out of `pages.go` and requires eight entries
  with Settings last, since the count is what makes "1…8" true.

- **Every destructive action is tested to ask first**: four writes cannot be
  undone — burning tokens, burning an asset, deleting a source, forgetting session
  keys — and each pairs a danger button with `destructiveConfirm`. The pairing is
  what a refactor can drop, because the button alone still looks right and still
  works. The check also requires the dialog to expose `RoleAlertDialog` with its
  label and description, since a modal that draws a title but carries no role is
  silent to a screen reader.

- **A value card is tested to draw no row for a value it has not read**: the
  empty-read contract has two halves, and only the list half had a test. The value
  cards read something that exists by definition — a token's supply, an account's
  balance, an asset's holder — so a blank means "not read yet", and a row whose
  label sits beside an empty value would claim the value is empty. `readout` draws
  each row as two text nodes, so the check looks for that pair; a joined-string
  check cannot see it, because there is no "Balance:" anywhere.

- **The four NFT cards are tested to keep their own ids**: Inspect, Transfer,
  Burn and History each hold an id; a selection fills all four on purpose, but a
  typed id is a local edit, so retargeting a transfer must not quietly aim the
  burn at the same asset. `docs/desktop.md` now states the separation for both
  pages — four read cards and four write cards on the token page, Allowance's
  owner and spender included, and four ids on the NFT page.

- **The allowance read card's inputs are tested to be its own**: the token page has
  four read cards with separate inputs, and the test for that covered three.
  Allowance is the fourth and the only one with two inputs, an owner and a
  spender — and Approve, the write card, has its own pair, so reading one
  spender's allowance must not silently retarget the grant Approve would sign.

- **The burn and the source toggle now have tests**: of seventeen writes, those
  two had none, and both change something the same page shows — a destroyed asset
  still listed, or a switch that disagrees with its row. The burn is driven
  through its confirmation dialog by keyboard, the only path that reaches the
  service; clicking the button by label hit the tab behind the dialog and never
  called it. The id is set and settled before the key, because the frame that sees
  a new id re-derives the holder and clears the previous subject's key.

- **Every failed read is now tested to be visible**: a card that stores its error
  and draws nothing leaves a stale value on screen with no sign of it, which reads
  as a current answer. All thirteen list-level reads are driven to fail with one
  message, and that message must appear in the rendered text of all seven pages —
  a read whose failure is not meant to be visible would have to be exempted with
  a reason.

- **Every field error now has a rendered-copy test**: an error that draws nothing
  is invisible — the write is refused and no sentence says which input to fix —
  and a second call reporting the same message reads as a duplicate. All eight
  form fields are counted and required to render exactly once. The setup differs
  per case because the cards do: Setup shows registration forms over an empty
  roster and the session card only once a voter exists, and Create Token keeps
  its decimals input behind a collapsed Advanced section.

- **The session time fields report their error once**: both controls called
  `Error` twice with the same message — `attachFieldError` on the wrapping Field,
  then `Error` on the inner `TimeInput`. Only the first draws anything, since the
  inner element carries no field of its own, so the second call was a no-op that
  read as if it were doing something. Measured before removing: the first renders
  the message once, the second renders it zero times. A test now asserts both
  messages appear exactly once.

- **The reference documents only name paths that exist**: a citation that drifts
  is worse than none, since the reader follows it to a path that is not there. The
  three reference documents now have their `internal/` and `desktop/` citations
  checked against the checkout, with a document that cites none passing —
  `docs/cli.md` is a command reference and needs no path. `docs/architecture.md`
  is excluded on purpose: its tree uses bare directory names, already covered by
  the structure check, and the rest quotes gone paths by design.

- **The structure tree in the architecture reference is checked too**: README.md's
  project tree has a test that every `internal/` directory appears in it, and
  `docs/architecture.md` carries the same package set in its 目标结构 section with no
  such check — so a new directory would leave one tree stale and the other
  correct, in the document a reader goes to for the layering. The tree also names
  the `cmd/`, `desktop/` and `tools/` roots, which are asserted with it.

- **The translator's catalog copies are tested**: `Init` clones each shipped
  catalog, and the copy is what makes a locale file safe — `LoadLocaleFile`
  installs into `messages[locale]`, which for a shipped locale would otherwise be
  the package table itself. The test asserts the helper's property directly and
  then on the real English catalog. A first attempt drove it through
  `LoadLocaleFile` and could not be falsified, because a locale file names its own
  locale and never writes the shipped maps.
- **`joinCSV`'s doc comment sits with its function**: it had ended up at the bottom
  of `native_nft.go` while the function lives in `native_services.go`. The
  architecture document's line count for that file said 80 and is now 98.

- **The container's uniqueness as the wiring point is now guarded**: the
  architecture document calls the composition root "the only place that imports
  the storage and ledger implementations", and the layering rule skips that
  directory to allow it — which is also what would let a second assembly point
  appear unseen, since nothing checked where `infra` may be imported. A test walks
  `internal/app` and `internal/api` without the skip: exactly the container may
  reach `infra`, and it must still do so.

- **The two presentation layers depend in one direction, and the reference now
  says which**: rule 4 said `api` and `desktop` "must not depend on each other",
  but the desktop imports `internal/api` to start an in-process server and reuse
  the REST handlers — so the sentence contradicted the code, and the code alone
  could not say which half was intended. The rule states the direction now: the
  desktop may reuse the REST surface, the gateway must not know a window exists.
  A test walks `internal/api` and fails on any desktop import, while asserting
  the other direction stays.

- **The desktop module keeps its own layering rules, and now a test says so**:
  `TestLayerDependencies` walks `internal/*` only, so the nested window module had
  no import-direction check at all — even though it is a second presentation
  layer with two packages of its own. `nativeui` (views and view state) must not
  reach `app`, `infra` or `api`, and `deskapp` (the adapter) must not import
  `infra`, because the container hands it ports. Both hold today; production
  imports only, since the deskapp tests build a real store from `infra` the way
  the root module's tests do.

- **The token name and symbol ceilings now reach the places a user reads**:
  both are enforced — a ten-byte symbol is accepted and an eleven-byte one is
  rejected with `TOKEN_SYMBOL_TOO_LONG` — but neither appeared in the CLI help
  or the REST reference. The same command's `-d` flag already said "0-127", so
  sitting next to a flag that stated its bound, `-n` and `-s` read as unbounded
  and a client learned the limit only by tripping it.
- **The flag help names the bound in all three catalogs**, and the create-body
  row in `docs/api.md` carries it next to the decimals range it already had.
- **The check reads the specific catalog entry**, not the file. The first
  version asserted the number appeared somewhere in a 600-line catalog and
  stayed green after the help text lost its bound, because other entries carry
  small integers too; the mutation run caught that and the assertion was
  tightened.

### Changed

- **The ledger's own past-the-end fallback is tested too**: the other seven
  lists recurse to page zero inside their loader, but the ledger's offset and
  rows arrive in one page document, so its view notices an empty page carrying
  an offset, resets it and re-reads. That branch had no test, and the existing
  one covered the opposite case. Reaching it needs the store to shrink under an
  open page plus a read, since the view re-reads on a changed query or page size
  only — which is what keeps a repaint from scanning the chain.

- **The page-past-the-end fallback is now tested**: seven window loaders share a
  path nothing exercised — a read that comes back empty with a positive offset
  recurses to page zero, because an empty page carrying an offset is not an
  answer and would leave the list blank at a position that no longer exists. The
  pager cannot normally ask for that page, which is why it needs a test: the path
  is reached when the store shrinks between two frames. It is now the fifth
  documented paging invariant, and each loader asserts both the offset it
  committed and the rows it read.

- **The paging invariants are documented, not only the behaviour**: the section
  said what paging does and left the rules implicit — a query survives a page
  move, a highlight is restored by id rather than row number, a successful read
  commits rows and offset together, and the row count comes from the extra-row
  probe. Each is a decision someone could undo while the prose stayed true, so
  they are stated as invariants a change has to satisfy, along with why three
  failed voting reads report one line (`errors.Join` keeps every cause,
  `dedupeLines` collapses the strip).

- **Every paginated list is checked for its extra-row probe**: six window files
  ask storage for one row beyond the page they show and enable Next only when it
  comes back. Five lists had a boundary test for that behaviour; a change
  dropping the +1 from one read would turn Next on the last page and open an
  empty one, and the list it happened to would be the one without a test. The
  check names each read, because `voting.go` makes three of them and a
  constant-only check passes while one has lost its probe.

- **The window's paging ceiling is documented as wider on purpose**: the Query
  card accepts a limit up to 1000 while `GET /oracle/query` and the CLI's
  `oracle data` cap the same query at 100. The code comment said the defaults
  matched but not why the ceilings did not, which read like a missing clamp. The
  reason is the one the other collection cards follow — the window reads in
  process and pages a long history, the gateway bounds a caller that can repeat
  the request for free — and the property that matters is that the value the
  field accepts is the value the store is asked for.

- **The paging caps are in one table**: the ceilings were stated in five
  separate paragraphs beside their own modules, so a client planning a deep page
  had to read the whole reference to learn that `/nft/list` and `/token/history`
  stop at 1000 while the rest reach 10000. The table names the reason those two
  are tighter — their queries join per row — and a test reads each row's three
  cells against the constant its handler uses, which the previous prose check
  could not: it only looked for the number somewhere in the document.

- **Both limit-zero exceptions are named**: the entry above called `oracle data`
  the only command that does not read `-l 0` as "everything". `token history`
  is the second: it clamps a non-positive limit to its default of 50, for the
  same reason its flag default differs from the other lists. Both now sit in one
  table with their fallback and the reason, so a script passing 0 for "no
  paging" can see which commands ignore it.

- **`oracle data` is documented as the one limit-zero exception**: every other
  list reads `-l 0` as "everything", while `oracle data` clamps to [1,100] and
  falls back to 10 — unbounded would scan the whole readings table. A script
  that passes 0 for "all" on every list silently got ten rows here. The rule and
  its reason are now stated, and a test keeps the flag default, the clamp and
  the sentence in agreement.
- **The oracle paragraph read as two broken halves**: an earlier edit left
  "`oracle data` 的" ending a sentence that resumed three lines later, splicing
  the limit rule into the empty-read rule.

- **Both empty oracle answers are shown as examples**: the reference stated in
  prose that `/oracle/query` sends `{"data":[]}` and `/oracle/latest` sends
  `{"data":null}`, leaving a reader to reconstruct the bytes. The two are now
  copyable, with the reason they differ — a collection's empty answer is an
  empty array, a single record's is null — and a test keeps both documented,
  because `if (!res.data)` treats an empty page as a missing record while
  `res.data === null` misses the empty page.

- **Clearing a search brings the whole page back, and now a test says so**: the
  search tests covered a query narrowing the rows and choosing from what was
  left; none covered emptying the field. A filter written as a one-way narrowing
  — "keep the row only when a query is set and it matches" — would leave a list
  permanently short with no way to widen it, and nothing failed. The four
  filtered helpers and the two inline filters are covered, and
  `docs/desktop.md` states the property every future filter must keep.

- **A search hint is now checked against what its search does**: eight lists
  carry a sentence beside the field, and the sentences say two different things
  — six filter the rows already loaded ("Search narrows the current page"), and
  the ledger plus lottery history query storage across the whole set. Swapping
  them leaves a reader hunting for an old record in a page they cannot see, or
  trusting a filtered count as the whole store; the words are the only signal.
- **The check ties each sentence to its code path**: the two adapter methods
  still take a search term and reach storage, the page-local lists still filter
  with `containsFold`, and exactly two hints may promise a whole-store search.

- **The i18n tests now check the call, not only the spelling**: the existing
  test compared the English and Chinese spellings of one key; nothing checked
  that the call site passes as many values as the message takes. A message with
  two verbs and one argument prints `%!d(MISSING)` in every language, and no
  test failed.
- **The check reads the verbs from `messages_en.go`** and the arguments from
  every `GetTextF`/`TFormat` call in `cmd/` and `desktop/`. The desktop's
  dialogs read the same catalog, so one verb table covers both. A guard test
  compares the argument counter against `fmt` itself, so a wrong counter cannot
  make the whole check vacuous.

- **The reference says how a client notices a clamped page**: an over-limit page
  request is reduced, not refused — `limit=9999` answers 200 with
  `"limit":100`. That is deliberate, but silent clamping is only safe when the
  caller can find out. `/blockchain/blocks` echoes the values it applied;
  `lottery history`, `token list` and the rest return a bare array and cannot.
- **The paragraph says what a client can do about it** — keep the request inside
  the ceiling, or notice the returned count stopped at it. A client that assumes
  it received everything drops records with no error anywhere.
- **A handler test pins the behaviour** (200, clamp, echo, and negative or
  garbage values falling back to the default), and a `repocheck` test pins the
  paragraph to it.

- **The reference states that every timestamp is unix seconds**: twenty-two
  places build timestamps with `time.Now().Unix()` and nothing uses
  `UnixMilli`/`UnixNano`, but only the voting session fields said so. A client
  reading `timestamp` or `created_at` had to guess, and the common guess is
  milliseconds — which renders 1970 instead of failing. The paragraph names the
  fields it covers, and the CLI's shape table spells the unit beside the one
  column that mentioned a timestamp.
- **A test asserts it from both ends**: no builder may call a sub-second
  constructor, no documented sample may carry a value outside the plausible
  seconds range, and the paragraph must remain.

- **The reference names the second request/response spelling difference**:
  `candidate_ids`/`candidates` was documented; `voter_public_key`/`public_key`
  was not. The vote sample showed the request key and the prose said "the
  registration response's key pair", leaving the reader to notice the response
  calls it `public_key`. Posting a registration response back returns 400 with
  an empty public key, and the error names neither field.
- **The session round-trip needs the same translation**, since `/session/{id}`,
  `/start` and `/end` all answer with `candidates`. A test marshals both DTOs and
  requires the paragraph, so a rename on either side fails there.

- **The reference says which write response has no `block_height`**: every token
  write carries it except `approve`, because `ApproveEvent` has no block height
  — the domain records the event id, both public keys, the amount and a
  timestamp. A client that infers success from the field's presence therefore
  misreads an approval, and the shape table lists all five writes together.
  The paragraph also says how to find the block an approval landed in.
- **The desktop guide says Settings has no Refresh**: the page reads no
  database, so the header button and the palette entry are both absent, and the
  guide listed Refresh without that qualifier.
- **Both are pinned to the code**: a RepoCheck test fails if `approve` gains the
  field or a sibling loses it, and the Settings rule already had a UI test.

- **Three catalog keys for one placeholder are now one**: `voting.none`,
  `migrate.none` and `common.none` all said "(none)" — three keys a translator
  keeps in step for a placeholder with no module meaning, and three places a
  reviewer checks when the wording changes. The four call sites read
  `common.none` now.
- **A test looks for the same text under two keys of one prefix**, which is the
  shape the duplicate took. It deliberately does not compare across prefixes:
  `token.owner` and `nft.owner` are both "Owner public key" because both modules
  have an owner, and collapsing them would name a key after the wrong module.

- **The API reference's pagination numbers are checked against the handlers**:
  nine endpoints state a default page size and a ceiling in prose, and every one
  of those numbers is a constant in a handler file. Nothing compared the two, so
  a changed ceiling left a wrong number in the document that a client plans its
  paging around.
- **The tighter offset ceilings are asserted separately**: `/nft/list` and token
  history stop at 1000 because paging that deep is a scan, while every other
  list reaches 10000. Flattening the two is the edit most likely to go
  unnoticed, and it now fails the check twice.

- **The CLI reference says the page envelope's zero limit means two things**:
  `aurora blockchain blocks -l 0 --json` echoes `"limit": 0` and lists
  everything, while `GET /blockchain/blocks?limit=0` falls back to the default
  and echoes `"limit": 20`. A script comparing `len(blocks)` with `limit` to
  decide whether a page was truncated read the CLI's zero as an empty page — the
  opposite of what it means.
- **The paragraph states both behaviours** and how a script should handle the
  zero, since the two numbers sit under the same key. A `repocheck` test pins
  the CLI's zero to "no cap" and the gateway's to "use the default".

- **The API reference says which health fields come and go**: `/oracle/health`
  omits `last_success_at` until a fetch has succeeded and `last_error` once a
  fetch recovers, but the endpoint was described without mentioning either. A
  client written from that description reads a missing key as zero — "succeeded
  in 1970" for a source that has never been reachable, or "no error" for a
  source whose last fetch failed.
- **The paragraph names both fields**, says a missing key is the answer rather
  than a zero value, and lists the four keys that are always present. A
  `repocheck` test marshals the real `SourceStat` in both states so the tags and
  the reference cannot drift.

- **The CLI write commands show how their flags fit together**: `aurora token
  mint --help` said "Mint tokens" and then listed `-t`, `-o`, `-a` and `-k`
  with nothing to say how the four belong in one line. All of them are
  required, so a reader learned the shape by failing. `token create` and
  `nft mint` two commands away each showed two runnable lines, and
  `docs/cli.md` had the examples — but `--help` is where a reader is when the
  command fails.
- **`token mint`, `transfer`, `approve`, `transfer-from`, `burn` and `nft
  transfer`, `burn` carry one example each**, run against a fresh database
  before the change landed.
- **A test requires every write command to carry an example**, that each
  `--flag` in it exists, and that every required flag appears, so a documented
  flag that no longer exists cannot send a reader to an error message.

- **The API reference no longer claims the gateway's search follows the reader's
  language**: `docs/api.md` said the blocks `q` filter matches "the same
  sentence the CLI's `-s` and the window's ledger search compare". That stopped
  being true when those two surfaces started also matching the words their
  reader sees — a Chinese window finds a record by 中奖者; the gateway does not.
- **The difference is intended**: a gateway request has to mean the same thing on
  every process, so `q` matches the English vocabulary the decoder emits. The CLI
  and the window can add the reader's language because each knows which language
  its reader sees; for the API that would make one query return different rows on
  two deployments of the same data. A handler test now proves it, and making the
  gateway translate fails the test.

- **`blockchain blocks` had the wrong row in the `--json` shape table**: it was
  listed among the commands that print an array. It prints
  `{"blocks","total","limit","offset"}` — the same page envelope as
  `GET /blockchain/blocks` — so a script written from that row and indexing the
  output as a list read null, quietly, on the first run of a fresh install.
- **A CLI test runs every command in the array row on an empty database** and
  requires an array, then checks the page's four keys and that an empty ledger
  sends `[]` rather than null. A one-row documentation error is invisible from
  the command succeeding.

- **The CLI reference says where the `--json` envelope differs from REST**: the
  page claimed the documents carry fields "consistent with REST". That holds for
  every list command but one — `oracle source list --json` prints a bare array
  while `GET /api/v1/oracle/sources` sends `{"sources":[...]}`. A script written
  from that sentence and pointed at the endpoint reads an array where the API
  sends an object.
- **The difference is deliberate**, since the oracle read endpoints wrap their
  payloads (`/oracle/query` and `/oracle/latest` send `{"data":...}`) and other
  modules' lists do not, so the reference states it rather than leaving a reader
  to discover it. The elements are identical; only the outer shape differs.
- **`repocheck` pins the claim to the code**: the envelope type, the CLI's
  `printJSON(resp.Sources)` and the paragraph all have to stay.

- **The API reference now names the production switch**: the auth section said
  "in production the key must be at least 32 characters" without saying how a
  process enters that mode. `AURORA_ENV=production` is the only channel — no
  config key, no flag — so a container missing the variable runs in development
  mode and generates a key instead of rejecting an empty one. That is the
  failure the paragraph warns about, and it did not name the switch.
- **`repocheck` pins that paragraph to the code**: the variable, the
  32-character minimum and the "no other channel" sentence all have to stay.

- **Two DB-path tests pointed at a variable nothing binds**: the success and
  failure cases in `internal/api/router_test.go` set `AURORA_DB_PATH`, which no
  code binds (`Load()` binds `api.key`, `server.host` and `server.port`;
  `AutomaticEnv` would look for `DB.PATH`), so both ran against the checkout's
  `./data/aurora.db`. They passed while proving nothing, and the failure case
  took the "no error" branch it tolerated. A real `aurora-api` run with
  `AURORA_DB_PATH` set created `./data` instead, which is how this surfaced.
- **The failure case now asserts the error and a nil server**, because
  accepting either outcome is what hid the defect.

- **The ledger search description names what it matches**: the field said it
  "matches module, payload or height". The adapter compares the module, the
  action, the subject, every fact label and value, the summary, the raw payload
  and the height — and, since the rows are translated, the words as the window
  draws them. A reader with 抽奖 on screen could not tell the field would find
  it. `docs/desktop.md` repeated the same short list in two places.

- **The English catalog writes base64 in lowercase**: four flag labels said
  "Public key (Base64)" while every document, both Chinese catalogs and the Go
  package say base64. The house-word check already listed the spelling, but it
  scanned only the Chinese catalogs, so a Latin word could drift in the one
  table nothing read. The scan now includes `messages_en.go`; the Chinese pairs
  cannot match English text, so it costs one pass.

- **Four module help pages now explain the module**: `aurora nft --help` opened
  with "NFT system" and then listed its subcommands. Lottery and blockchain
  already had a `Long` sentence; voting, oracle, NFT and token pointed `Long` at
  the same one-line `Short`, so the paragraph a reader lands on repeated the
  name and added nothing. Each now names the operations and the rule a reader
  needs before choosing a subcommand — one signed ballot per voter per session,
  a source's method/JSON path/interval, a creator key for minting, and the
  ERC-20-style allowance flow.
- **The check compares `Long` with `Short`** rather than matching words: a module
  may restate its name, but its help has to add something the summary does not.

- **The window calls its one action Refresh everywhere**: the header button, the
  command palette and the shortcut reference all reach the same method, but the
  Settings card listed the chord as "Reload (data pages)" and the palette's
  detail as "Reload the data shown on this page". A reader looking for the
  Refresh button found a different name for the same key.
- **The dead `Reload` key left both catalogs**: nothing drew it once the card
  was renamed, and an unread key reads as coverage the window does not have.

### Fixed

- **Every highlight lookup now has a structural check**: eleven helpers move a
  list's highlight to a named record — four after a reload, seven after a create —
  and a miss must leave nothing highlighted, because the write cards read their
  subject from it. The same shape was written wrong twice, so the check requires
  each function's first assignment to clear before the loop: reverting last
  round's three files fails it five times, the count of helpers that were wrong.

- **A lookup that finds nothing clears the highlight**: five helpers highlight
  the record the page was just told to show — after a mint, a draw, a source add,
  a registration, a session create. Each looked the record up by id and, on a
  miss, returned without touching the highlight, leaving the previous row
  selected. The write cards read their subject from that highlight, so a miss
  left the next transfer, burn or ballot naming a record the operator never
  chose. All five now clear before the lookup, as the four restore helpers
  already did.

- **A mint's highlight is tested where position and id disagree**: the existing
  mint test had the new asset land at index 0, where selecting by id and
  selecting the first row give the same answer. The collection shows whatever
  order the store chose, so this test has another asset come first and asserts
  both the index and the row the highlight lands on — selecting by position there
  would point every write card at the wrong asset.

- **The session highlight is tested like the other two**: three helpers restore
  a reload's highlight by record id rather than row index, so a filter that
  reorders the list still lands on the same record. Source, voter and candidate
  were covered for a reorder, a vanished record and an empty id; the session
  helper has the same shape and had none of it — and the write path depends on
  its roster as well. The three tests now cover it.

- **A loader commits its offset only after the read, and now a check says so**:
  the paging invariants state that rows, offset and `HasMore` are committed
  together once the read succeeded. Every loader does that today and nothing
  failed when it stopped — a commit moved above the error return leaves the page
  label describing rows that were never loaded, so a failed refresh's stale note
  points at the wrong page. The check covers all nine loaders, including the
  ledger's, which takes its offset from the page document.

- **The ordering now has a structural check**: all six lists record the applied
  query before their empty-list return, and the rule is invisible in normal use,
  so a repocheck test requires each guard to be preceded by — or contain — that
  comparison within its own function. It reads the guard's block rather than the
  whole function, because the fixed code keeps a second comparison on the normal
  path and a function-wide search accepts the broken version.

- **The token, NFT and oracle lists had the same ordering**: their applied-query
  update also sat after the empty-list return, and all three gate their
  first-record panel on that field, so a query cleared over an empty store showed
  a work bench and hid the guide to creating the first record. Six lists in two
  rounds, all one shape — the field describes the field, not whether rows
  survived the read.

- **A query cleared while the list is empty is still recorded**: the voter,
  candidate and session lists updated their applied-query field after the
  empty-list return, so a query cleared at that moment kept looking applied. Two
  things followed: `votingNeedsFirstRecord` counted the stale query as "the
  reader has entered", so an empty store showed a work bench instead of the
  first-record guide, and the next read announced itself as a search result. The
  field's state is a fact about the field, not about whether rows survived.

- **Changing the oracle limit re-reads the card, and now a test says so**: the
  Query card compares the Limit field against the limit its rows answer and
  re-reads on a change, so the number in the field describes what is on screen.
  Nothing covered it — the existing cases changed the subject, not the page size,
  the same gap the NFT collection had. `docs/desktop.md` now states which cards
  re-read on a size change (those whose list loads itself) and which return to
  "not loaded" for their own button (the explicit-read history cards), because a
  silent re-read there would leave the reader unsure whether the rows came from
  the button or the edit.

- **A new NFT page size re-reads the collection**: the Show field only changed
  how the *next* page would be read, so the number in the field contradicted the
  rows under it until something else forced a load. The ledger's Show field
  re-reads on a size change; the collection now does too, starting at the newest
  rows and clearing the highlight. The token History card is deliberately not
  changed: it is an explicit read with its own button, so a new limit returns it
  to "not loaded yet" and the reader presses Load History — the collection has no
  such button, it loads itself.

- **`/blockchain/verify` now says what `length` measures**: it counts the chain
  including genesis, while `/blockchain/blocks` reports `total` without it, so
  the two endpoints answer 3 and 2 for one ledger. The reference states the
  difference and ties it to the CLI labels that say the same thing.
- **The block-page example contradicted itself**: `total: 2` beside one row with
  `limit: 20`, a limit large enough to hold both. `total` is 1, matching the row.
  A new check derives the expected row count from `total`, `limit` and `offset`
  and compares it with `len(blocks)`, because the parse check that already
  existed only asked whether the JSON was well formed.

- **db-info and doctor name the block count they report**: a ledger with one
  record answered three ways — `lottery db-info` said "Total blocks: 1",
  `blockchain blocks` said "Audit records: 1", `blockchain verify` said
  "Blocks (including genesis): 2". The first two count the same rows
  (`height > 0`), but only one said so, and "Total blocks" was the phrase verify
  had just dropped for reading as the chain's length. db-info now says "Audit
  records" and the doctor says "ledger blocks including genesis", so the two
  camps are recognisable from the text alone.

- **The verify count says it includes the genesis block**: `blockchain verify`
  printed "Blocks: 2" on a ledger whose `blockchain blocks` printed "Audit
  records: 1". Both were right — verify reports the chain's length, which counts
  genesis, and blocks reports audit records, which does not — but the labels
  differed by one plural, so reading them side by side looked like a miscount.
  It now reads "Blocks (including genesis)", and `docs/cli.md` states the
  difference beside the commands.

- **The desktop latest-value card says when a read came back empty**: a source
  with no readings drew nothing — no rows, no line — so an answered empty read
  looked like a card that had never been used. `latestLoadedSource` marks the
  read, so the line appears only after one; before that the field and its button
  are the card's content. The value cards keep their rule of never inventing
  rows, and Latest is documented as the exception, because "the newest row so
  far" is legitimately absent before the first poll.

- **The latest reading says which card answered and what to do**: `oracle
  latest` printed a bare "No data found" for a source with no readings — no
  heading, no next step. Its sibling `oracle data` prints "📊 Oracle Data:" over
  an indented "(none)", so the two cards of one module answered an empty read in
  two different shapes. It now prints "📈 Latest Data:" over a sentence naming
  the next step; `--json` keeps returning `{"data":null}`.

- **The ledger heading names which count it shows**: `blockchain blocks --limit 2`
  printed "Audit records: 4" above two rows. The number is the total, the rows
  are the page, and the heading said neither — a reader had to count rows to
  tell whether the ledger held four entries or the filter had narrowed it. When
  the page holds the whole ledger the plain total stands; when it does not, the
  heading names both: "Audit records on this page: 2 shown of 3". The empty cases
  keep their own sentences, so "0 shown of 0" never replaces the line that says
  whether the ledger is empty, the page is past its end, or nothing matched.

- **The NFT history names what its number counts**: `nft history --limit 5`
  printed "Operations: 1 (offset 0)" — the number is the page's rows, but the
  heading did not say so — while the unpaged run printed "Operations: 1" for the
  whole history. One label served two different counts. Both modes now name
  their own: "Total operations: 1" and "Operations on this page: 1 (offset 0)".

- **Every list heading carries the count it describes**: four lists dropped the
  number on the unpaged path — `📋 Candidates:`, `📡 Data Sources:`, the voters
  and sessions headings. A heading with only a colon reads as a section label,
  so an empty store and a populated one looked the same until the rows were
  read, and `docs/cli.md` already promised the count. Each list now has two
  labelled headings: `Total candidates: 3` without paging flags, `Candidates on
  this page: 2 (offset 0)` with them.
- **The used-key check matched keys as substrings**: `voting.candidates` counted
  as read because `voting.candidates_page` contains it, so three labels with no
  call site stayed in the catalog. Requiring the quotes makes the check what the
  call sites actually contain, and the three dead labels are gone.

- **Every paged list answers about the page, not the store**: `token list
  --offset 9`, `oracle source list --offset 9` and the NFT history read only
  `--limit` when deciding whether a request was a page, so an offset-only
  request kept the plain heading and answered "Total: 0" or "(none)" — claims
  about the store that a non-empty store contradicts. All three now take the
  paged heading and answer "No entries on this page", as `lottery history` and
  the voting lists already did.

- **An offset alone asks for a page in the voting lists**: `voting candidate
  list --offset 9` and `voting session list --offset 9` printed `(none)` when
  the store had records. Both read only `--limit` when deciding whether a
  request was a page, so an offset-only request fell into the empty-store branch
  and claimed there was nothing to list. Voters already had the three answers;
  both lists now take the paged heading and answer "No entries on this page".

- **An empty list dropped the heading that carries its count**: `token list`
  printed a bare "No tokens yet", and `lottery history` did the same for an
  empty store or an empty search, while their non-empty runs start with a
  heading ("Total: 1", "Lotteries on this page: 0 (offset 0)"). The bare
  sentence dropped the number, so a reader and a script could not tell an empty
  store from an empty page — which is what the three distinct messages exist for.
  Every branch now prints the heading first, with the empty sentence indented
  under it, as `nft list` and the voting lists already did.

- **An empty oracle query drew nothing**: a source that had been read and
  returned no readings left the Query card blank — no rows, no line, nothing
  where the list belongs. A blank region reads as "this card has no content",
  not as "this source has no readings yet".
- **The other two states already had answers and keep them**: the
  missing-source prompt sits beside the field it fills, and a failure returns
  through `loadWarning`, which has already reported it. A "not loaded yet" line
  was written and removed — opening the Readings tab reads the selected source,
  so that sentence was unreachable, and only the line that renders was added to
  the catalogs.

- **The token history named a missing first step the wrong way**: an empty
  Token ID printed "No transfers loaded yet.", the same line a chosen but unread
  token gets. The NFT history and the voting tally already answer the two cases
  differently, and the difference matters — one asks the reader to pick a
  subject, the other points at a Load button that is disabled until an ID is
  typed. A read that returns nothing still says "No transfers found for this
  query.", so the card now has three answers where it had two.

- **A stage capture photographed the top of the page and passed**: every NFT
  stage and the token Burn stage came out byte-identical to their page captures
  — `nft.png`, `nft-Burn.png` and `nft-History.png` were one file. The capture
  still passed because its assertion looks for text in the accessibility tree,
  and the tree lists every element whether or not the viewport shows it.
- **Cause**: `-scroll-to` dispatched the accessibility action
  `scrollIntoView`, which scrolls only as far as the view's containers allow in
  that frame. A stage body below a tall card reported success and did not move.
- **The capture now wheels over the content area**, a notch at a time, until the
  element's box is inside the window's content area — the protocol reports both,
  so the check is measured. A target above the fold scrolls the other way, and
  the loop is bounded. A repocheck test forbids going back to the accessibility
  action and requires the box check before the capture.

- **Four handlers said "required" four different ways**: the message a client
  meets while wiring its first request was `"token_id required"`,
  `"token_id is required"`, `"token_id and owner required"` and
  `"token_id, owner and spender required"`, while the oracle endpoints already
  said `"source parameter is required"`. A reader comparing two endpoints could
  not tell whether the missing word meant anything.
- **All four read "… is required" / "… are required" now.** A `repocheck` test
  pins the shape rather than the nouns, so an endpoint that gains a field can
  extend its list without touching the rule.

- **An empty page printed a zero count and no explanation**:
  `aurora voting voter list --limit 5` and `aurora lottery history --limit 5`
  printed their heading — "Voters on this page: 0 (offset 0)" — and nothing
  else. Candidate, session and oracle-source lists print "(none)", and the
  ledger already distinguishes three cases; these two did not, so an operator
  saw a zero with no statement and could not tell a silent failure from a page
  past the end.
- **Both print a page-empty line** that names the two ways out (`--offset`, or
  drop `--limit`). A filtered miss keeps its own sentence, because "No draw
  matches" and "no rows on this page" are different answers, and a test covers
  all three states for both commands.

- **Six list commands advertised a token ceiling**: every list command registers
  the same `--limit` flag, and all of them read the token catalog's entry.
  `aurora lottery history --help` said "Maximum tokens to list",
  `aurora oracle source list --help` said it too, and a reader checking how many
  sources they could list was told about tokens.
- **The token list keeps that wording**, since it is what the command lists; the
  other six read a new `common.list_limit`, "Maximum records to list". A
  `repocheck` test maps each paging flag to the key it reads, so a copy-pasted
  flag line cannot reintroduce the leak.

- **A break at the first block reported no broken index**: `IntegrityReport.
  FirstBrokenIndex` carried `omitempty`. The field is `-1` for an intact chain
  and the break's index otherwise, so `0` is a real answer — the nil-chain path
  reports it. That case serialized as `{"valid":false,"length":3,
  "first_broken_reason":"nil chain"}`, so a caller following the documented
  contract decoded the field's zero value and could not tell "broken at the
  first block" from "intact".
- **The field is present in all three cases now**, and both a ledger test and a
  `repocheck` guard pin it — the defect only appeared in one of them.

- **The doctor report and the chain verdict line up too**: these are the two
  blocks a user reads when something is wrong, and they kept the old shape — a
  two-space indent with a hand-written `Printf` per line. `doctor` printed five
  labels of different lengths at five different value columns, and
  `blockchain verify` stacked `Blocks` over a padded `Head` over `State`.
- **`printMarkedFields` puts the status mark in its own column** so the row that
  warns does not shift its value away from the four that do not.
- **A test measures where each doctor value starts** in display columns and
  requires one column, so the padding cannot silently disappear again.

- **The CLI's field blocks line up their values**: a result block printed each
  "Label: value" line at a fixed indent with no column — "ID: MTK" beside
  "Total Supply: 1000000" beside "Owner: …" — so the eye had to re-find the
  value on every line. In Chinese it was worse: 所有者 is six terminal columns
  and nine bytes, so padding by byte length would have left the block less
  aligned than before.
- **`printFields` takes the whole block** and pads each label to the widest one
  in display columns. `i18n.DisplayWidth` counts a Han character and a
  full-width mark as two columns, which is what the Chinese catalogs print.
- **The blocks were also split between two- and three-space indents**, one
  helper here and a hand-written `Printf` there; they share the helper now.
- **Tests assert against the value rather than the exact spacing**, so a wider
  label elsewhere in a block does not break an unrelated check, and the
  alignment has its own test comparing the value's column in both languages.

- **`--interval` advertised its value type as "source add"**: `aurora oracle
  source add --help` printed `-i, --interval source add`. The flag is an int;
  pflag reads a pair of backticks inside a usage string as the type
  placeholder, and the English catalog wrote "on `source add` -i is the
  interval", so those words replaced the type on every help screen that flag
  appears on.
- **A repocheck test rejects backticks in any flag usage string**, because the
  rule is a pflag behaviour no compiler or linter reports.

- **A capture could be filed under a size it never had**: `AURORA_E2E_SIZE`
  accepted any `WIDTHxHEIGHT` and named the output directory after it. The
  window refuses to open below 900x600, so `AURORA_E2E_SIZE=700x520` opened at
  900x600 and wrote those PNGs under `700-520` — a reader would take them as
  evidence of a layout nobody saw. The script refuses such a size now.
- **`AURORA_E2E_OUT` was resolved after `cd "$root/desktop"`**, so a relative
  value was relative to the desktop module: `AURORA_E2E_OUT=dist/e2e2` compiled
  the app into `desktop/dist/e2e2` and then failed to launch
  `dist/e2e2/automation-shot`. The path is made absolute before the build.
- **Two repocheck tests keep both rules**: the refused floor must equal
  `WindowOptions`' `MinWidth`/`MinHeight`, and the normalisation must happen
  before the build changes directory.

- **The window's ledger search only matched the English words**: in a Chinese
  window the row reads 抽奖 · A（中奖者：A）, and typing 中奖者 answered zero —
  the adapter compared the words the chain stores. The CLI's `-s` had the same
  defect until it started matching both renderings.
- **`recordSearchText` also carries the words as the window draws them**, while
  the English ones stay, so the vocabulary in the documentation and in `--json`
  still matches in every language. `nativeui.RecordVocabulary` is exported for
  the adapter, so the search and the row resolve a word through one key.

- **An NFT transfer read 转账 in the window — the token word**: the desktop
  resolved a record's action through its label table by the bare English word,
  and `Transfer` is one entry, so the NFT ledger row printed the word the
  token page uses. The repository's own term rule says an NFT transfer is 转让
  and a token transfer is 转账; the row and the rule disagreed on the same
  screen.
- **`records.TextL` hands the record's module to the resolver**, and
  `records.ScopedKey` builds `record.<kind>.<word>@<module>`. A per-module entry
  is tried first and falls back to the bare one, so only the word that differs
  per module carries a second entry.
- **The window and the CLI now share one vocabulary**: the desktop resolves all
  three kinds through the same `record.*` keys, which is the drift this removes
  — two catalogs had been carrying the same words and had already disagreed.
- **`repocheck` also checks the scoped entries**: a module the decoder never emits
  or a missing bare fallback fails the check.

- **Translating the row made the ledger filter read one language**: in a Chinese
  process `aurora blockchain blocks -s Winners` found nothing, although the
  documentation, the `--json` payload and every log call the fact that. The
  reader who sees 中奖者 on screen and the reader who arrives from the docs have
  to match the same records, so `blockMatches` compares both renderings; an
  English window still does not answer a Chinese query.
- **The first attempt panicked on the filter path**: it passed a nil lookup to
  `TextL`. Parsing once and resolving through `recordLookup` removes the
  possibility instead of guarding it.

- **A Chinese ledger row printed its vocabulary in English**: in a Chinese
  process `aurora blockchain blocks` showed
  `lottery · Draw · C, A   (Winners: C, A; Participants: 3; Seed: seed-abc)`
  under a 审计记录 heading, while the desktop's ledger page rendered the same
  record translated. The decoder names its words in English and the CLI printed
  them verbatim; `docs/cli.md` claimed the two surfaces already shared the
  vocabulary.
- **`records.Record.TextL` resolves each word through the caller's lookup**, and
  `TextIn` takes the punctuation separately, because Chinese sets ：；（） where
  the Latin spelling uses `":; ()"`. An action that resolves to the module's own
  name is dropped, as the window does.
- **Fact values and subjects stay verbatim**: they are recorded data, not
  vocabulary. A translated public key or seed would no longer match the payload
  an auditor compares the row against.
- **`repocheck` reads the decoder** and requires a `record.*` entry for every
  module, action and fact it can emit, so a new module cannot arrive
  untranslated.

- **A single ballot printed as "1 votes"**: `voting results` and
  `voting candidate list` render every tally through one plural entry, so a
  candidate with one ballot read "Alice (Greens): 1 votes" — the one place a
  reader compares candidates by number. `voteCountText` picks a singular key at
  one; both keys carry the same 票 in Chinese, which has no plural form.
- **The tests pinned the wrong string**: they asserted "1 votes" and would have
  defended the defect. They check the singular form now, and the plural case
  keeps its "0 votes" assertion so both stay covered.

- **`genzhhant -check` could never pass**: it compares the checked-in
  Traditional catalogs with what the generator produces, and three reviewed
  wordings — 批准額度 and its two neighbours, and 字節 where OpenCC says 位元組 —
  existed only in the generated file. A correction applied by hand read as
  permanent drift: the gate failed on every run, and nothing distinguished a
  reviewed wording from an accident.
- **The corrections live in the generator now**, in `reviewedWords`, applied
  after conversion. 上游 is the entry both catalogs needed: OpenCC's character
  table turns 游 into 遊, and 上遊 means "to go on a trip" rather than an
  upstream feed. The two byte-size entries settle on 字節, the spelling the core
  catalog already uses for the same unit.
- **`-check` names the entries that differ**, not only the file. The old message
  sent the reader to regenerate everything and commit the result, so a one-word
  review arrived as a diff of every entry the generator also disagreed about. A
  key with no counterpart on the other side reports `(missing)`.
- **The desktop doc described the old workflow**: it told a reader to change the
  generated file and that `-check` would catch a hand edit, which is precisely
  what made the gate permanently red. It now points at `reviewedWords`.

- **The Token guide's last step was not a sentence**: it read "Keys are never
  stored. Use generated owner key works only for the token that key created."
  The second sentence ran the button's name into its own predicate, so it stated
  something about the key where the guide meant to point at a control — and it
  sat under three steps written as sentences.
- **The control's name is quoted now**, the way the Blockchain guide writes
  "Payload" and "Show" and the NFT guide writes Collection. The Simplified and
  Traditional entries carry the quoted name too, so a Chinese window shows the
  same pointer.

- **A tagged desktop build showed its version as `vv1.1`**: the sidebar footer
  prepended "v" to the value the shell passed, and the shell passes
  `git describe`'s output — already prefixed — in every build that carries
  ldflags. Settings and `--version` both read `v1.1`, so the one place the
  window's footer disagreed was the one a user glances at. Only an unstamped
  `go build` left the bare `0.0.1` the prefix was written for.
- **`versionBadge` adds the prefix only when there is none**, and the test
  covers both shapes: handling only the tagged case would drop the "v" from the
  default footer.

- **The desktop's Copy labels put a space between two Chinese words**: the
  labels built at run time were `tr("Copy") + " " + tr(target)`, so a Chinese
  window showed 复制 诊断信息. The catalog's own composed entries — 复制载荷,
  复制代币 ID, 复制诊断信息 — carry no space, because Chinese does not separate
  two Chinese words; only the one label assembled in code disagreed with the
  rest of the window.
- **`i18n.JoinWords` owns the rule now**: two Han characters meet with nothing
  between them, and a boundary involving Latin keeps the space the project's
  Chinese prose uses everywhere else, so 复制 token-1 stays spaced. The desktop
  reaches it through `copyActionLabel`, which the three run-time Copy labels
  share.
- **A byte that does not decode keeps the space**: `DecodeLastRuneInString`
  returns `utf8.RuneError`, and reading that as Han would glue two Latin words
  together.

- **The desktop's readout label column was estimated from bytes, not
  measured**: `labelColumnWidth` multiplied the byte length of a label by 7.5
  instead of asking the text engine. Every Chinese character is three bytes, so
  数据库路径 — 55px at the label size — reserved 112px and 已记住的密钥 reserved
  135px. The Chinese About card then spent half its width on an empty label
  column and pushed its values into the remaining space, where the database
  path ellipsised.
- **Each row also sized its own column.** `Version`, `Build time` and
  `Database` produced three different label widths, so the values beneath them
  started at three different x positions and the card read as a ragged list
  instead of a table. `readout` now measures every label in the panel once and
  gives them one shared column.
- **The measurement is the text engine's**: `labelColumnWidth` calls
  `Context.MeasureText`, the same call MyGo lays the text out with, and keeps
  the 72px floor and 160px ceiling. The label size moved to a `labelSize`
  constant because the column is derived from it.
- **Keeping the byte estimate fails the new test in all three locales**: the
  zh case reports `"133.5" is not less than or equal to "107"`, and the per-row
  column fails with values beginning at 93, 96 and 111px.
- **`Amount.Cmp` panicked on an amount whose integer was never set**:
  `String`, `Sign` and `Int64` all treat a nil embedded `*big.Int` as zero —
  they check
  `a.Int == nil` before dereferencing. `Cmp` checked only the receivers, so the
  same value printed `"0"`, reported `Sign 0`, and then panicked the moment a
  caller compared it. A money type is the wrong place for three of four methods
  to agree and the fourth to crash.
- **The fix is the smallest one that makes the four agree**: a nil integer now
  counts as absent, exactly as a nil receiver already did, and the existing
  ordering is preserved for every input that did not panic. No current
  production path builds a bare `Amount{}` — every repository read goes through
  `NewAmount` or `NewAmountFromString` — so this is a latent trap rather than an
  active failure.
- **Dropping the nil-integer guard reproduces the panic** in the table test.

### Changed

- **The audit outbox retry ladder had no direct assertion**: `backoff` was
  reachable only through a drain pass, so its rung sequence and ceiling were
  never checked — the function sat at 62.5% statement coverage with none of its
  behaviour pinned.
- **Both directions matter**: a gap that failed to grow would re-attempt a
  failing destination every pass forever, because the drainer ticks once a
  second; a gap that grew past the cap would delay an audit record a recovering
  destination could already accept.
- **The ladder is spelled out rather than derived**, so it also pins that this
  implementation still matches the oracle scheduler's, which the code comment
  claims it mirrors. A small positive ceiling is checked separately to prove the
  doubling is clipped rather than overshot.
- **Halting the doubling fails the ladder case**; ignoring the ceiling fails the
  cap case with `backoff(attempts=10)=8m32s, want the cap 5m0s`.

- **The parent-cancellation shutdown order is covered**: the two existing
  `background.Start` tests cover stop-waits-for-exit and stop-is-idempotent, both
  starting from a live parent context. Production shuts down the other way
  round — the API server's root context is cancelled first, the work observes it
  and leaves, and only then does the caller reach its deferred `stop`.
- **That order reaches a different path**: the completion channel is already
  closed before `stop` runs, and nothing exercised it. The new test cancels the
  parent, waits for the work to exit, then requires `stop` to return promptly
  instead of waiting for a completion it missed.
- **Ignoring the parent context in `Start` fails it**: "cancelling the parent
  context must stop the background work".

- **Two pools over one file is the shape a goroutine test misses**: the existing
  concurrency regression exercises one `*sql.DB` with many goroutines. Two pools
  share no Go state — no mutex, no shared connection — and contend only through
  SQLite's file lock, which is what two CLI processes, or a CLI and the API, do
  against one `aurora.db`. That is a real deployment shape (a cron job and a
  server) and the one where a missing `_busy_timeout` hurts most.
- **The test moves value between two accounts from both pools at once** and then
  checks the totals still add up, so a lost update fails it even when no error
  surfaces.
- **Dropping the DSN hardening reproduces the failure the suffix prevents**:
  `try subtract balance: database is locked (5) (SQLITE_BUSY)`.

- **A cmd test that writes into the checkout now fails the suite**: the store
  resolves `db.path` on first open and defaults to `./data`, so a test that
  opens it without first moving to a temp directory writes `aurora.db` into the
  checkout. Every test there uses `withTempDir`, but nothing enforced the
  convention, and the cost of forgetting is not local — an earlier flaky
  repocheck failure came from a sibling test's `ResetForTest` removing `./data`
  mid-walk while another binary was still reading it.
- **`TestMain` records whether `./data` existed before the run** and fails the
  suite if this run created it. A checkout that already had one is left alone,
  so a developer who ran the CLI is not the suite's problem.
- **Verified both directions**: the untouched suite passes and leaves no
  directory; a probe that opens `store.OpenShared` in the checkout reports the
  run as failed even though the probe itself passed.

- **The pagination ceilings are bound to the reference**: seven list endpoints
  each carry a limit ceiling and an offset ceiling. The behaviour is tested per
  handler, but nothing compared those numbers with `docs/api.md`, so a tuning
  change would leave the prose advertising bounds the server no longer honours.
- **Two endpoints deliberately differ** and are why a blanket check would be
  wrong: `/token/history` and `/nft/list` stop at 1000 where the other five
  allow 10000. The check pins every constant and asserts the reference still
  calls out both divergent cases.
- **Changing a ceiling or editing the documented bound fails it.**

- **A wrapped overflow rejection now keeps its cause translatable in the
  window**: the repository wraps the domain cause behind a long English
  sentence, so the catalog holds only the tail. `localizeServiceErrorMessage`
  tries the head, then the tail, and leaves the sentence before the cause alone.
- **The pairing only works now that the repository wraps the sentinel**: before
  `TryAddBalance` and `TryAddToSupply` wrapped `token.ErrAmountTooLarge`, neither
  `try add balance` nor `balance would exceed maximum` was a key, so the whole
  line arrived in English in a Chinese window.
- **The bare form stays verbatim**, so a diagnostic still matches the log line a
  reader searches for. Removing the tail lookup fails the new test.

### Fixed

- **A saturated balance answered 500 instead of a client error**:
  `TryAddBalance`'s overflow guard returned a bare `fmt.Errorf`, so a credit that
  would push a balance past `MaxInt64` reached the client as an unclassified 500
  `INTERNAL_ERROR`. The supply path was fixed for exactly this reason and says so
  in a comment — "a plain message reached clients as an unclassified 500, which
  tells them to retry a mint that can never succeed" — and balances were missed.
- **The sentinel is the right one**: `token.ErrAmountTooLarge` documents itself
  as covering an amount the persistence layer cannot store exactly, naming
  "balances, supply, and allowances" as what would otherwise be corrupted. The
  API maps it to `400 AMOUNT_TOO_LARGE`.
- **The existing test could not have caught it**: it built the account id by
  concatenating token id and owner without the `-` that `TryAddBalance` uses, so
  `ON CONFLICT(id)` never matched the seeded row and the insert failed on
  `UNIQUE(token_id, owner)` instead. Asserting only "an error occurred", it
  accepted the unique violation and never reached the overflow guard.

- **A transfer of an asset that vanished mid-flight answered 500 instead of
  404**: `TryTransferOwnership` disambiguates `RowsAffected==0` into "gone" and
  "ownership moved", and its own comment says the point is letting the caller
  return the right error code. The gone branch returned
  `fmt.Errorf("nft %q not found")` — a string no caller can classify — while the
  sibling `TryDeleteNFTIfOwned` returns `nft.ErrNFTNotFound` for the same
  situation.
- **The path is reachable**: `NFTService.Transfer` checks existence and then
  writes inside a transaction, so a concurrent burn can delete the row in
  between — exactly the race the `Try*` primitives exist to close.
- **The old test locked the wording instead of the contract**: it asserted
  `err.Error()` contains `"not found"`, which the formatted string satisfies. It
  now uses `require.ErrorIs`, matching the sibling ownership-changed test.

### Changed

- **A test named `TestOracleData_Serialization` asserted nothing**: it built an
  `OracleData`, then discarded every field with `_ = data.ID` and friends. It
  called no production function and checked no value — the `_ =` lines exist
  only to silence the compiler, which is the shape a half-written test takes
  when it is committed anyway.
- **It now pins the stored record format**: the struct is marshalled straight
  onto the chain in `oracleapp/usecase.go`, so its JSON tags are what an auditor
  reads and a client parses. The test checks all six keys and their values, plus
  a round trip; renaming or dropping a tag fails it.

- **Three of `DurationSeconds`' five type branches were unreached**: viper's
  stored type depends on how a value was written — a TOML float arrives as
  `float64`, an explicit `time.Duration` stays one, a large integer can arrive
  as `int64` — and the existing test exercised only `string` and `int`.
- **The failure mode would have been invisible**: a branch that multiplied by
  the wrong unit turns `checkInterval = 250.0` into a 250-nanosecond ticker that
  busy-polls, while the integer test stays green.
- **The new test repeats the positive-only rule for each type**: a zero or
  negative value of any stored form must fall back, because a zero window or
  timeout disables the limit it was meant to set.

- **The schedule limits in the CLI reference are tied to the code**: it states a
  data source's interval ceiling (30 days) and the retry backoff (1s, 2s, 4s …
  capped at five minutes). Neither was compared against the constants that
  enforce them, so a tuning change would have left the prose describing an older
  schedule.
- **The interval is read from the domain constant** through the existing
  `readConst` helper. The two backoff values are `time.Duration` expressions,
  which that helper does not evaluate, so their declarations are matched as
  written and the test says why.
- **Changing either side fails it**: the code constant or the documented number.

- **The fetch response ceiling is documented now**: a data source that serves
  more than 10 MiB fails the whole fetch with `502 FETCH_FAILED`, and nothing in
  the reference said why. The paragraph beside it already documented the 10-hop
  redirect ceiling, so the omission read as "there is no size limit".
- **It is stated as a different limit from this server's own**: the cap bounds
  the *upstream feed*; the 4 MiB figure in the request table bounds incoming
  calls. An oversized response fails rather than being truncated into a stored
  reading.
- **`readConst` folds multi-factor multiplication**: `maxResponseBytes` is
  written `10 * 1024 * 1024`, which the old two-operand parser rejected.

- **The two modules do not share one patch version**: `docs/desktop.md` said
  `go.mod` and `desktop/go.mod` were both `go 1.27.2`. The nested module had
  already been raised to `1.27.3` by the MyGo dependency, whose own `go`
  directive is 1.27.3. A reader comparing the two files was told the difference
  was not there.
- **The prose says why they differ**: `go get` raises a module to its
  dependency's floor, and CI keeps them apart — the root jobs pin `GO_VERSION`,
  the desktop jobs read `go-version-file: desktop/go.mod`.
- **A check holds it**: both modules stay in the same Go major.minor, and the
  doc must name each module's exact directive. Reverting the doc, or moving the
  desktop module to another minor, fails it.

- **The README coverage table follows the code again**: three of the seven
  figures had fallen behind — lottery domain said 91.7% against 94.2%
  measured, NFT domain 82.0% against 87.5%, oracle domain 87.6% against 92.3% —
  and the SQLite and oracle app cells were stale too. The paragraph explaining
  an older drift still ended "stable at 91.7%".
- **A check holds what a test can see without running the suite**: every module
  keeps a row, no row records an app figure below the documented floor, the two
  layer-less rows keep their `-`, and the prose floor still matches the number
  the recipes pass. Deleting a row or listing one below the floor fails it.
  Re-measuring stays `just coverage-floor`'s job.

- **Both base-transport branches are covered**: `secureBaseTransport` decides
  which transport carries the fetcher's dial guard — a clone of
  `http.DefaultTransport` normally, explicit `net/http` defaults when an
  embedding application installed a different `RoundTripper`. Nothing tested
  either. The fallback is the one that matters: a zero transport there would
  drop proxy resolution and HTTP/2 tuning while the SSRF check silently applied
  only to the pre-request URL validation.

- **An unrecognised audit payload no longer prints its raw control
  characters**: both display fallbacks returned the stored bytes unchanged —
  the CLI's `blockLine` and the desktop's `decodePipePayload`. The sanitising
  rule covered the fields it could name, so the one path with no fields to name
  had no rule at all. A forged record could recolour the ledger table or split
  the row it sat in.
- **One helper now covers the whole value**: `records.ReadableText` applies the
  same encoding `ReadableFields` applies per field, and `records.IsPrintable` is
  exported so callers can assert the result. `--json` still carries the exact
  bytes; the decoded sentence is a reading aid, never the only copy.
- **Three tests, one per layer**: the records helper, the CLI table line and the
  desktop summary. Reverting any of the three makes its test fail.

- **A JSON audit payload can no longer carry a live terminal escape**: the pipe
  payloads were always rendered through `ReadableFields`, which base64-encodes
  a field a terminal cannot show, but the JSON ones were not — `jsonString`
  returned whatever `json.Unmarshal` produced. A draw whose winning name was
  `\u001b[31mRED\u001b[0m` reached the CLI's `blockchain blocks` table and the
  window's ledger card as a colour change, and an embedded newline split the row
  it labelled.
- **Both stored shapes share one rule now**: `jsonString` and `jsonStringSlice`
  go through the same `readableField` helper as the pipe path. The unprintable
  list and the encoding are unchanged; only the JSON route was missing them.
- **Two tests hold both halves**: control characters are encoded, and ordinary
  names and seeds pass through untouched — base64-ing a winner's name is the
  failure the encoding exists to prevent.

- **The backup checksum guard is exercised by a real archive**: every restore
  test built its archive by hand with `writeMeta`, which records no
  per-database checksums. `verifyArchiveContents` then filled the map from the
  file it had just read, so the comparison at the copy step compared a value
  with itself and could not fail — the guard those tests seemed to cover was
  never reached.
- **A round trip through `Create` and `Restore` covers the operator's path**:
  `Create` records the checksum before verification, and the restore compares
  against that recorded value. A second test replaces the archived database
  after a successful `Verify` and expects the refusal.

- **An upstream fetch failure is a 502 now, not a 500**: a feed that answered a
  non-2xx status, timed out or returned an unreadable body came back as an
  unclassified error, so `POST /oracle/fetch` answered `500 INTERNAL_ERROR`.
  That says Aurora is broken and invites a retry of the same doomed request;
  the far side is what failed. Measured over HTTP, a feed returning 404 gave
  500.
- **There is a domain sentinel for it**: `oracle.ErrFetchFailed`, classified as
  `502 FETCH_FAILED`. It lives in the domain rather than next to `ErrHTTPError`
  in `infra/http`, because the API must classify it without importing the HTTP
  client — the layer rule the architecture test enforces.
- **The conditions the domain already names keep their own codes**: a missing,
  disabled, rate-limited or SSRF-refused source still maps to 404/400/429/400.
  Only what nothing else claimed becomes a fetch failure.
- **The fetch-time SSRF guard was unwrapped too**: a source edited to point at
  loopback answered 500 on every fetch and never named the URL. It wraps
  `ErrInvalidSource` (400) now.

- **A supply overflow is a 400 now, not a 500**: `TryAddToSupply` returned a
  bare `fmt.Errorf` when a mint would push `total_supply` past `MaxInt64`. The
  mint use case passes the error through unchanged, the classifier found no
  sentinel in it, and the caller got `500 INTERNAL_ERROR` — which invites a
  retry of a mint that can never succeed. It wraps `token.ErrAmountTooLarge`,
  which the API already maps to 400 `AMOUNT_TOO_LARGE` and the CLI already
  translates.
- **The repository test asserted only `require.Error`**, which is how the gap
  survived. It names the sentinel now, and a smoke test drives the real server
  so the status and code are pinned end to end.

- **A first-touch allowance decrease no longer stores a negative amount**:
  `TryAdjustApproval` tested the delta's sign with `? >= 0`, but `amount` is a
  TEXT column and the driver binds the delta as TEXT — and SQLite orders every
  TEXT value above every INTEGER, so `'-100' >= 0` was TRUE. A decrement that
  created the row stored `'-100'` and returned it, while the primitive's
  contract and `DecreaseAllowance` both promise clamping at zero.
- **The existing clamp test could not see it**: it seeds the allowance first, so
  its clamp ran in the `ON CONFLICT` arm, which already cast both operands. A
  new test covers the `VALUES` arm a first use takes.

- **A repository walk now survives an entry that vanished**: the suite failed
  intermittently with `open .../cmd/aurora/cmd/data: no such file or directory`.
  The repocheck package walks the checkout while other packages test, and
  `cmd/aurora/cmd`'s tests chdir into temp dirs and call `store.ResetForTest`,
  which removes `./data`. All eight walks share one tolerance helper that treats
  ENOENT as "nothing left to read" and still propagates every other error.
- **Two walks had reimplemented the skip list**, and `doc_links_test`'s copy was
  already missing `.rumdl_cache`, `.planning` and `testdata`; both use the shared
  rule now.

- **The ownerless NFT list is tested with assets in it**: the only test for
  `GET /nft/list` without `?owner` used an empty store and asserted `[]`, which
  proves the encoding and nothing about the endpoint's purpose — listing assets
  the caller has no owner key for. A version that quietly filtered by some owner
  would have passed.
- **The new test uses two owners and asserts the order**: both must appear,
  newest first, with each row carrying its owner. Retargeting the handler at
  `ListByOwner` makes it fail with a 400 rather than a short list.

- **The stored VRF key codec has its own tests**: `EncodePublicKey` and
  `DecodePublicKey` are how a lottery record's key survives a round trip to
  SQLite and back, and `lottery verify` decides a stored record is tampered by
  failing to decode it. Only the success path had coverage — reached
  incidentally through the draw tests — so none of the four rejection branches
  was exercised.
- **Each rejection names its reason**: absent, not base64, too short, too long,
  and correct length but not a curve point. The last one is the interesting
  case: 32 bytes of `0x7F` pass the length check and are refused only by
  `SetBytes`, so a change that let the zero point through would have reached
  verify as a valid-looking key.

- **The audit renderer's terminal guard covers the characters a terminal acts
  on**: `ReadableFields` base64-encodes a field a terminal cannot show, and the
  only test used NUL and high bytes — both of which already fail the UTF-8 check
  on their own. DEL, ESC, BEL, CR, LF, TAB and the unit separator are valid
  UTF-8 and were held back by `isPrintable` alone, so dropping that half would
  have let an ESC sequence recolour a ledger row and a CR overwrite a line.
- **Both halves are now pinned**: one test asserts each unprintable field is
  encoded and equals its base64, the other asserts printable text — including
  Chinese and an empty field — passes through untouched, because base64-ing an
  amount is the failure the encoding exists to prevent.

- **The fetch-error scrubber has tests**: `sanitizeFetchError` carries a promise
  the reference makes in prose — a failed fetch reports the status or the
  underlying cause, never the source URL, because userinfo, path segments and
  query parameters can all carry credentials and `last_error` reaches every
  API-key holder. Nothing tested it.
- **Six tests, one per shape that reaches it**: a plain `*url.Error`, a nested
  one from a redirect chain, an empty cause (which would otherwise print
  `%!w(<nil>)`), an error that is not a `*url.Error` at all, and the
  `net.OpError` a refused connection produces. Dropping the URL is only half of
  it: the wrapped cause must survive, or `ErrBlockedDestination` reaches the API
  as an unclassified 500 instead of the 400 it maps to.

- **`/oracle/query` states its page size in the reference now**: it was the one
  collection read whose default (10) and cap (100) appeared nowhere, while every
  other list documents both. A caller asking for 500 readings got 100 with
  nothing saying so.
- **A test holds the three places together**: the handler's constants, the
  CLI's `maxCLIQueryLimit` (which says it mirrors the API) and the sentence in
  `docs/api.md`. The first draft of that test matched any "`limit` 上限 100" in
  the document — eight endpoints say it — so it checked the wrong sentence and
  passed with the real one deleted; it reads the query note's own slice now.

- **The allowance-spending endpoint has a success test now**: `POST
  /token/transfer_from` had one test, for malformed JSON, so its 200 response,
  its receipt fields and its status mapping were unobserved. The handler is
  fully covered; the sentinel behind a refused spend reaches the client as
  `INSUFFICIENT_ALLOWANCE` (400) rather than a generic 500.
- **The test fake had to return a real event**: it answered `(nil, nil)`, so a
  success-path test panicked on the nil dereference — which is why the gap had
  survived.

- **An oracle reading query without a limit reads every reading**: the SQLite
  repository turned `limit <= 0` into `LIMIT 100` while every other repository
  and the in-memory store treat it as unbounded, so a caller asking for the
  whole history got a truncated list with nothing saying more existed. Every
  production caller passes an explicit limit, so the visible behaviour is
  unchanged; the contract now matches the rest of the store.
- **The test that pins it starts its timestamps at 1**: `SaveData` replaces a
  zero timestamp with "now", which silently made the first fixture row the
  newest one and turned a correct query into a confusing failure.

- **Four use-case tests asserted only that something failed**: the oracle
  enabled/disabled lookups, the NFT empty-creator mint and the vote
  transaction failure each checked `require.Error`. Any error passes those, so
  a use case that reported the wrong sentinel — or masked a storage failure as
  a rejected ballot — kept them green. They name the sentinel now.
- **The oracle pair pins the normalization too**: an unknown id reaches the
  API as `SOURCE_NOT_FOUND` (404) only because the use case translates the
  storage sentinel, and nothing tested that translation.

- **The in-memory NFT store lists in the stored order too**: `ListNFTs` sorted
  by id and `GetNFTsByOwner` returned map order, while the SQLite repository
  runs newest first for both. App and transport tests read through this store,
  so a paging assertion could pass on a sequence the API never serves.
- **It follows `INSERT OR REPLACE` and `UPDATE`**: a re-saved asset moves to the
  front, an updated one keeps its place, and a rolled-back save restores the
  order along with the rows.

- **The in-memory oracle store reads like the SQLite one**: `GetDataBySource`
  walked a Go map, so it returned readings oldest first and `GetLatestData`
  picked the earlier of two same-second rows. Every handler and use-case test
  reads through this store, so the suite was pinning the opposite of what
  `/oracle/query` and `/oracle/latest` answer in production (timestamp
  descending, insertion order within a second).
- **A rewritten reading takes the newest place in its second**, because the
  store's `INSERT OR REPLACE` assigns a fresh rowid; three tests now hold the
  order, the tie and that rewrite to the store's behaviour.

- **`/nft/list?owner=` runs newest first like the ownerless list**: the filtered
  path ordered by `rowid ASC` while the whole collection ordered by `rowid DESC`,
  so turning the owner filter on reversed the page a reader was looking at.
  `docs/api.md` promises every collection endpoint runs newest first.
- **The two readings now share one order in a test**: the filtered page must
  match the ownerless page's order for the assets they share, and a query-plan
  test keeps the filtered read on `idx_nft_owner` without a temp B-tree.

- **The Ballot tab offers the stored sessions under both of its id fields**:
  Cast Vote and Session Controls each take a session id, and neither offered a
  stored one, so naming a session meant copying an id out of Setup. Both cards
  now carry the Results tab's `Stored sessions` picker under their own field;
  choosing one also loads that session's roster, so the candidate chips resolve
  by name. The picker stays visible once a session is chosen, which makes
  switching sessions one click instead of clearing the field first, and the
  Session Controls hint now points at the picker below it rather than only at
  Setup.
- **The Results tab's session chips sit under the field they fill**: the
  newest sessions were offered inside the empty tally card in the right column,
  so a reader saw a chip named for a session while the "Session ID" input it
  writes sat across the gutter in another card. The picker is under that field
  now, labelled "Stored sessions", and the empty tally's prompt names that
  picker instead of sending the reader back to Setup.

- **The capture guide names every switch the script reads**: `AURORA_E2E_OUT`
  and `AURORA_E2E_DISPLAY` were only in the script's own header, so a reader
  following `docs/desktop.md` met a run that wrote somewhere unexpected or
  failed to start because it wanted a display it was never told about. A test
  now holds the two files together.

- **An unfinished marker cannot reach a release**: the tree carries no `TODO`,
  `FIXME`, `XXX` or `HACK` in shipped Go today, and a test now keeps it that way.
  A TODO in shipped code is a promise nobody tracked; a FIXME is a known defect
  left in place. A line may keep the word when it says the state is intended, so
  a deliberate exception is documented where a reader finds it.
- **The guard had to skip itself**: it names the four markers in order to forbid
  them, so the first run failed on its own source.

- **A Simplified character can no longer hide in a Traditional value**: the
  generator converts character by character, so a value that never went through
  it keeps its Simplified form and mixes 简体 into a 繁體 page. The 219 characters
  that the Simplified catalogs use and the Traditional ones never do are listed in
  the test; a value holding one fails by key and by character.
- **The list is committed, not derived**: a Simplified character typed into the
  Traditional table removes itself from a set computed from those same tables, so
  a derived set would pass on exactly the entry the check exists to catch. That
  is how the first draft behaved when an injected 创 still came back green.

- **The Simplified and Traditional catalogs must share one key set**: `T` falls
  back to English and then to the key itself, so a key added to one catalog and
  forgotten in another degrades quietly — the window shows English, or a raw
  identifier, and every test passes because the value exists somewhere. Both
  catalogs hold 561 core keys and 737 window keys today, and a missing one now
  fails by name.
- **The scan has to read raw strings too**: the long help paragraphs are backtick
  literals, so a double-quote-only pattern reported three keys as Traditional-only
  when the English catalog has them.

- **The JSON examples in the API reference are parsed by a test now**: eleven
  fenced blocks show the shape of a request or a response, and nothing checked
  that they parse. A trailing comma there teaches the wrong shape and costs the
  reader a debugging round before they discover the server is fine. A block
  tagged `json` that holds nothing fails too, because it reads as an example the
  writer meant to fill in.

- **The docs may no longer name a flag a command does not register**: the path
  check stops at the first word that looks like a value, so it never reaches a
  flag — a line could say `-x` where the command registers `-s` and nothing
  failed. The new check maps each command variable to its path and each `Flags()`
  call to the flags it registers, then holds every documented flag to that.
- **Two shapes had to be found**: the root command's own `Use` is `aurora`, so
  adding the prefix again keyed every path as `aurora aurora …`; and `--confirm`
  is registered through a helper that takes the command as an argument, so the
  variable appears at the call site rather than before `Flags()`.
- **Renaming one documented flag fails with the line number**:
  `docs/cli.md:77 documents "-x" on "aurora lottery create", which does not
  register it`.

- **Four reserved sentinels say so where they are declared**: `INVALID_SIGNATURE`
  on both token and nft, plus `NOT_TOKEN_OWNER`, `DUPLICATE_TRANSFER` and
  `ErrStorageNotInit`, are classified and never returned. The reference explained
  why; a reader in the errors file had to go and find it.

- **The CLI's translation table got the same check**: 82 entries name a sentinel
  and nothing compared them against the code that returns the errors. It found
  `nft.ErrStorageNotInit`, declared and translated and never returned — the fourth
  such entry across the two tables.
- **The two tables list different sets for a reason**: six sentinels translate
  only in the CLI, where lottery import and ledger integrity surface; the HTTP
  reference lists none that the CLI lacks.

- **A test now proves every classified code can be returned**: the HTTP table
  lists 75 sentinels and nothing checked that a request can produce them. It
  matches on package plus name, not message text, because token, nft and voting
  each declare an `ErrInvalidPrivateKey` with identical wording.
- **The check found two more dead mappings**: `token.ErrInvalidSignature` and
  `nft.ErrInvalidSignature` are classified as 401 and no code returns either. Both
  services report a bad signature as a false verification result rather than an
  error, so `docs/api.md` now lists them with the two older reserved codes.

- **`SignVote` returned an error nobody could classify**: a short private key
  produced an inline `fmt.Errorf("invalid private key size")` while the package
  already declared `ErrInvalidPrivateKey` for exactly that case, and the API maps
  that sentinel to 400. The mapping had been dead since it was written.
- **The use case had grown its own length check** to compensate — its test
  comment said the window between the two was a deep dive finding. Both layers
  return the sentinel now, so the check holds whichever one catches it first.

- **Three token tests proved the wrong thing**: `TestApprove_InvalidAmount`,
  `TestApprove_InvalidOwner` and `TestApprove_InvalidSpender` left `PrivateKey`
  empty, so the key check failed first and the condition in the test's name never
  ran. Each asserted only that something failed, which it did.
- **They set the key and assert the sentinel now**, so the nil amount really
  reaches `ValidateAmount`. Suppressing that call makes the test report
  `got <nil>, want ErrAmountMustBePositive` instead of passing.
- **Eleven more token tests assert `errors.Is`**: the nil key, nil recipient and
  zero amount cases in Mint, Transfer and Transfer From named no sentinel either.

- **The lottery validators asserted only that something failed**: eleven tests
  checked `err == nil` and never the sentinel, so a validator returning the wrong
  error — or an unrelated one from a bug — passed. They assert `errors.Is` now, and
  swapping one sentinel makes both affected tests name the expectation.
- **The same round found the rest of the domain already doing this**: token and
  nft tests use `errors.Is` throughout; lottery was the one package that did not.

- **The capture guide says why text scale has no flag**: the window's text size
  comes from the framework's preferences, which the app cannot pre-write, so a
  large-text check runs through the offscreen tester rather than a screenshot. A
  reader who looked for the fourth flag found three and no reason.

- **Four Traditional entries said 資料源 where the other 42 say 數據源**: the
  Oracle page's error messages disagreed with the page's own labels, and each
  spelling is a legitimate word, so nothing failed. They are 數據源 now, and a
  test compares the two forms in both Traditional catalogs.
- **The test reads the census before it objects**: the desktop table is written
  in mainland vocabulary throughout, so 數據 wins there and 資料 would be the
  stray form; a table written the other way would be judged the other way. A
  single stray occurrence fails, because mixing is the defect.
- **預設 and 默認 are not a pair**: they translate preset and default, two
  different things, and the Simplified table makes the same distinction.

- **The dashboard called one number two things**: the card's label is 投票会期
  and its caption was 治理时间窗口, both reading `dashboard.Sessions`. The window
  counts governance windows, and the Voting page calls each one 会期, so the
  caption is 会期数量 now and the section sentence lost its 治理时间窗口 too.

- **The NFT page no longer calls its holder an owner**: the burn form labelled
  the current holder's key 所有者, and the Inspect card's filter did the same,
  while three sentences on the same page say 持有者 — 创建者会成为第一个持有者,
  按持有者筛选. The catalog could not fix it by itself: the English string
  "Owner (base64 public key)" is right on the token page, where the account signs
  writes, so it had to stay 所有者 there.
- **The NFT fields got their own keys instead**: Holder (base64 public key),
  Holder Private Key, Filter by Holder and its hint, in both scripts. The four
  older entries stayed for the token page and their two stale variants were
  removed.
- **The unused-key test caught the leftovers**: it failed with the two keys the
  view no longer reads, which is what that check exists for.

- **The shared walk helper has tests of its own**: `skipWalkDir` decides what
  nine checks never look at, and `walkGoSource` decides what they do. Both were
  untested, and both fail in ways the callers cannot see — a walk that skips a
  source directory passes on an empty set, which is the same green as a tree with
  nothing wrong in it.
- **`.rumdl_cache` joined the skip list**: it is a tool cache that carries its own
  `.gitignore`, so it never reached a commit, but it already holds 640 files and
  grows with the prose it indexes.

- **Two punctuation tests became one**: `punctuation_test.go` checked the ASCII
  colon after Chinese on two catalogs, and `cjk_punctuation_test.go` — added later
  — checked the same marks plus brackets and periods on all four. The older test
  was a subset of the newer one and had been running alongside it for sixty
  rounds.
- **Its two good ideas moved instead of disappearing**: the merged test reports
  the offending line rather than only the value, and keeps the matcher's own
  self-check, so a rule that stopped matching fails loudly instead of passing on
  zero hits.
- **The layering test now counts what it walked**: a rule whose directory was
  renamed would pass on an empty walk, which is the failure mode that makes a
  layering check decorative. Each of the four rules must examine more than
  fifteen files.

- **Every page is rendered in the stacked layout a narrow window gets**: below
  `twoColumnMinWidth` the two columns become one, and only the token page had
  ever been drawn at that width. The other seven share the branch by
  construction, but nothing checked the result; the new test walks all eight at
  900×700 and fails when the widest string runs past the window.
- **The test says why it is not vacuous**: it asserts its own width is below the
  threshold, and a 320-pixel run fails with a named string, so a future change to
  either number cannot leave it passing on the wrong branch.

- **Four failed reads that nobody had exercised now warn in a test**: the fake
  can fail `chain`, `tokenInfo`, `tokenBalance` and `nftDetails`, and no test had
  ever set one of them. A card that swallowed an error shows an empty readout,
  which reads as "there is nothing there" rather than "the read did not work".
- **Three of them run behind a button, so the test clicks it**: assigning the id
  before the first frame does not work, because selecting a token clears the
  subject's inputs; the button, disabled until its fields are set, then swallows
  the click and the test passes for the wrong reason. The ids are set inside the
  view closure instead.

- **Every write form behind the Token and NFT tabs is rendered by a test**: only
  Burn on the token page and Burn plus History on the NFT page had ever been
  drawn, so a form that stopped rendering — or drew another tab's body — would
  have shipped. Eight tabs are covered now, asserted on each form's own purpose
  line rather than its tab label, because the label is on screen whichever tab is
  selected.
- **Two attempts before that were worthless and looked green**: the first
  asserted the tab label, which passes even when the body draws nothing; the
  second ran against an empty fake, where the page shows its first-record guide
  and every tab draws the same thing. Both were found by breaking the code and
  watching the test still pass.

- **A page missing from one of the window's tables is caught now**: a page keys
  four tables (`pageIDs`, `pageTitles`, `pageIcons`) and two order slices, and
  every one of them fails quietly when a page is missed. A missing id makes the
  sidebar's click resolve to nothing — `pageByID` returns false and the page
  never opens — while a missing title renders an empty header.
- **The ids are checked for what a choice is remembered as**: distinct,
  non-empty, and each one resolving back to the page it came from.
- **The two sections are checked against each other**: the module section is the
  page order without Settings, and Settings is last.

- **A configuration key the sample file never mentions is caught now**: 19 keys
  are read through viper across `internal/config` and the CLI's own defaults, and
  all 19 appear in `config/aurora.toml` with a value, a commented-out default or
  a note. A key the loader honours and the sample omits is a setting nobody knows
  exists; the reverse — a key the sample shows and the code ignores — silently
  does nothing.
- **The values agree too**: the commented-out defaults match the code
  (`api.rateLimit.requests` 120, its window `1m`, `oracle.scheduler.checkInterval`
  1), and `log.path` differs only in a trailing slash that `NormalizePath`
  removes.

- **A callback the view asks for and the command forgets to fill is caught now**:
  `Actions` is a struct of callbacks, and adding a field is how the view asks for
  something new. A field the command never sets is a nil callback reached on the
  frame that needs it, and the view tests build their own value so they would not
  notice. All 24 are set today.
- **The rest of the seam holds**: every field is also read somewhere in the view,
  so the struct has no unused member, and all seven domain interfaces are named
  under `infra`.

- **A handler without a route is caught now**: the route table passes method
  values (`r.Post("/create", h.Create)`), so a method missing from it compiles,
  passes its own test against a fake router, and 404s in production. The new
  check parses each module's Routes table and requires every `*Handler` method to
  appear in it; all 43 do today.
- **The same round confirmed the rest of the surface**: 43 module routes all
  appear in the reference and the reference names no route that is not
  registered; 37 use cases are all named by a presentation layer once the
  constructors are counted rather than the type names.

- **The README lists `check-verify`**: the recipe exists so `verify` does not
  rewrite the tree, and only the justfile knew about it. A reader who wanted a
  format check without side effects found `check` and got the writing one.
- **The recipe description names both formatters**: it said "Format the core
  module and run vet" while also running goimports.
- **A test holds the first**: it parses the justfile's recipes and requires each
  one to appear as `just <name>` in the README, so a new recipe cannot be added
  for a reason the reader needs and stay invisible.

- **A desktop translation could name an error no code produces**: the window
  catalog holds English service errors as keys, matched by exact text, and
  "blockchain not initialised" sat in it while `infra/blockchain` and `deskapp`
  said "initialized". Two spellings of one word meant the window printed the
  English while the catalog looked complete — no test compared the two.
- **The catalog and the code now share one spelling**, and the removed duplicate
  it exposed is gone: the same key appeared twice, once in the oracle error group
  and once among the general ones.
- **A test keeps the link**: it reads both catalogs, skips the keys the view names
  verbatim, and requires every remaining key to be recoverable from a Go file —
  as a whole literal, a `%w:` fragment or a concatenated prefix. 196 keys are
  checked.

- **Two listed error codes are never produced**: `NOT_TOKEN_OWNER` and
  `DUPLICATE_TRANSFER` sit in the classifier and their sentinels have existed
  since the domain layer was written, but no use case returns either. The
  reference listed them among the errors a caller can receive, which invites a
  branch that never runs.
- **The reference says so now**, and a test keeps the list honest: it reads the
  classifier, finds the codes no other file mentions, and requires both to be
  named in the note. The day a use case starts returning one, the test asks for
  the note to change.

- **A failed lookup says 未找到, and a person takes 一位**: the window used 找不到
  and 未找到 for the same four "not found" errors, and the CLI counted candidates
  with 一名 while the window used 一位 for the same people. Both are one word now,
  and the vocabulary test covers them.
- **A token Amount is 金额 in both layers**: the CLI said 数量 for the amount, the
  same word it uses for a count of people.
- **That last pair is deliberately not a rule**: 数量 is correct for 参与者数量 and
  wrong for a token amount, and the two cannot be told apart from the string, so
  a rule would fire on correct copy. The exclusion is written in the test beside
  the rules rather than left to be rediscovered.

- **Four more words had a second spelling in one layer**: the CLI called a
  mintable token 可增发 while both surfaces and the guide say 铸造; the window
  called a disabled feed 已停用 once against 禁用 everywhere else; the encoding
  was 大小写两项 and the guide writes it lowercase; and one error said 非法字符
  where every other rejection says 无效. Each is one word now, and the vocabulary
  test covers all four.
- **Two of those rules judge the catalogs only**: 停用记忆 means "stop
  remembering" and 非法 JSON means a syntax error, so running text keeps both
  spellings. The rule records why it stops at the catalogs instead of pretending
  the word is always wrong.

- **`Owner` had three spellings across two surfaces**: the CLI said 持有者, the
  window said 授权人, and the window's own test asserted 所有者. Since the guide
  fixes 持有者 for an NFT's holder and 所有者 for the account that signs a token's
  writes, the two surfaces now agree with the rule: `common.owner` and the
  window's `Owner` are 所有者, the NFT key-mismatch error keeps 持有者, and the
  three `token.owner*` entries in the CLI catalog follow the token sense.
- **Four field labels were prepositions in the window**: `From` read 从 and `To`
  read 到 above a transfer record, which is how a sentence fragment appears where
  a column name belongs. They are 转出方 and 转入方, matching the CLI's
  `common.from`/`common.to`.
- **`Spender` and `Party` follow one word each**: 被授权方 for the account an
  approval names, and 党派 for the candidate field; the CLI said 消费方 and
  党派/组织 for the same two.

- **`just verify` no longer repairs the tree it is validating**: `check` runs
  `gofmt -w` and `goimports -w`, so the gate that exists to report a formatting
  problem fixed the file, exited 0, and left the change in the working tree for
  the next commit to pick up unread. `verify` now calls `check-verify`, which
  runs the same three checks with `-l` and fails; `check` keeps repairing on
  purpose, because a developer running it is about to commit.
- **The README says which is which**: the command list annotated `just check` as
  "gofmt + goimports + go vet" without mentioning that it writes.

- **The coverage floor reuses the profile the test step wrote**: `just verify`
  ran the desktop suite twice — once plain, once for coverage — and Go cannot
  share cached results between the two modes, so every run paid 44 seconds for
  the same assertions. `desktop-check` now writes the profile as it tests, and
  the floor step reads it; the gate went from 100s to 81s on a warm cache.
- **The floor still stands alone**: run by itself, `desktop-coverage-floor` tests
  first when no profile exists, so the recipe means what it says either way.
- **Two guards keep the duplication from returning**: the justfile test asserts
  the floor step reads the file instead of re-running, and the CI test asserts
  the workflow's test step leaves the profile behind.
- **`-coverpkg` was tried and rejected**: it makes every package count every
  other package's statements, which reported deskapp at 41% instead of 83%.

- **One test spent twelve of the suite's twelve-and-a-half seconds in the build
  cache**: `TestEveryTranslationKeyExists` walks the repository, and its skip
  list named `dist`, `data` and `testdata` but not `.gocache`, `.gomodcache`,
  `.gopath` or `.gotmp`. A working checkout holds 122,000 files, almost all of
  them in those four directories. The suite now runs in 0.39s instead of 12.8s.
- **The skip list moved to one function** the walks in this package share, so the
  next test cannot forget part of it. Nine tests walk; the other eight either
  start below the repository root or already skip every dotted directory.
- **A literal pre-filter runs before the parser**: 551 Go files hold 29 that call
  a translation helper, and parsing the rest bought nothing. The filter is a
  superset — any mention of a helper name — so it cannot skip a call the parser
  would have found.

- **A nil error is pinned as "nothing to announce"**: both `errStatus` and
  `localizeServiceError` start with a nil branch — every successful write handler
  reaches it — and neither branch was covered. The tests now fix what a caller
  gets: an empty line with a zero revision, so the empty line is not announced as
  a fresh outcome, and an empty string rather than a panic.
- **The revision assertion had to change shape**: `nextStatusRevision` increments
  on every call, so comparing before and after measured the test's own call. The
  line's `revision` field is what carries the meaning.

- **The same rule covers the core catalog, against its own reference**: the
  key there is an identifier (`backup.restore_done`) and the sentence lives in
  `messages_en.go`, so the argument list to match is the English one rather than
  the key. 75 entries per Chinese table are checked, and a dropped `%s` in
  已恢复数据库 is caught.

- **A translation cannot drop or invent a format verb**: the desktop catalog
  uses the English literal as its key, so a translation that loses `%s` prints
  `%!s(MISSING)` to the user and one that adds a verb prints `%!d(string=...)`.
  Neither fails a test, because `fmt.Sprintf` does not report a bad format. The
  new check compares the argument count on both sides of all 37 entries that
  carry one.
- **Positional verbs are counted by distinct index**: 从 %[2]s 销毁 %[1]s？ reorders
  the two arguments on purpose, and a highest-index-only count accepted a
  translation that had dropped `%[1]s` entirely. The first version of the test
  missed exactly that, which is what its own regression run showed.

- **The narrow-window captures explain the half-visible tab bar**: the script
  scrolls the stage's control into view, and the stage tabs live inside the
  scrolling body while the page header is fixed. The result photographs a tab
  strip cut at the viewport edge, which reads as a layout defect. The guide now
  says the stage is named by the file name and the tabs scrolled out on purpose.
- **The layout is right**: the header is fixed chrome above the body, and the
  tabs belong to the body — that is the design, not a regression to fix.

- **The reference names all four completion shells**: the paragraph said "four
  shell subcommands" and the examples showed two, so a fish or PowerShell user
  could not tell whether their shell was covered. The names are listed now, and
  a test holds them.
- **That family is the one the tree parser cannot see**: Cobra adds the shell
  subcommands itself, so no source file declares `completion fish` and
  `TestCLIDocsMentionEveryCommand` never demanded it. Every other subcommand in
  the CLI was already documented.

- **Every package in the tree opens with a sentence saying what it is for**:
  nineteen were missing a doc comment — the five use-case packages, the whole
  api layer, five infra packages, and the two command packages. `go doc` showed
  a wall of member listings with no orientation, and a reader new to the
  repository had to infer the layering from import statements.
- **A test keeps it that way**: it walks `internal/` and `cmd/`, and fails with
  the list of directories whose Go files do not begin with `// Package` or
  `// Command`. A new package cannot be added blank.
- **`repocheck` gained a `doc.go`**: the package is tests only, so the comment
  lives in the one non-test file Go needs for `go doc` to show it.

- **Three packages that carry the hardest constraints now say so**: `ledger`,
  `i18n` and `infra/sqlite` had no package comment, while `records`,
  `background`, `netguard` and `app/container` did. Their comments state what a
  reader would otherwise have to reconstruct: why the ledger ports name
  `database/sql.Tx`, the order the locale variables are read in, and that every
  repository is pure Go so the binary builds with `CGO_ENABLED=0`.
- **The architecture tree lists the domain subpackages the way it lists infra's**:
  it named `errdetail` alone and left `lottery`, `voting`, `nft`, `token`,
  `oracle`, `events` and `repoerr` unmentioned, so the one subpackage that
  appeared looked like the only one.

- **The desktop guide explains why deskapp stops at 83%**: the coverage floor is
  70% and the package sits at 83%, which reads as room to grow. The twenty
  uncovered functions are one-line wrappers over `mygo.App`/`mygo.Shell`, window
  lifecycle setters, and three that read operating-system state — none has a
  branch to assert. Raising the number would mean tests that check a global was
  called, which pin the implementation rather than the behaviour. The guide now
  says so, sorted into those three groups, so the next reader does not repeat the
  attempt.

- **The lint configuration states what it excludes and why, with counts**: the
  errcheck note claimed its findings were "stdout writes and test-only Close
  calls". Run against this tree it reports 28 sites — nine `fmt.Fprint*` writes
  to a help screen the process cannot act on, and nineteen unchecked Close,
  Remove and Unsetenv calls in test cleanup. The note now carries those numbers,
  and it records why four further linters were tried and left off, so the next
  reader does not repeat the experiment.

- **The vocabulary rule reaches the documents**: the catalogs were checked for
  the retired spellings and the prose was not, which is where an old word comes
  back — copied from a release note, or written from memory. A document that
  says 会话 for a voting session carries a word neither surface uses.
- **Five more term pairs were examined and left alone**: 删除/移除/销毁,
  获取/读取, 保存/存储, 清除, 小数位. Each pair separates an operator's action from
  the state it leaves or the English verb behind it, and collapsing them would
  lose the distinction rather than tidy it.

- **The drain window in the reference is checked against the constant**: the
  API section states that a stopped server finishes within 15s, and nothing tied
  that number to `shutdownTimeout`. A server that stops sooner than documented
  drops in-flight writes, and a reader planning a rollout plans around the
  number. This round also confirmed the rest of the section against the code —
  the metric names, the log permissions and the two ID formats — without finding
  anything to change.

- **A token's precision is 精度 on both surfaces**: the CLI spelled it out as
  小数位数 in three entries while the guide and the window said 精度. The
  vocabulary test covers the pair now.
- **持有者 and 所有者 stay separate**, checked before changing anything: the first
  names an NFT's holder and the second the account that signs a token's writes.
  They look like synonyms and are not.

- **An NFT transfer and a token transfer keep their own words**: the guide and
  the window move an asset with 转让 and money with 转账, and the CLI used 转移
  for both. The distinction is not decoration — the two records are different
  operations, and one verb made them read alike in the history and in the help.
- **The vocabulary test covers the pair**: its rule for a Traditional spelling
  used to be a lookup table of four words, which silently mapped a new entry to
  the empty string. Each concept now carries both spellings.

- **The command line and the window name the voter the same word**: the CLI
  said 投票人 twelve times and the window and every document said 选民 — nobody
  sees both at once, so nothing failed until an operator moved from one to the
  other and read two names for one thing. 会话/会期 had the same split, 31 against
  58, with one sentence using both.
- **A test holds the vocabulary**: it reads all four catalogs and rejects the
  spelling the house word replaced, with an exclusion for 会话密钥, where 会话 is
  the right word for a process session rather than a voting one.

- **Both Chinese catalogs and both window edges**: the test added in the
  previous change ran the Simplified table and checked only the right edge. The
  two tables are separate, so a line can be short in one and long in the other,
  and content can be pushed off to the left as well. It now runs 简体 and 繁體 as
  subtests and reports the leftmost and rightmost edge, each with the string
  that produced it.

- **Chinese at double text is checked against the window edge**: the suite had
  eighty-six assertions at 2x and none of them in Chinese, where every
  non-Latin character is twice as wide before the scale applies. The new test
  walks all eight pages at 2x in Chinese and fails when the widest string runs
  past the window. The layout holds at every width above 400px; the check is
  what proves it, and a 320px window is what proves the check.

- **A printed bracket follows the language too**: two lines built "(%s)"
  themselves — the doctor's database check and a candidate's party on the results
  list — so a Chinese window showed 数据库：./data/aurora.db (80.1 KiB) next to
  labels that all use full-width brackets. `i18n.Paren` owns the bracket and the
  space before it, because Chinese sets it against the preceding word while the
  Latin spelling keeps one.

- **The bracket rule covers the spelling the catalog does not use**: it matched
  a Chinese character, spaces, then a half-width bracket, which is how the fix in
  the previous change reads. A bracket with no space at all went uncaught, so a
  later edit in that style would have passed. Both spellings are covered now.

- **Parentheses follow the same rule the other marks do**: six entries wrote
  公钥 (Base64) and 代币 (FT) 系统 with half-width brackets while seventy wrote
  会话开始时间（Unix）with full-width ones. The punctuation test covered the marks
  that end a clause and not the pair that surrounds a note, so the rule held
  everywhere except where a Latin term was being explained.

- **The remaining source assertions match a call, not an expression**: two of
  them spelled out the full argument list, and one of those was rewritten twice
  for the separator and the label helper — each rewrite read as an
  accessibility regression. They now match `ActiveDescendant(`, `.Description(`
  and `c.Announce(`: rewriting the expression is a refactor, dropping the call
  is the regression.
- **Those failures no longer print the file**: `require.Contains` on a whole
  source file dumps every line, which buried the one message the test exists to
  give. The comparisons are on the search result instead.

- **The card-group test checks rendering as well as the source**: it read
  `layout.go` for the label call and asserted nothing about the frame, so it
  would have passed had `formCard` stopped drawing its title. It now renders a
  card and checks both strings reached the screen, and matches the source
  loosely — a rename in `tr(title)` is not an accessibility regression, while
  the panel losing its label is.

- **The guide's account of the capture script matches what it does**: it said a
  language run captured Settings and Dashboard and stopped, and that it wrote to
  `dist/e2e/zh/`. Neither was true any more — the run walks the same page loop in
  all three languages and names its directory after the variants it combines
  (`light-zh`, `light-zh-900-600`). A reader planning a release would have
  believed six pages were unverified in the other two languages.
- **Two tests hold that**: one reads the stage table and the page-loop sentence
  the script needs, the other rejects the two phrases that described the old
  behaviour.
- **A wrapped error is described in the guide**: the section on copy and
  diagnostics now says how a twice-wrapped cause is localized segment by
  segment, because that is the shape a failed fetch actually arrives in.

- **A cause that is wrapped twice is translated all the way down**: the Oracle
  health card shows a failed fetch verbatim, and the fetcher wraps it as
  "failed to fetch data: request failed: Service Unavailable". The localizer
  translated the head once and stopped, so the line stayed English even though
  the middle segment had an entry added for exactly this card. It now localizes
  the head and then the remainder, and the four fetch wrappers gained entries of
  their own so the outer layer is translated too.

- **The punctuation rule the spacing rule starts from is checked too**: the
  catalogs use full-width marks after Chinese text, and nothing held that. The
  new test reads all four catalogs and fails on a half-width mark that follows a
  Chinese character.

- **The CLI spells its labels the way the window does**: the core catalog said
  抽奖ID, 候选人ID and 会话ID in three places while the desktop table said 抽奖 ID
  for the same string. The project writes Chinese prose with a space between a
  Chinese character and a Latin word; a rule applied in three of four places is
  the kind of thing a reader notices without being able to name.
- **A test holds that rule across all four catalogs**: it strips placeholders
  and escaped newlines first, because those separate as much as a space does.

- **The ledger calls an NFT's owner a holder, like the page does**: the NFT
  page and the transfer record were renamed to "Holder" a while back, but the
  decoder kept emitting "Owner" for a mint, so a ledger row read Owner in
  English while the card above it said 持有者. An approve still names an owner —
  the account that granted the allowance — which is a different role, so that
  one keeps the word and gains the entry it was missing.
- **A test ties the decoder's vocabulary to the catalogs**: the labels live in
  `internal/records` and the card looks them up in the desktop table, so the two
  can drift without either package noticing.

- **The source type is a word, so the window translates it**: the Oracle
  source row showed the domain's own "custom" beside translated labels. The
  first language run over every page is what surfaced it — the earlier runs
  stopped at Settings and Dashboard. "json" names a format and keeps its
  spelling, which tr() already does by falling back to the key.

- **A language run captures every page, not two of them**: the window script
  stopped after Settings and Dashboard because its title, page-marker and stage
  tables were English-only, so a Chinese or Traditional run proved the switch
  took effect and nothing else. The four tables now carry all three languages
  and the run walks the same loop: 15 captures per language, seven of them
  stage switches. Pressing matches a control's label exactly, so the stage tab
  needs its own translation — the tab reads 销毁 while the button under it
  reads 销毁…, and using one for the other fails the capture.

- **The window joins its own labels the same way**: the desktop built
  "label: value" by hand in six places, and its Chinese labels carry no colon —
  the code added one — so a Chinese window showed 版本: 0.0.1 in the copied
  diagnostics text and announced "失败: 读取: …" to a screen reader, beside
  translated words. `labelText` and `colonText` take the separator from the
  language, as the CLI does now.

- **A printed label uses the colon its language uses**: Chinese prose puts a
  full-width colon after a label, and all 55 Chinese entries in the message table
  already did — but the labels printed by the CLI passed a hard-coded half-width
  one, so a line read 版本: 0.0.1 next to translated text. `i18n.Label` and
  `i18n.Colon` now pick the separator from the active locale, and the 84 call
  sites that built the string by hand go through them: version output, ledger
  and token summaries, lottery and voting receipts, oracle source listings,
  doctor checks. English output is unchanged, because its colon is the one that
  was hard-coded.

- **The desktop adapter's own three errors are translated**: these are not
  domain rules with a sentence around them, they are the whole explanation —
  "blockchain not initialised" on a ledger card, "no data recorded for this
  source yet" under Get Latest, "desktop settings path is empty" beside the save
  that failed. All three are in both catalogs.
- **Three such checks now share one walker**: the domain and adapter i18n tests
  each repeated the same directory walk, so it moved into one helper. The
  adapter test deduplicates its report, because one message is declared twice
  and a list that repeats it reads like two problems.

- **The eight fetch-failure reasons are translated, including the wrapped
  ones**: a source that fails to fetch shows its reason verbatim in the health
  card's Last error field, and none of the eight — empty body, invalid JSON,
  size cap, redirect cap, path extraction, request failure, timeout — had an
  entry, so that line stayed English between Chinese labels. The translator
  also learned to translate the cause of a wrapped error: "http error response:
  status 503" now reads "HTTP 响应错误：status 503", where trying the tail alone
  left the head in English.

- **The oracle's four error messages are translated**: the domain reports in
  English and the desktop looks each whole sentence up in its Chinese table, so
  a missing entry reaches a Chinese window in English while every surrounding
  word is Chinese. Fetch now, Delete and Enable/Disable could show "rate limit
  exceeded", "source not found", "invalid source" and "source is disabled"
  untranslated. All four are in both Chinese catalogs now.
- **A test holds the two halves together**: it collects every message the
  domain declares and fails when one has no desktop entry, with an explicit
  list of the three that never reach the window — two programming faults and
  the not-found sentinel the desktop rewrites into a module-specific sentence.

- **The architecture document says what it closed, not what it found**: the
  section was headed 已识别的边界问题 and every one of its seven entries began
  with 已做, so a reader reaching the heading could not tell whether it listed
  open work or finished work; finding out meant reading all of it. It is headed
  已解决的边界问题 now, opens by saying the entries record conclusions rather
  than plans, and points at the one item that is still open. The checklist
  section says the same about its progress.

- **Registering a candidate is a primary action like its neighbours**: on the
  Voting setup page, Register Voter and Create Session were primary buttons and
  Register Candidate, the same kind of write sitting between them, was
  secondary. Each card has one submit action, so the three now look alike
  instead of making one of them read as optional.

- **Every metric the API reference names has to exist**: the metrics section is
  correct today, but a documented series that stops being emitted is the quiet
  kind of wrong — the dashboard or alert built on it draws an empty graph rather
  than failing, so nothing points at the mistake. A test reads the series names
  out of the registry, the API gauges and the oracle scheduler, and fails when
  the reference mentions one that no package registers.

- **The lottery reference states the id resolution order instead of pointing at
  the CLI**: it said `{id}` accepts an id, a block height or a substring "in the
  same order as the CLI", which meant opening a second document to learn an
  order that decides behaviour — a seed that is all digits is read as a height
  before it is tried as a substring. The three steps are a table now. The
  paragraph also says plainly that a failed proof is still a 200 with
  `integrity_ok: false`, so a script that only checks the status code reads a
  broken draw as valid.
- **A test reads the resolver's order out of the source**: `GetByID` before
  `GetByBlockHeight` before `GetAll`, which is what makes the table true.

- **The ledger card's page size and three states are covered**: the page-size
  clamp is pure arithmetic feeding a query, so its bounds are worth pinning
  rather than reading — zero, a negative value and one under the floor all
  become the floor; one over the ceiling becomes the ceiling; a fraction
  truncates. The card's three states are told apart too: an unverified ledger
  asks for a refresh, a verified one shows the head as its identity, and a
  broken one says so instead of leaving the counters to imply health. An
  unverified page turned out to be reachable only before the services are
  wired, because showing the page loads it. `nativeui` coverage rose from 94.4%
  to 94.7%.

- **The request limits are a table instead of four separate bullets**: the body
  cap, the header cap, the rate limiter and CORS each held one sentence about a
  limit or a switch, so comparing them meant reading four places to learn which
  apply before auth and what a client sees when one is hit. They are one table
  with the default, the consequence and the status code. A test reads the two
  caps out of `MaxRequestBody` and `maxHeaderBytes`, and the request-id and log
  widths out of their declarations, so a number that changes has to reach the
  reference.

- **The vote gate is a table of outcomes in the order they are checked**: the
  section said which conditions reject a ballot but not which error each one
  produces, and it left out the case that surprises clients — a draft session
  inside its window is refused with `SESSION_NOT_STARTED`, not accepted and not
  `SESSION_ENDED`, which decides whether calling start can fix it. The table
  lists all five outcomes, and a test reads the `return` statements out of
  `CanAcceptVotes` and fails when a row's code or position disagrees.

- **The latest reading and the transfer write are covered**: both sat at 43%
  with only their guard exercised. The transfer tests pin the rule the code is
  built around — a key that worked is cleared from the screen, a key that was
  rejected stays so the retry does not need it re-entered — and the failure case
  is what proves the rule, since clearing unconditionally passes the success
  test. The latest-reading tests cover success, a failed upstream and the guard
  in front of both. `nativeui` coverage rose from 94.2% to 94.4%; the two
  functions went from 43% to 100%.

- **The voting guide points at the tab it means**: it said 打开“投票”, but the
  tab it sends the reader to is 选票 — the page is called 投票, the section
  inside it is not. A reader following that sentence looks for a control that
  is not on screen. Fixed in both Chinese catalogs.
- **The voting first-run path says how it ends**: the other three modules close
  their first-record guidance with what becomes possible once the first write
  exists — inspect the asset, read the balances, review the health. Voting
  stopped at creating a session, leaving the ballot and the tally unmentioned.

- **The guide describes the language card as it now is**: it still said the
  card leaves all three radios unchecked while following and that Follow System
  is a separate control outside the group — both true before the options became
  one group of four. It now names the four options in order and says following
  the system is the group's first entry, matching the Appearance card below it.
- **A test ties that sentence to the options the card draws**: the card changed
  shape twice and the guide described the old shape both times, so the names it
  promises are checked against the list in `languageChooser`.

- **Follow-the-system is a radio in the language group, not a button beside
  it**: the card answered the same question as the Appearance card below —
  "follow the system, or pick one" — with a different control, and no radio in
  the group meant "back to the system", so returning to it was a separate move.
  Both cards now present it the same way: one group of four. The sentence that
  explained following the system moved from the button's description to that
  radio's, which the unused-key audit caught when the button went away — the
  alternative was deleting it and leaving keyboard users with a bare label.

- **The desktop guide matches how the language card now behaves**: it said the
  current language carries the checked radio and that Follow System is a
  separate action. Since the choice is recorded, that is no longer what the card
  shows — following the system leaves every radio unchecked and writes "Follow
  System · English" in the current row. The guide now explains what the check
  marks answer, and the Settings summary says the same.
- **The three choice restorers are covered**: `restoreSourceChoice`,
  `restoreVoterChoice` and `restoreCandidateChoice` store a row's id rather than
  its index, so a filter that reorders the list still lands on the same record
  and a vanished record leaves nothing highlighted. Tests cover all three
  properties — position after filtering, missing record, and an empty id.
  `nativeui` coverage rose from 93.9% to 94.2%.

- **The page-status clear and the services accessor are covered**:
  `clearPageStatus` switches over the five pages that carry an outcome line, and
  one missing case would keep a stale "Done" on that page while the rest kept
  working. Every branch is now exercised, with the sibling page checked to
  prove only the open one is emptied. `App.Services` — the accessor the deskapp
  tests reach the window's dependencies through — is checked to hand back what
  was injected. `nativeui` coverage rose from 93.7% to 93.9%.

- **A failed language write is reported instead of swallowed**: `SetLanguage`
  applied the locale and then ignored the result of saving the choice, so a
  read-only preferences file left the operator with a changed window and no
  hint that the next launch reverts. It reports the error now, the way
  `SetThemeMode` already does, and Settings shows the same "Couldn't save
  settings" line the theme control uses. The language is still applied either
  way — the window and the native controls must not disagree for the session.

- **The blocked-header rules are a table, and the table found a missing
  header**: the same paragraph carried the header rules for a data source, and
  it named six of the seven hop-by-hop headers the domain rejects —
  `Proxy-Connection` was absent. Grouped by why a header is refused (the
  transport derives it, it belongs to one connection, or it is a proxy
  credential), with the URL and JSON-path rules as their own sentences. A test
  reads both sets out of `internal/domain/oracle/service.go` and fails when the
  table omits one.

- **The SSRF rules are a table of reasons, not a run-on list of prefixes**:
  the API reference named blocked ranges inline in a twenty-line paragraph that
  also carried the header rules and the redirect policy. It listed about half of
  the 30 prefixes in the policy and gave no way to tell which half. They are now
  grouped by why they are blocked — loopback, private, link-local, CGNAT,
  multicast, documentation ranges, IPv6 transition prefixes — with one example
  each and a pointer to `internal/netguard/blocked.go` for the full set. A test
  checks that every prefix the table names exists in that file and that the
  count in the prose matches the list.

- **Settings says whether the language was chosen or follows the system**: the
  card showed the language in effect as an active radio plus a separate Follow
  System button, so a window following an English system and one set to English
  on purpose looked identical — and the difference decides what a later system
  change does. The stored choice is now recorded in `desktop-settings.json`
  (`language`, absent means follow the system), `Shell.LanguageFollowsSystem`
  answers from that record, and the card reads it: following shows "Follow
  System · English" with no radio active and the follow control filled in,
  choosing shows the language alone with the follow control quiet. When no
  getter is supplied the card says nothing about the policy rather than
  guessing one.

- **The current-language row is covered**: `languageLabel` was exercised on
  one of its branches. The cases that matter are the two the row gets wrong
  when it guesses — an unsupported tag must be shown as itself, because
  displaying "English" for a French setting is a false statement about the
  window, and an empty choice stays an em dash rather than claiming a language
  nobody selected. `nativeui` coverage rose from 93.5% to 93.7%; the function
  itself from 25% to 87.5%.

- **The ledger status, session controls and manual fetch are covered**:
  `LedgerStatus`, `StartSession`, `EndSession` and `FetchOracleSource` had no
  test. The first derives its height as `Len()-1`, which is only right because
  a chain always carries its genesis block; the test records that invariant
  next to the assertion. The session pair is read-then-write on purpose — a
  missing id must fail instead of reporting a session started that was never
  stored — and the fetch test keeps a failed upstream from leaving the card
  looking freshly updated. `deskapp` coverage rose from 82.3% to 83.5%.

- **The guide's control names are checked against the page**: every "How this
  works" step tells the reader where to go by naming a card, a tab or a button
  — "Open Ballot", "press Verify". A renamed control leaves the sentence
  pointing at nothing and nothing fails: the guide still reads well, it just no
  longer matches the page. A test now opens each guided page with records and
  checks that every control its steps name is drawn.
- **The token guide's comment counted its steps wrong**: it said "the same
  three moves" above four steps, carried over from before the signing rule and
  the storage rule were split apart.

- **The dashboard names its composition section once**: the page opened the
  section with a ruled "Ledger composition" label and then repeated itself as
  "What the ledger is made of" in the card immediately below. The rule is gone,
  the card's heading and its explanation stay, and with them the last caller of
  `sectionRule` — the helper and its pixel test are removed rather than left as
  dead code. `TestDashboardCompositionIsMoreThanACounterGrid` required both
  labels, which is how the duplication survived; it now requires one.

- **The desktop's token paths are covered**: `TokenInfo`, `TransferToken`,
  `ApproveToken`, `TransferFromToken`, `BurnToken`, `TokenAllowance` and
  `TokenHistory` had no test. They are assembled here the way the container
  assembles them — a real SQLite-backed repository, event store, replay
  protection and chain — because these paths sign, persist and read back, so a
  fake service would only prove the adapter calls it. `deskapp` coverage rose
  from 79.5% to 82.3%.
- Writing them surfaced two facts the tests now record: creating a token
  credits its whole supply to the owner, so a balance after a mint is the
  supply plus the minted amount; and the transfer stream holds transfer events
  only, so a mint is not history until something moves. The bus also needs the
  audit subscription for events to reach the reader at all.

- **The desktop's oracle lifecycle is covered**: `SetOracleSourceEnabled`,
  `DeleteOracleSource`, `OracleData` and `OracleLatest` had no test. The first
  one carries the only branch among them — the Sources card's Enable and
  Disable buttons both land there — so a reversed condition would flip both
  silently. The tests also pin the empty-source message: a feed that was never
  fetched must say so rather than render a zero row that reads like a real
  measurement of zero. `deskapp` coverage rose from 78.1% to 79.5%.

- **The desktop's NFT writes and operation trail are covered**: `TransferNFT`,
  `BurnNFT` and `NFTHistory` had no test, so the three paths that sign with a
  private key, check ownership and record an operation were only exercised by
  hand. The new tests mint, transfer, read the trail and burn over an in-memory
  store and a real chain, and assert the parts a wrong implementation would
  still let through: the asset changes owner, the previous owner's key is
  refused afterwards, a key that parses but belongs to someone else is refused,
  and a burned asset is gone while its history still explains what happened.
  `deskapp` coverage rose from 76.8% to 78.1%.

- **The desktop's NFT reads are covered**: `GetNFT`, `ListNFTs` and the paging
  path they share had no test, so a field dropped from the adapter — the
  Inspect card simply showing nothing where the store has a value — could only
  be found by comparing the window with `aurora nft get`. The new test mints
  over an in-memory repository and a real chain, then checks every mapped field,
  the owner filter and the missing-id path. `deskapp` coverage rose from 75.4%
  to 76.8%.

- **The CLI reference's operations section is grouped by command**: it ran as
  one block of prose under the example commands, and the `blockchain` and
  `migrate` notes appeared in two places each — the same facts about `verify`'s
  output and `blocks`'s payload restated eleven paragraphs apart. Four
  subsections — 账本, 备份, 自检, 迁移 — now hold one command family each, the
  duplicated sentences are one statement the first time, and the `backup`
  JSON shapes moved from the migrate paragraph to the backup one. A contents
  list at the top links every heading, including the new ones.

- **A wrapped tab row loses its underline, not its meaning**: the active tab's
  underline exists to join it to the hairline the list draws beneath, so the tab
  reads as part of the panel. When a narrow window at a large text size wrapped
  the labels onto two rows, that underline landed on the first row and read as a
  divider cutting the trailing tabs off — the NFT page showed Transfer NFT
  underlined with Burn beside it and History stranded below the line. The
  underline is skipped when the row wraps; the weight and accent colour already
  mark the selection. A test compares the wrap prediction against the rows the
  list actually drew, because a disagreement would put the underline back on a
  wrapped row.

- **A disabled button drops its hint instead of clipping it**: both runs in
  the button are single-line, so a row with no room cut them: at 760px with 2x
  text the Create Token button read "Complete the re" and lost the rest. The
  disabled fill and the muted label already say the action is not ready, and
  the sentence is the explanation — so it is drawn only when it measures to
  fit, which is what the label and the hint are measured for at the page's own
  content width.

- **The page header stacks instead of shredding its parts**: in a narrow window
  at a large text size the header's single row squeezed the purpose sentence
  into a four-line column and clipped the Refresh button's Ctrl+R chord to two
  characters — the shortcut the button exists to teach. Below the breakpoint
  the action now drops under the title row, and the button keeps its own
  measure instead of stretching to the column's width. The breakpoint is
  expressed per text scale like the two-column one, so a wide window at 2x
  keeps the one-line header.

- **The privacy statement lists each file and its mode**: it described the
  owner-only policy in one long paragraph, so answering "what does Aurora write
  to disk, and who can read it" meant parsing prose. A table now names every
  path — the database, its sidecars, the log, the desktop settings, a backup
  set, the restore undo snapshot — with the mode Aurora gives it. The row for
  `window-state.json` credits MyGo, because Aurora does not set that one and
  claiming a number it does not choose would be false. A test reads the modes
  out of the hardening code (`RestrictSQLiteFiles`, the backup and restore
  directory calls) and fails when the table disagrees.

- **The NFT readout labels its account Holder, not Owner**: the token page
  calls the account behind a token its 所有者; the NFT page calls the account
  holding an asset its 持有者, and that word appears in thirteen of its
  sentences. One shared "Owner" key could only render one of them, so the NFT
  collection readout showed 持有者 while the token inspect readout showed 所有者
  from a different key. The NFT field is "Holder" now, and a vocabulary test
  keeps the two apart.

- **The NFT page says transfer, not remit**: the English "Transfer" key is
  shared with the token page, where 转账 is right — moving value between
  accounts. The NFT page moves an asset between owners, and every sentence
  around its tab already said 转让 while the tab itself said 转账. The NFT page
  now keys off "Transfer NFT" (转让), so the tab, the card and the button agree
  with the prose. English reads "Transfer NFT" there too, which is clearer next
  to Burn and History than a bare "Transfer".
- **The Chinese catalog states where a new entry belongs**: the table is
  sectioned by page and had no rule for which section a new key joins. Its
  header now says to file a key under the page that draws it, and to put copy
  shared by several pages in the shared section that names its condition rather
  than in whichever module needed it first.

- **The repeat-causing dedup moved to the shared warning**: the previous fix
  un-joined repeated causes inside the Voting page's own aggregation, which left
  every other caller of `loadWarning` unprotected and put the rule in the wrong
  place — the strip is what renders the message. `loadWarning` now drops a
  reason already contained in what it kept, so any page that joins errors gets
  the same treatment, and `syncVotingLoadErr` is a plain join again that still
  reports all three failures to anyone reading the model.

- **A failed Voting load names its cause once**: the page joins the three list
  reads into one warning, and `errors.Join` renders every error it is given —
  so when all three failed with "database is locked" the strip read the same
  sentence three times under a heading that already said which module failed.
  The join now drops repeated messages and keeps distinct ones, because a
  different cause per list is a different fact about the failure.

- **The oracle commands are grouped by what they act on**: `oracle.go` held
  the repository helper, the group and all eleven subcommands in 486 lines.
  Feeds are `oracle_source.go`, reads are `oracle_read.go`, presets are
  `oracle_template.go`; `oracle.go` keeps the helper, the group and the single
  `init` that registers everything, so no command moved in the tree. Names,
  flags and help output are unchanged.

- **Completed removals in the architecture checklist are verified**: the
  document contradicted itself once, marking the `DBPath`/`InitDB`/`OpenDB`
  removal done while another file still listed those symbols as open work. A
  `repocheck` test now reads every completed item that records a removal, finds
  the package it names, and fails if the package still declares one of those
  symbols — so re-adding a deleted API cannot leave a stale "done" behind. Only
  removals are checked automatically; renames and moves depend on judgement.

- **The ablation report stopped calling two solved problems open**: its
  "known architecture debt" section listed three pieces of evidence that
  `blockchain` owned the store; two of them — the `dbInstance` singleton and
  the `utils.RestrictSQLiteFiles` call — were moved into `infra/store` since,
  and `blockchain.{DBPath,InitDB,OpenDB}` no longer exists. The section now
  strikes those through, marks step 2 of its own plan done, and leaves the one
  claim that still holds: `Chain` reads and writes the `blocks` table itself,
  with no repository interface.

- **The REST reference opens with an endpoint index**: the document went
  straight from the connection rules into per-module sections, so learning what
  routes exist meant reading all 500 lines. A bullet index now lists all 48
  routes grouped by module, without parameters or response shapes — those stay
  in the module sections a reader lands on from it. Two tests keep it honest:
  one fails if a registered route is missing from the index or the index names
  a route that does not exist, and one fails if the index is ever rewritten as
  a table, because the existing "every route is documented" check reads tables
  and would then pass without any module section documenting a field.

- **The blockchain package separates appending from reading, verifying and
  constructing**: `block.go` held all four in 576 lines, so the append path —
  the one with the concurrency argument written into its comments — sat among
  read projections and a proof-of-work re-check that never share its locks.
  Reading is `chain_read.go`, integrity is `chain_verify.go`, and block
  construction with the two constructors is `chain_lifecycle.go`; `block.go`
  keeps the types, the append path and the transaction hooks in 334 lines. The
  exported API is byte-identical (`go doc` diff before and after: 30 symbols).

- **The HTTP client separates the request from the transport and the budget**:
  `fetcher.go` held the Fetcher, the per-source rate limiter and the guarded
  transport in 636 lines, so three lifetimes shared one file — a request, a
  window of requests, and the connection the request crosses. The limiter is
  `ratelimit.go` and the transport `transport.go`; `fetcher.go` keeps the
  client, its options and the fetch path in 453 lines. The redirect policy
  stays with the fetch path because it decides what a request may carry.

- **The desktop reference is navigable**: its "架构" section was a flat list of
  58 bullets, three of them longer than 25 lines, so finding the one that
  answered a question meant scrolling the whole thing. It is now eight
  subsections — window and modules, visuals, write feedback, collections,
  copies, page skeleton, forms and selection, keyboard and screen reader, data
  and language — each with a one-line lead, plus a contents list at the top.
  Every bullet kept its text; the count is 73 before and after.
- **The link audit's slug rule matches GitHub for CJK punctuation**: the check
  treated every non-ASCII rune as a letter, so a heading "模块、工具链与 CI"
  produced an anchor no browser would generate and a correct link failed.
  It now keeps only letters, and a new test pins the rule with the CJK and
  trailing-punctuation cases that were wrong.

- **The token guide separates the signing rule from the storage rule**: its
  third step packed three signing keys and a storage disclaimer into one
  sentence, so the page's most important safety fact — which key signs which
  write — arrived as the tail of a 200-character paragraph. It is two steps
  now: which key each write uses, and that no key is stored.

- **The voting commands are grouped by registry**: `voting.go` held the shared
  repository singletons, all eleven subcommands and the registration table in
  596 lines. The commands moved by what they act on — `voting_registry.go`
  (candidates, voters), `voting_ballot.go` (the vote), `voting_sessions.go`
  (session lifecycle and results) — while `voting.go` keeps the singletons,
  the group and the single `init` that registers everything in order. Command
  names, flags and help output are unchanged.

- **The voting use cases are split by stage**: `usecase.go` held all nine use
  cases — the ballot transaction, voter and candidate registration, session
  lifecycle and the tally — in 585 lines. They now follow the flow an operator
  moves through: `castvote.go` (the one transaction that writes the vote, the
  claim, the tally and the ledger block), `register.go` (voters and
  candidates), `sessions.go` (window lifecycle) and `results.go` (the counting
  rule). Constructors and types keep their names, so callers are untouched.

- **The voting repository is split by entity**: 578 lines held the schema, the
  shared transaction plumbing and all four registries — votes, voters,
  candidates, sessions — in one file, so a reader following a ballot through
  the store scrolled past three unrelated CRUD groups. Votes and the plumbing
  stay in `voting.go`; the voter, candidate and session registries moved to
  `voting_voters.go`, `voting_candidates.go` and `voting_sessions.go`. The
  method set is unchanged (22 across the four files) and the concurrency tests
  pass untouched. A doc comment that described a `DeleteVote` this file never
  defined — it sat above `GetVotesByCandidate` — is gone.

- **The verify section states its failure conditions as a list**: the
  sentence broke mid-enumeration ("记录没有中奖者、" ended a line), so the four
  conditions that reject a record read as prose fragments. They are a
  condition list now, with the exit-code rule stated after it.

- **The Chinese catalog's sections match the pages they serve**: the table
  already had section comments, but the Oracle block opened inside the Token
  block and two of its guide sentences closed the NFT one, so a reader editing
  feed copy had to know the file's history to find it. Oracle now has its own
  section with the page's three stages in order, and the destructive
  confirmation prompts and required-field refusals — both shared across modules
  — are marked as their own groups instead of trailing whichever page happened
  to precede them.

- **The lottery commands are grouped by what an operator is doing**:
  `lottery.go` held all nine commands in 581 lines — making a draw, checking
  one, moving records between installations, and inspecting or resetting the
  store. They now follow the split `token` already uses: `lottery.go` keeps
  create/history/verify, `lottery_portability.go` holds export/import, and
  `lottery_admin.go` holds stats/reset/db-info. Command names, flags and help
  output are unchanged; `aurora lottery --help` lists the same eight.

- **The backup package separates the happy path from crash recovery**:
  `create.go` held `Create`'s two-phase write beside ten helpers whose job is
  repairing a run that SIGKILL or power loss interrupted before it began — two
  lifetimes in one file, and the reader could not tell which a function served.
  Those helpers are now `create_recovery.go` with a header naming when they
  run; `create.go` keeps the snapshot/promote/metadata sequence in 260 lines.

- **Cross-document links are checked**: a link into another file's heading
  rots silently when that heading is renamed, which is how the README's new
  gate link could have broken on the next edit to the CLI reference. A
  `repocheck` test now resolves every relative markdown link in the current
  docs, checks that the file exists, and derives the same slug GitHub does to
  confirm the heading is really there. Historical material under
  `docs/superpowers/` and `.planning/` stays out of scope.

- **The dashboard's failure copy names the button that exists**: the broken
  chain's sentence said "Inspect the ledger" while the button beside it reads
  "Open ledger", so a reader looking for an Inspect action on that page found
  none. The sentence now says "Open the ledger", matching the control.

- **The pager's announcement has one definition**: `pagerInline` and
  `pagerControls` carried the same five-line comment and the same
  announce-once bookkeeping, so a fix to one could leave the other silently
  repeating or dropping the range a screen reader hears. `announcePagerMove`
  now holds it, with the reason written once.

- **The REST reference explains errors once**: "错误响应" opened the document
  with the envelope and the full code table, then a separate "错误格式" near the
  end repeated the envelope and stated the status-code rule again, 440 lines
  later. The second section is gone; its status-code rule and the
  unknown-resource rule now sit with the envelope at the top, next to the code
  table a reader consults when one of them fires.

- **The CLI reference's closing section is split by question**: one
  "确认门与错误输出" list held eight unrelated policies — the confirmation gate,
  exit-code rules, log levels, file permissions, config validation, duration
  parsing, timeout overrides and language resolution — so looking up "what
  does this exit code mean" meant scanning all of them. It is now five
  sections (确认门, 错误与退出码, 配置诊断, 日志与文件权限, 语言), the gate is a
  table naming what each command destroys, and the README links to it.

- **The token repository is split by question, not by table**: its 641 lines
  held the schema, the token rows, the allowance primitives and the balance
  primitives in one file, so a reader looking for "may this spender move this
  amount" scrolled past three unrelated groups. `token_approvals.go` now holds
  the allowance half and `token_balances.go` the account half, each with a
  header naming the atomicity argument it makes; `token.go` keeps the schema,
  the token rows and the transaction plumbing. No API moved — same package,
  same method set — and the existing concurrency tests cover both halves.

- **The outbound-address policy lives in one package**: the block list was in
  `internal/netguard` while the host and dial checks that apply it stayed in
  the HTTP client, so a reader had to open two packages to see one policy and
  the package comment claimed call sites it did not contain. `IsBlockedHost`,
  `HostBlocked` (the DNS-aware redirect check) and `ValidateDialAddress` moved
  into `netguard`, their unit tests moved with them, and the HTTP client keeps
  only the wiring: its redirect policy and dialer call the shared checks, and
  `ErrBlockedDestination` is an alias of `netguard`'s sentinel so existing
  `errors.Is` call sites keep working. `fetcher.go` drops from 720 to 636
  lines.

- **The README's project tree lists every package**: it is the first map a
  contributor reads and seven shipped packages were missing from it
  (`config`, `logger`, `metrics`, `netguard`, `records`, `background` and
  `tools/`), so the tree described a smaller repository than the one on disk.
  A `repocheck` test now fails when an `internal/` package is absent from it,
  the same guard the architecture document already had.

- **The architecture document's dependency table is checked, not trusted**:
  the table claims to be verifiable against `go list` and had already drifted —
  `domain`'s row said standard library and same-layer only while
  `domain/oracle` imported `netguard`, and `nativeui`'s row listed one
  dependency while the package also imports the lottery domain for pre-write
  validation. Every row now names the packages its directory actually imports,
  a new `repocheck` test compares the table to the production imports, and a
  row that says "no internal dependencies" fails if one appears.
- **The dashboard's module rails are declared once**: the four rails were
  written out in both layouts, so a rail's title, note or summed total could
  drift between them. They are one ordered list now, with each column naming
  which rails it takes. The arrangement is pinned by a test: wide, the record
  feeds lead the two scales and the second rail of each column shares a row;
  narrow, the same four stack in module order. The hero's own stacking width
  moved into a named constant beside the two-column one instead of a bare 860.

- **The Chinese window says "Failed to load allowance" and "Use this draw for
  Verify" in Chinese**: two labels had no catalog entry, so the allowance
  read's failure and the History card's reuse action stayed English. Both are
  translated now, and the audit that should have caught them was the real bug:
  its index table pointed at each helper's `*ui.Context` instead of its label,
  so the walk found a non-literal argument there and skipped every call. A new
  test asserts each registered index names a string parameter, which fails
  loudly the next time the table drifts from a helper's signature.

- **A failed list read looks like the other strips**: the in-card failure was
  plain danger-red text, so the one state that must not be missed read as a hint
  line among the card's other copy, and its retained-data note looked like part
  of the error. The warning now uses the busy and status strips' shape — a
  fading wash over a lit left edge, a mark and the word "Failed" — with the
  reason in body ink and the stale note muted beneath it. All three strips draw
  that wash through one `statusTintFill`.
- **The dashboard's ledger route looks like a button**: "Open ledger" was quiet
  text at the hero's edge, so the one navigation action on the landing panel
  read like a caption. It uses the secondary button shape now, with the hover
  and press feedback every other card action has.
- **A selectable record says so with a radio mark**: token, NFT, draw, voter,
  candidate, session and data-source lists drew the same recessed card whether
  or not a row was a choice, so the only cues were the hint above the list and
  a tint after the first click. A leading radio circle now marks every
  selectable row, fills with the page accent when chosen, and the row's border
  warms on hover; read-only streams keep plain records.
- **The ledger page size sits on the search row**: the `Show` field stacked its
  label above a number input at the card's far right, so the word floated a line
  away from the control it named while the search field occupied a small part of
  an otherwise empty band. The size is an inline form field beside the query now,
  and the query fills the row.
- **The one-click voter, candidate and session pickers look clickable**: the
  shortcuts were quiet text under their label, so "Voters / Ada" read as a
  label-value pair and the action that fills the public key was invisible. Each
  choice is a bordered pill now, with hover and press states in the page accent.
- **Chinese labels use the full-width colon**: thirteen strings wrote the label
  as `当前版本: 8` while the other forty-seven use `当前版本：8`; the odd ones out
  read as an English label glued to a Chinese value, and the migrate table showed
  the mix side by side.
- **The database's directory is owner-only like the file**: the shared helper
  created the parent directory 0755 while the database file itself, the logs, the
  backups and the desktop data directory are all 0700. A newly created parent is
  0700 now; an existing one keeps the mode its owner chose, because MkdirAll only
  applies permissions it creates.
- **The lint gate also watches SQL handles and error handling**: `nilerr`,
  `durationcheck`, `rowserrcheck` and `sqlclosecheck` join the shared config, so a
  `rows.Err()` nobody checks, a `*sql.Rows` nobody closes, a `nil` returned next
  to a non-nil error, and a duration multiplied by a duration fail `just lint` in
  both modules. Four test helpers scoped their rows so `defer` covers every exit;
  the NFT rebuild keeps its explicit close (the transaction follows immediately)
  with the reason written beside it.
- **The lint gate also watches error wrapping**: `errorlint` joins the shared
  config, so a sentinel compared with `==` (a wrap would slip past it) and an
  accidental `%v` fail `just lint` in both modules. The deliberate
  "sentinel plus printed detail" shape now lives in `domain/errdetail.Wrap`, one
  documented place instead of twenty-two repetitions, and eighteen comparisons
  across the token/oracle/blockchain code became `errors.Is`.
- **The lint gate also watches response bodies and spelling**: `bodyclose` and
  `misspell` join the shared config, so a leaked `http.Response.Body` and a
  comment typo fail the gate. `bodyclose` found one real leak, in the redirect-
  loop test that discarded a response; it closes the body now.
- **The database handle has a home of its own**: `internal/infra/store` now owns
  the process-wide SQLite handle — `db.path` resolution, the pragmas every writer
  needs, directory creation and the 0600 tightening — so the ledger package no
  longer has to carry a database bootstrap to read one row. Every caller moved to
  `store.Path`/`store.OpenShared`/`store.Open`, and `blockchain` no longer exports
  `DBPath`/`InitDB`/`OpenDB` or holds a database singleton.
- **A fresh export is owner-only**: the export lists participant names and their
  draws, yet a new file was written 0644 while the database, the logs and the
  backups are 0600. A new document now gets 0600; a destination that already
  exists keeps its own bits, as `WriteFileAtomic` promises, so re-exporting does
  not silently change a mode the operator chose.
- **The lint gate lives in one place**: `.golangci.yml` still enabled errcheck
  while both `just lint` invocations passed `--disable=errcheck`, so a bare
  `golangci-lint run` checked a different set than the documented gate. errcheck
  is now simply absent from the config (its 9 non-test findings are stdout
  `fmt.Fprint*` writes nobody can act on), and the recipes no longer override it.
- **The compose image is not a service, so the service recipes are gone**:
  `just dev`, `prod`, `start`, `stop` and `restart` brought up a container that
  exits immediately — the image is a one-shot CLI toolbox, as `compose.yml`
  documents. The README now shows `just image` plus
  `docker compose run --rm aurora <command>`.
- **The Oracle Query card asks for the same page as the other surfaces**: it
  started at 20 readings while REST's `/oracle/query` and the CLI's `oracle data`
  default to 10. The card now starts at 10 (the cap of 100 is unchanged), so an
  untouched window and an untouched script read the same page size.
- **The token page names the step that is actually missing**: the amber band said
  "No token selected — Create or inspect one first", while inspecting also needs
  an id. It now says "Choose one in the list, or create a token", which is what
  selects the subject the write cards act on.
- **The Data card states what actually leaves the machine**: it said "No data
  leaves this machine", while an oracle fetch contacts the source the operator
  configured. The line now reads "Nothing is uploaded; the window opens no
  network listener and contacts only the oracle sources you configure", matching
  the privacy statement.
- **A disabled secondary action looks disabled**: controls such as Start/End
  Session, Query, Get Latest, Enable/Disable/Fetch now, Get NFT and Register
  Candidate blocked the click but kept the normal surface and hover accent, so
  they read as available; `secondaryButton` now takes the `ready` flag the
  primary and danger buttons already had and renders a muted, hover-free form.
- **First-step hints belong to the empty state only**: the "Choose a session in
  Setup or enter its ID first" hint sat beside a filled Session ID field, and the
  Oracle cards did the same with "Choose a source in Sources…" after a selection.
  Both now show the hint only while the field is empty, whether the object came
  from a list, the Results shortcuts or a new record.
- **`compose.yml` drops the obsolete `version:` key**: Compose v2 ignores it and
  warns on every command; the service, volume and network definitions are the
  whole file, and the reason is now written beside the header.
- **The lottery history table reads field by field**: each draw used to be a
  JSON blob cut at 200 characters, which hid fields mid-value; the table now
  prints ID, seed, winners, participant count, block height and time on their own
  lines, and `--json` still returns the whole record.
- **The CLI's audit list reads like the ledger page**: `blockchain blocks`
  printed each stored payload raw — JSON blobs and pipe-delimited key bytes. The
  table now renders the same module · action · facts sentence the window shows,
  from the shared decoder, and leaves an unrecognised payload untouched;
  `--json` still carries the exact `payload`. Its `-s` filter compares that
  sentence and the height as well, so a word the table shows finds its record,
  and the REST endpoint's `q` compares the same sentence.
- **Token history has one shape on every surface**: `token history --json`
  printed `{"transfers": […]}` while the REST endpoint answered a top-level
  array, and the use case wrapped the page in a one-field struct so each
  consumer unwrapped it differently. The use case now returns the slice, the CLI
  prints the array the endpoint serves, and the wrapper type is gone.
- **The window's draw search runs in storage**: typing in Draw history used to
  filter the loaded page, so a draw older than the first hundred read as "no
  draws match". A new term now reloads from the newest match through the same
  repository read the API and CLI use, with the pager paging inside the matches;
  creating a draw clears a hiding search before the reload so the new row is
  highlighted on the unfiltered page.
- **The version document is locale-independent**: `aurora version --json`
  printed the translated application name, so a script keying on `name` broke
  when the locale switched; the document now carries the stable `Aurora` while
  the table keeps its translated title.
- **The chain head lives in the integrity report**: the head used to be hex
  encoded and trimmed independently by the doctor, the desktop status and the
  diagnostics, each reading the chain beside its verdict. The integrity report
  now carries it as a hex field, present whenever a last block exists (a broken
  chain still reports the head it read), so one verification supplies both the
  verdict and the identity; `aurora blockchain verify` prints its 12-character
  prefix beside the block count.
- **`BackupResult.File` is now `Dir`**: the value the backup returns is the
  output directory (the file names inside are fixed), and the field name plus
  its JSON tag (`file`) said otherwise; the struct, the CLI message, the
  container accessor and the tests use `Dir`/`dir` now.
- **The REST verify endpoint returns the full report**: `GET /lottery/{id}/verify`
  answered `{id, valid, reason}` while the CLI and the desktop show the record,
  every check and the verdict. The endpoint now returns `lotteryapp.VerifyReport`
  (`record`, `checks`, `integrity_ok`, `proof_verified`), resolves its input like
  the CLI (id, block height or substring), and the API-only use case that reduced
  the report is gone.
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

- **`lottery verify` lists what it checked**: the table printed the conclusion
  alone (`✅ 抽奖记录已验证！`), leaving the reason — a proof checked against the
  stored key, or the deterministic winner match a pre-v1.31 draw falls back to —
  to the JSON document. A `🔍 校验项:` block now prints each passing check, so two
  machines can be compared without reading the document.
- **The punctuation guard**: a repocheck test fails when a Chinese character in
  either zh catalog is followed by an ASCII comma, colon or semicolon, so the next
  label cannot drift back to `当前版本: 8`; it leaves Latin labels (`ID: %s`) and
  URLs alone, and checks its own matcher against both shapes.
- **`just test-shuffle`**: the suite runs in registration order, so an
  order-dependent test only fails when the full run happens to hit it (that is how
  the store singleton leak above surfaced). The recipe runs the same packages with
  `-shuffle=on`, and both modules are green under it today.
- **`migrate down` names the migrations it removed**: the mirror of the `up`
  change. "current version 4" alone does not say which three migrations a
  `down 3` rolled back, so the command now prints `Rolled back: 5, 6, 7`
  (已回滚 in Chinese) in the same ascending order every other version list uses;
  the no-op answer at the bottom still says there is nothing to roll back.
- **`migrate up` names the migrations it ran**: "current version 8" alone left
  an operator guessing whether a repair or backfill migration was part of the
  run. One status read now serves both the pending cap and the applied list, so
  the command prints `Applied: 1, 2, …` (and `已应用: …` in Chinese) after the
  version line; a second run, which applies nothing, prints no such line.
- **`aurora-desktop --version` names the MyGo release too**: the binary that draws
  the window is two toolchains deep, and only the Go half was printed. The output
  now ends with `MyGo Version: v0.7.3` when the build info carries a module list,
  mirroring the diagnostics line, in both languages.
- **The doctor checks who can read the configuration**: the file may hold
  `[api] key`, so a mode that lets another user read it now produces a warning
  with the advice (`mode 0644; other users can read it — chmod 600`) instead of a
  silent OK. A failure to stat it is not a warning: the configuration was read.
- **The About card names the toolchain**: the first question for a window bug
  ("which Go and which MyGo?") now has an answer —
  `Runtime: go1.27.3 · mygo v0.7.3`, read from the build info, with the MyGo half
  omitted when a `go run` build carries no module list. The card and the copied
  diagnostics draw from one row list, so they cannot tell different stories.
- **The outbox warning says how long it has waited**: `aurora doctor`'s outbox
  check and the window's copied diagnostics now append the age of the oldest
  parked event ("pending events: 1; oldest waits 2m0s"), which is what separates a
  just-parked event from a stuck retry loop.
- **The scrape says how long the outbox has been waiting**: alongside
  `aurora_outbox_pending_events`, `/metrics` now carries
  `aurora_outbox_oldest_pending_seconds`, the age of the oldest parked event. A
  count can be normal on a busy node; an age that keeps growing means the retry
  loop is stuck. An empty outbox or a failed read omits the line.
- **The capture script can prove a language**: `AURORA_E2E_LANG=zh` (or
  `zh-hant`) switches the real window's language from Settings, then captures the
  Settings and Dashboard pages with their localized titles under
  `dist/e2e/<lang>/`, so a translation regression is visible without a manual
  window run.
- **The Results tab can start from a stored session**: its empty tally asked for
  a session id or a trip to Setup. It now lists the newest three sessions as
  buttons and loads the chosen tally in place; Setup keeps the full, searchable
  list, and the choice also sets the roster the other tabs read.
- **The ballot offers the session roster and the voters by name**: a session
  stores candidate ids and a voter key is long base64, so the Cast Vote card
  asked for both copied out of Setup. With a session chosen it now lists the
  roster as name chips (writing the candidate id) and the registered voters as
  name chips (writing the public key, clearing another voter's private key). The
  private key itself stays manual.
- **The metrics endpoint exposes the node's state**: `/metrics` now carries
  `aurora_outbox_pending_events`, the gauge an operator watches when a delivery
  keeps failing while the ledger records (the same count `aurora doctor`
  prints), and `aurora_ledger_blocks`, the ledger height over time. A read
  failure omits the line for that scrape instead of reporting a stale zero. The
  scrape also carries `aurora_build_info{go,version[,build_time]}`, so a
  dashboard can tell which build an instance runs.
- **The lottery history endpoint filters in storage**: `GET /lottery/history?q=`
  matches id, seed, participants and winners case-insensitively, with paging
  inside the match set, mirroring the audit trail's `q`. Filtering a loaded page
  would have returned only the hits that happened to land on it. The repository
  gained `ListMatching`, the two paged reads now share one scan helper, and
  `aurora lottery history -s/--search` reads through the same filter; its header
  then prints the match count instead of the store total.
- **The Blockchain page names the chain's head**: the summary line had blocks and
  height but no identity, so two installations could not confirm they follow the
  same chain; a third tile shows the newest block's 12-character hash prefix,
  taken from the ledger status that already carries the chain's verdict.
- **The desktop copies a draw's verdict**: the Verify card keeps its findings on
  screen, but a support reply needs text; one action copies `Draw ID …: verified`
  plus every check line, in the language the window is showing.
- **Settings copies its diagnostics**: the About card's rows (version, build
  time, database path and size) go to the clipboard as the same text with one
  action, plus one ledger line (block count, short head hash and whether the
  chain verified) and the audit outbox's pending count, so a support reply
  quotes the window instead of a typed paraphrase. The same action sits in the Ctrl/Cmd+K palette and stays available
  while a background write owns the database.
- **`aurora doctor`**: the first support question — "is this installation
  healthy?" — needed `db-info` + `blockchain verify` + `migrate status` three
  times over. One command now checks config, database, migrations, ledger
  integrity (with the head block's height and short hash) and the audit outbox in
  a fixed order, with `--json` for scripts and
  an exit code that fails only on real problems. A backlogged outbox warns; a
  dirty migration fails; pending migrations are informational, because a
  CLI-created database always reports schema 0 (the repositories create their
  tables lazily and the API migrates on boot).
- **The CLI's read commands print JSON**: every list (`lottery history`,
  `token list`, `nft list`, the three voting lists, `oracle source list`), the
  single-record reads (`lottery verify`, `token info`/`balance`/`allowance`/`history`,
  `nft get`/`history`, `oracle data`/`latest`, `voting results`), the creation
  commands (`lottery create`, `nft mint`, `voting candidate add`,
  `voting session create`), the token writes (`mint`, `transfer`, `approve`,
  `transfer-from`, `burn`), `nft transfer`/`burn`, `voting vote` and the session
  start/end transitions, `lottery export` and `import` (`path`/`exported` and
  `total`/`imported`/`rejected`),
  `migrate status|up|down|repair`, `backup create`/`verify`/`restore`, the two
  `oracle template` commands, `version`, `lottery stats`,
  `lottery db-info` and the two
  blockchain commands take `--json` and print the same fields the REST API
  returns, with `[]` for an empty collection and no banner before the document,
  so a script can pipe the CLI into `jq` instead of parsing tables. Commands that
  print a private key (`token create`, `voter register`) have no JSON mode on
  purpose. `voting results` also stopped tallying its own map and now reports the
  app layer's roster-ordered result, the same one the REST endpoint returns.
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

- **The store tests no longer inherit a closed handle**: one test closed the
  shared database without resetting the singleton, so a later test in the package
  (when run as part of the full suite) opened a fixture the store never created.
  Both now reset through `ResetForTest`, which closes the handle and clears the
  once, and the package passes under `-shuffle=on`.
- **Test suites no longer share one database in the checkout**: the NFT and e2e
  suites called `blockchain.Init()` with no `db.path`, so the process-wide store
  opened `./data/aurora.db` inside the repository, and two suites running at once
  reset each other's chain mid-test (seen as a panic in the burn-atomicity test
  while a second suite ran). Each suite now points `db.path` at its own temporary
  directory through `TestMain`, and `ResetForTest` removes the default `./data`
  only when that is the configured location, so a configured fixture survives.
- **`--help` ends with the same version facts as `--version`**: the usage block
  stopped at the Go line, so the toolkit the window actually runs (MyGo) was
  visible only in `--version`; both now print one shared tail.
- **The last untranslated sentinel**: an audit payload that cannot be decoded
  (`ErrInvalidPayload`, "invalid event payload") had no CLI translation entry, so
  a Chinese line would carry the English sentence. With it mapped, all
  eighty-five domain sentinels the CLI can report have a Chinese form.
- **Five more service messages read Chinese in the window**: a fetch with no
  body, an oversized import, a draw that failed its integrity or could not be
  found, and an empty public key were the last reachable sentinel texts without
  desktop entries, so a Chinese window showed them verbatim.
- **The window's export failure speaks Chinese too**: the CLI localizes the two
  destination checks, and the window showed their English text because the path
  inside the sentence defeats a full-message lookup. The sentence around the path
  is translated now, and an unrecognised variant still keeps its words so a
  diagnostic matches the log.
- **`nft get` keeps empty metadata out of the table**: an NFT minted without a
  description, image URL or token URI printed those three labels with nothing
  after them; `printLocalizedOptionalField` skips a field that has no value, so
  the table shows only what the asset carries.
- **A source without a path no longer prints a dangling label**: the list line
  read `Method: GET  Path:   Interval: 60s`, with the optional path empty; the
  field is omitted when the source has none, so the line says what it means.
- **An unusable export destination says so**: an empty path answered
  `replace : rename ..tmp : no such file or directory`, and a missing directory
  named the temporary file instead of the destination. Both are rejected up front
  ("export path is empty", "export directory … does not exist"), and a plain
  error's detail now goes through the same Chinese table as a wrapped sentinel's.
- **Sentinels are matched through a wrap where it matters**: a wrapped
  `*cliError` lost its localisation to a plain type assertion, and the approve /
  lookup paths compared `ErrTokenNotFound` and `sql.ErrNoRows` with `==`, so a
  repository that wraps them fell into the unknown-error branch. Those use
  `errors.As`/`errors.Is` now, as do the test assertions that would have failed
  spuriously on the same wrap.
- **A config path Viper cannot describe says why**: `--config <directory>` and
  `--config <file without an extension>` both answered
  `failed to read config: Unsupported Config Type ""`. Both the CLI and
  `aurora-api` now report "config path … is a directory" or "config file … has no
  extension; Aurora reads .toml", while a missing `.toml` keeps the OS message
  that already names the path.
- **A database path that names a directory says so**: SQLite answered the first
  write with "unable to open database file (14)", which names neither the path nor
  the reason; the shared directory check now reports "database path … is a
  directory" before any connection is opened.
- **`lottery create --help` no longer promises a "3"**: the flag carried a
  literal default while the effective value is `lottery.defaultCount`, so a
  configured install contradicted its own help page. The flag keeps the zero
  value, the usage text names the config key, and pflag's misleading annotation
  is gone; behaviour is unchanged (no `-c` still resolves through the config).
- **A Chinese `--help` no longer ends in English**: pflag renders the
  "(default 3)" annotation itself, so every localised flag table still carried it.
  The annotation now follows the translator and reads `（默认 3）`, with the
  full-width bracket keeping Chinese spacing; English output is unchanged.
- **The burn question follows the window language, all of it**: the token burn
  dialog kept an English title, an English detail and a reversed argument order
  in Chinese ("从 25 销毁 owner-pk？"), because the helper did not translate the
  title and the detail's format string was never looked up. The helper translates
  the title now, the detail goes through `tr`, and the Chinese format names the
  account first with indexed verbs.
- **A rejected write says it once**: five token write use cases, the lottery
  participant check and the voting key-length check wrapped a sentinel in its own
  text, so the API answered `{"error":"invalid amount: invalid amount"}` (and
  `duplicate participant: duplicate participant`, `invalid private key length:
  invalid private key length`), with the CLI printing the duplicate too. Each now
  returns the sentinel directly; `errors.Is` classification is unchanged.
- **A rejected oracle source explains itself in Chinese**: the source validator's
  stable English details ("disallowed scheme ...", "missing host", a blocked
  private address) reached the Chinese CLI verbatim. Every detail it can produce
  has a translation now, and an unknown detail still passes through so a new one
  stays visible.
- **A wrong-shaped import names the format, not a Go type**: `lottery import`
  answered a JSON object with "json: cannot unmarshal object into Go value of type
  []lottery.LotteryRecord". The document is now described as it should be
  ("expected a JSON array of lottery records, got object"), an unparsable file
  says so, and the Chinese catalogue translates both.
- **Oracle readings from the same second keep their order**: `/oracle/data`
  ordered by timestamp alone and `/oracle/latest` did the same, so two polls in
  one second could come back in either order and "latest" could differ from the
  list's head. Both queries now break the tie on insertion order, and the
  reference states the ordering rule for every paged collection.
- **The lottery history search is literal**: `q` was compiled straight into a
  LIKE pattern, so a seed containing `%` matched everything and `_` matched any
  single character; both are escaped now, matching the audit trail's
  contains-based search.
- **The doctor's ledger and outbox lines read "1 blocks"**: an earlier wording
  fix never reached the catalogs (the edit targeted unpadded lines), so the
  English text kept the plural and the new head-hash format would have printed
  `%!(EXTRA …)`. The details now say `ledger blocks: N; head …` and
  `pending events: N`, and a test rejects any unconsumed format argument.
- **Creating a session returned a different shape from reading it**:
  `POST /voting/session` answered with title/description/status/candidates only,
  while GET, list and the start/end endpoints also carry `start_time`,
  `end_time` and `created_at`; the create response now goes through the same
  mapping, so one client struct parses every session document.
- **A failed verify no longer passes through `--json`**: `lottery verify
  --json` printed `integrity_ok: false` and still exited 0, and
  `blockchain verify --json` returned before its verdict check, so a script
  that trusts the exit code read a broken record as verified. Both modes now
  print the document and then exit non-zero, matching the table path.
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
