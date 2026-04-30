# Coding Conventions

**Analysis Date:** 2026-04-30

## Style & Formatting

**Go Tools:**
- `gofmt` - Format code (tabs indentation for Go files)
- `goimports` - Format and organize imports
- `go vet` - Static analysis

**EditorConfig** (`.editorconfig`):
- `charset: utf-8`
- `end_of_line: lf`
- `indent_size: 4` (spaces, except Go files use tabs)
- `max_line_length: 120`
- `insert_final_newline: true`
- `trim_trailing_whitespace: true`

## Linting Configuration

**golangci-lint** (`.golangci.yml`):
Enabled linters:
- `errcheck` - Check error returns (disabled in justfile lint)
- `govet` - Go vet analyzer
- `ineffassign` - Detect unused variable assignments
- `staticcheck` - Static code analysis (excludes `SA5011` in tests)
- `unused` - Detect unused code

```bash
just lint    # Run golangci-lint (errcheck disabled)
just check   # gofmt + goimports + go vet
```

## Naming Conventions

**Files:**
- Go source: `snake_case.go` or `lowerCamelCase.go`
- Test files: `*_test.go`
- E2E tests: `e2e/*_e2e_test.go`

**Types/Structs:**
- PascalCase: `LotteryRecord`, `TokenRepository`
- Interfaces: `Service`, `Repository` suffixes
- Private structs: lowercase prefix: `lotteryService`

**Functions:**
- Public: PascalCase: `CreateLotteryRecord`, `NewService`
- Private: camelCase: `validateSeed`, `generateKeyPair`
- Test helpers: often prefixed with `setup`

**Variables:**
- camelCase: `participants`, `winnerCount`
- Constants: PascalCase for exported, camelCase for private: `MaxParticipants`
- Error variables: `err` or descriptive: `errTokenNotFound`

## Package Structure

```
github.com/pplmx/aurora/
├── cmd/                  # Entry points
│   ├── api/             # HTTP server
│   └── aurora/          # CLI application
│       └── cmd/         # Cobra commands
├── internal/
│   ├── api/             # HTTP handlers, middleware
│   ├── app/             # Application layer (use cases)
│   │   ├── lottery/
│   │   ├── nft/
│   │   ├── oracle/
│   │   ├── token/
│   │   └── voting/
│   ├── domain/          # Domain layer (entities, services)
│   │   ├── lottery/
│   │   ├── nft/
│   │   ├── oracle/
│   │   ├── token/
│   │   ├── voting/
│   │   ├── blockchain/
│   │   └── events/
│   ├── infra/           # Infrastructure layer
│   │   ├── backup/
│   │   ├── events/
│   │   ├── http/
│   │   ├── migrate/
│   │   └── sqlite/
│   ├── ui/              # TUI components (Bubble Tea)
│   ├── i18n/            # Internationalization
│   ├── logger/          # Zerolog configuration
│   └── utils/           # Shared utilities
├── config/              # Configuration files
├── e2e/                 # End-to-end tests
├── migrations/          # Database migrations
└── docs/                # Documentation
```

## Import Organization

**Order** (goimports handles automatically):
1. Standard library
2. External packages (third-party)
3. Internal packages (`github.com/pplmx/aurora/...`)

**Example:**
```go
import (
    "encoding/hex"
    "fmt"

    "filippo.io/edwards25519"
    "github.com/stretchr/testify/require"

    "github.com/pplmx/aurora/internal/domain/lottery"
)
```

## Error Handling

**Pattern:** Return errors with context using `fmt.Errorf`
```go
func (r *LotteryRecord) Validate() error {
    if err := ValidateSeed(r.Seed); err != nil {
        return fmt.Errorf("seed: %w", err)
    }
    if err := ValidateParticipants(r.Participants); err != nil {
        return fmt.Errorf("participants: %w", err)
    }
    return nil
}
```

**Custom Errors:** Define sentinel errors in domain packages
```go
// internal/domain/token/errors.go
var ErrTokenNotFound = errors.New("token not found")
```

## Logging

**Framework:** `github.com/rs/zerolog`

**Pattern:**
```go
logger.Info().
    Str("version", Version).
    Str("build_time", BuildTime).
    Msg("Aurora starting")

logger.Error().Err(err).Msg("Application error")
```

## Comments & Documentation

**Package-level:** Required for exported packages
```go
// Package lottery provides VRF-based transparent lottery functionality.
// It implements verifiable random function (VRF) to ensure fair and
// transparent winner selection that can be verified on-chain.
package lottery
```

**Public functions:** Brief comment explaining purpose
```go
// CreateLotteryRecord creates a new lottery record with the given parameters.
func CreateLotteryRecord(...) *LotteryRecord { ... }
```

## Interface Patterns

**Service Interfaces:** Define in domain layer, implement in same package
```go
type Service interface {
    DrawWinners(participants []string, seed string, count int) ([]string, []string, []byte, []byte, error)
    VerifyDraw(record *LotteryRecord, publicKey *edwards25519.Point) (bool, error)
}
```

**Repository Interfaces:** Define in domain, implement in infra
```go
// In domain
type Repository interface {
    Save(*LotteryRecord) error
    GetByID(id string) (*LotteryRecord, error)
    GetAll() ([]*LotteryRecord, error)
}

// In infra/sqlite
type LotteryRepository struct { ... }
```

## Constructor Patterns

**Factory functions:** `New*` prefix for constructors
```go
func NewService() Service {
    return &lotteryService{}
}

func NewTokenRepository(dbPath string) (*TokenRepository, error) { ... }
```

## Test Patterns

**Co-located tests:** `*_test.go` in same package as implementation
```go
package lottery  // Not "lottery_test"
```

**Table-driven tests** for multiple test cases:
```go
func TestValidateParticipantName_Valid(t *testing.T) {
    valid := []string{"Alice", "Bob 123", "test-name", "Name_With", "日本語", "中文"}
    for _, name := range valid {
        err := ValidateParticipantName(name)
        if err != nil {
            t.Errorf("ValidateParticipantName(%q) should not error: %v", name, err)
        }
    }
}
```

**Use testify for assertions:**
```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// require - fatal on failure
require.NoError(t, err)
require.Len(t, results, 2)

// assert - non-fatal
assert.Equal(t, http.StatusOK, rr.Code)
assert.NotNil(t, resp)
```

## Architecture Patterns

**Layered Architecture:**
```
cmd/        → Entry points (CLI, API server)
  ↓
internal/app/   → Use cases, orchestration
  ↓
internal/domain/ → Business logic, entities, services
  ↑
internal/infra/  → External concerns (DB, HTTP, events)
```

**Dependency Injection:** Constructor injection via interfaces
```go
func NewCreateLotteryUseCase(repo Repository, chain BlockChain) *CreateLotteryUseCase {
    return &CreateLotteryUseCase{repo: repo, chain: chain}
}
```

## CLI Framework

**Tool:** `github.com/spf13/cobra` with `github.com/spf13/viper`

**Pattern:**
```go
var rootCmd = &cobra.Command{
    Use:   "aurora",
    Short: i18n.GetText("app.name"),
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        // Initialization
    },
}
```

## Configuration

**Format:** TOML
**Location:** `$HOME/aurora.toml` or `./config/aurora.toml`
**Defaults:** Set via `viper.SetDefault()`

---

*Convention analysis: 2026-04-30*
