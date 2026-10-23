# Technology Stack

**Analysis Date:** 2026-10-23

## Languages

**Primary:**

- Go 1.27.2 - All application code (CLI, TUI, HTTP server, blockchain logic)

## Runtime

**Environment:**

- Go 1.27 (Alpine Linux-based container builds)
- `CGO_ENABLED=0` for every production build and the default test gate

**Package Manager:**

- Go modules (go.mod/go.sum)
- GOPROXY: https://goproxy.cn,direct

## Frameworks

**Core:**

- `github.com/spf13/cobra v1.10.2` - CLI command framework
- `github.com/spf13/viper v1.21.0` - Configuration management (TOML)

**TUI (Terminal UI):**

- `charm.land/bubbletea/v2 v2.1.0` - Interactive TUI framework (Bubble Tea)
- `charm.land/bubbles/v2 v2.2.1` - TUI components (text input, table, viewport, etc.)
- `charm.land/lipgloss/v2 v2.0.6` - Terminal styling system

**HTTP/REST:**

- `github.com/go-chi/chi/v5 v5.5.0` - Lightweight HTTP router (REST API)
- Custom HTTP Fetcher in `internal/infra/http/fetcher.go` - Rate-limited HTTP client

**Cryptography:**

- `filippo.io/edwards25519 v1.2.0` - Ed25519 signatures, VRF proofs (via RFC 8037)
- Standard library crypto for EdDSA signing

**Logging:**

- `github.com/rs/zerolog v1.35.1` - Structured JSON logging

## Testing

**Framework:**

- `github.com/stretchr/testify v1.12.1` - Assertions and mocking

**Run Commands:**

```bash
go test ./...                 # All tests
go test -race ./...           # With race detector
go test -cover ./...          # With coverage
```

**CI/CD Testing:**

- Go 1.27.2 on Ubuntu latest
- Cgo-free unit tests plus an explicit cgo-backed race-detector gate
- Codecov integration for coverage tracking

## Build & Linting

**Linter:**

- golangci-lint v2.15.0 in CI (via `.golangci.yml`)
- Enabled: errcheck, govet, ineffassign, staticcheck, unused

**Build Tool:**

- Go toolchain (cross-platform builds for darwin/linux/windows, amd64/arm64)
- GoReleaser for release artifacts (`.goreleaser.yaml`)

## Database

**SQLite:**

- `modernc.org/sqlite v1.61.0` - Pure Go SQLite driver
- WAL mode enabled for concurrency
- Foreign keys enforced

**Migration:**

- Custom migration runner in `internal/infra/migrate/`
- SQL migration files in `migrations/` directory

## Key Dependencies Summary

| Package                 | Version  | Purpose              |
| ----------------------- | -------- | -------------------- |
| spf13/cobra             | v1.10.2  | CLI framework        |
| spf13/viper             | v1.21.0  | Config (TOML)        |
| charm.land/bubbletea/v2 | v2.1.0   | TUI framework        |
| charm.land/lipgloss/v2  | v2.0.6   | Terminal styling     |
| go-chi/chi              | v5.5.0   | HTTP router          |
| rs/zerolog              | v1.35.1  | Logging              |
| modernc.org/sqlite      | v1.61.0  | Database             |
| filippo.io/edwards25519 | v1.2.0   | Crypto (Ed25519/VRF) |
| google/uuid             | v1.6.0   | UUID generation      |
| stretchr/testify        | v1.12.1  | Testing              |

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
- `oracle.scheduler` - Oracle scheduler cadence

## Platform Requirements

**Development:**

- Go 1.27.2+
- No C toolchain for production builds or the default test gate
- Optional: golangci-lint, goreleaser

**Production:**

- Linux/macOS/Windows
- Static binary (no external dependencies)
- Optional: Docker (Alpine-based image available on Docker Hub)

---

*Stack analysis: 2026-10-23*
