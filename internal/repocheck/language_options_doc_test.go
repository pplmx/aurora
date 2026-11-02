package repocheck

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// languageOptionDecl matches one entry of the language radio list in the view:
// `{"Follow System", languageFollowsSystem},`.
var languageOptionDecl = regexp.MustCompile(`\{"([^"]+)",\s*(?:"[A-Za-z-]+"|[A-Za-z][A-Za-z0-9]*)\},`)

// TestLanguageOptionsMatchTheDoc keeps the desktop guide's account of the
// language card tied to the options the card actually draws. The card has
// changed shape twice — a button beside the group, then a radio inside it — and
// the guide described the old shape both times until it was corrected by hand.
func TestLanguageOptionsMatchTheDoc(t *testing.T) {
	code := readRepoFile(t, "desktop", "internal", "nativeui", "settings.go")

	start := strings.Index(code, "func languageChooser(")
	require.GreaterOrEqual(t, start, 0, "settings.go must keep languageChooser")
	end := strings.Index(code[start:], "}).Row().Wrap().Gap(space4)")
	require.Greater(t, end, 0, "the option list must end before the group's row settings")

	options := languageOptionDecl.FindAllStringSubmatch(code[start:start+end], -1)
	require.GreaterOrEqual(t, len(options), 4, "the group must offer at least four options")
	require.Equal(t, "Follow System", options[0][1], "following the system is the first option")

	doc := readRepoFile(t, "docs", "desktop.md")
	at := strings.Index(doc, "设置页的语言是一个命名单选组")
	require.GreaterOrEqual(t, at, 0, "docs/desktop.md must describe the language group")
	// The sentence that names the options ends at the em dash that introduces
	// the comparison with the Appearance card.
	rest := doc[at:]
	if end := strings.Index(rest, "方向键即可切换"); end > 0 {
		rest = rest[:end]
	}

	// The guide is written in Chinese, so the first option appears under its
	// translated name; the other three are language names and stay as they are.
	assert.Contains(t, rest, "跟随系统", "the guide must name the follow-system option")
	for _, m := range options[1:] {
		assert.Containsf(t, rest, m[1], "docs/desktop.md does not name %q", m[1])
	}
}
