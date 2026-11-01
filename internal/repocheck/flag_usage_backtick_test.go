package repocheck

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFlagUsageHasNoBackticks covers a pflag rule that is invisible in the
// source: a pair of backticks inside a flag's usage string names the value type
// shown in --help. `aurora oracle source add --help` printed
//
//	-i, --interval source add   Refresh interval in seconds …
//
// because the usage contained "`source add`". The flag is an int, and the words
// between the backticks replaced the type in the help screen.
//
// Only flag usage strings are affected: backticks in a command's Long text are
// printed verbatim, which is why the map below is narrowed to the catalog keys
// a flag registration reads.
func TestFlagUsageHasNoBackticks(t *testing.T) {
	// Every catalog key passed to a flag setter, with the setter that reads it.
	checked := 0
	require.NoError(t, walkGoSource(filepath.Join(repoRoot(t), "cmd", "aurora", "cmd"), func(path, body string) {
		for _, m := range flagUsageKey.FindAllStringSubmatch(body, -1) {
			key := m[1]
			checked++
			entry := englishCatalogValue(t, key)
			assert.NotContainsf(t, entry, "`",
				"%s registers %s with a usage string containing backticks; pflag reads "+
					"the text between them as the value type, so --help shows %q instead of the type",
				path, key, betweenBackticks(entry))
		}
	}))
	require.GreaterOrEqual(t, checked, 20, "the scan must reach the flag registrations")
}

// flagUsageKey matches a flag registration whose usage is a catalog lookup:
// Flags().IntP("interval", "i", 60, i18n.GetText("oracle.source_interval")).
var flagUsageKey = regexp.MustCompile(`Flags\(\)\.\w+\([^)]*i18n\.GetText\("([a-z_.]+)"\)`)

// englishCatalogValue returns the English text of one catalog key.
func englishCatalogValue(t *testing.T, key string) string {
	t.Helper()
	pattern := regexp.MustCompile(`(?m)^\t"` + regexp.QuoteMeta(key) + `":\s+("(?:[^"\\]|\\.)*"),$`)
	m := pattern.FindStringSubmatch(readRepoFile(t, "internal", "i18n", "messages_en.go"))
	require.NotNilf(t, m, "messages_en.go has no entry for %s", key)
	return strings.Trim(m[1], `"`)
}

// betweenBackticks is the text pflag would show as the value type, for the
// failure message.
func betweenBackticks(s string) string {
	if i := strings.Index(s, "`"); i >= 0 {
		if j := strings.Index(s[i+1:], "`"); j >= 0 {
			return s[i+1 : i+1+j]
		}
	}
	return ""
}
