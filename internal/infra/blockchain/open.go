package blockchain

import (
	"database/sql"

	_ "modernc.org/sqlite"

	"github.com/pplmx/aurora/internal/utils"
)

// OpenDB opens an independent handle to the ledger database. Unlike InitDB it
// does not publish a process-wide singleton, so a composition root can own the
// handle and close it without affecting a CLI or test that still uses InitDB.
func OpenDB(path string) (*sql.DB, error) {
	if err := utils.EnsureSQLiteDirectory(path); err != nil {
		return nil, err
	}
	// _txlock=immediate + _busy_timeout: the shared DB backs voting writes
	// through a TxManager over a real pool, so a deferred read-then-write
	// transaction racing a concurrent writer hits the un-waitable
	// SQLITE_BUSY_SNAPSHOT — serialize writers at BEGIN and wait up to 5s for
	// the lock instead (v1.70, ISS-076; same values as infra/sqlite/dsn.go).
	db, err := sql.Open("sqlite", utils.SQLiteDSN(path, "_foreign_keys=ON&_txlock=immediate&_busy_timeout=5000"))
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := utils.RestrictSQLiteFiles(path); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}
