package utils

import (
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
)

// SQLiteDSN builds a driver DSN without treating characters in dbPath as
// connection-parameter syntax. The modernc driver splits a plain filename at
// the first '?', so paths containing that character must be sent as an escaped
// SQLite file URI instead.
//
// Explicit ":memory:" and "file:" DSNs are already URI-style inputs and their
// existing query is preserved. Callers that pass "file:" own its escaping.
// A URI fragment is dropped when parameters are added: SQLite ignores it, but
// the modernc driver folds it into the final query value and would reject the
// hardened parameters. On Windows, UNC paths need one extra slash before the
// path component so SQLite does not mistake the server name for a URI
// authority (the supported authority is empty or exactly "localhost").
func SQLiteDSN(dbPath, query string) string {
	if query == "" {
		return dbPath
	}
	if dbPath == ":memory:" {
		return dbPath + "?" + query
	}
	if strings.HasPrefix(dbPath, "file:") {
		if index := strings.IndexByte(dbPath, '#'); index >= 0 {
			dbPath = dbPath[:index]
		}
		separator := "?"
		if strings.Contains(dbPath, "?") {
			separator = "&"
		}
		return dbPath + separator + query
	}

	return sqliteFileURIForOS(dbPath, runtime.GOOS) + "?" + query
}

// sqliteFileURIForOS converts a filesystem path into a SQLite file URI. It takes
// GOOS explicitly so the Windows-only UNC and drive-letter branches can be
// tested on every development platform.
func sqliteFileURIForOS(path, goos string) string {
	normalized := path
	if goos == "windows" {
		normalized = strings.ReplaceAll(normalized, `\`, "/")
	} else {
		normalized = filepath.ToSlash(normalized)
	}

	if goos == "windows" {
		if strings.HasPrefix(normalized, "//") {
			// "file:" plus an empty authority needs four slashes before a
			// UNC path: file:////server/share/db. Three slashes would make
			// SQLite treat the first path segment as a single-root path.
			return "file://" + (&url.URL{Path: normalized}).EscapedPath()
		}
		if isWindowsDrivePath(normalized) {
			// A drive-qualified path needs the slash before C: in a file URI.
			normalized = "/" + normalized
		}
	} else if filepath.IsAbs(path) && strings.HasPrefix(normalized, "//") {
		// In a SQLite file URI, a second leading slash starts an authority.
		// POSIX treats repeated leading slashes as the same root, so collapse
		// them instead of producing file://host-style names.
		normalized = "/" + strings.TrimLeft(normalized, "/")
	}

	return "file:" + (&url.URL{Path: normalized}).EscapedPath()
}

func isWindowsDrivePath(path string) bool {
	return len(path) >= 3 &&
		isASCIILetter(path[0]) &&
		path[1] == ':' &&
		(path[2] == '/' || path[2] == '\\')
}
