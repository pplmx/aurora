package repocheck

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestArchitectureDocStatesWhatIsDone keeps the two history sections of the
// architecture document honest about their status. Both describe work that was
// finished, and a reader arriving at a heading that says "problems" cannot tell
// whether it lists open items or closed ones — the answer took reading every
// entry to find out.
func TestArchitectureDocStatesWhatIsDone(t *testing.T) {
	doc := readRepoFile(t, "docs", "architecture.md")

	// The section says the problems are solved, and says so in the heading.
	// The assertion is on the result of the search rather than the text, so a
	// failure names the missing heading instead of printing the whole document.
	assert.NotEqual(t, -1, strings.Index(doc, "## 已解决的边界问题"),
		"the section must name its outcome, not just its subject")
	assert.Equal(t, -1, strings.Index(doc, "## 已识别的边界问题"),
		"a heading that only says the problems were identified reads as an open list")

	start := strings.Index(doc, "## 已解决的边界问题")
	require.GreaterOrEqual(t, start, 0)
	end := strings.Index(doc[start:], "## 目标结构")
	require.Greater(t, end, 0)
	section := doc[start : start+end]

	// Every numbered problem records what was done about it.
	problems := regexp.MustCompile(`(?m)^### \d+\.`).FindAllString(section, -1)
	require.GreaterOrEqual(t, len(problems), 7, "the section lists its problems")
	done := strings.Count(section, "**已做**")
	assert.GreaterOrEqual(t, done, len(problems),
		"each of the %d problems must record what was done, found %d markers", len(problems), done)

	// The checklist's unchecked items are the remaining work, and the prose
	// says how many states that is.
	order := doc[strings.Index(doc, "## 执行顺序"):]
	unchecked := regexp.MustCompile(`(?m)^- \[ \]`).FindAllString(order, -1)
	assert.LessOrEqual(t, len(unchecked), 1,
		"the checklist states that only the last item is left, but %d are unchecked", len(unchecked))
	if len(unchecked) == 1 {
		assert.Contains(t, order, "最后一条是仍然待做的评估项",
			"the prose must name the item it leaves open")
	}
}
