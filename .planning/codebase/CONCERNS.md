# Codebase Concerns

**Analysis Date:** 2026-04-26

## Tech Debt

### VRF Implementation Quality
- **Issue:** The VRF implementation in `internal/domain/lottery/vrf.go` uses SHA-256 hashing for `hashToPoint()` which doesn't guarantee a valid curve point. The standard ECVRF uses SHA-512 and elliptic curve operations differently.
- **Files:** `internal/domain/lottery/vrf.go:39-49`
- **Impact:** VRF output may not be cryptographically optimal for random selection
- **Fix approach:** Consider implementing RFC 9380 (Internet-Draft draft-irtf-cfrg-vrf-09) or using a library that implements it properly
- **Workaround:** Current implementation is documented in code comments. For lottery use case where strict cryptographic properties are less critical than for security-sensitive applications, this simplified approach is acceptable. See `vrf.go` for detailed limitations and RFC 9380 references.

### Oracle Service Returns Mock Data
- **Issue:** The `FetchData` implementation in `internal/domain/oracle/service.go:66-79` returns hardcoded "sample-value" instead of actual fetched data.
- **Files:** `internal/domain/oracle/service.go:70-77`
- **Impact:** Oracle data fetching is non-functional
- **Fix approach:** Wire up the HTTP fetcher (`internal/infra/http/fetcher.go`) properly to the oracle service

### Missing Transaction Boundaries
- **Issue:** Token transfer operations update multiple tables (balances, approvals, events) without transactional protection. A failure mid-operation could leave inconsistent state.
- **Files:** `internal/domain/token/service.go:224-289`, `internal/domain/token/service.go:292-378`
- **Impact:** Partial transfers possible during errors
- **Fix approach:** Wrap balance updates and event publication in database transactions

### SQLite Concurrency Limits
- **Issue:** SQLite uses file-level locking which limits write concurrency. Multiple concurrent token operations will serialize.
- **Files:** `internal/infra/sqlite/*.go`
- **Impact:** Poor performance under concurrent load
- **Fix approach:** Consider connection pooling or switching to PostgreSQL for production

### No Connection Pooling
- **Issue:** Each repository creates its own database connection without pooling.
- **Files:** `internal/infra/sqlite/token.go:26`, `internal/infra/sqlite/nft.go:25`, etc.
- **Impact:** Resource inefficiency, potential connection exhaustion
- **Fix approach:** Implement a shared connection pool

## Known Bugs

### VRF Output Validation Edge Case
- **Symptoms:** `VRFVerify` in `internal/domain/lottery/vrf.go:68-85` returns `true` if length check passes even when the proof is invalid
- **Files:** `internal/domain/lottery/vrf.go:84`
- **Trigger:** Pass incorrect proof bytes of correct length
- **Workaround:** Ensure proof generation uses the same `VRFProve` function

### Lottery ID Collisions
- **Issue:** Lottery ID is derived from first 16 characters of SHA-256 hash of seed, which has collision risk with enough participants
- **Files:** `internal/domain/lottery/entity.go:114-115`
- **Trigger:** Create lotteries with similar seeds
- **Workaround:** None

## Security Considerations

### Private Keys in Memory
- **Risk:** Private keys passed as `[]byte` in memory during signing operations could be observed by memory scraping attacks
- **Files:** `internal/domain/token/service.go:85`, `internal/domain/nft/service.go:16`
- **Current mitigation:** Standard Go runtime memory management
- **Recommendations:** Consider using hardware security modules or key derivation for production

### Signature Replay (Token Transfers)
- **Risk:** Token transfer signatures include a nonce but the signature itself could be replayed if nonce tracking is lost
- **Files:** `internal/domain/token/service.go:260-262`
- **Current mitigation:** Replay protection via `infraevents.ReplayProtection`
- **Recommendations:** Ensure replay protection persists across restarts

### Voting Session Time Boundaries Not Enforced
- **Risk:** Votes can be cast outside of active session windows because session timing is not checked in the vote execution
- **Files:** `internal/app/voting/usecase.go:25-78`
- **Current mitigation:** None
- **Recommendations:** Add session status validation before accepting votes

### No Rate Limiting
- **Risk:** External APIs (oracle fetching) have no rate limiting
- **Files:** `internal/infra/http/fetcher.go`
- **Current mitigation:** None
- **Recommendations:** Add rate limiting for external HTTP calls

### HTTP Fetcher Missing Security Headers
- **Risk:** HTTP client doesn't set user-agent, origin, or other security headers; relies on defaults
- **Files:** `internal/infra/http/fetcher.go:42-45`
- **Current mitigation:** 10-second timeout
- **Recommendations:** Add security headers, consider TLS certificate validation options

