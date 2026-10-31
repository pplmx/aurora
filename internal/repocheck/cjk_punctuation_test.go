package repocheck

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// halfWidthAfterChinese matches a half-width mark that follows Chinese text,
// which is the spelling the project's Chinese prose avoids. An opening bracket
// is included: the catalogs write 会话开始时间（Unix）with full-width parentheses
// around the Latin note, and a half-width pair beside them reads as a different
// hand wrote the line.
var halfWidthAfterChinese = regexp.MustCompile(`[\p{Han}][,.;:!?(]|[\p{Han}] +[(]`)

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

	offenders := []string{}
	checked := 0
	han := regexp.MustCompile(`\p{Han}`)
	for _, rel := range catalogs {
		name := filepath.Join(rel...)
		for i, line := range strings.Split(readRepoFile(t, rel...), "\n") {
			m := catalogEntry.FindStringSubmatch(line)
			if m == nil || !han.MatchString(m[2]) {
				continue
			}
			checked++
			if halfWidthAfterChinese.MatchString(m[2]) {
				// The line number is what makes this actionable: a value can be
				// translated twice, and the fixing edit is one line.
				offenders = append(offenders, fmt.Sprintf("%s:%d %s", name, i+1, strings.TrimSpace(line)))
			}
		}
	}
	require.Greater(t, checked, 500, "the scan must reach the catalogs")

	sort.Strings(offenders)
	assert.Empty(t, offenders, "these entries put a half-width mark after Chinese text")

	// The matcher must recognise the shape it exists for and leave a Latin label
	// alone, or a green run would prove nothing.
	require.True(t, halfWidthAfterChinese.MatchString(`"key": "当前版本: %d"`),
		"the matcher must recognise an ASCII colon after Chinese")
	require.False(t, halfWidthAfterChinese.MatchString(`"key": "ID: %s"`),
		"the matcher must not flag a Latin label")
}
