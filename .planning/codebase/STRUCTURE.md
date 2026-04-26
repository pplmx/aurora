# Codebase Structure

**Analysis Date:** 2026-04-26

## Directory Layout

```
aurora/
├── cmd/aurora/           # CLI entry points (Cobra commands)
│   ├── main.go          # Package main, app initialization
│   └── cmd/             # Command implementations
│       ├── root.go      # Root command, config, GlobalApp
│       ├── lottery.go   # Lottery subcommands
│       ├── voting.go    # Voting subcommands
│       ├── nft.go       # NFT subcommands
│       ├── token.go     # Token subcommands
│       └── oracle.go    # Oracle subcommands
├── internal/             # Application code (no external deps in domain)
│   ├── app/             # Application layer (use cases)
│   ├── domain/          # Domain layer (entities, services, repos)
│   ├── infra/           # Infrastructure layer (SQLite, events, HTTP)
│   ├── ui/              # TUI layer (Bubble Tea)
│   ├── config/          # Config structs
│   ├── i18n/            # Internationalization
│   ├── logger/          # Logging setup
│   └── utils/           # Utility functions
├── config/              # Default config file (aurora.toml)
├── data/                # Runtime data (blocks.db, etc.)
├── e2e/                 # End-to-end tests
├── scripts/             # Build/dev scripts
├── docs/                # Documentation
├── .planning/           # Planning artifacts (this directory)
│   └── codebase/        # Codebase mapping documents
├── go.mod               # Go module definition
├── go.sum               # Dependency checksums
├── justfile             # Just command runner
├── Makefile             # Makefile for common tasks
└── AGENTS.md            # Project guide for AI agents
```

## Directory Purposes

**`cmd/aurora/`:**
- Purpose: Executable entry points and CLI command definitions
- Contains: `main.go`, `cmd/` directory
- Key files: `cmd/root.go` (GlobalApp wiring), `cmd/lottery.go` (subcommands)

**`internal/domain/`:**
- Purpose: Pure business logic with zero external dependencies
- Contains: Entities, services, repository interfaces, domain events
- Key files:
  - `domain/lottery/entity.go` - `LotteryRecord`, validation, `SelectWinners()`
  - `domain/lottery/service.go` - `DrawWinners()` with VRF
  - `domain/lottery/repo.go` - `Repository` interface
  - `domain/blockchain/init.go` - `BlockChain` singleton, `InitBlockChain()`
  - `domain/events/types_*.go` - Event type definitions

**`internal/app/`:**
- Purpose: Use case orchestration, DTO transformation
- Contains: Use cases (`*UseCase`), DTOs (`*Request`, `*Response`)
- Key files:
  - `app/wire.go` - Dependency injection setup
  - `app/lottery/usecase.go` - `CreateLotteryUseCase`
  - `app/token/transfer.go` - Token transfer logic

**`internal/infra/`:**
- Purpose: External system integrations
- Contains: SQLite repositories, event bus, HTTP handlers
- Key files:
  - `infra/sqlite/lottery.go` - `LotteryRepository` implementation
  - `infra/sqlite/blockchain.go` - Block persistence
  - `infra/events/bus.go` - `SyncEventBus`, `EventBus` interface
  - `infra/events/handlers.go` - `AuditHandler`, `StatsHandler`

**`internal/ui/`:**
- Purpose: Terminal user interface (TUI)
- Contains: Bubble Tea models
- Key files:
  - `ui/lottery/tui.go` - `RunLotteryTUI()`, `model` struct
  - `ui/components/` - Shared styling with `lipgloss`

**`internal/config/`:**
- Purpose: Configuration struct definitions
- Contains: `config.go` - AppConfig struct

**`internal/i18n/`:**
- Purpose: Internationalization
- Contains: Locale detection, translation lookup

**`internal/logger/`:**
- Purpose: Logging infrastructure
- Contains: `zerolog` initialization

## Key File Locations

**Entry Points:**
- `cmd/aurora/main.go` - Executable entry, calls `cmd.Execute()`
- `cmd/aurora/cmd/root.go` - Root Cobra command, `GlobalApp` setup

**Configuration:**
- `config/aurora.toml` - Default TOML config
- `internal/config/config.go` - Config struct
- `cmd/aurora/cmd/root.go` - Viper initialization

**Core Logic:**
- `internal/domain/lottery/entity.go` - VRF lottery entities and logic
- `internal/domain/voting/entity.go` - Ed25519 voting logic
- `internal/domain/token/service.go` - Token operations
- `internal/domain/nft/service.go` - NFT operations

**Testing:**
- `e2e/*_test.go` - End-to-end tests
- `internal/domain/lottery/*_test.go` - Domain unit tests
- `internal/app/lottery/usecase_test.go` - Use case tests
- `internal/infra/sqlite/*_test.go` - Repository tests

