package repocheck

import (
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// simplifiedOnlyChars lists characters the Simplified catalogs use and the
// Traditional ones never do. It is committed rather than derived at test time on
// purpose: a Simplified character typed into the Traditional table would remove
// itself from a derived set, and the check would pass on exactly the entry it
// exists to catch.
const simplifiedOnlyChars = "与个为么义习于产仅从仓仪价优会传体余侧储党关内册写决准减几则创删别务动区单占历参发台号后听启响图坏块声壳处备复头夹奖实审对导将尝属币帮并库应开张强当录径忆态总惯户才执护报拦择损据数断无时显暂机权条来构标栏样核档检毁没注浅测浏滚滤点状现监盖盘码确种称竞筛签类级纲线组细织终经绑结给络绝统继绪续编缩网脏脚节获补装观览计认让议记许论设访证诊试话询该详语误说请读调负败账贴资转轮载输达迁过运还这进适选递里钥钮铸链销错键长闭问间随页项须预领频题额验"

// TestTheTraditionalCatalogsUseNoSimplifiedCharacters catches the entry someone
// forgot to translate. The generator converts character by character, so a value
// that never went through it keeps its Simplified form and reads as a different
// language on the same screen.
func TestTheTraditionalCatalogsUseNoSimplifiedCharacters(t *testing.T) {
	entry := regexp.MustCompile(`(?m)^\t"((?:[^"\\]|\\.)*)":\s*"((?:[^"\\]|\\.)*)",$`)

	require.Greater(t, len([]rune(simplifiedOnlyChars)), 150,
		"the committed set must keep its size; a shrinking set hides entries")

	for _, group := range []struct {
		label string
		path  []string
	}{
		{"core", []string{"internal", "i18n", "messages_zh_hant.go"}},
		{"desktop", []string{"desktop", "internal", "nativeui", "messages_zh_hant.go"}},
	} {
		values := map[string]string{}
		for _, m := range entry.FindAllStringSubmatch(readRepoFile(t, group.path...), -1) {
			values[m[1]] = m[2]
		}
		require.Greaterf(t, len(values), 500, "%s: the catalog must be parsed", group.label)

		offenders := []string{}
		for key, value := range values {
			bad := []string{}
			for _, r := range value {
				if strings.ContainsRune(simplifiedOnlyChars, r) {
					bad = append(bad, string(r))
				}
			}
			if len(bad) > 0 {
				sort.Strings(bad)
				offenders = append(offenders, key+" ("+strings.Join(bad, "")+")")
			}
		}
		sort.Strings(offenders)
		assert.Emptyf(t, offenders,
			"%s: Traditional values holding Simplified characters: %v", group.label, offenders)
	}
}
