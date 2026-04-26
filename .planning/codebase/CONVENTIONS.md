# Coding Conventions

**Analysis Date:** 2026-04-26

## Naming Conventions

### Files

- **Go source files:** `snake_case.go` (e.g., `service.go`, `entity.go`, `usecase_test.go`)
- **Test files:** `*_test.go` suffix (e.g., `service_test.go`, `usecase_test.go`)
- **E2E test files:** `e2e/*_e2e_test.go`

### Packages

- **Package names:** Lowercase, single-word when possible (e.g., `lottery`, `voting`, `token`)
- **Import aliases:** Use module alias for clarity (e.g., `lottery "github.com/pplmx/aurora/internal/domain/lottery"`)

### Types & Interfaces

- **Structs:** `PascalCase` (e.g., `LotteryRecord`, `CreateLotteryUseCase`)
- **Interfaces:** `PascalCase` with `Service`/`Repository` suffix (e.g., `Service`, `Repository`)
- **Type aliases:** Clear, descriptive names (e.g., `type Service interface`)

### Variables & Functions

- **Variables:** `camelCase` (e.g., `lotteryRepo`, `publicKey`)
- **Functions:** `PascalCase` exported, `camelCase` unexported (e.g., `NewService()`, `DrawWinners()`)
- **Constructor functions:** `New` prefix (e.g., `NewService()`, `NewLotteryUseCase()`)
- **Validation functions:** `Validate` prefix (e.g., `ValidateSeed()`, `ValidateParticipants()`)
- **Test helpers:** Descriptive names (e.g., `setupVotingTestDB()`, `newInMemoryTokenRepo()`)

### Constants

- **Constants:** `PascalCase` for exported, `camelCase` or `PascalCase` for unexported (e.g., `MaxParticipants`, `MaxSeedLength`)
- **Multiple constants:** Use `const (...)` block

### JSON Fields

- **JSON tags:** `snake_case` (e.g., `json:"block_height"`, `json:"vrf_proof"`)

## Code Style

### Formatting

- **Tool:** `gofmt` with `-s` flag (simplify)
- **Import formatting:** `goimports` (sorts and formats imports)
- **Line length:** No strict limit, but keep lines readable

### Indentation & Braces

- **Standard Go formatting:** `gofmt` handles automatically
- **Brace placement:** Same line for control structures (Go style)

## Import Organization

**Order (via `goimports`):**
1. Standard library imports
2. Third-party imports (e.g., `github.com/`, `filippo.io/`)
3. Internal package imports (e.g., `github.com/pplmx/aurora/...`)

**Import aliases:** Use when package name conflicts or for clarity:
```go
import (
    "encoding/hex"
    
    "filippo.io/edwards25519"
    
    lottery "github.com/pplmx/aurora/internal/domain/lottery"
    "github.com/pplmx/aurora/internal/domain/voting"
)
```

## Error Handling

### Error Patterns

**Domain layer errors:** Return errors directly, wrap with context using `fmt.Errorf`:
```go
func ValidateSeed(seed string) error {
    if len(seed) < MinSeedLength {
        return fmt.Errorf("seed too short (min %d chars)", MinSeedLength)
    }
    return nil
}
```

**Use case layer errors:** Wrap errors with `fmt.Errorf("action: %w", err)`:
```go
if err := lottery.ValidateSeed(seed); err != nil {
    return nil, fmt.Errorf("invalid seed: %w", err)
}
```

**Service layer errors:** Propagate with context:
```go
winners, winnerAddrs, output, proof, err := uc.service.DrawWinners(...)
if err != nil {
    return nil, fmt.Errorf("failed to draw winners: %w", err)
}
```

### Error Checking

- Use explicit error checks: `if err != nil { ... }`
- Use `errors.Is()` for sentinel errors when available
- Test errors with `t.Fatal`/`t.Fatalf` for fatal failures
- Test errors with `t.Error`/`t.Errorf` for non-fatal assertions

