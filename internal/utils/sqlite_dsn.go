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
// Explicit ":memory:" and "file:" DSNs are already URI-style inputs and are
// preserved. Callers that pass "file:" own its escaping and existing query.
func SQLiteDSN(dbPath, query string) string {
	if query == "" {
		return dbPath
	}
	if dbPath == ":memory:" {
		return dbPath + "?" + query
	}
	if strings.HasPrefix(dbPath, "file:") {
		separator := "?"
		if strings.Contains(dbPath, "?") {
			separator = "&"
		}
		return dbPath + separator + query
	}

	path := filepath.ToSlash(dbPath)
	if runtime.GOOS != "windows" && filepath.IsAbs(dbPath) && strings.HasPrefix(path, "//") {
		// In a SQLite file URI, a second leading slash starts an authority.
		// POSIX treats repeated leading slashes as the same root, so collapse
		// them instead of producing file://host-style names.
		path = "/" + strings.TrimLeft(path, "/")
	}
	if runtime.GOOS == "windows" && filepath.IsAbs(dbPath) && !strings.HasPrefix(path, "//") {
		// A Windows drive path needs the slash before C: in a file URI.
		path = "/" + path
	}
	escaped := (&url.URL{Path: path}).EscapedPath()
	return "file:" + escaped + "?" + query
}
