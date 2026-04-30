# Architecture

**Analysis Date:** 2026-04-30

## System Overview

Aurora is a blockchain-based digital voting system with VRF (Verifiable Random Function) lottery, built with Go. It provides both CLI and TUI interfaces for managing lotteries, voting, NFTs, fungible tokens, and oracles.

```text
┌─────────────────────────────────────────────────────────────────┐
│                        ENTRY POINTS                              │
├───────────────────────────┬─────────────────────────────────────┤
│   cmd/aurora/main.go      │   cmd/api/main.go                   │
│   (CLI/TUI)               │   (HTTP API Server)                 │
│   Cobra commands          │   Chi router                        │
└─────────────┬─────────────┴─────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        UI LAYER                                  │
│   internal/ui/{lottery,nft,oracle,token}/                       │
│   Bubble Tea TUI components                                     │
└─────────────────────────────┬───────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                     APPLICATION LAYER                            │
│   internal/app/{lottery,voting,nft,token,oracle}/usecase.go     │
│   Use cases, DTOs, input validation                             │
└─────────────────────────────┬───────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                      DOMAIN LAYER                                │
│   internal/domain/{lottery,voting,nft,token,oracle,             │
│                    blockchain,events}/                           │
│   Business logic, entities, services, interfaces                 │
└─────────────────────────────┬───────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                   INFRASTRUCTURE LAYER                           │
│   internal/infra/{sqlite,events,http,migrate,backup}/           │
│   Database, event bus, HTTP client, migrations                  │
└─────────────────────────────────────────────────────────────────┘
```

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| Entry (CLI) | CLI commands, TUI orchestration | `cmd/aurora/cmd/*.go` |
| Entry (API) | HTTP server, request routing | `cmd/api/main.go` |
| UI | Terminal user interface | `internal/ui/*/tui.go` |
| Use Cases | Application-specific workflows | `internal/app/*/usecase.go` |
| Domain Services | Core business logic | `internal/domain/*/service.go` |
| Domain Entities | Data models, validation | `internal/domain/*/entity.go` |
| Repositories | Data persistence | `internal/infra/sqlite/*.go` |
| Event Bus | Async event handling | `internal/infra/events/*.go` |
| Handlers | HTTP request handling | `internal/api/handler/*.go` |

## Pattern Overview

**Overall:** Clean Architecture with three-tier separation

**Key Characteristics:**
- Domain layer has no external dependencies (pure business logic)
- Use cases orchestrate domain services with input validation
- Infrastructure implements domain interfaces (repository pattern)
- Event-driven architecture via composite event bus
- Dual entry points: CLI (Cobra) and HTTP API (Chi)

## Layers

### Entry Points Layer (`cmd/`)

**CLI Entry (`cmd/aurora/`):**
- Location: `cmd/aurora/main.go`, `cmd/aurora/cmd/`
- Contains: Cobra command definitions for each module
- Depends on: `internal/app`, `internal/domain`, `internal/infra`
- Used by: End users via terminal

**API Entry (`cmd/api/`):**
- Location: `cmd/api/main.go`
- Contains: HTTP server initialization
- Depends on: `internal/api`, `internal/infra`
- Used by: Client applications

### UI Layer (`internal/ui/`)

**Purpose:** Terminal user interfaces using Bubble Tea
- Location: `internal/ui/{lottery,nft,oracle,token}/tui.go`
- Contains: TUI models and view logic
- Depends on: `internal/domain`, `internal/app`
- Used by: `cmd/aurora` via `Run*TUI()` functions

### Application Layer (`internal/app/`)

**Purpose:** Use case orchestration, DTOs, input transformation
- Location: `internal/app/{lottery,voting,nft,token,oracle}/`
- Contains: `usecase.go`, `dto.go`, module-specific operations
- Depends on: `internal/domain` (interfaces)
- Used by: `cmd/aurora`, `internal/api/handler`

**Example Use Case Pattern (`internal/app/lottery/usecase.go`):**
```go
type CreateLotteryUseCase struct {
    lotteryRepo lottery.Repository
    blockRepo   interface { AddLotteryRecord(data string) (int64, error) }
    service     lottery.Service
}
func (uc *CreateLotteryUseCase) Execute(req CreateLotteryRequest) (*LotteryResponse, error)
```

### Domain Layer (`internal/domain/`)

**Purpose:** Pure business logic, no external dependencies
- Location: `internal/domain/{lottery,voting,nft,token,oracle,blockchain,events}/`
- Contains: Entities, services, repository interfaces, domain events

