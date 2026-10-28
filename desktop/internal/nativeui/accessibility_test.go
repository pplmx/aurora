package nativeui

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestPageChangeAnnouncesTheNewTitle covers the custom navigation path: unlike
// MyGo's Router, Aurora swaps pages itself, so a screen reader otherwise keeps
// reading the old page after a sidebar, shortcut or deep-link change.
func TestPageChangeAnnouncesTheNewTitle(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 900)
	tester.Frame()
	require.Empty(t, tester.Announcements())

	tester.Key(platformRefreshModifier(), ui.Key2)
	tester.Frame()
	require.Equal(t, pageLottery, app.page)
	require.Equal(t, []string{tr(pageTitles[pageLottery])}, tester.Announcements())

	tester.Frame()
	require.Empty(t, tester.Announcements(), "an unrelated frame must not repeat the page title")

	app.ShowPage(pageIDs[pageLottery])
	tester.Frame()
	require.Empty(t, tester.Announcements(), "choosing the current page is not a transition")
}

// TestCommandPaletteTakesTypingImmediately keeps Ctrl/Cmd+K a real keyboard
// shortcut: opening the dialog must put the caret in its search field, not
// leave the operator tabbing before the first letter can narrow the list.
func TestCommandPaletteTakesTypingImmediately(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 900)
	tester.Frame()

	tester.Key(platformRefreshModifier(), ui.KeyK)
	tester.Frame()
	require.True(t, app.paletteOpen)
	require.True(t, tester.Focused(tr("Search pages")), "the palette search field must own the keyboard")

	tester.Type("settings")
	tester.Frame()
	require.Equal(t, "settings", app.paletteQuery)

	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	require.Equal(t, pageSettings, app.page)
	require.False(t, app.paletteOpen)
}

// TestStatusOutcomeIsAnnouncedOnce covers feedback that appears away from the
// focused control. Every new outcome carries a successful/failed word, but a
// screen reader needs the change announced rather than silently painted.
func TestStatusOutcomeIsAnnouncedOnce(t *testing.T) {
	app := New(&fakeServices{})
	app.ShowPage(pageIDs[pageToken])
	tester := ui.NewTester(app.View, 1280, 900)
	tester.Frame()
	tester.Announcements() // the initial page transition

	app.token.status = okStatus("Saved.")
	tester.Frame()
	require.Equal(t, []string{"Done: " + app.token.status.text}, tester.Announcements())

	tester.Frame()
	require.Empty(t, tester.Announcements(), "rebuilding the same status must stay silent")

	// Repeating an action creates a new status revision even when its visible
	// text is unchanged, so the second save is confirmed too.
	app.token.status = okStatus("Saved.")
	tester.Frame()
	require.Equal(t, []string{"Done: " + app.token.status.text}, tester.Announcements())

	app.token.status = errStatus(errors.New("token not found"))
	tester.Frame()
	require.Equal(t, []string{"Failed: " + app.token.status.text}, tester.Announcements())
}

// TestActionCardIsKeyboardOperable pins the whole-card dashboard shortcut to
// the button behavior it visually imitates: Tab reaches it and Enter activates
// it without a pointer.
func TestActionCardIsKeyboardOperable(t *testing.T) {
	activated := false
	tester := ui.NewTester(func(c *ui.Context) {
		if actionCard(c, "Open token", func() {
			ui.Text(c, "Token body")
		}).Clicked() {
			activated = true
		}
	}, 420, 180)
	tester.Frame()

	tester.Key(0, ui.KeyTab)
	require.True(t, tester.Focused(tr("Open token")))
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	require.True(t, activated)
}

// TestTabsAreKeyboardOperable covers the custom focus ring and wrapping tab
// row: arrows select adjacent tabs and follow the focus, including at an end.
func TestTabsAreKeyboardOperable(t *testing.T) {
	selected := 0
	tester := ui.NewTester(func(c *ui.Context) {
		tabs(c, &selected, "Token actions", "Mint", "Transfer", "Approve")
	}, 500, 100)
	tester.Frame()

	tester.Key(0, ui.KeyTab)
	require.True(t, tester.Focused(tr("Mint")))
	tester.Key(0, ui.KeyRight)
	tester.Frame()
	require.Equal(t, 1, selected)
	require.True(t, tester.Focused(tr("Transfer")))

	tester.Key(0, ui.KeyEnd)
	tester.Frame()
	require.Equal(t, 2, selected)
	tester.Key(0, ui.KeyRight)
	tester.Frame()
	require.Equal(t, 0, selected, "Right on the last tab wraps to the first")
	require.True(t, tester.Focused(tr("Mint")))
}