### SQL Parameterization (Good Pattern)
- **Positive:** All SQL queries use parameterized statements, no SQL injection risk observed
- **Files:** `internal/infra/sqlite/*.go`

## Performance Bottlenecks

### Proof-of-Work Mining
- **Problem:** Mining new blocks uses CPU-intensive SHA-256 hashing in a loop
- **Files:** `internal/domain/blockchain/proof.go:50-69`
- **Cause:** `math.MaxInt64` iterations potentially needed per block
- **Improvement path:** Make mining interruptible, add difficulty adjustment

### Large Event History Loading
- **Problem:** `GetTransferHistory` loads all events then truncates
- **Files:** `internal/domain/token/service.go:506-519`
- **Cause:** No SQL LIMIT applied before loading
- **Improvement path:** Use SQL pagination with LIMIT/OFFSET

### In-Memory Blockchain
- **Problem:** Entire blockchain kept in memory during operation
- **Files:** `internal/domain/blockchain/block.go:24-26`
- **Cause:** `BlockChain.Blocks` grows unboundedly
- **Improvement path:** Implement pruning or use LevelDB/IPFS for storage

## Fragile Areas

### Global i18n Translator Singleton
- **Files:** `internal/i18n/i18n.go:17`
- **Why fragile:** Global mutable state makes testing difficult and can cause race conditions
- **Safe modification:** Use dependency injection instead
- **Test coverage:** Limited due to singleton pattern

### Gob Encoding for Blockchain Serialization
- **Files:** `internal/domain/blockchain/block.go:69-92`
- **Why fragile:** Gob encoding is Go-specific and not stable across versions
- **Safe modification:** Use JSON or Protocol Buffers for cross-version compatibility
- **Test coverage:** Unit tests exist

### Hardcoded Default Timeout
- **Files:** `internal/infra/http/fetcher.go:15`
- **Why fragile:** 10-second timeout hardcoded, not configurable
- **Safe modification:** Make timeout configurable via config file

## Scaling Limits

### Single SQLite Database
- **Current capacity:** Suitable for single-node development/demo
- **Limit:** No horizontal scaling, write throughput limited by SQLite
- **Scaling path:** Migrate to PostgreSQL, add read replicas

### In-Memory Event Bus
- **Current capacity:** Suitable for single instance
- **Limit:** Events don't persist across restarts
- **Scaling path:** Use message queue (Kafka, RabbitMQ) for distributed deployments

## Dependencies at Risk

### mattn/go-sqlite3
- **Risk:** CGO dependency requiring C compiler toolchain
- **Impact:** Cross-compilation more complex, larger Docker images
- **Migration plan:** Consider modernc.org/sqlite (pure Go) for easier deployment

### google/uuid
- **Risk:** Low risk, stable library
- **Impact:** Used for ID generation
- **Migration plan:** None needed

### charmbracelet/bubbletea
- **Risk:** TUI framework, not suitable for web deployments
- **Impact:** Limits CLI-only use case
- **Migration plan:** None needed (CLI project)

## Missing Critical Features

### No Backup/Restore
- **Problem:** No mechanism to backup or restore database state
- **Blocks:** Production deployment without disaster recovery

### No Data Migration System
- **Problem:** Schema changes require manual migration
- **Blocks:** Safe upgrades in production

### No Metrics/Observability
- **Problem:** No Prometheus metrics, OpenTelemetry tracing, or structured health checks
- **Blocks:** Production monitoring requirements

## Test Coverage Gaps

### Untested Crypto Operations
- **What's not tested:** Ed25519 signature edge cases, VRF verification edge cases
- **Files:** `internal/domain/lottery/vrf.go`, `internal/domain/voting/service.go`
- **Risk:** Cryptographic bugs could go unnoticed
- **Priority:** High

### Untested Concurrency Scenarios
- **What's not tested:** Concurrent transfer operations, race conditions in event bus
- **Files:** `internal/domain/token/service.go`, `internal/infra/events/bus.go`
- **Risk:** Race conditions in production
- **Priority:** Medium

### No Integration Tests for External APIs
- **What's not tested:** Oracle HTTP fetching, real HTTP responses
- **Files:** `internal/infra/http/fetcher.go`
- **Risk:** Oracle integration could break silently
- **Priority:** Medium

### No E2E Tests for Voting Session Lifecycle
- **What's not tested:** Create session → start → vote → end → results
- **Files:** `internal/app/voting/usecase.go`
- **Risk:** Session workflow could break unnoticed
- **Priority:** Medium

---

*Concerns audit: 2026-04-26*