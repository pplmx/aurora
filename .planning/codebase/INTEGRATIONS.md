# External Integrations

**Analysis Date:** 2026-04-26

## Containerization

**Docker:**
- Multi-stage build Dockerfile at `/Dockerfile`
- Build stage: `golang:1.26-alpine`
- Deploy stage: `gcr.io/distroless/static:nonroot`
- Exposed ports: 6666, 8888, 12345
- Cross-platform builds: linux/amd64, linux/arm64

**Docker Compose:**
- File: `compose.yml`
- Services: `aurora` container
- Named volumes: `aurora-data`, `aurora-logs`
- Health check: `./aurora lottery stats`
- Network: `aurora-network`

**Kubernetes:**
- File: `k8s.yml`
- Resources:
  - ConfigMap: `aurora-config` (LOG_LEVEL)
  - PersistentVolumeClaim: `aurora-data` (1Gi storage)
  - Deployment: 1 replica (configurable 1-3 via HPA)
  - Service: ClusterIP on port 6666
  - HorizontalPodAutoscaler: CPU-based scaling (70% target)
- Resource limits: 128-256Mi memory, 100-500m CPU
- Liveness/readiness probes: `./aurora lottery stats`

## Data Storage

**SQLite (Embedded):**
- Driver: `github.com/mattn/go-sqlite3`
- Databases per module:
  - `tokens.db` - Token state and balances
  - `events.db` - Event store for audit trail
  - `nonces.db` - Replay protection
- Location: `internal/infra/sqlite/*.go`
- Pragma settings:
  - `journal_mode=WAL`
  - `foreign_keys=ON`

**File Storage:**
- Data directory: `~/.aurora/data/` (configurable via `data.dir`)
- Logs directory: `./logs/` (configurable via `log.path`)
- No external file storage services

**Persistent Storage (K8s):**
- PersistentVolumeClaim with 1Gi ReadWriteOnce storage
- Mounted at `/app/data`

## HTTP Integrations

**Oracle Data Fetching:**
- Component: `internal/infra/http/fetcher.go`
- Standard library `net/http` client
- Configurable timeout: 10 seconds
- Supports JSON path extraction from responses
- Used by oracle module for external data sources

**No external API integrations** - Aurora is self-contained

## Testing Infrastructure

**Test Types:**
- Unit tests: `internal/*/*_test.go`
- E2E tests: `e2e/*_test.go`
- Coverage reporting: `coverage.out`, `infra_coverage.out`

## Configuration Sources

**TOML Config File:**
- Default: `config/aurora.toml`
- Sections: `[server]`, `[log]`, `[db]`, `[lottery]`, `[i18n]`

**Environment Variables:**
- Supported via Viper's `AutomaticEnv()`
- LOG_LEVEL (via ConfigMap in K8s)
- Auto-discovered from config file

**Build-time Configuration:**
- Version info injected via ldflags
- Build metadata: Version, BuildTime

## Internationalization

**i18n System:**
- In-code translation map in `internal/i18n/i18n.go`
- Supported locales: English (en), Chinese (zh)
- Auto-detection via LANG environment variable
- Configurable via `i18n.locale` setting

## Build & Release

**GoReleaser:**
- GitHub releases: github.com/pplmx/aurora
- Docker Hub: pplmx/aurora
- Platforms: Linux (amd64, arm64), Darwin (amd64, arm64), Windows (amd64)

**Pre-commit Hooks:**
- Config: `.pre-commit-config.yaml`

## Dependencies Summary

**Direct External Dependencies:**
| Dependency | Purpose | External Service |
|------------|---------|------------------|
| filippo.io/edwards25519 | VRF cryptography | No |
| spf13/cobra | CLI framework | No |
| spf13/viper | Configuration | No |
| rs/zerolog | Logging | No |
| mattn/go-sqlite3 | Database | No |
| charmbracelet/* | TUI components | No |
| google/uuid | UUID generation | No |

**Infrastructure Dependencies:**
- Docker (optional)
- Kubernetes (optional, for production deployment)

**No external services required:**
- No cloud provider dependencies
- No third-party API calls (except oracle HTTP fetches which are user-defined)
- No external authentication services

---

*Integration audit: 2026-04-26*
