package sqlite

import "github.com/pplmx/aurora/internal/domain/repoerr"

// ErrNotFound is kept as an alias for existing SQLite tests and callers; the
// sentinel itself lives in domain/repoerr so application code does not depend
// on this package.
var ErrNotFound = repoerr.ErrNotFound
