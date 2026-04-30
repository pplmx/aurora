# Technology Stack

**Analysis Date:** 2026-04-30

## Languages

**Primary:**
- Go 1.26.0 - All application code (CLI, TUI, HTTP server, blockchain logic)

## Runtime

**Environment:**
- Go 1.26 ( Alpine Linux-based container builds)
- CGO_ENABLED=0 (static linking, no C dependencies)

**Package Manager:**
- Go modules (go.mod/go.sum)
- GOPROXY: https://goproxy.cn,direct

## Frameworks

**Core:**
- `github.com/spf13/cobra v1.10.2` - CLI command framework
- `github.com/spf13/viper v1.21.0` - Configuration management (TOML)

**TUI (Terminal UI):**
- `charm.land/bubbletea/v2 v2.0.2` - Interactive TUI framework (Bubble Tea)
- `charm.land/bubbles/v2 v2.1.0` - TUI components (text input, table, viewport, etc.)
- `charm.land/lipgloss/v2 v2.0.2` - Terminal styling system

**HTTP/REST:**
- `github.com/go-chi/chi/v5 v5.2.5` - Lightweight HTTP router (REST API)
- Custom HTTP Fetcher in `internal/infra/http/fetcher.go` - Rate-limited HTTP client

**Cryptography:**
- `filippo.io/edwards25519 v1.2.0` - Ed25519 signatures, VRF proofs (via RFC 8037)
- Standard library crypto for EdDSA signing

**Logging:**
- `github.com/rs/zerolog v1.35.0` - Structured JSON logging

## Testing

**Framework:**
- `github.com/stretchr/testify v1.11.1` - Assertions and mocking

**Run Commands:**
```bash
go test ./...                 # All tests
go test -race ./...           # With race detector
go test -cover ./...          # With coverage
```

**CI/CD Testing:**
- Go 1.26 on Ubuntu latest
- Codecov integration for coverage tracking

## Build & Linting

**Linter:**
- golangci-lint (via `.golangci.yml`)
- Enabled: errcheck, govet, ineffassign, staticcheck, unused

**Build Tool:**
- Go toolchain (cross-platform builds for darwin/linux/windows, amd64/arm64)
- GoReleaser for release artifacts (`.goreleaser.yaml`)

## Database

**SQLite:**
- `github.com/mattn/go-sqlite3 v1.14.41` - Pure Go SQLite driver (cgo)
- WAL mode enabled for concurrency
- Foreign keys enforced

**Migration:**
- Custom migration runner in `internal/infra/migrate/`
- SQL migration files in `migrations/` directory

## Key Dependencies Summary

| Package | Version | Purpose |
|---------|---------|---------|
| spf13/cobra | v1.10.2 | CLI framework |
| spf13/viper | v1.21.0 | Config (TOML) |
| charmbracelet/bubbletea | v2.0.2 | TUI framework |
| charmbracelet/lipgloss | v2.0.2 | Terminal styling |
| go-chi/chi | v5.2.5 | HTTP router |
| rs/zerolog | v1.35.0 | Logging |
| mattn/go-sqlite3 | v1.14.41 | Database |
| filippo.io/edwards25519 | v1.2.0 | Crypto (Ed25519/VRF) |
| google/uuid | v1.6.0 | UUID generation |
| stretchr/testify | v1.11.1 | Testing |

## Configuration

**Format:** TOML (`config/aurora.toml`)

**Lookup Order:**
1. CLI flags
2. `$HOME/aurora.toml`
3. `./config/aurora.toml`

**Key Config Sections:**
- `server` - HTTP server host/port (default: 0.0.0.0:8080)
- `log` - Log level and path
- `db` - SQLite database path
- `http.rateLimit` - Requests per window for external fetches
- `oracle.sources` - Data source configurations

## Platform Requirements

**Development:**
- Go 1.26+
- SQLite3 (via go-sqlite3 driver)
- Optional: golangci-lint, goreleaser

**Production:**
- Linux/macOS/Windows
- Static binary (no external dependencies)
- Optional: Docker (Alpine-based image available on Docker Hub)

---

*Stack analysis: 2026-04-30*