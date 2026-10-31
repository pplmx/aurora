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

// SQLiteDiskUsage sums the database file and the sidecars that hold committed
// pages (-wal/-shm) or a rollback journal, so a reported size matches what
// `du` shows for the same files. Missing files contribute nothing: a database
// that was never written reports 0 rather than failing.
func SQLiteDiskUsage(dbPath string) int64 {
	var size int64
	for _, file := range append([]string{dbPath}, SQLiteSidecarPaths(dbPath)...) {
		if info, err := os.Stat(file); err == nil {
			size += info.Size()
		}
	}
	return size
}

// SQLiteSidecarPaths returns every file SQLite can create beside dbPath.
// Callers that protect or remove one generation of database state must handle
// the complete set; omitting the rollback journal leaves the previous
// generation's pages available for replay.
func SQLiteSidecarPaths(dbPath string) []string {
	return []string{dbPath + "-wal", dbPath + "-shm", dbPath + "-journal"}
}

// SQLiteFilePath returns the filesystem path named by a database location as
// interpreted by SQLiteDSN. Plain paths and explicit file: URIs are both
// accepted. The ephemeral result is true for SQLite's memory and empty-name
// temporary databases, which have no on-disk state to create or protect.
//
// SQLite treats file: values as URIs, including percent escapes and an
// optional file://localhost/ authority. Callers that perform filesystem work
// on a configured database must use this helper rather than path/filepath on
// the raw DSN.
func SQLiteFilePath(dsn string) (path string, ephemeral bool, err error) {
	if dsn == "" || dsn == ":memory:" {
		return "", true, nil
	}
	if !strings.HasPrefix(dsn, "file:") {
		// SQLiteDSN escapes '?' in plain paths before handing them to the
		// driver, so the entire plain value is a filesystem path.
		return dsn, false, nil
	}

	raw := strings.TrimPrefix(dsn, "file:")
	if strings.HasPrefix(raw, "//") {
		authorityAndPath := raw[2:]
		slash := strings.IndexByte(authorityAndPath, '/')
		authority := authorityAndPath
		if slash >= 0 {
			authority = authorityAndPath[:slash]
			raw = authorityAndPath[slash:]
		} else {
			raw = ""
		}
		// SQLite rejects every authority except its exact, case-sensitive
		// localhost spelling. In particular, treating another host as a Unix
		// path would chmod an unrelated local file.
		if authority != "" && authority != "localhost" {
			return "", false, fmt.Errorf("SQLite file URI has unsupported authority %q", authority)
		}
	}

	rawPath, rawQuery := splitSQLiteURI(raw)
	decodedPath, err := decodeSQLiteURI(rawPath)
	if err != nil {
		return "", false, fmt.Errorf("decode SQLite file URI path: %w", err)
	}
	path = sqliteURIPath(decodedPath)
	mode := lastSQLiteURIParameter(rawQuery, "mode")
	if path == "" || path == ":memory:" || mode == "memory" {
		return "", true, nil
	}
	return path, false, nil
}

// EnsureSQLiteDirectory creates the parent directory needed by a disk-backed
// SQLite location. Memory and empty-name temporary databases need no directory.
func EnsureSQLiteDirectory(dsn string) error {
	path, ephemeral, err := SQLiteFilePath(dsn)
	if err != nil {
		return err
	}
	if ephemeral {
		return nil
	}
	// A directory where the database file belongs is a configuration mistake, not
	// a database failure: without this check SQLite answers the first write with
	// "unable to open database file (14)", which names neither the path nor the
	// reason.
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return fmt.Errorf("database path %s is a directory", path)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	return nil
}

func splitSQLiteURI(raw string) (path, query string) {
	pathEnd := strings.IndexAny(raw, "?#")
	if pathEnd < 0 {
		return raw, ""
	}
	path = raw[:pathEnd]
	if raw[pathEnd] != '?' {
		return path, ""
	}
	query = raw[pathEnd+1:]
	if fragment := strings.IndexByte(query, '#'); fragment >= 0 {
		query = query[:fragment]
	}
	return path, query
}

func decodeSQLiteURI(raw string) (string, error) {
	var decoded strings.Builder
	decoded.Grow(len(raw))
	for i := 0; i < len(raw); i++ {
		if raw[i] != '%' || i+2 >= len(raw) || !isHex(raw[i+1]) || !isHex(raw[i+2]) {
			decoded.WriteByte(raw[i])
			continue
		}
		value := hexValue(raw[i+1])<<4 | hexValue(raw[i+2])
		if value == 0 {
			return "", errors.New("NUL escapes are not valid in a filesystem path")
		}
		decoded.WriteByte(value)
		i += 2
	}
	return decoded.String(), nil
}

func lastSQLiteURIParameter(rawQuery, key string) string {
	value := ""
	for _, parameter := range strings.Split(rawQuery, "&") {
		rawName, rawValue, ok := strings.Cut(parameter, "=")
		if !ok {
			continue
		}
		name, err := decodeSQLiteURI(rawName)
		if err != nil || name != key {
			continue
		}
		if decoded, err := decodeSQLiteURI(rawValue); err == nil {
			value = decoded
		}
	}
	return value
}

func sqliteURIPath(path string) string {
	native := filepath.FromSlash(path)
	if runtime.GOOS == "windows" &&
		len(native) >= 3 &&
		os.IsPathSeparator(native[0]) &&
		isASCIILetter(native[1]) &&
		native[2] == ':' {
		// file:///C:/data/aurora.db names C:\data\aurora.db on Windows.
		native = native[1:]
	}
	return native
}

func isHex(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
}

func hexValue(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	default:
		return c - 'A' + 10
	}
}

func isASCIILetter(c byte) bool {
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}

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
		dbPath == "" {
		return nil
	}

	path, ephemeral, err := SQLiteFilePath(dbPath)
	if err != nil {
		return fmt.Errorf("resolve SQLite database path: %w", err)
	}
	if ephemeral {
		return nil
	}

	// SQLite resolves a database symlink before naming its WAL/SHM/journal
	// sidecars. Chmod through the link would secure the main file but leave
	// the real sidecars at the process umask.
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", path, err)
	}

	paths := append([]string{resolvedPath}, SQLiteSidecarPaths(resolvedPath)...)
	for i, path := range paths {
		if err := os.Chmod(path, 0o600); err != nil {
			if i > 0 && errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return fmt.Errorf("restrict %s to owner: %w", path, err)
		}
	}
	return nil
}
