package api

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	cssCommentRE = regexp.MustCompile(`(?s)/\*.*?\*/`)
	cssRuleRE    = regexp.MustCompile(`(?s)([^{}]+)\{([^{}]*)\}`)
)

// TestWebUIFormControlsUseDarkTheme keeps every form-control type on the same
// visual contract. Native selects have their own browser chrome and do not
// inherit the input background/border/text styling, which made the Voting and
// Oracle pickers fall out of the dark theme.
func TestWebUIFormControlsUseDarkTheme(t *testing.T) {
	cssBytes, err := os.ReadFile(filepath.Join(realWebDir(), "css", "style.css"))
	require.NoError(t, err)
	css := string(cssBytes)

	root := cssRuleForSelectors(t, css, ":root")
	require.Equal(t, "dark", cssDeclaration(t, root, "color-scheme"),
		"the root must opt native controls into the dark palette")

	controls := cssRuleForSelectors(t, css,
		".form-group input",
		".form-group select",
		".form-group textarea",
	)
	require.Equal(t, "var(--bg-primary)", cssDeclaration(t, controls, "background-color"))
	require.Equal(t, "1px solid var(--border)", cssDeclaration(t, controls, "border"))
	require.Equal(t, "10px", cssDeclaration(t, controls, "border-radius"))
	require.Equal(t, "var(--text-primary)", cssDeclaration(t, controls, "color"))
	require.Equal(t, "1rem", cssDeclaration(t, controls, "font-size"))

	focus := cssRuleForSelectors(t, css,
		".form-group input:focus",
		".form-group select:focus",
		".form-group textarea:focus",
	)
	require.Equal(t, "var(--accent-lottery)", cssDeclaration(t, focus, "border-color"))
}

func cssRuleForSelectors(t *testing.T, css string, selectors ...string) string {
	t.Helper()
	want := append([]string(nil), selectors...)
	for i := range want {
		want[i] = normalizeCSSSelector(want[i])
	}
	slices.Sort(want)

	css = cssCommentRE.ReplaceAllString(css, "")
	for _, match := range cssRuleRE.FindAllStringSubmatch(css, -1) {
		got := strings.Split(match[1], ",")
		for i := range got {
			got[i] = normalizeCSSSelector(got[i])
		}
		slices.Sort(got)
		if slices.Equal(got, want) {
			return match[2]
		}
	}
	require.FailNow(t, "style.css must define the selector group", strings.Join(selectors, ", "))
	return ""
}

func normalizeCSSSelector(selector string) string {
	return strings.Join(strings.Fields(selector), " ")
}
