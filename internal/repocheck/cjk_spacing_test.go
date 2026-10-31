package repocheck

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	catalogEntry  = regexp.MustCompile(`(?m)^\t"((?:[^"\\]|\\.)*)":\s+"((?:[^"\\]|\\.)*)",$`)
	cjk           = `\p{Han}`
	cjkTouchLatin = regexp.MustCompile(`[\p{Han}][A-Za-z]|[A-Za-z][\p{Han}]`)
)

// TestChineseCopySpacesLatinWords keeps the one typographic rule the catalogs
// share. The project writes Chinese prose with a space between a Chinese
// character and a Latin word, and the CLI labels were the exception: they said
// 抽奖ID while the desktop table said 抽奖 ID for the same string. A rule applied
// in three of four places is the kind of thing a reader notices without being
// able to name.
func TestChineseCopySpacesLatinWords(t *testing.T) {
	catalogs := [][]string{
		{"internal", "i18n", "messages_zh.go"},
		{"internal", "i18n", "messages_zh_hant.go"},
		{"desktop", "internal", "nativeui", "messages_zh.go"},
		{"desktop", "internal", "nativeui", "messages_zh_hant.go"},
	}

	offenders := map[string]bool{}
	checked := 0
	for _, rel := range catalogs {
		name := filepath.Join(rel...)
		for _, m := range catalogEntry.FindAllStringSubmatch(readRepoFile(t, rel...), -1) {
			value := m[2]
			// A placeholder or an escaped newline separates as much as a space
			// does, so both become a space before the check: "%sID" and
			// "命令。\\n输入" are not examples of a missing space.
			stripped := strings.NewReplacer(
				"%s", " ", "%d", " ", "%v", " ", "%q", " ",
				`\n`, " ", `\t`, " ",
			).Replace(value)
			if !regexp.MustCompile(cjk).MatchString(stripped) {
				continue
			}
			checked++
			if cjkTouchLatin.MatchString(stripped) {
				offenders[name+" :: "+value] = true
			}
		}
	}
	require.Greater(t, checked, 500, "the scan must reach the catalogs")

	names := make([]string, 0, len(offenders))
	for name := range offenders {
		names = append(names, name)
	}
	sort.Strings(names)
	assert.Empty(t, names, "these entries put a Chinese character against a Latin word")
}
