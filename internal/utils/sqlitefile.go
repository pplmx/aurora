package utils

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// RestrictSQLiteFiles keeps the live SQLite database and its WAL/SHM/journal
// sidecars owner-only. SQLite creates files with the process umask (commonly
// 0644 even inside a 0755 data directory), while the Oracle table can contain
// bearer headers and the other tables contain operator state. The desktop shell
// and backup path already enforce this boundary; every CLI/API constructor must
// enforce it too.
//
// The main database must exist. Sidecars are optional because journal_mode=WAL
// may not have created them yet.
func RestrictSQLiteFiles(dbPath string) error {
	if runtime.GOOS == "windows" ||
		dbPath == "" ||
		dbPath == ":memory:" ||
		strings.HasPrefix(dbPath, "file:") {
		return nil
	}

	// SQLite resolves a database symlink before naming its WAL/SHM/journal
	// sidecars. Chmod through the link would secure the main file but leave
	// the real sidecars at the process umask.
	resolvedPath, err := filepath.EvalSymlinks(dbPath)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", dbPath, err)
	}

	for i, path := range []string{resolvedPath, resolvedPath + "-wal", resolvedPath + "-shm", resolvedPath + "-journal"} {
		if err := os.Chmod(path, 0o600); err != nil {
			if i > 0 && errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return fmt.Errorf("restrict %s to owner: %w", path, err)
		}
	}
	return nil
}
