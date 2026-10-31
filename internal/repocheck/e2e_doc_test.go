package repocheck

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2EDocMatchesTheScript keeps the guide's account of the capture script
// tied to what the script does. The two drifted once already: the guide said a
// language run captured two pages and wrote to `dist/e2e/zh/`, while the script
// had grown a title table for every page and named the directory after the
// variants it combined (light-zh). Both statements read as fact and neither
// failed anything.
func TestE2EDocMatchesTheScript(t *testing.T) {
	script := readRepoFile(t, "desktop", "scripts", "e2e-window.sh")
	guide := readRepoFile(t, "docs", "desktop.md")

	// The directory is the variant name, so the guide must not promise a fixed
	// per-language path.
	// The assertions compare search results rather than whole documents: a
	// failing NotContains prints every line, which buries the message.
	assert.Equal(t, -1, strings.Index(guide, "dist/e2e/zh/"),
		"the guide must not name a path the script does not use")

	// A language run now walks the same page loop, which means it carries a
	// stage table. If the script ever drops that, the guide has to say so.
	require.Contains(t, script, "stageTab", "the script needs a stage table for language runs")
	require.NotEqual(t, -1, strings.Index(guide, "语言变体同样逐页截图并断言"),
		"the guide must state that a language run covers every page")

	// Both variant spellings the guide promises are the ones the script builds.
	for _, want := range []string{"light", "zh"} {
		assert.Contains(t, script, want, "the script must build the %q variant", want)
	}
}

// TestE2EDocDoesNotPromiseTheOldTwoPageRun is the half that keeps the removed
// behaviour from creeping back as prose: the guide used to end a language run
// after two pages, and a reader planning a release would believe the rest of
// the pages were verified in every language.
func TestE2EDocDoesNotPromiseTheOldTwoPageRun(t *testing.T) {
	guide := readRepoFile(t, "docs", "desktop.md")
	for _, stale := range []string{
		"不参与后续页面循环",
		"只截取设置页与仪表盘",
	} {
		assert.False(t, strings.Contains(guide, stale),
			"the guide still describes the two-page language run: %q", stale)
	}
}
