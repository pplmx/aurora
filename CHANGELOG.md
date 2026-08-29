# Changelog

All notable changes to this project will be documented in this file.

The v1.x line is milestone-tracked in `.planning/milestones/` and `.planning/STATE.md`
(release `Version` is injected at build time via `-ldflags -X cmd.Version=`). The
entries below summarise v1.64–v1.88; earlier v1.x milestones (v1.0–v1.63) are
documented in their per-milestone ROADMAP files.

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
