package api

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWebUIDashboardComposition pins the overview hierarchy that replaced the
// generic auto-fit card wall. Six metrics must form an explicit 3x2 desktop
// grid, remain two-up on phones, and use a real list for recent activity.
func TestWebUIDashboardComposition(t *testing.T) {
	bodyBytes, err := os.ReadFile(filepath.Join(realWebDir(), "index.html"))
	require.NoError(t, err)
	body := string(bodyBytes)

	require.NotContains(t, body, `<div class="stat-card`,
		"dashboard metrics should be semantic article elements")
	require.Equal(t, 6, strings.Count(body, `<article class="stat-card`),
		"dashboard must present all six metrics in one deliberate composition")
	for _, label := range []string{
		"Total Lotteries", "Total Votes", "Candidates",
		"Voting Sessions", "Ledger Integrity", "Oracle Feeds",
	} {
		require.Contains(t, body, label)
	}
	require.Contains(t, body, `class="overview-hero"`)
	require.Contains(t, body, `class="stats-panel"`)
	require.Contains(t, body, `class="activity-feed"`)
	require.Contains(t, body, `class="activity-item"`)

	cssBytes, err := os.ReadFile(filepath.Join(realWebDir(), "css", "style.css"))
	require.NoError(t, err)
	css := string(cssBytes)
	stats := cssRuleForSelectors(t, css, ".stats-grid")
	require.Equal(t, "repeat(3, minmax(0, 1fr))", cssDeclaration(t, stats, "grid-template-columns"))
	require.Contains(t, css, "grid-template-columns: repeat(2, minmax(0, 1fr));",
		"phones must keep two compact metric columns instead of one long column")
}

// TestWebUILotteryAndOracleWorkspaces keeps the two busiest workspaces on the
// same navigation contract as the token page: a page intro with an in-page
// operation index, a two-column workspace grid, and one numbered panel per
// operation whose anchor matches the index entry pointing at it.
func TestWebUILotteryAndOracleWorkspaces(t *testing.T) {
	cases := []struct {
		file      string
		titleAttr string
		indexAria string
		panels    int
	}{
		{"lottery.html", "lottery-workspace-title", "Lottery operations", 3},
		{"oracle.html", "oracle-workspace-title", "Oracle operations", 6},
		{"voting.html", "voting-workspace-title", "Voting operations", 7},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			bodyBytes, err := os.ReadFile(filepath.Join(realWebDir(), tc.file))
			require.NoError(t, err)
			body := string(bodyBytes)

			require.Contains(t, body, `class="page-intro"`)
			require.Contains(t, body, `aria-labelledby="`+tc.titleAttr+`"`)
			require.Contains(t, body, `role="navigation" aria-label="`+tc.indexAria+`"`)
			require.Contains(t, body, `class="workspace-grid"`)
			require.Equal(t, tc.panels, strings.Count(body, `class="panel-number"`),
				"%s: every operation needs a numbered workspace panel", tc.file)

			prefix := strings.SplitN(tc.file, ".", 2)[0] + "-"
			idRE := regexp.MustCompile(`<section id="(` + prefix + `[^"]+)"`)
			hrefRE := regexp.MustCompile(`href="#(` + prefix + `[^"]+)"`)
			ids := idRE.FindAllStringSubmatch(body, -1)
			hrefs := hrefRE.FindAllStringSubmatch(body, -1)
			require.Len(t, ids, tc.panels, "%s: section ids", tc.file)
			require.Len(t, hrefs, tc.panels, "%s: index links", tc.file)
			for i := range ids {
				require.Equal(t, ids[i][1], hrefs[i][1],
					"%s: workspace index order must match panel order", tc.file)
			}
		})
	}
}

// TestWebUITokenWorkspaceNavigation keeps every operation in the token page
// discoverable from the workspace index and preserves the two-column form
// grouping that makes the long transaction surface usable on desktop.
func TestWebUITokenWorkspaceNavigation(t *testing.T) {
	bodyBytes, err := os.ReadFile(filepath.Join(realWebDir(), "token.html"))
	require.NoError(t, err)
	body := string(bodyBytes)

	require.Contains(t, body, `class="page-intro"`)
	require.Contains(t, body, `role="navigation" aria-label="Token operations"`)
	require.Contains(t, body, `class="workspace-grid"`)
	require.Equal(t, 10, strings.Count(body, `class="panel-number"`),
		"all ten token operations need a numbered workspace panel")
	require.Equal(t, 10, strings.Count(body, `<form class="form-grid`),
		"all token forms should share the responsive form grid")

	idRE := regexp.MustCompile(`<section id="(token-[^"]+)"`)
	hrefRE := regexp.MustCompile(`href="#(token-[^"]+)"`)
	ids := idRE.FindAllStringSubmatch(body, -1)
	hrefs := hrefRE.FindAllStringSubmatch(body, -1)
	require.Len(t, ids, 10)
	require.Len(t, hrefs, 10)
	for i := range ids {
		require.Equal(t, ids[i][1], hrefs[i][1],
			"workspace navigation order must match panel order")
	}

	cssBytes, err := os.ReadFile(filepath.Join(realWebDir(), "css", "style.css"))
	require.NoError(t, err)
	css := string(cssBytes)
	workspace := cssRuleForSelectors(t, css, ".workspace-grid")
	require.Equal(t, "repeat(2, minmax(0, 1fr))", cssDeclaration(t, workspace, "grid-template-columns"))
	form := cssRuleForSelectors(t, css, ".form-grid")
	require.Equal(t, "repeat(2, minmax(0, 1fr))", cssDeclaration(t, form, "grid-template-columns"))
}

// TestWebUIDesignMotionHonorsPreferences keeps the entrance sequence
// intentional without making reduced-motion users wait through it.
func TestWebUIDesignMotionHonorsPreferences(t *testing.T) {
	cssBytes, err := os.ReadFile(filepath.Join(realWebDir(), "css", "style.css"))
	require.NoError(t, err)
	css := string(cssBytes)

	require.Contains(t, css, "@keyframes rise-in")
	require.Contains(t, css, "@media (prefers-reduced-motion: reduce)")
	require.Contains(t, css, "animation-duration: 0.01ms !important;")
	require.Contains(t, css, "animation-delay: 0ms !important;")
}
