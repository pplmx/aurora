package nativeui

import (
	"strconv"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestEveryPageHasItsOwnIcon pins the sidebar's glyph set. A page added to
// pageOrder without an icon would otherwise render a row that is only text,
// and nil would be passed straight into SidebarItem.
func TestEveryPageHasItsOwnIcon(t *testing.T) {
	seen := map[*ui.SVG]page{}
	for _, p := range pageOrder {
		icon := pageIcons[p]
		require.NotNilf(t, icon, "page %q has no sidebar icon", pageTitles[p])
		if other, dup := seen[icon]; dup {
			t.Fatalf("pages %q and %q share the same icon", pageTitles[other], pageTitles[p])
		}
		seen[icon] = p
	}
}

// TestNumberShortcutsSwitchPages pins Ctrl/Cmd+1..7 in sidebar order, the
// desktop convention the reference MyGo apps follow. It runs through the real
// view so the shortcut has to be registered where the header is built, not
// only in a helper.
func TestNumberShortcutsSwitchPages(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 860)
	tester.Frame()

	for i, want := range pageOrder {
		tester.Key(platformRefreshModifier(), ui.Key(int(ui.Key1)+i))
		tester.Frame()
		require.Equalf(t, want, app.page, "shortcut for %q must select it", pageTitles[want])
	}
}

// TestHeaderNamesThePageAndItsPurpose keeps the chrome honest: the title and
// the one-line purpose are part of the page's identity, and the glyph sits
// beside them rather than replacing either.
func TestHeaderNamesThePageAndItsPurpose(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 860)

	for _, p := range pageOrder {
		app.ShowPage(pageTitles[p])
		tester.Frame()
		require.Truef(t, tester.HasText(pageTitles[p]), "header must name %q", pageTitles[p])
		require.Truef(t, tester.HasText(pagePurpose[p]), "header must state the purpose of %q", pageTitles[p])
	}
}

// TestStatusStaysInTheFixedChrome pins where an action's outcome is drawn.
// The status line used to be the first child of each page's scrolling body, so
// a failure from a form near the bottom could sit above the viewport: the
// operator saw the form unchanged and assumed the write had worked.
func TestStatusStaysInTheFixedChrome(t *testing.T) {
	app := New(&fakeServices{tokenRows: []TokenRow{{ID: "t1", Name: "Gold", TotalSupply: "100"}}})
	tester := ui.NewTester(app.View, 1000, 600)
	app.ShowPage("Token")
	app.token.status = okStatus("Saved.")
	tester.Frame()

	headingBefore, ok := tester.Find("Create Token")
	require.True(t, ok, "the form heading must be on screen before scrolling")
	statusBefore, ok := tester.Find("Saved.")
	require.True(t, ok, "the status line must be on screen before scrolling")

	tester.Scroll(700, 400, 0, 120)
	tester.Frame()

	headingAfter, ok := tester.Find("Create Token")
	require.True(t, ok, "the body must still show the form heading")
	require.Less(t, headingAfter.Y, headingBefore.Y, "the body must actually scroll")

	statusAfter, ok := tester.Find("Saved.")
	require.True(t, ok, "the status line must remain on screen while the body scrolls")
	require.InDelta(t, float64(statusBefore.Y), float64(statusAfter.Y), 0.5,
		"the status line is fixed chrome: only the body may move")
}

// TestDashboardListsTheNewestChainEntries pins the overview's activity card:
// the counters say how much the store holds, the list says what happened, and
// only the newest few belong on the landing page.
func TestDashboardListsTheNewestChainEntries(t *testing.T) {
	blocks := make([]BlockRow, 7)
	for i := range blocks {
		blocks[i] = BlockRow{
			Height:  int64(len(blocks) - i),
			Module:  "module-" + strconv.Itoa(i),
			Summary: "payload-" + strconv.Itoa(i),
		}
	}
	app := New(&fakeServices{blocks: blocks})
	tester := ui.NewTester(app.View, 1280, 1000)
	tester.Frame()

	require.True(t, tester.HasText("payload-0"), "the newest entry must be listed")
	require.True(t, tester.HasText("payload-4"), "the fifth newest entry must be listed")
	require.False(t, tester.HasText("payload-5"), "the card must show only the newest five")
}

// TestSwitchingPagesStartsAtTheTop pins the scroll of a newly chosen page. The
// page body is a Scroll at the same place in the tree on every page, so its
// offset can leak from the page just left; a short page then opens halfway
// down, with its title area and first cards above the viewport.
func TestSwitchingPagesStartsAtTheTop(t *testing.T) {
	tokens := make([]TokenRow, 30)
	for i := range tokens {
		tokens[i] = TokenRow{ID: "t" + strconv.Itoa(i), Name: "Token " + strconv.Itoa(i), Symbol: "T", TotalSupply: "1"}
	}
	app := New(&fakeServices{tokenRows: tokens})
	tester := ui.NewTester(app.View, 1000, 600)
	app.ShowPage("Token")
	tester.Frame()

	before, ok := tester.Find("Tokens")
	require.True(t, ok, "the collection heading must be visible before scrolling")
	tester.Scroll(900, 500, 0, 900)
	tester.Frame()
	after, ok := tester.Find("Tokens")
	require.True(t, ok)
	require.Less(t, after.Y, before.Y, "the body must have scrolled before the page switch is meaningful")

	// A fresh app that never scrolled gives the position of the top of the
	// Lottery page; the switched-to page must match it.
	fresh := New(&fakeServices{tokenRows: tokens})
	freshTester := ui.NewTester(fresh.View, 1000, 600)
	freshTester.Frame()
	fresh.ShowPage("Lottery")
	freshTester.Frame()
	want, ok := freshTester.Find("Create Draw")
	require.True(t, ok)

	app.ShowPage("Lottery")
	tester.Frame()
	got, ok := tester.Find("Create Draw")
	require.True(t, ok, "a page chosen after scrolling another must open at its top")
	require.InDelta(t, float64(want.Y), float64(got.Y), 1,
		"the new page must start at the scroll position of a fresh one")
}

// TestFreshDashboardCanOpenTheLottery pins the onboarding line's action: the
// empty dashboard names Lottery as the first move, and the button takes it
// instead of leaving the operator to hunt the sidebar.
type emptyDashboard struct{ *fakeServices }

func (emptyDashboard) Dashboard() (DashboardStats, error) {
	return DashboardStats{Integrity: "verified"}, nil
}

func TestFreshDashboardCanOpenTheLottery(t *testing.T) {
	app := New(emptyDashboard{fakeServices: &fakeServices{}})
	tester := ui.NewTester(app.View, 1280, 860)
	tester.Frame()
	require.Equal(t, pageDashboard, app.page)

	require.NoError(t, tester.Click("Open Lottery"))
	tester.Frame()
	require.Equal(t, pageLottery, app.page)
}