### Sentinel Errors

- Define as `var ErrXxx = errors.New("xxx")` when needed
- Currently, most errors are inline `fmt.Errorf` calls

## Logging

### Framework

- **Library:** `github.com/rs/zerolog`
- **Initialized in:** `internal/logger/zero.go`

### Log Levels

- **Info:** General application messages
- **Debug:** Detailed debugging information
- **Warn:** Warning conditions
- **Error:** Error conditions
- **Fatal:** Fatal errors (exits application)

### Usage Pattern

```go
import "github.com/pplmx/aurora/internal/logger"

// Using logger package
logger.Info().Str("key", "value").Msg("message")
logger.Error().Err(err).Msg("Operation failed")

// In CLI commands
logger.Error().Err(err).Msg("Application error")
```

### Configuration

- **Log level:** Configurable via `log.level` in config
- **Log path:** Configurable via `log.path` (writes to `aurora.log`)
- **Default level:** `info`
- **Default path:** `./log/`
- **Format:** RFC3339 timestamps, console output by default

## Comments & Documentation

### Package Documentation

At the top of each file with package-level context:
```go
// Package lottery provides VRF-based transparent lottery functionality.
// It implements verifiable random function (VRF) to ensure fair and
// transparent winner selection that can be verified on-chain.
package lottery
```

### Function Comments

For exported functions and complex logic:
```go
// NewService creates a new lottery service instance.
func NewService() Service { ... }

// DrawWinners selects winners from participants using VRF.
func (s *lotteryService) DrawWinners(...) (...) { ... }
```

### TODO Comments

Format: `// TODO(description)` - used sparingly for future work

## Linting Configuration

**Config file:** `.golangci.yml`

### Enabled Linters

```yaml
linters:
  default: none
  enable:
    - errcheck      # Check error returns
    - govet         # Vet tool
    - ineffassign   # Detect unused variable assignments
    - staticcheck   # Static analysis
    - unused        # Detect unused code
```

### Excluded Rules

Test files exclude `SA5011` (possible nil pointer dereference):
```yaml
issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - staticcheck
      text: "SA5011"
```

### In justfile

```justfile
lint:
    golangci-lint run --disable=errcheck
```

## Pre-commit Hooks

**Config file:** `.pre-commit-config.yaml`

### Built-in Hooks

| Hook | Purpose |
|------|---------|
| `end-of-file-fixer` | Ensures files end with newline |
| `trailing-whitespace` | Removes trailing whitespace |
| `check-toml` | Validates TOML files |
| `check-yaml` | Validates YAML files |
| `check-merge-conflict` | Detects merge conflict markers |
| `mixed-line-ending` | Fixes line endings to LF |

### Commitizen

Enforces conventional commits:
```yaml
- repo: https://github.com/commitizen-tools/commitizen
  rev: v4.13.9
  hooks:
    - id: commitizen
    - id: commitizen-branch
```

### Go Hooks (Local)

```yaml
- id: gofmt
  entry: gofmt -l -s -w
- id: goimports
  entry: goimports -l -w
- id: go-vet
  entry: go vet ./...
- id: golangci-lint
  entry: golangci-lint run --timeout=5m --disable=errcheck,staticcheck
```

### Markdown Hooks

```yaml
- id: rumdl
  entry: rumdl fmt && rumdl check
```

## Code Organization

### Layer Structure

```
internal/
├── app/          # Application use cases
├── domain/       # Domain entities and business logic
├── infra/        # Infrastructure (DB, HTTP, events)
├── ui/           # CLI/TUI interfaces
├── i18n/         # Internationalization
├── logger/       # Logging setup
└── utils/        # Utility functions
```

### Dependency Direction

- Domain has no dependencies on other layers
- App layer depends on domain
- Infra layer implements domain interfaces
- UI layer depends on app layer

---

*Convention analysis: 2026-04-26*
