package api

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWebUINavigationMarksCurrentPage gives every shipped page one
// machine-readable current-location marker. Without it the seven identical
// navigation bars force operators to infer the active module from the title,
// and assistive technology cannot announce the current page (WCAG 2.4.8).
func TestWebUINavigationMarksCurrentPage(t *testing.T) {
	handler := injectAPIKey(http.FileServer(http.Dir(realWebDir())), "test-serve-key")
	pages := []struct {
		path   string
		module string
		href   string
	}{
		{path: "/", module: "dashboard", href: "/"},
		{path: "/lottery.html", module: "lottery", href: "/lottery.html"},
		{path: "/voting.html", module: "voting", href: "/voting.html"},
		{path: "/token.html", module: "token", href: "/token.html"},
		{path: "/oracle.html", module: "oracle", href: "/oracle.html"},
		{path: "/blockchain.html", module: "blockchain", href: "/blockchain.html"},
		{path: "/nft.html", module: "nft", href: "/nft.html"},
	}

	navRE := regexp.MustCompile(`(?s)<nav\b[^>]*>(.*?)</nav>`)
	linkRE := regexp.MustCompile(`<a\b[^>]*>`)
	hrefRE := regexp.MustCompile(`\bhref="([^"]+)"`)
	for _, page := range pages {
		t.Run(page.module, func(t *testing.T) {
			body := requireServedAsset(t, handler, page.path)
			require.Contains(t, body, `data-module="`+page.module+`"`,
				"%s must identify its module for shared active-accent styling", page.path)

			nav := navRE.FindStringSubmatch(body)
			require.Len(t, nav, 2, "%s must contain one header navigation", page.path)

			current := 0
			for _, link := range linkRE.FindAllString(nav[1], -1) {
				href := hrefRE.FindStringSubmatch(link)
				require.Len(t, href, 2, "%s: navigation link has no href: %s", page.path, link)
				isCurrent := href[1] == page.href
				if isCurrent {
					current++
					require.Contains(t, link, `aria-current="page"`,
						"%s: current link %s must set aria-current=page", page.path, href[1])
				} else {
					require.NotContains(t, link, `aria-current=`,
						"%s: only the current link may set aria-current, got %s", page.path, link)
				}
			}
			require.Equal(t, 1, current,
				"%s must mark exactly one navigation link current", page.path)
		})
	}
}

// TestWebUIResponsiveShell keeps the module navigation in a compact,
// horizontally scrollable command bar instead of wrapping it into four rows
// on phones. The header itself is a deliberate grid: brand and action stay on
// the first row while navigation owns the second row on small screens.
func TestWebUIResponsiveShell(t *testing.T) {
	cssBytes, err := os.ReadFile(filepath.Join(realWebDir(), "css", "style.css"))
	require.NoError(t, err)
	css := string(cssBytes)

	header := cssRuleForSelectors(t, css, "header")
	require.Equal(t, "grid", cssDeclaration(t, header, "display"))
	require.Equal(t, "auto minmax(0, 1fr) auto", cssDeclaration(t, header, "grid-template-columns"))

	nav := cssRuleForSelectors(t, css, "nav")
	require.Equal(t, "nowrap", cssDeclaration(t, nav, "flex-wrap"))
	require.Equal(t, "auto", cssDeclaration(t, nav, "overflow-x"))

	active := cssRuleForSelectors(t, css, `nav a[aria-current="page"]`)
	require.Contains(t, cssDeclaration(t, active, "box-shadow"), "var(--accent-page)",
		"the current navigation link must carry the page accent")
	require.Contains(t, css, "@media (max-width: 48rem)",
		"the compact navigation must have a phone/tablet breakpoint")
	require.Contains(t, css, "grid-column: 1 / -1;",
		"mobile navigation must move below the brand and action")

	for module, accent := range map[string]string{
		"dashboard":  "var(--text-secondary)",
		"lottery":    "var(--accent-lottery)",
		"voting":     "var(--accent-voting)",
		"token":      "var(--accent-token)",
		"oracle":     "var(--accent-oracle)",
		"blockchain": "var(--accent-blockchain)",
		"nft":        "var(--accent-nft)",
	} {
		rule := cssRuleForSelectors(t, css, `.container[data-module="`+module+`"]`)
		require.Equal(t, accent, cssDeclaration(t, rule, "--accent-page"))
	}
}

// TestWebUITokenActionsUseTokenAccent keeps the Token module from wearing
// the Lottery purple. The token page had no .btn-token despite the stylesheet
// already carrying --accent-token for the Token module.
func TestWebUITokenActionsUseTokenAccent(t *testing.T) {
	bodyBytes, err := os.ReadFile(filepath.Join(realWebDir(), "token.html"))
	require.NoError(t, err)
	body := string(bodyBytes)
	require.NotContains(t, body, `class="btn-lottery"`,
		"token actions must not use the Lottery accent")
	require.GreaterOrEqual(t, strings.Count(body, `class="btn-token"`), 10,
		"all token actions should use .btn-token")
}