// TestNavigationAndTabListsExposeNames gives each tree and tab list a name.
// Without them assistive technology announces an unnamed navigation tree or
// tab list, forcing the operator to infer the region from its first item.
func TestNavigationAndTabListsExposeNames(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 1800)
	tester.Frame()
	_, ok := tester.Find(tr("Navigation"))
	require.True(t, ok, "the sidebar tree must be named Navigation")

	for _, tc := range []struct {
		page page
		name string
	}{
		{pageVoting, "Voting sections"},
		{pageToken, "Token actions"},
		{pageOracle, "Oracle sections"},
		{pageNFT, "NFT actions"},
	} {
		app.ShowPage(pageIDs[tc.page])
		tester.Frame()
		_, ok := tester.Find(tr(tc.name))
		require.Truef(t, ok, "%s must name its tab list %q", pageTitles[tc.page], tc.name)
	}
}

// TestSessionTimeControlsNameTheirPurpose distinguishes the date button and
// each time segment. MyGo names a TimeInput's spin buttons after its Label and
// "hours"/"minutes"; without the Label both start and end controls announce
// only those generic segment words.
func TestSessionTimeControlsNameTheirPurpose(t *testing.T) {
	app := New(&fakeServices{})
	app.ShowPage(pageIDs[pageVoting])
	tester := ui.NewTester(app.View, 1280, 1800)
	tester.SetLocale("en")
	tester.Frame()

	for _, label := range []string{"Start date", "Start time", "End date", "End time"} {
		_, ok := tester.Find(tr(label))
		require.Truef(t, ok, "the session form must expose %q", label)
	}
}

// TestDataTablesHaveAccessibleNames checks the shared table path and its
// callsites. MyGo gives the table role an empty name by default, so every grid
// must pass through namedTable instead of calling ui.Table directly.
func TestDataTablesHaveAccessibleNames(t *testing.T) {
	state := &ui.ListState{}
	tester := ui.NewTester(func(c *ui.Context) {
		namedTable(c, "Draw history", state, []ui.TableColumn{{Title: tr("ID")}}, 0, func(row, col int) {})
	}, 480, 200)
	tester.Frame()
	_, ok := tester.Find(tr("Draw history"))
	require.True(t, ok, "a named table must expose its accessible name")

	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	tableCalls := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") ||
			strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, err := os.ReadFile(name)
		require.NoError(t, err)
		calls := strings.Count(string(source), "ui.Table(")
		tableCalls += calls
		if name != "controls.go" {
			require.Zerof(t, calls,
				"%s calls ui.Table directly; use namedTable so the grid has an accessible name", name)
		}
	}
	require.Equal(t, 1, tableCalls, "ui.Table must only be called inside the namedTable helper")

	controls, err := os.ReadFile("controls.go")
	require.NoError(t, err)
	require.Contains(t, string(controls), "ui.Table(c, state, columns, n, cell).Label(tr(name))")
}

// TestFormCardsNameTheirSections keeps repeated field labels in context. The
// ballot and session controls both have a Session ID field, while every token
// write card has an owner field; the containing group names the task.
func TestFormCardsNameTheirSections(t *testing.T) {
	layout, err := os.ReadFile("layout.go")
	require.NoError(t, err)
	require.True(t, strings.Contains(string(layout), "panel.Label(tr(title))"),
		"formCard must name its card group after the section title")
}

// TestHeadingsCarrySemanticRanks keeps titles from being only visually bold.
// The toolkit exposes RoleHeading and Level to screen readers; retaining the
// exact call here protects the shared heading path.
func TestHeadingsCarrySemanticRanks(t *testing.T) {
	layout, err := os.ReadFile("layout.go")
	require.NoError(t, err)
	require.Contains(t, string(layout), "Role(ui.RoleHeading).Level(level)")

	app, err := os.ReadFile("app.go")
	require.NoError(t, err)
	require.Contains(t, string(app), "heading(c, tr(pageTitles[a.page]), 26, 1)")
}
