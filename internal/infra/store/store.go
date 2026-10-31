// Package store owns the process-wide SQLite handle: where the database lives
// and how it is opened.
//
// The ledger, the repositories and both entry points (CLI/API/desktop via the
// composition root) ask for the same handle. Its location is whatever
// `internal/config` resolved into `db.path` (`--config`, the environment, the
// `$HOME`/`./config` search, then the built-in default). Keeping that bootstrap
// here rather than in a feature package means a package that wants a connection
// no longer has to own a database singleton to get one.
package store

import (
	"database/sql"
	"os"
	"sync"

	"github.com/spf13/viper"
	_ "modernc.org/sqlite"

	"github.com/pplmx/aurora/internal/utils"
)

// DefaultPath is where the database lives when nothing is configured.
const DefaultPath = "./data/aurora.db"

var (
	shared     *sql.DB
	sharedErr  error
	sharedOnce sync.Once
)

// Path returns the configured database path without touching the filesystem.
// Opening the database (or constructing a repository) owns directory creation
// and must surface its error; returning an empty path here would make SQLite
// silently open a temporary database and report writes that vanish at exit.
func Path() string {
	if p := viper.GetString("db.path"); p != "" {
		return p
	}
	return DefaultPath
}

// Open opens one handle at path with the pragmas every writer needs.
func Open(path string) (*sql.DB, error) {
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

// OpenShared returns the process-wide handle, opening it on the first call.
// Concurrent callers all observe the same pointer: reading the variable, calling
// sql.Open and assigning it without synchronization would let two callers both
// see nil, both open, and leak the first connection.
func OpenShared() (*sql.DB, error) {
	sharedOnce.Do(func() {
		shared, sharedErr = Open(Path())
	})
	return shared, sharedErr
}

// Close closes the shared handle, if one was opened.
func Close() error {
	if shared != nil {
		return shared.Close()
	}
	return nil
}

// ResetForTest closes and forgets the shared handle so the next OpenShared
// reopens at the then-current path. It also removes ./data, the directory the
// default path names, so a test that ran with defaults leaves nothing behind.
func ResetForTest() {
	if shared != nil {
		_ = shared.Close()
		shared = nil
	}
	sharedOnce = sync.Once{}
	sharedErr = nil
	_ = os.RemoveAll("./data")
}
