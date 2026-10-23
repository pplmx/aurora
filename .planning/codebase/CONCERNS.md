# Codebase Concerns

**Analysis Date:** 2026-10-23

## Resolved Assumptions

**Backup restore is implemented:**

- `BackupService.Restore` verifies checksums and metadata before replacing a
  live database, snapshots the previous database under `.pre_restore`, copies
  through a temporary sibling, and removes stale `-wal`/`-shm` files.
- Regression coverage includes round-trip restore, corrupt archives, path
  traversal, alias/self-restore rejection, WAL completeness, and temp cleanup.

## Test Coverage Gaps

**Current lower-coverage areas (2026-10-23):**

- `internal/app/lottery`: 50.8%; the verify path is covered mainly through CLI
  and desktop integration rather than direct unit tests.
- `internal/utils`: 69.6%; shared helpers have branch gaps.
- `internal/infra/backup`: 78.4%; failure-path coverage remains the main gap.
- `internal/infra/sqlite`: 80.4%; repository-specific error branches remain.

`internal/api`, `internal/api/handler`, every TUI package, and
`internal/domain/blockchain` now have substantial direct coverage. The README
coverage table is the current single source for published numbers.

## Security Considerations

**API key defaults are guarded:**

- `aurora-api-key-default` is rejected in production rather than accepted as a
  working default. A missing production key fails with `ErrMissingAPIKey`;
  development mode generates a random 32-byte key.
- `AURORA_API_KEY` is bound explicitly, and the Web UI receives the resolved
  key only in same-origin HTML under a nonce-based CSP.

**Backup file permissions:**

- Backup metadata and restored database files are created with `0640`.
  Directories remain `0755` so the owner can traverse and share them; operators
  who need stricter directory privacy should choose the backup parent
  accordingly.

**API error redaction:**

- Unclassified 500 responses return a generic `internal server error`; only
  known domain errors echo a caller-actionable message.
- The special `AUDIT_PUBLISH_FAILED` response deliberately says the operation
  committed and must not be retried, without exposing the underlying store error.

## Performance Bottlenecks

**Large Token Service:**

- File: `internal/domain/token/service.go` (813 lines)
- Issue: Single monolithic service with 10+ methods, each 50-100+ lines
- Risk: Harder to optimize independently, larger blast radius on changes
- Recommendation: Consider extracting replay protection, event publishing as separate components

**Large Test Files:**

- `internal/domain/token/service_test.go`: 3,555 lines
- `internal/infra/http/fetcher_test.go`: 1,189 lines
- `internal/app/token/usecase_test.go`: 1,142 lines
- Issue: These files are difficult to navigate, may indicate over-testing or complex setup
- Risk: Slower test runs, harder to maintain
- Recommendation: Consider splitting into multiple test files by method/feature

**Large i18n File:**

- File: `internal/i18n/i18n.go` (1,152 lines)
- Issue: Single large file with all translations
- Risk: Difficult to manage translations, potential merge conflicts
- Recommendation: Consider splitting by feature/module

## Maintainability Issues

**Magic Numbers in Token Service:**

- File: `internal/domain/token/service.go:17`
- Issue: `defaultHistoryLimit = 50` defined but could be configurable
- Recommendation: Consider making configurable via config

## Anti-Patterns

**Silent Error Ignoring:**

- No silent production error paths were found in the audited oracle fetch and
  backup create/verify flows: marshal, chain recording, and persistence errors
  are wrapped and returned.
- Remaining `_ =` uses are cleanup/close paths or intentional best-effort
  removals, where replacing the primary error with a cleanup error would hide
  the actionable failure.

## Concurrency Concerns

**Appropriate Use of Mutexes:**

- Rate limiter (`internal/infra/http/fetcher.go`): Uses `sync.RWMutex` correctly
- Event bus (`internal/infra/events/bus.go`): Uses `sync.RWMutex` correctly
- In-memory NFT repo (`internal/domain/nft/inmem_repo.go`): Uses `sync.RWMutex` correctly
- No issues detected

**Race Detector:**

- CI runs both the core and desktop modules under `-race`.
- The race detector requires cgo even though production builds and the default
  test gates are cgo-free. `just desktop-race` is the explicit local opt-in.

## Code Organization

**Missing Documentation:**

- No godoc comments on exported functions in `internal/domain/token/service.go`
- Recommendation: Add package-level documentation explaining the ERC-20-like token system

## Infrastructure Gaps

**Observability:**

- `/healthz`, `/readyz`, `/health`, `/metrics`, and `/metrics/oracle` are
  implemented and covered by regression tests.
- Remaining opportunity: OpenTelemetry tracing and richer request correlation;
  the stdlib Prometheus-text registry is intentionally dependency-light.

---

*Concerns audit: 2026-10-23*
