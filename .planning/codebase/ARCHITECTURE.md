# Architecture

**Analysis Date:** 2026-04-26

## System Overview

Aurora is a CLI/TUI blockchain-based digital system implementing VRF lotteries, voting, NFTs, fungible tokens, and oracles. It uses a layered architecture with clear separation between domain logic, application use cases, and user interfaces.

```text
┌─────────────────────────────────────────────────────────────┐
│                    CLI Commands Layer                        │
│         `cmd/aurora/cmd/*.go` (Cobra commands)               │
├──────────────────┬──────────────────┬───────────────────────┤
│   Lottery CLI    │   Voting CLI     │   NFT/Token/Oracle    │
│  `lottery.go`    │  `voting.go`     │  `nft.go`, `token.go` │
└────────┬─────────┴────────┬─────────┴──────────┬────────────┘
         │                  │                     │
         ▼                  ▼                     ▼
┌─────────────────────────────────────────────────────────────┐
│                    UI Layer (TUI)                            │
│     `internal/ui/{lottery,voting,nft,token,oracle}/`        │
│         Bubble Tea TUI components with lipgloss styling      │
└─────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────┐
│                 Application Layer (Use Cases)                │
│           `internal/app/{lottery,voting,nft,token}/`         │
│    DTOs → Use Cases → Orchestrate domain services + repos   │
└─────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────┐
│                     Domain Layer                             │
│  `internal/domain/{lottery,voting,nft,token,oracle,blockchain}/`
│  Entities, Services (business logic), Repository interfaces  │
└─────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────┐
│                  Infrastructure Layer                        │
│     `internal/infra/{sqlite,events,http}/`                   │
│         SQLite repos, Event bus, HTTP handlers               │
└─────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────┐
│                   Cross-Cutting Concerns                     │
│     `internal/{i18n,logger,config,utils}/`                   │
└─────────────────────────────────────────────────────────────┘
```

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| `cmd/` | CLI entry points, command wiring | `cmd/aurora/cmd/*.go` |
| Domain Services | VRF proofs, Ed25519 signing, validation | `internal/domain/*/service.go` |
| Domain Entities | Business objects, value objects | `internal/domain/*/entity.go` |
| Domain Repos | Repository interfaces | `internal/domain/*/repo.go` |
| App Use Cases | Orchestration, DTO transformation | `internal/app/*/usecase.go` |
| UI/TUI | Bubble Tea models, rendering | `internal/ui/*/tui.go` |
| Infra SQLite | Database implementation | `internal/infra/sqlite/*.go` |
| Infra Events | Event bus, handlers, replay protection | `internal/infra/events/*.go` |

## Pattern Overview

**Overall:** Layered Architecture with Dependency Injection

**Key Characteristics:**
- **Domain layer** has zero external dependencies (pure Go business logic)
- **Application layer** depends only on domain interfaces
- **Infrastructure layer** implements domain repository interfaces
- **UI/CLI layer** orchestrates app use cases
- Event-driven architecture with `CompositeEventBus` for cross-cutting concerns

## Layers

**CLI Commands (`cmd/aurora/cmd/`):**
- Purpose: Parse CLI flags, create use cases, execute commands
- Location: `cmd/aurora/cmd/`
- Contains: Cobra commands, flag definitions
- Depends on: `internal/app/*` (use cases), `internal/infra/sqlite` (repos)
- Used by: User via `aurora [command]`

**UI Layer (`internal/ui/`):**
- Purpose: TUI rendering and user interaction
- Location: `internal/ui/{lottery,voting,nft,token,oracle}/`
- Contains: Bubble Tea models (`tea.Model`)
- Depends on: `internal/domain/*`, `internal/infra/*`
- Used by: CLI commands via `Run*TUI()` functions

**Application Layer (`internal/app/`):**
- Purpose: Use case orchestration, DTO transformation, transaction boundaries
- Location: `internal/app/{lottery,voting,nft,token,oracle}/`
- Contains: Use cases (`*UseCase`), DTOs (`*Request`, `*Response`)
- Depends on: Domain interfaces (services, repositories)
- Used by: CLI commands, TUI

**Domain Layer (`internal/domain/`):**
- Purpose: Business logic, entity definitions, repository contracts
- Location: `internal/domain/{lottery,voting,nft,token,oracle,blockchain,events}/`
- Contains: Entities, value objects, domain services, repository interfaces
- Depends on: None (pure Go)
- Used by: Application layer, Infrastructure layer

**Infrastructure Layer (`internal/infra/`):**
- Purpose: External integrations (SQLite, HTTP, events)
- Location: `internal/infra/{sqlite,events,http}/`
- Contains: Repository implementations, event handlers
- Depends on: Domain repository interfaces
- Used by: Application layer (via dependency injection)

## Data Flow

### Primary Request Path (Lottery Create)

1. **CLI Entry** (`cmd/aurora/cmd/lottery.go:createCmd.RunE`)
   - Parses `--participants`, `--seed`, `--count` flags
   - Creates repository: `sqlite.NewLotteryRepository()`
   - Creates blockchain: `blockchain.InitBlockChain()`
   - Instantiates use case: `lotteryapp.NewCreateLotteryUseCase()`

2. **Use Case Execution** (`internal/app/lottery/usecase.go:Execute`)
   - Validates input (sanitization, participant count)
   - Calls domain service: `uc.service.DrawWinners()`
   - Creates record: `lottery.CreateLotteryRecord()`
   - Persists to blockchain: `uc.blockRepo.AddLotteryRecord()`
   - Saves to repository: `uc.lotteryRepo.Save()`

3. **Domain Service** (`internal/domain/lottery/service.go:DrawWinners`)
   - Validates participants, seed, winner count
   - Generates key pair: `GenerateKeyPair()`
   - Computes VRF: `VRFProve(sk, seed)`
   - Selects winners: `SelectWinners(output, participants, count)`
   - Maps names to addresses: `NameToAddress()`

4. **Response** (`internal/app/lottery/usecase.go:Execute`)
   - Returns `LotteryResponse` with winners, VRF proof/output
   - CLI prints formatted output

### Secondary Flow (Token Transfer via Events)

1. **CLI Command** (`cmd/aurora/cmd/token.go`) → `transferCmd`
2. **App Use Case** (`internal/app/token/transfer.go:TransferUseCase.Execute`)
3. **Domain Service** (`internal/domain/token/service.go`) validates & executes
4. **Event Published** via `infraevents.CompositeEventBus`
5. **Handlers Execute**: `AuditHandler` (persists to event store), `StatsHandler` (updates counters)
6. **Repository Updated** via `sqlite.NewTokenRepository()`

## Key Abstractions

**Repository Pattern:**
- Interface defined in `internal/domain/*/repo.go`
- Implemented in `internal/infra/sqlite/*.go`
- Example: `lottery.Repository` interface, `sqlite.LotteryRepository` implementation

**Service Pattern (Domain):**
- Interface defined in `internal/domain/*/service.go`
- Pure business logic with no external dependencies
- Example: `lottery.Service` with `DrawWinners()` and `VerifyDraw()` methods

**Event Bus Pattern:**
- `infraevents.EventBus` interface with publish/subscribe
- `CompositeEventBus` combines `SyncEventBus` and `AsyncEventBus`
- Handlers: `AuditHandler`, `StatsHandler`
- Example usage in `wire.go:Wire()` - subscribes handlers to sync bus

**Use Case Pattern:**
- Application service wrapping domain services + repositories
- Input DTOs in `internal/app/*/dto.go` or dedicated files
- Single `Execute()` method per use case

**Singleton Pattern:**
- Blockchain: `InitBlockChain()` uses `sync.Once`
- Database: `InitDB()` uses `sync.Once`
- Accessible via `GetBlockChain()`, `GetDB()`

## Entry Points

**Main Entry:**
- Location: `cmd/aurora/main.go`
- Triggers: Executable invocation
- Responsibilities: Initialize logger, i18n, invoke `cmd.Execute()`

**Root Command:**
- Location: `cmd/aurora/cmd/root.go`
- Triggers: Any `aurora` subcommand
- Responsibilities: Config loading (Viper), data directory setup, `GlobalApp` wiring

**Subcommands:**
- `cmd/aurora/cmd/lottery.go` - Lottery create/history/tui/verify/export/import/stats/reset
- `cmd/aurora/cmd/voting.go` - Voting create/vote/tui
- `cmd/aurora/cmd/nft.go` - NFT mint/transfer/get/list/tui
- `cmd/aurora/cmd/token.go` - Token create/mint/transfer/balance/history/tui
- `cmd/aurora/cmd/oracle.go` - Oracle sources/fetch/query/tui

## Architectural Constraints

- **Threading:** Single-threaded event loop in Bubble Tea TUI; `sync.Once` for singletons
- **Global state:** Module-level singletons in `blockchain/init.go` and `app/wire.go`
- **Circular imports:** None detected - domain has no external dependencies
- **Configuration:** Viper with TOML, paths: `$HOME/aurora.toml` → `./config/aurora.toml`
- **Data persistence:** SQLite via `github.com/mattn/go-sqlite3`; files in `$HOME/.aurora/data/`

## Anti-Patterns

### Direct SQLite Usage in CLI Commands

**What happens:** Commands like `lottery.go` create repositories directly in `RunE`:
```go
lotteryRepo, err := sqlite.NewLotteryRepository(blockchain.DBPath())
```
**Why it's wrong:** Bypasses the dependency injection pattern; makes testing harder; inconsistent with app layer
**Do this instead:** Use the wired `GlobalApp` from `root.go` or inject via `wire.go`

### Genesis Block Side Effects

**What happens:** `blockchain/init.go:InitBlockChain()` inserts genesis block on first run as a side effect
**Why it's wrong:** Hidden mutation during initialization; non-idempotent if genesis changes
**Do this instead:** Explicit initialization function or migration-based genesis creation

## Error Handling

**Strategy:** Errors propagate up the call stack with wrapped context

**Patterns:**
- Domain validation returns typed errors (e.g., `ValidateSeed()` returns `fmt.Errorf`)
- Use cases wrap errors: `fmt.Errorf("failed to draw winners: %w", err)`
- CLI commands print formatted errors and exit with code 1
- Silent errors in some TUI handlers (`_ = rows.Close()`)

## Cross-Cutting Concerns

**Logging:** `zerolog` via `internal/logger.Init()`
- Log levels configurable via `log.level` in config
- Log path configurable via `log.path`

**Internationalization:** `gobwas/i18n` via `internal/i18n/`
- Locale auto-detected or set via `i18n.locale`
- Text accessed via `i18n.GetText("key")`

**Configuration:** `spf13/viper`
- TOML format in `aurora.toml`
- Environment variable override via `viper.AutomaticEnv()`

---

*Architecture analysis: 2026-04-26*
