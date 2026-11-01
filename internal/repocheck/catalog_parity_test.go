package repocheck

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTheCatalogsShareOneKeySet pins what the fallback hides. Translator.T falls
// back to English and then to the key itself, so a key added to one catalog and
// forgotten in another degrades quietly: the Simplified window shows English, or
// a raw identifier, and every test still passes because the value exists
// somewhere.
//
// The core catalogs are keyed by identifiers; the desktop ones are keyed by the
// English text itself, so each group is compared within itself.
func TestTheCatalogsShareOneKeySet(t *testing.T) {
	// A value may be a double-quoted string or a backtick raw string: the
	// long help paragraphs use the raw form so they can span lines.
	entry := regexp.MustCompile("(?m)^\\t\"((?:[^\"\\\\]|\\\\.)*)\":\\s*[\x60\"]")

	for _, group := range []struct {
		label    string
		base     []string
		catalogs [][]string
	}{
		{
			label: "core",
			base:  []string{"internal", "i18n", "messages_en.go"},
			catalogs: [][]string{
				{"internal", "i18n", "messages_zh.go"},
				{"internal", "i18n", "messages_zh_hant.go"},
			},
		},
		{
			label: "desktop",
			base:  []string{"desktop", "internal", "nativeui", "messages_zh.go"},
			catalogs: [][]string{
				{"desktop", "internal", "nativeui", "messages_zh_hant.go"},
			},
		},
	} {
		base := keysOf(t, entry, group.base...)
		require.Greaterf(t, len(base), 500, "%s: the reference catalog must be parsed", group.label)

		for _, rel := range group.catalogs {
			other := keysOf(t, entry, rel...)
			missing := difference(base, other)
			extra := difference(other, base)
			assert.Emptyf(t, missing, "%s misses keys the reference has: %v", join(rel), missing)
			assert.Emptyf(t, extra, "%s defines keys the reference lacks: %v", join(rel), extra)
		}
	}
}

// keysOf reads the top-level string keys of a catalog.
func keysOf(t *testing.T, entry *regexp.Regexp, rel ...string) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, m := range entry.FindAllStringSubmatch(readRepoFile(t, rel...), -1) {
		out[m[1]] = true
	}
	return out
}

// difference returns the keys of want that have is absent.
func difference(want, have map[string]bool) []string {
	out := []string{}
	for k := range want {
		if !have[k] {
			out = append(out, k)
		}
	}
	return out
}

func join(rel []string) string {
	s := ""
	for i, part := range rel {
		if i > 0 {
			s += "/"
		}
		s += part
	}
	return s
}
