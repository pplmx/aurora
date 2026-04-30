# Codebase Structure

**Analysis Date:** 2026-04-30

## Directory Layout

```
aurora/
├── cmd/                    # Entry points
│   ├── aurora/            # CLI application
│   │   ├── cmd/           # Cobra commands
│   │   └── main.go        # CLI entry
│   └── api/               # HTTP API server
│       └── main.go        # API entry
├── internal/              # Core application code
│   ├── api/               # HTTP layer
│   ├── app/               # Application layer (use cases)
│   ├── config/            # Configuration
│   ├── domain/            # Domain layer (business logic)
│   ├── i18n/              # Internationalization
│   ├── infra/             # Infrastructure layer
│   ├── logger/            # Logging setup
│   ├── ui/                # Terminal UI (Bubble Tea)
│   └── utils/             # Utilities
├── config/                # Configuration files
├── migrations/            # Database migrations
├── e2e/                   # End-to-end tests
├── web/                   # Static web assets
├── docs/                  # Documentation
└── scripts/               # Build/deployment scripts
```

## Directory Purposes

### `cmd/` - Application Entry Points

**Purpose:** Executable entry points

**`cmd/aurora/`** - CLI/TUI Application
- `main.go`: CLI entry point
- `cmd/`: Cobra command definitions
  - `root.go`: Root command, config loading, dependency injection
  - `lottery.go`: Lottery subcommands (create, tui, history, verify, export, import)
  - `voting.go`: Voting subcommands
  - `nft.go`: NFT subcommands
  - `token.go`: Token subcommands
  - `oracle.go`: Oracle subcommands
  - `backup.go`: Backup subcommands
  - `migrate.go`: Migration commands

**`cmd/api/`** - HTTP API Server
- `main.go`: HTTP server entry point

### `internal/` - Core Application Code

**`internal/domain/`** - Domain Layer (Business Logic)
- Purpose: Pure business logic, no external dependencies
- Files per module: `entity.go`, `service.go`, `repo.go`, `errors.go`
- Key modules:
  - `lottery/`: VRF lottery with Ed25519 signatures
  - `voting/`: Digital voting system
  - `nft/`: Non-fungible tokens
  - `token/`: Fungible tokens
  - `oracle/`: Data oracle sources
  - `blockchain/`: Simple blockchain storage
  - `events/`: Domain event definitions

**`internal/app/`** - Application Layer (Use Cases)
- Purpose: Orchestrate domain services, DTOs, input validation
- Structure: One directory per module
  - `lottery/`: `usecase.go`, `dto.go`
  - `voting/`: `usecase.go`, `dto.go`
  - `nft/`: `usecase.go`, `dto.go`
  - `token/`: `usecase.go`, `dto.go`, `mint.go`, `transfer.go`, etc.
  - `oracle/`: `usecase.go`, `dto.go`
- Shared: `wire.go` - Dependency injection wiring

**`internal/ui/`** - Terminal UI
- Purpose: Bubble Tea TUI components
- Structure: One directory per module
  - `lottery/tui.go`: Lottery TUI
  - `voting/tui.go`: Voting TUI
  - `nft/tui.go`: NFT TUI
  - `token/tui.go`: Token TUI
  - `oracle/tui.go`: Oracle TUI
  - `components/`: Shared UI components

**`internal/api/`** - HTTP API Layer
- Purpose: HTTP request handling
- Structure:
  - `server.go`: Server initialization
  - `router.go`: Chi router setup
  - `response.go`: JSON response helpers
  - `handler/`: HTTP handlers per module
    - `lottery.go`, `voting.go`, `nft.go`, `token.go`, `oracle.go`
  - `middleware/`: HTTP middleware
    - `auth.go`: API key authentication
    - `cors.go`: CORS headers
    - `logger.go`: Request logging
    - `recovery.go`: Panic recovery

**`internal/infra/`** - Infrastructure Layer
- Purpose: External integrations, persistence
- Structure:
  - `sqlite/`: SQLite repositories
    - `lottery.go`, `voting.go`, `nft.go`, `token.go`, `oracle.go`
    - `blockchain.go`: Blockchain storage
    - `tx.go`: Transaction management
    - `errors.go`: Database errors
  - `events/`: Event bus system
    - `bus.go`: Sync event bus
    - `async_bus.go`: Async event bus
    - `composite_bus.go`: Combined bus
    - `event_store.go`: Event persistence
    - `handlers.go`: Event handlers (audit, stats)
    - `replay.go`: Replay protection
  - `http/`: HTTP client
    - `fetcher.go`: External data fetching
  - `migrate/`: Database migrations
    - `migrate.go`: Migration runner
  - `backup/`: Backup functionality
    - `backup.go`: Backup/restore logic

**`internal/config/`** - Configuration
- Purpose: Configuration loading and access

