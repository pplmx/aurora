package blockchain

import (
	"database/sql"

	"github.com/pplmx/aurora/internal/infra/store"
)

// OpenDB opens an independent handle to the ledger database. Opening lives in
// internal/infra/store so the pragmas, the directory creation and the file
// permissions are the same wherever a handle is opened.
func OpenDB(path string) (*sql.DB, error) { return store.Open(path) }
