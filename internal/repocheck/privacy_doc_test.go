package repocheck

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// privacyRow matches one row of the permission table in docs/privacy.md:
// "| `aurora.db` | `0600` | ... |". The path cell is taken up to its closing
// backtick, so a row naming several sidecars keys off the first one.
var privacyRow = regexp.MustCompile("^\\| `([^`]+)`[^|]*\\| ([^|]+?)\\s*\\|")

// TestPrivacyTableMatchesTheHardeningCode keeps the privacy statement's file
// list tied to the code that sets those modes. The statement is a promise to
// the operator, so a mode that drifts from the implementation is a factual
// error in the promise, not a stale comment.
func TestPrivacyTableMatchesTheHardeningCode(t *testing.T) {
	text := readRepoFile(t, "docs", "privacy.md")

	rows := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		if m := privacyRow.FindStringSubmatch(line); m != nil {
			rows[m[1]] = strings.TrimSpace(m[2])
		}
	}
	require.NotEmpty(t, rows, "docs/privacy.md must carry the permission table")

	// The database and its sidecars are hardened by RestrictSQLiteFiles, which
	// chmods the main file and every sidecar to 0600.
	sqliteHardening := readRepoFile(t, "internal", "utils", "sqlitefile.go")
	assertMode(t, rows, "aurora.db", 0o600, sqliteHardening)
	assertMode(t, rows, "aurora.db-wal", 0o600, sqliteHardening)

	// The data directory is created 0700 where the database is opened.
	dirCreated := regexp.MustCompile(`os\.MkdirAll\(dir, (0o[0-7]{3})\)`)
	match := dirCreated.FindStringSubmatch(sqliteHardening)
	require.NotNil(t, match, "sqlitefile.go must create its directory with an explicit mode")
	assertMode(t, rows, "data directory", parseMode(t, match[1]), sqliteHardening)

	// A new backup set is a 0700 directory; restore keeps a 0700 undo snapshot.
	create := readRepoFile(t, "internal", "infra", "backup", "create.go")
	assertMode(t, rows, "backup directory (new)", firstMode(t, create, "mkdirAllDurable"), create)
	restore := readRepoFile(t, "internal", "infra", "backup", "restore.go")
	assertMode(t, rows, "restore undo snapshot", firstMode(t, restore, "mkdirAllDurable"), restore)

	// Aurora does not choose the toolkit's window-state mode, and the document
	// says so rather than claiming a number it does not set.
	require.Contains(t, rows["window-state.json"], "MyGo",
		"the window-state row must credit the toolkit, since Aurora does not set that mode")
}

// assertMode checks that the table row exists and names want as an octal mode.
func assertMode(t *testing.T, rows map[string]string, path string, want uint64, from string) {
	t.Helper()
	cell, ok := rows[path]
	require.Truef(t, ok, "docs/privacy.md has no row for %q", path)
	wantOctal := "0" + strconv.FormatUint(want, 8)
	assert.Truef(t, strings.Contains(cell, wantOctal),
		"docs/privacy.md says %q for %q, but the code uses %s", cell, path, wantOctal)
}

// firstMode returns the first explicit mode passed to the named helper.
func firstMode(t *testing.T, source, helper string) uint64 {
	t.Helper()
	pattern := regexp.MustCompile(helper + `\([^)]*?(0o[0-7]{3})`)
	match := pattern.FindStringSubmatch(source)
	require.NotNilf(t, match, "%s must take an explicit mode", helper)
	return parseMode(t, match[1])
}

func parseMode(t *testing.T, literal string) uint64 {
	t.Helper()
	value, err := strconv.ParseUint(strings.TrimPrefix(literal, "0o"), 8, 32)
	require.NoErrorf(t, err, "mode literal %q", literal)
	return value
}