**`internal/i18n/`** - Internationalization
- Purpose: Translation support
- Files: `i18n.go`, locale data

**`internal/logger/`** - Logging
- Purpose: Zerolog setup

**`internal/utils/`** - Utilities
- Purpose: Shared utility functions

### `config/` - Configuration Files

- `aurora.toml`: Default configuration

### `migrations/` - Database Migrations

- SQL migration files for schema changes

### `e2e/` - End-to-End Tests

- Integration tests spanning multiple modules

### `web/` - Static Assets

- Web files served by API server

## Key File Locations

**Entry Points:**
| File | Purpose |
|------|---------|
| `cmd/aurora/main.go` | CLI application entry |
| `cmd/api/main.go` | HTTP API server entry |
| `cmd/aurora/cmd/root.go` | Root command, DI setup |

**Configuration:**
| File | Purpose |
|------|---------|
| `config/aurora.toml` | TOML configuration |
| `cmd/aurora/cmd/root.go` | Config loading (viper) |

**Core Logic:**
| File | Purpose |
|------|---------|
| `internal/domain/lottery/service.go` | VRF lottery logic |
| `internal/domain/voting/service.go` | Voting logic |
| `internal/domain/token/service.go` | Token logic |
| `internal/app/wire.go` | Dependency injection |

**Persistence:**
| File | Purpose |
|------|---------|
| `internal/infra/sqlite/lottery.go` | Lottery repository |
| `internal/infra/sqlite/blockchain.go` | Blockchain storage |
| `internal/infra/events/event_store.go` | Event persistence |

**API:**
| File | Purpose |
|------|---------|
| `internal/api/server.go` | Server initialization |
| `internal/api/router.go` | Route definitions |
| `internal/api/handler/lottery.go` | Lottery endpoints |

## Naming Conventions

**Files:**
- Go source: `*.go` (lowercase, snake_case for multi-word)
- Test files: `*_test.go` suffix
- Examples: `entity.go`, `service.go`, `usecase.go`, `dto.go`

**Directories:**
- Go packages: lowercase, single word or snake_case
- Module directories: match module name (e.g., `lottery/`, `voting/`)

**Types:**
- Entities: PascalCase noun (e.g., `LotteryRecord`, `Vote`)
- Services: PascalCase noun ending in `Service` (e.g., `TokenService`)
- Use Cases: PascalCase ending in `UseCase` (e.g., `CreateLotteryUseCase`)
- DTOs: PascalCase ending in `Request`/`Response` (e.g., `CreateLotteryRequest`)

**Interfaces:**
- PascalCase (e.g., `Repository`, `Service`)
- Method signatures in documentation comments

## Where to Add New Code

### New Domain Module

**Primary code locations:**
- Domain logic: `internal/domain/{module}/entity.go`, `service.go`
- Repository interface: `internal/domain/{module}/repo.go`
- Application use case: `internal/app/{module}/usecase.go`, `dto.go`
- Infrastructure: `internal/infra/sqlite/{module}.go`
- CLI command: `cmd/aurora/cmd/{module}.go`
- HTTP handler: `internal/api/handler/{module}.go`
- TUI: `internal/ui/{module}/tui.go`

**Tests:**
- Domain tests: `internal/domain/{module}/*_test.go`
- App tests: `internal/app/{module}/*_test.go`
- Repo tests: `internal/infra/sqlite/{module}_test.go`

### New CLI Command

**Location:** `cmd/aurora/cmd/`

**Pattern:**
1. Create new file (e.g., `newcmd.go`)
2. Define `newCmd` variable of type `*cobra.Command`
3. Add to parent command in `init()` function
4. Register flags in `init()`

### New HTTP Endpoint

**Location:** `internal/api/handler/`

**Pattern:**
1. Add method to existing handler or create new handler
2. Register route in `internal/api/router.go`
3. Add middleware if needed in `internal/api/middleware/`

### New Use Case

**Location:** `internal/app/{module}/`

**Pattern:**
1. Create `*{Module}UseCase` struct
2. Implement `Execute(*Request) (*Response, error)` method
3. Inject dependencies via constructor
4. Register in `internal/app/wire.go` if needed

## Special Directories

**`internal/domain/events/`**
- Purpose: Domain event type definitions
- Generated: No
- Contains: Event types for lottery, voting, token, nft, oracle

**`internal/infra/events/`**
- Purpose: Event bus implementation
- Generated: No
- Committed: Yes
- Contains: Sync/async event processing

**`migrations/`**
- Purpose: Database schema migrations
- Generated: No
- Committed: Yes
- Note: Applied at startup if configured

**`web/`**
- Purpose: Static files for API server
- Generated: No
- Committed: Yes
- Note: Served at root path

---

*Structure analysis: 2026-04-30*
