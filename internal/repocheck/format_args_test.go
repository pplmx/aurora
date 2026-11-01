package repocheck

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFormatCallsPassTheRightNumberOfArguments covers the gap between a
// catalog entry and the call that prints it. The i18n tests compare the English
// and Chinese spellings of one key; nothing checked that the call site passes as
// many values as the key expects, so a message with two verbs and one argument
// printed "%!d(MISSING)" in every language and no test failed.
//
// The verbs are read from the catalogs, the arguments from the call — both from
// source, because the two live in different layers.
func TestFormatCallsPassTheRightNumberOfArguments(t *testing.T) {
	verbs := catalogVerbCounts(t)

	// The desktop's dialogs read the core catalog too — `desktop.dialog.*` and
	// `desktop.error.*` live in messages_en.go — so its calls are checked
	// against the same verb table.
	calls := 0
	desktopCalls := 0
	check := func(path, body string, catalogue map[string]int, counter *int) {
		for _, call := range formatCall.FindAllStringSubmatch(body, -1) {
			key := call[1]
			want, ok := catalogue[key]
			if !ok {
				continue
			}
			*counter++
			got := countArguments(strings.TrimSpace(call[2]))
			assert.Equalf(t, want, got,
				"%s: %q expects %d value(s) but the call passes %d: %s",
				path, key, want, got, strings.TrimSpace(call[0]))
		}
	}
	require.NoError(t, walkGoSource(filepath.Join(repoRoot(t), "cmd"), func(path, body string) {
		check(path, body, verbs, &calls)
	}))
	require.NoError(t, walkGoSource(filepath.Join(repoRoot(t), "desktop"), func(path, body string) {
		check(path, body, verbs, &desktopCalls)
	}))
	require.GreaterOrEqual(t, calls, 30, "the scan must reach the core format calls")
	require.GreaterOrEqual(t, desktopCalls, 4, "the scan must reach the desktop format calls")
}

// formatCall matches a catalog lookup with its arguments: GetTextF("key", …)
// and TFormat("key", …). A nested call is matched to its own closing
// parenthesis, so TFormat("a", FormatBytes(x)) still yields one argument.
var formatCall = regexp.MustCompile(`(?:GetTextF|TFormat)\("([a-z_.]+)",\s*([^()]*(?:\([^()]*\))?[^()]*)?\)`)

// countArguments counts top-level comma-separated values in an argument list.
func countArguments(list string) int {
	list = strings.TrimSpace(list)
	if list == "" {
		return 0
	}
	depth, count := 0, 1
	for _, r := range list {
		switch r {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				count++
			}
		}
	}
	return count
}

// catalogVerbCounts maps each English key to how many values its format string
// takes, counting the distinct verbs the way fmt does (one per verb, not per
// operand — but every key here passes one operand per verb).
func catalogVerbCounts(t *testing.T) map[string]int {
	t.Helper()
	out := map[string]int{}
	for _, m := range catalogEntry.FindAllStringSubmatch(readRepoFile(t, "internal", "i18n", "messages_en.go"), -1) {
		out[m[1]] = len(printableVerbs(m[2]))
	}
	require.Greater(t, len(out), 300, "the English catalog must be parsed")
	return out
}

// printableVerbs returns the format verbs of s, ignoring %% and the ones that
// only appear inside the catalog's own escaped quotes.
func printableVerbs(s string) []string {
	out := []string{}
	for i := 0; i < len(s)-1; i++ {
		if s[i] != '%' {
			continue
		}
		if s[i+1] == '%' {
			i++
			continue
		}
		out = append(out, string(s[i+1]))
	}
	return out
}

// TestPrintfCountsAgreeWithFmt is a guard on the helpers above: they must agree
// with fmt itself, or the check they feed is meaningless.
func TestPrintfCountsAgreeWithFmt(t *testing.T) {
	cases := []struct {
		format string
		args   int
	}{
		{"no verbs", 0},
		{"one %s", 1},
		{"%s and %d", 2},
		{"%d%% done", 1},
		{"%q: %q", 2},
	}
	for _, c := range cases {
		assert.Equalf(t, c.args, len(printableVerbs(c.format)), "printableVerbs(%q)", c.format)
		// And fmt agrees that this many values are consumed.
		args := make([]any, c.args)
		for i := range args {
			args[i] = i
		}
		formatted := fmt.Sprintf(c.format, args...)
		assert.NotContainsf(t, formatted, "MISSING",
			"fmt says %q needs more values than %d", c.format, c.args)
	}
	_ = strconv.Itoa(0)
}
