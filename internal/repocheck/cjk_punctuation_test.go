package repocheck

import (
	"path/filepath"
	"regexp"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// halfWidthAfterChinese matches a half-width mark that follows Chinese text,
// which is the spelling the project's Chinese prose avoids.
var halfWidthAfterChinese = regexp.MustCompile(`[\p{Han}][,.;:!?]`)

// TestChineseCopyUsesFullWidthPunctuation holds the rule the spacing test next
// to it starts from: Chinese prose uses the full-width mark. It matters most
// where the two catalogues meet — a label translated in one and a sentence in
// the other — because a reader sees them on the same line.
func TestChineseCopyUsesFullWidthPunctuation(t *testing.T) {
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
			if !regexp.MustCompile(`\p{Han}`).MatchString(value) {
				continue
			}
			checked++
			if halfWidthAfterChinese.MatchString(value) {
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
	assert.Empty(t, names, "these entries put a half-width mark after Chinese text")
}
