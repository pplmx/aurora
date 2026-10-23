package utils

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
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

	for i, path := range []string{dbPath, dbPath + "-wal", dbPath + "-shm", dbPath + "-journal"} {
		if err := os.Chmod(path, 0o600); err != nil {
			if i > 0 && errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return fmt.Errorf("restrict %s to owner: %w", path, err)
		}
	}
	return nil
}
