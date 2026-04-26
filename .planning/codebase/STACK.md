# Technology Stack

**Analysis Date:** 2026-04-26

## Languages

**Primary:**
- Go 1.26.0 - All application code (CLI, domain logic, TUI, infrastructure)

## Runtime

**Environment:**
- Go runtime (pure Go, no CGO dependencies for main application)
- CGO_ENABLED=0 for cross-platform builds

**Package Manager:**
- Go modules (go.mod/go.sum)
- GOPROXY: https://goproxy.cn,direct

## Frameworks

**CLI Framework:**
- [spf13/cobra](https://github.com/spf13/cobra) v1.10.2 - Command structure and argument parsing
- Pattern: Nested subcommands per module (lottery, voting, nft, token, oracle)
- Location: `cmd/aurora/cmd/`

**Configuration:**
- [spf13/viper](https://github.com/spf13/viper) v1.21.0 - Configuration management
- Format: TOML
- Config lookup order: CLI flag → `$HOME/aurora.toml` → `./config/aurora.toml`
- Supports environment variable overrides via `viper.AutomaticEnv()`

**Logging:**
- [rs/zerolog](https://github.com/rs/zerolog) v1.35.0 - Structured logging
- Supports console (colorized) and file output
- Configurable log levels and output path
- Location: `internal/logger/zero.go`

**TUI Framework:**
- [charm.land/bubbletea/v2](https://github.com/charmbracelet/bubbletea) v2.0.2 - Elm-style TUI
- [charm.land/bubbles/v2](https://github.com/charmbracelet/bubbles) v2.1.0 - UI components (textinput, viewport)
- [charm.land/lipgloss/v2](https://github.com/charmbracelet/lipgloss) v2.0.2 - Terminal styling
- Pattern: tea.Model interface with Update/View methods
- Location: `internal/ui/*/tui.go`

**Database:**
- [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) v1.14.41 - SQLite driver
- WAL mode enabled for concurrent access
- Foreign keys enforced
- Location: `internal/infra/sqlite/*.go`

**Testing:**
- [stretchr/testify](https://github.com/stretchr/testify) v1.11.1 - Assertions and mocks
- Standard Go testing package

**HTTP Client:**
- Standard library `net/http` - Used in `internal/infra/http/fetcher.go`
- 10-second default timeout for oracle data fetching

## Cryptography Libraries

**VRF (Verifiable Random Function):**
- [filippo.io/edwards25519](https://filippo.io/edwards25519) v1.2.0 - Elliptic curve operations
- Used for VRF proof generation and verification in lottery module
- Location: `internal/domain/lottery/vrf.go`

**Ed25519 Signing:**
- Standard library `crypto/ed25519` - Standard library implementation
- Used for voting signatures and NFT ownership
- Location: `internal/domain/voting/service.go`

**Hashing:**
- Standard library `crypto/sha256` - VRF hash generation
- Location: `internal/domain/lottery/vrf.go`

## Key Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| spf13/cobra | v1.10.2 | CLI command framework |
| spf13/viper | v1.21.0 | Configuration management |
| rs/zerolog | v1.35.0 | Structured logging |
| charm.land/bubbletea/v2 | v2.0.2 | TUI framework |
| charm.land/bubbles/v2 | v2.1.0 | TUI components |
| charm.land/lipgloss/v2 | v2.0.2 | Terminal styling |
| filippo.io/edwards25519 | v1.2.0 | VRF/Ed25519 elliptic curve |
| mattn/go-sqlite3 | v1.14.41 | SQLite database driver |
| google/uuid | v1.6.0 | UUID generation |
| go-chi/chi/v5 | v5.2.5 | HTTP router |
| stretchr/testify | v1.11.1 | Testing framework |

## Build System

**Build Tool:** GoReleaser (.goreleaser.yaml)
- Cross-platform builds (Linux, macOS, Windows)
- Multi-arch support (amd64, arm64)
- Docker image publishing to Docker Hub (pplmx/aurora)
- Flags: `-trimpath -ldflags="-s -w"` for minimal binaries

**Task Runner:** Just (justfile)
- Cross-platform command aliasing
- Lint, test, build, and Docker operations

**Linter:** golangci-lint (.golangci.yml)
- Enabled: errcheck, govet, ineffassign, staticcheck, unused

## Platform Targets

**Development:**
- Go 1.26+
- Docker (optional for containerized development)

**Production:**
- Linux (primary), macOS, Windows
- amd64 and arm64 architectures
- Containers: gcr.io/distroless/static:nonroot (minimal image)

---

*Stack analysis: 2026-04-26*