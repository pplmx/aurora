package sqlite

import "fmt"

// writeDSNSuffix hardens every repository's SQLite DSN for workload run over
// the service's real connection pool (production wiring sets no
// SetMaxOpenConns(1)) so concurrent writers no longer fail with SQLITE_BUSY:
//
//   - _busy_timeout=5000 — wait up to 5s for another connection's write lock
//     instead of failing immediately. Same value as the replay-protection DSN
//     from the v1.67 fix (ISS-075).
//   - _txlock=immediate — every db.Begin() on this DB executes BEGIN IMMEDIATE,
//     acquiring the write lock at BEGIN. A deferred read-then-write transaction
//     racing a concurrent writer is precisely what surfaces the fatal, un-waitable
//     SQLITE_BUSY_SNAPSHOT that _busy_timeout alone cannot mask; serializing
//     the writers at BEGIN removes that class entirely (v1.70, ISS-076).
//
// Verified: 16 concurrent read-then-write transactions over an unlimited pool
// failed 10/16 with "database is locked" before this suffix, and 16/16
// succeeded with it (regression: TestTokenRepository_WithTransaction_ConcurrentNoSQLiteBusy).
// Only apply to DSNs of databases whose connections perform writes.
const writeDSNSuffix = "?_foreign_keys=ON&_txlock=immediate&_busy_timeout=5000"

// dsn returns dbPath plus the hardened write suffix for repository DSNs.
func dsn(dbPath string) string {
	return fmt.Sprintf("%s%s", dbPath, writeDSNSuffix)
}
