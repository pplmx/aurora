package repocheck

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTheTraditionalCatalogUsesOneWordPerConcept catches a mainland word left
// behind in a Traditional table. Four entries said 資料源 where the other 42 say
// 數據源 — the same page could show both — and no existing check noticed,
// because each spelling is a legitimate word.
//
// Only pairs where one form is the standard Traditional spelling and the other
// is the mainland one are listed. A pair the project deliberately distinguishes
// is not: 預設 (preset) and 默認 (default) name different things.
func TestTheTraditionalCatalogUsesOneWordPerConcept(t *testing.T) {
	values := regexp.MustCompile(`(?m)^\t"[^"]*":\s*"((?:[^"\\]|\\.)*)"`)
	pairs := []struct{ standard, mainland, why string }{
		{"資料", "數據", "data is 資料 in Traditional usage, but this catalog says 數據 everywhere"},
		{"網路", "網絡", "network is 網路 in Traditional usage"},
		{"軟體", "軟件", "software is 軟體 in Traditional usage"},
		{"檔案", "文件", "file is 檔案 in Traditional usage"},
		{"資訊", "信息", "information is 資訊 in Traditional usage"},
		{"選單", "菜單", "menu is 選單 in Traditional usage"},
	}

	checked := 0
	for _, rel := range [][]string{
		{"internal", "i18n", "messages_zh_hant.go"},
		{"desktop", "internal", "nativeui", "messages_zh_hant.go"},
	} {
		body := readRepoFile(t, rel...)
		name := filepath.Join(rel...)
		var text strings.Builder
		for _, m := range values.FindAllStringSubmatch(body, -1) {
			text.WriteString(m[1])
			checked++
		}
		require.Greaterf(t, checked, 100, "%s must be parsed", name)

		all := text.String()
		for _, p := range pairs {
			hasStandard := strings.Contains(all, p.standard)
			hasMainland := strings.Contains(all, p.mainland)
			if hasStandard && hasMainland {
				// 數據/資料 is settled by the census: whichever form dominates
				// wins, so a table that is consistently mainland is left alone.
				standard, mainland := strings.Count(all, p.standard), strings.Count(all, p.mainland)
				dominant := p.standard
				if mainland > standard {
					dominant = p.mainland
				}
				minor := p.mainland
				if dominant == p.mainland {
					minor = p.standard
				}
				n := strings.Count(all, minor)
				assert.Equalf(t, 0, n,
					"%s: %q appears %d times beside the dominant %q; %s",
					name, minor, n, dominant, p.why)
			}
		}
	}
}
