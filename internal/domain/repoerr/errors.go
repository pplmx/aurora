// Package repoerr holds repository-level errors shared by application code and
// storage implementations. Keeping them outside infra lets a use case classify
// a missing row or a lost race without importing SQLite.
package repoerr

import "errors"

var (
	// ErrNotFound means a repository lookup did not find the requested row.
	ErrNotFound = errors.New("record not found")
)
