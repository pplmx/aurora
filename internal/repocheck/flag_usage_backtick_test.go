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

// TestPagingFlagsReadAModuleNeutralLabel covers a wording leak. Seven list
// commands share the same --limit/--offset flags, and every one of them read
// the token catalog's entry: `aurora lottery history --help` said "Maximum
// tokens to list", `aurora oracle source list --help` said it too, and a reader
// checking how many sources they could list was told about tokens.
//
// The token command is the one place that wording is right, so it keeps its own
// key and the other six read the neutral one.
func TestPagingFlagsReadAModuleNeutralLabel(t *testing.T) {
	registered := map[string]string{}
	require.NoError(t, walkGoSource(filepath.Join(repoRoot(t), "cmd", "aurora", "cmd"), func(path, body string) {
		for _, m := range pagingFlagKey.FindAllStringSubmatch(body, -1) {
			registered[m[1]] = m[2]
		}
	}))
	require.GreaterOrEqual(t, len(registered), 7, "the scan must reach the paging flags: %v", registered)

	for variable, key := range registered {
		want := "common.list_limit"
		if strings.Contains(strings.ToLower(variable), "token") {
			// "Maximum tokens to list" is correct for the token list.
			want = "token.list_limit"
		}
		assert.Equalf(t, want, key,
			"%s reads %s for its --limit; a token-specific label in another module's help names the wrong noun",
			variable, key)
	}
}

// pagingFlagKey matches a --limit registration: the receiver names the command,
// and the usage is a catalog lookup.
var pagingFlagKey = regexp.MustCompile(`(\w+)\.Flags\(\)\.IntP\("limit",\s*"l",\s*0,\s*i18n\.GetText\("([a-z_.]+)"\)\)`)