## Naming Conventions

**Directories:**
- Lowercase, single words or module names: `domain/`, `app/`, `infra/`
- Module subdirs match module names: `domain/lottery/`, `app/token/`

**Files:**
- Lowercase with underscores: `entity.go`, `service_test.go`, `usecase.go`
- DTOs: `dto.go` or `*_request.go`/`*_response.go`

**Types/Interfaces:**
- PascalCase: `LotteryRecord`, `Repository`, `Service`
- Interfaces often named after role: `Repository`, `EventBus`, `Handler`

**Functions:**
- CamelCase: `NewService()`, `DrawWinners()`, `SelectWinners()`
- Constructor helpers: `New*()`, `Init*()`, `Create*()`
- Test helpers: `ResetForTest()`, `NewTestApp()`

**Packages:**
- Lowercase: `package lottery`, `package sqlite`
- Avoid `internal` package name (Go convention handles this)

## Where to Add New Code

**New Feature Module:**
- Primary code:
  - Domain logic: `internal/domain/{feature}/` (entity.go, service.go, repo.go)
  - Application logic: `internal/app/{feature}/` (usecase.go, dto.go)
  - UI: `internal/ui/{feature}/` (tui.go)
  - Infrastructure: `internal/infra/sqlite/{feature}.go`
- Tests: `internal/domain/{feature}/*_test.go`, `internal/app/{feature}/*_test.go`
- CLI command: `cmd/aurora/cmd/{feature}.go`

**New Use Case:**
- Implementation: `internal/app/{module}/usecase.go`
- DTOs: `internal/app/{module}/dto.go` or co-located structs
- Tests: `internal/app/{module}/usecase_test.go`

**New Repository Implementation:**
- Interface: `internal/domain/{module}/repo.go`
- Implementation: `internal/infra/sqlite/{module}.go`
- Tests: `internal/infra/sqlite/{module}_repo_test.go`

**New TUI Component:**
- Main component: `internal/ui/{module}/tui.go`
- Shared components: `internal/ui/components/` (lipgloss styles)

**Utilities:**
- Shared utilities: `internal/utils/`
- Module-specific: co-located in module directory

## Special Directories

**`.planning/codebase/`:**
- Purpose: Architecture and structure documentation for AI agents
- Generated: Yes (this file)
- Committed: Yes

**`data/`:**
- Purpose: Runtime SQLite databases (blocks.db, tokens.db, etc.)
- Generated: Runtime
- Committed: No (gitignored)

**`config/`:**
- Purpose: Default configuration files
- Generated: Template committed
- Committed: Yes

**`e2e/`:**
- Purpose: End-to-end integration tests
- Generated: Test code
- Committed: Yes

## Module Boundaries

```
┌─────────────────────────────────────────────────────────────────┐
│                        cmd/ (CLI Layer)                          │
│  lottery.go → voting.go → nft.go → token.go → oracle.go        │
└─────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                    internal/ui/ (TUI Layer)                      │
│        Components ──────────────────────────────────► Models     │
└─────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                  internal/app/ (Use Case Layer)                  │
│  CreateLotteryUseCase ──► TransferUseCase ──► MintUseCase       │
│         │                      │                    │           │
│         └──────────────────────┼────────────────────┘           │
│                                ▼                                 │
└─────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                    internal/domain/ (Domain Layer)               │
│  ┌─────────┐ ┌────────┐ ┌───────┐ ┌───────┐ ┌────────┐        │
│  │ lottery │ │ voting │ │  nft  │ │ token │ │ oracle │        │
│  └────┬────┘ └────┬───┘ └───┬───┘ └───┬───┘ └───┬────┘        │
│       └───────────┴─────────┴─────────┴─────────┘              │
│                               │                                 │
│                    ┌──────────┴──────────┐                     │
│                    │    blockchain       │                     │
│                    │    (BlockChain)     │                     │
│                    └─────────────────────┘                     │
└─────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│               internal/infra/ (Infrastructure Layer)             │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  sqlite/  │  events/  │  http/                           │   │
│  │  ───────  │  ───────  │  ─────                           │   │
│  │ lottery.go│  bus.go   │  (handlers)                      │   │
│  │ token.go  │ handlers.go                                 │   │
│  │ nft.go    │ event_store.go                              │   │
│  │ voting.go │ replay.go                                   │   │
│  └───────────┴───────────┴─────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## Dependency Rules

1. **Domain** depends on nothing external
2. **App** depends on Domain only (interfaces)
3. **Infra** depends on Domain (implements interfaces)
4. **UI** depends on Domain/Infra (uses services)
5. **Cmd** depends on App/Domain/Infra (orchestrates)

---

*Structure analysis: 2026-04-26*