**Domain Services:** Implement business rules
- `lottery.Service`: VRF drawing, winner selection
- `voting.Service`: Vote tallying, signature verification
- `token.Service`: Token transfers, minting, burning
- `nft.Service`: NFT ownership, transfers
- `oracle.Service`: Data source management

**Domain Entities:** Core data models
- `LotteryRecord`: Participants, winners, VRF proof
- `Vote`: Proposal, choices, signatures
- `Token`: Supply, decimals, metadata
- `NFT`: Owner, metadata, edition

### Infrastructure Layer (`internal/infra/`)

**SQLite Repositories (`internal/infra/sqlite/`):**
- Purpose: Database persistence
- Files: `lottery.go`, `voting.go`, `nft.go`, `token.go`, `oracle.go`
- Implements: Domain repository interfaces

**Event System (`internal/infra/events/`):**
- Purpose: Event-driven communication
- Components:
  - `bus.go`: Sync event bus
  - `async_bus.go`: Async event processing
  - `composite_bus.go`: Combines sync/async
  - `event_store.go`: Event persistence
  - `handlers.go`: Event handlers (audit, stats)

**HTTP Client (`internal/infra/http/`):**
- Purpose: External data fetching
- Files: `fetcher.go`
- Used by: Oracle module for data sources

**Migrations (`internal/infra/migrate/`):**
- Purpose: Database schema migrations
- Files: `migrate.go`
- Used by: CLI at startup if configured

### API Layer (`internal/api/`)

**Purpose:** HTTP request handling
- Location: `internal/api/handler/`, `internal/api/middleware/`
- Contains: HTTP handlers, middleware, routing

**Handler Pattern:**
```go
type LotteryHandler struct {
    repo lottery.Repository
}
func (h *LotteryHandler) Routes(r chi.Router)
func (h *LotteryHandler) Create(w http.ResponseWriter, r *http.Request)
```

**Middleware:**
- `auth.go`: API key authentication
- `cors.go`: CORS handling
- `logger.go`: Request logging
- `recovery.go`: Panic recovery

## Data Flow

### CLI Lottery Creation Flow

1. **Command Entry** (`cmd/aurora/cmd/lottery.go:createCmd`)
   - Parses flags: participants, seed, count
   - Creates `CreateLotteryUseCase`

2. **Use Case Execution** (`internal/app/lottery/usecase.go:Execute`)
   - Validates input
   - Calls domain service for VRF drawing

3. **Domain Service** (`internal/domain/lottery/service.go:DrawWinners`)
   - Generates key pair
   - Computes VRF proof
   - Selects winners

4. **Persistence** (`internal/infra/sqlite/lottery.go`)
   - Saves to SQLite
   - Records in blockchain

5. **Response** (formatted output to terminal)

### HTTP API Flow

1. **Server Initialization** (`cmd/api/main.go`)
   - Creates `Server` via `NewServer()`

2. **Router Setup** (`internal/api/router.go`)
   - Chi router with middleware
   - Route registration per module

3. **Handler** (`internal/api/handler/lottery.go`)
   - Parses request
   - Calls use case or domain directly
   - Returns JSON response

4. **Response** (`internal/api/response.go`)
   - Standard JSON envelope
   - Error formatting

## Cross-Cutting Concerns

**Logging:** Zerolog via `internal/logger/`
**Configuration:** Viper with TOML files via `internal/config/`
**Internationalization:** i18n via `internal/i18n/`
**Error Handling:** Custom errors in `internal/domain/*/errors.go`
**Event Handling:** Composite event bus with replay protection

## Entry Points

| Entry Point | File | Purpose |
|-------------|------|---------|
| CLI Main | `cmd/aurora/main.go` | Initialize logger, i18n, run Cobra |
| API Server | `cmd/api/main.go` | Start HTTP server |
| Root Command | `cmd/aurora/cmd/root.go` | Config loading, app wiring |
| Module Commands | `cmd/aurora/cmd/{lottery,voting,nft,token,oracle}.go` | Module-specific CLI |

## Key Interfaces

**Repository Interface Pattern:**
```go
// internal/domain/lottery/repo.go
type Repository interface {
    Save(record *LotteryRecord) error
    FindByID(id string) (*LotteryRecord, error)
    FindAll() ([]*LotteryRecord, error)
}
```

**Service Interface Pattern:**
```go
// internal/domain/lottery/service.go
type Service interface {
    DrawWinners(participants []string, seed string, count int) ([]string, []string, []byte, []byte, error)
    VerifyDraw(record *LotteryRecord, publicKey *edwards25519.Point) (bool, error)
}
```

---

*Architecture analysis: 2026-04-30*