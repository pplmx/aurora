# Codebase Concerns

**Analysis Date:** 2026-04-30

## Deferred Items

**BCK-04: Backup Restore Not Implemented:**
- Issue: `Restore()` method in `internal/infra/backup/backup.go:89` returns error "restore not implemented - requires schema migration"
- Files: `internal/infra/backup/backup.go`
- Impact: Users cannot restore from backups, blocking disaster recovery
- Fix approach: Implement restore logic with schema version migration support for v1.2

## Test Coverage Gaps

**Critical - API Handlers:**
- API package: 3.6% coverage
- API handler package: 9.5% coverage
- Files: `internal/api/server.go`, `internal/api/router.go`, `internal/api/handler/*.go`
- Risk: API endpoints not validated, potential routing issues undiscovered
- Priority: High

**Critical - UI Packages (0% coverage):**
- `internal/ui/components`: 0.0%
- `internal/ui/lottery`: 0.0%
- `internal/ui/nft`: 0.0%
- `internal/ui/oracle`: 0.0%
- `internal/ui/token`: 0.0%
- Risk: TUI rendering bugs undetected, UI state issues undiscovered
- Priority: Medium

**Low Coverage Areas:**
- `internal/config`: 0.0% - Config loading not tested
- `internal/app`: 0.0% - App layer orchestration not tested
- `internal/infra/sqlite`: 49.1% - Database operations gaps
- `internal/domain/blockchain`: 30.9% - Mining/blockchain logic not fully tested
- `internal/domain/oracle`: 76.1% - Oracle domain has room for improvement

## Security Considerations

**Hardcoded Default API Key:**
- File: `internal/config/config.go:40`
- Issue: Default API key `"aurora-api-key-default"` is hardcoded
- Current mitigation: Viper allows override via config file/env
- Recommendation: Add runtime validation that default key was changed, or generate random default on first run

**File Permissions on Backups:**
- File: `internal/infra/backup/backup.go:54`
- Issue: Uses `0644` permissions which allows world-read on backup files containing sensitive data
- Recommendation: Use `0640` to restrict to owner/group only

**Missing Input Validation in API:**
- Files: `internal/api/handler/*.go`
- Issue: API handlers call use cases but error responses may expose internal details
- Risk: Potential information disclosure via error messages
- Recommendation: Audit all error responses for safe user-facing messages

## Performance Bottlenecks

**Large Token Service:**
- File: `internal/domain/token/service.go` (595 lines)
- Issue: Single monolithic service with 10+ methods, each 50-100+ lines
- Risk: Harder to optimize independently, larger blast radius on changes
- Recommendation: Consider extracting replay protection, event publishing as separate components

**Large Test Files:**
- `internal/domain/token/service_test.go`: 2,246 lines
- `internal/app/token/usecase_test.go`: 1,075 lines
- `internal/infra/http/fetcher_test.go`: 956 lines
- Issue: These files are difficult to navigate, may indicate over-testing or complex setup
- Risk: Slower test runs, harder to maintain
- Recommendation: Consider splitting into multiple test files by method/feature

**Large i18n File:**
- File: `internal/i18n/i18n.go` (705 lines)
- Issue: Single large file with all translations
- Risk: Difficult to manage translations, potential merge conflicts
- Recommendation: Consider splitting by feature/module

## Maintainability Issues

**Manual Byte Comparison:**
- File: `internal/domain/nft/inmem_repo.go:52-62`
- Issue: Uses manual loop for byte comparison instead of `bytes.Equal()`
```go
for i := range nft.Creator {
    if nft.Creator[i] != creator[i] {
        match = false
        break
    }
}
```
- Fix: Replace with `bytes.Equal(nft.Creator, creator)`

**Transaction Error Handling:**
- Files: `internal/domain/token/service.go` (lines 298-331, 394-434)
- Issue: Uses closure variable `transferErr` in addition to transaction error for error handling
- Risk: Could miss errors if pattern not followed consistently
- Recommendation: Standardize error handling pattern across all transaction operations

**Magic Numbers in Token Service:**
- File: `internal/domain/token/service.go:17`
- Issue: `defaultHistoryLimit = 50` defined but could be configurable
- Recommendation: Consider making configurable via config

## Anti-Patterns

**Unused Transaction Parameter:**
- File: `internal/domain/token/service.go:225`
- Issue: Transaction callback accepts `*sql.Tx` but never uses it
```go
err = s.txManager.WithTransaction(func(tx *sql.Tx) error {
    // tx parameter unused
```
- Recommendation: Change callback signature to `func() error` or document why tx is needed

**Silent Error Ignoring:**
- Files: Multiple locations
- Pattern: `_ = someFunction()` where errors are silently ignored
- Risk: Errors go unnoticed
- Examples:
  - `internal/app/oracle/usecase.go:61-62` - JSON marshal and chain AddLotteryRecord errors ignored
  - `internal/infra/backup/backup.go:53,79` - JSON marshal error ignored in Create/Verify

## Concurrency Concerns

**Appropriate Use of Mutexes:**
- Rate limiter (`internal/infra/http/fetcher.go`): Uses `sync.RWMutex` correctly
- Event bus (`internal/infra/events/bus.go`): Uses `sync.RWMutex` correctly
- In-memory NFT repo (`internal/domain/nft/inmem_repo.go`): Uses `sync.RWMutex` correctly
- No issues detected

**Missing Race Detector Testing:**
- Recommendation: Run `go test -race` regularly to catch data races
- Add to CI: Consider running race detector in CI pipeline

## Code Organization

**Large Modules Need Refactoring:**
- Token service at 595 lines is approaching the threshold where it should be split
- Voting use case needs similar attention if it grows

**Missing Documentation:**
- No godoc comments on exported functions in `internal/domain/token/service.go`
- Recommendation: Add package-level documentation explaining the ERC-20-like token system

## Infrastructure Gaps

**No Health Check Endpoint:**
- Missing `/health` or `/ready` endpoint for container orchestration
- Files: `internal/api/router.go`, `internal/api/handler/`
- Recommendation: Add health check for Kubernetes/load balancer integration

**No Metrics/Observability:**
- No Prometheus metrics, OpenTelemetry tracing, or structured logging correlation IDs
- Recommendation: Add observability for production deployment readiness

---

*Concerns audit: 2026-04-30*