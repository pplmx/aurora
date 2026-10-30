package handler

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// documentedCodeRow matches one row of the error-code table in docs/api.md:
// "| 代币 | `TOKEN_NOT_FOUND` | 404 |" (the formatter pads the cells).
var documentedCodeRow = regexp.MustCompile(`\|\s+[^|]+\s+\|\s+` + "`([A-Z0-9_]+)`" + `\s+\|\s+(\d{3})\s+\|`)

// TestEveryClientErrorCodeIsDocumented keeps the reference complete and its
// status column honest: clients switch on these codes and their HTTP status, so
// a new sentinel (or a reclassified one) must reach docs/api.md, not only
// helpers.go.
func TestEveryClientErrorCodeIsDocumented(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "the test must know where it lives")
	data, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "..", "docs", "api.md"))
	require.NoError(t, err)

	documented := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		if m := documentedCodeRow.FindStringSubmatch(line); m != nil {
			documented[m[1]] = m[2]
		}
	}
	require.NotEmpty(t, documented, "docs/api.md must carry the error-code table")

	for _, entry := range errorClassification {
		status := strconv.Itoa(entry.statusCode)
		documentedStatus, ok := documented[entry.code]
		require.Truef(t, ok, "%s (%s) is missing from the docs/api.md error table", entry.code, status)
		require.Equalf(t, status, documentedStatus,
			"%s is documented as HTTP %s but classified as HTTP %s", entry.code, documentedStatus, status)
	}
}
