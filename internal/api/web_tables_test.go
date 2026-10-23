package api

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWebUITablesUseScrollFrames keeps wide data inside its card on phones and
// split desktop panes. Unwrapped tables with UUIDs, participant lists, feed
// errors or five columns widened the whole document instead.
func TestWebUITablesUseScrollFrames(t *testing.T) {
	tableFrameRE := regexp.MustCompile(`(?s)<div class="table-scroll"(?:[^>"']|"[^"]*"|'[^']*')*>\s*<table\b`)
	frameTagRE := regexp.MustCompile(`(?s)<div class="table-scroll"(?:[^>"']|"[^"]*"|'[^']*')*>`)
	labelRE := regexp.MustCompile(`\baria-label="[^"]+"`)
	for _, page := range []string{"lottery.html", "voting.html", "oracle.html"} {
		t.Run(page, func(t *testing.T) {
			bodyBytes, err := os.ReadFile(filepath.Join(realWebDir(), page))
			require.NoError(t, err)
			body := string(bodyBytes)
			tables := strings.Count(body, "<table")
			require.Positive(t, tables, "%s should contain at least one data table", page)
			require.Equal(t, tables, len(tableFrameRE.FindAllString(body, -1)),
				"%s must wrap every table in .table-scroll", page)
			for _, frame := range frameTagRE.FindAllString(body, -1) {
				require.Contains(t, frame, `role="region"`,
					"%s: a scrollable table needs a named landmark", page)
				require.Contains(t, frame, `tabindex="0"`,
					"%s: a scrollable table must be keyboard reachable", page)
				require.Regexp(t, labelRE, frame,
					"%s: a scrollable table needs an accessible name", page)
			}
		})
	}
}

// TestWebUITableResponsiveStyles pins the frame and cell behavior that lets
// the table scroll instead of overflowing, while keeping long values readable
// once they reach a cell boundary.
func TestWebUITableResponsiveStyles(t *testing.T) {
	cssBytes, err := os.ReadFile(filepath.Join(realWebDir(), "css", "style.css"))
	require.NoError(t, err)
	css := string(cssBytes)

	frame := cssRuleForSelectors(t, css, ".table-scroll")
	require.Equal(t, "auto", cssDeclaration(t, frame, "overflow-x"))
	require.Equal(t, "contain", cssDeclaration(t, frame, "overscroll-behavior-x"))
	focus := cssRuleForSelectors(t, css, ".table-scroll:focus-visible")
	require.Contains(t, cssDeclaration(t, focus, "outline"), "2px solid var(--accent-page)")

	table := cssRuleForSelectors(t, css, ".table-scroll table")
	require.Equal(t, "40rem", cssDeclaration(t, table, "min-width"))

	cells := cssRuleForSelectors(t, css, ".table-scroll td")
	require.Equal(t, "anywhere", cssDeclaration(t, cells, "overflow-wrap"))
	require.Equal(t, "top", cssDeclaration(t, cells, "vertical-align"))
}

// TestWebUIOracleActionsAreGrouped keeps the state-changing and destructive
// row controls separated instead of rendering as one continuous button slab.
func TestWebUIOracleActionsAreGrouped(t *testing.T) {
	bodyBytes, err := os.ReadFile(filepath.Join(realWebDir(), "oracle.html"))
	require.NoError(t, err)
	require.Contains(t, string(bodyBytes), `<div class="table-actions">`)

	cssBytes, err := os.ReadFile(filepath.Join(realWebDir(), "css", "style.css"))
	require.NoError(t, err)
	actions := cssRuleForSelectors(t, string(cssBytes), ".table-actions")
	require.Equal(t, "flex", cssDeclaration(t, actions, "display"))
	require.Equal(t, "wrap", cssDeclaration(t, actions, "flex-wrap"))
	require.Equal(t, "8px", cssDeclaration(t, actions, "gap"))
}
