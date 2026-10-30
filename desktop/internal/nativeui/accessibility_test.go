package nativeui

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

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

// TestCommandPaletteEscapeReturnsFocusToNavigation keeps the keyboard from
// falling into the window root when the search field that owned it disappears.
// Escape leaves the current page untouched and lands on the sidebar's single
// focus stop, from which Tab can continue through the window.
func TestCommandPaletteEscapeReturnsFocusToNavigation(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 900)
	app.ShowPage("Settings")
	tester.Frame()

	tester.Key(platformRefreshModifier(), ui.KeyK)
	tester.Frame()
	require.True(t, tester.Focused("Search pages"))

	tester.Key(0, ui.KeyEscape)
	tester.Frame()
	require.False(t, app.paletteOpen)
	require.Equal(t, pageSettings, app.page)
	require.True(t, tester.Focused("Navigation"),
		"closing the palette must hand focus to the sidebar")
}

// TestCommandPaletteExposesTheActiveRow keeps the keyboard highlight from
// being visual-only. The search field owns typing, while its active descendant
// names the row the arrows are on and that row's description states the action.
func TestCommandPaletteExposesTheActiveRow(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 900)
	tester.Frame()
	tester.Key(platformRefreshModifier(), ui.KeyK)
	tester.Frame()

	_, ok := tester.Find(tr("Pages and commands"))
	require.True(t, ok, "the result list must have an accessible name")
	tester.Key(0, ui.KeyDown)
	tester.Frame()
	require.Equal(t, 1, app.paletteChoice)

	source, err := os.ReadFile("palette.go")
	require.NoError(t, err)
	require.True(t, strings.Contains(string(source), "search.ActiveDescendant(row)"),
		"the focused search field must point at the highlighted command")
	require.True(t, strings.Contains(string(source), "Description(tr(entry.detail))"),
		"each command must describe what selecting it does")
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

// TestStatusAppearanceKeepsPageFocus keeps a result from stealing the keyboard
// from the control that produced it. The status strip is inserted above the
// page body, so an unkeyed body looked like a different subtree to MyGo and
// focus fell back to the window root.
func TestStatusAppearanceKeepsPageFocus(t *testing.T) {
	app := New(&fakeServices{})
	app.token.owner = "owner-pk"
	app.token.ownerPriv = "owner-sk"
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage(pageIDs[pageToken])
	tester.Frame()

	copyLabel := tr("Copy Owner private key")
	for range 80 {
		if tester.Focused(copyLabel) {
			break
		}
		tester.Key(0, ui.KeyTab)
		tester.Frame()
	}
	require.True(t, tester.Focused(copyLabel), "the Copy action must be keyboard reachable")
	require.True(t, app.token.status.empty())

	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	require.False(t, app.token.status.empty(), "Copy must create the status strip")
	require.True(t, tester.Focused(copyLabel),
		"inserting the status strip must not remove focus from the page body")
}

// TestLoadFailuresAreStatusRegions keeps a failed refresh from being only a
// red line painted above retained rows. The warning and whether those rows are
// stale form one status message, announced once per distinct failure.
func TestLoadFailuresAreStatusRegions(t *testing.T) {
	fake := &fakeServices{draws: []DrawRow{{
		ID: "draw-1", BlockHeight: 1, Seed: "seed",
		Participants: []string{"Alice"}, Winners: []string{"Alice"},
	}}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 1800)
	app.ShowPage("Lottery")
	tester.Frame()
	tester.Announcements()

	announcement := tr("Failed to load history") + ": database is locked " +
		tr("Showing the last successful load.")
	fake.loadErrs.history = errors.New("database is locked")
	app.refresh()
	tester.Frame()
	require.Equal(t, []string{announcement}, tester.Announcements())

	tester.Frame()
	require.Empty(t, tester.Announcements(), "an unrelated repaint must not repeat a load failure")

	fake.loadErrs.history = nil
	app.refresh()
	tester.Frame()
	require.Empty(t, tester.Announcements())

	fake.loadErrs.history = errors.New("database is locked")
	app.refresh()
	tester.Frame()
	require.Equal(t, []string{announcement}, tester.Announcements(),
		"a later failure after a good load must be announced again")
}

// TestBackgroundWorkIsStatusAndAnnouncedOnce gives the busy banner the same
// treatment as an action result: assistive technology hears that work started,
// then the one-time warning when it has run unusually long.
func TestBackgroundWorkIsStatusAndAnnouncedOnce(t *testing.T) {
	app := New(&fakeServices{})
	app.SetBackgroundWrites(true)
	tester := ui.NewTester(app.View, 1280, 900)
	app.ShowPage("Token")
	tester.Frame()
	tester.Announcements()

	started := make(chan struct{})
	release := make(chan struct{})
	app.runWrite("Minting…", func() func() {
		close(started)
		<-release
		return func() {}
	})
	<-started
	tester.Frame()
	require.Equal(t, []string{tr("Minting…")}, tester.Announcements())
	tester.Frame()
	require.Empty(t, tester.Announcements(), "the running banner must be announced only once")

	app.busySince = time.Now().Add(-21 * time.Second)
	tester.Frame()
	slow := tr("Still working…") + " " +
		tr("This can take a while; the operation will finish on its own.")
	require.Equal(t, []string{slow}, tester.Announcements())
	tester.Frame()
	require.Empty(t, tester.Announcements(), "the long-running warning must also be announced once")

	close(release)
	tester.Advance(60 * time.Millisecond)
	tester.Frame()
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
	app := New(workbenchFakes())
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
	// One voter keeps the session form on the page: an untouched store opens
	// on the registration path instead.
	app := New(&fakeServices{voterRows: []VoterRow{{PublicKey: "voter-pk-1", Name: "Ada"}}})
	app.ShowPage(pageIDs[pageVoting])
	tester := ui.NewTester(app.View, 1280, 1800)
	tester.SetLocale("en")
	tester.Frame()

	for _, label := range []string{"Start date", "Start time", "End date", "End time"} {
		_, ok := tester.Find(tr(label))
		require.Truef(t, ok, "the session form must expose %q", label)
	}
}

// TestCollectionsHaveAccessibleNames checks the shared collection path and its
// callsites. MyGo gives a list role an empty name by default, so every
// collection must pass through collectionList instead of drawing an anonymous
// group of records.
func TestCollectionsHaveAccessibleNames(t *testing.T) {
	tester := ui.NewTester(func(c *ui.Context) {
		collectionList(c, "Draw history", []collectionRecord{
			{title: "draw-1", facts: []collectionFact{{label: "Block", value: "7"}}},
		}, -1, nil)
	}, 480, 240)
	tester.Frame()
	_, ok := tester.Find(tr("Draw history"))
	require.True(t, ok, "a collection must expose its accessible name")
	require.True(t, tester.HasText("draw-1"), "the collection must draw its record")

	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") ||
			strings.HasSuffix(name, "_test.go") || name == "collection.go" {
			continue
		}
		source, err := os.ReadFile(name)
		require.NoError(t, err)
		require.Zerof(t, strings.Count(string(source), "ui.Table("),
			"%s builds a table; collections are records drawn by collectionList", name)
	}

	controls, err := os.ReadFile("collection.go")
	require.NoError(t, err)
	require.Contains(t, string(controls), `Role(ui.RoleList).Label(tr(name))`,
		"collectionList must name the list it draws")
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

// TestValidatedFieldsAttachErrorsToTheirControls keeps an error from becoming
// an unrelated red line below a field. MyGo's Field.Error marks the control
// invalid and puts the message in its accessible description.
func TestValidatedFieldsAttachErrorsToTheirControls(t *testing.T) {
	value := ""
	message := "Voter name is required."
	tester := ui.NewTester(func(c *ui.Context) {
		validatedField(c, "Voter Name", &value, &message)
	}, 480, 180)
	tester.Frame()
	require.True(t, tester.HasText(tr("Voter name is required.")))

	controls, err := os.ReadFile("controls.go")
	require.NoError(t, err)
	require.True(t, strings.Contains(string(controls), "attachFieldError(c, f, label, message)"),
		"validated fields must pass their error through the shared field-error path")
	require.True(t, strings.Contains(string(controls), "c.Announce(fmt.Sprintf(\"%s: %s: %s\", tr(\"Failed\"), tr(label), localized))"),
		"a new validation error must be announced even while focus remains on Submit")
}

// TestValidatedFieldClearsErrorWhenEdited prevents a corrected field from
// keeping the previous attempt's invalid state until the next submit.
func TestValidatedFieldClearsErrorWhenEdited(t *testing.T) {
	value := ""
	message := "Voter name is required."
	tester := ui.NewTester(func(c *ui.Context) {
		validatedField(c, "Voter Name", &value, &message)
	}, 480, 180)
	tester.Frame()
	require.True(t, tester.HasText(tr("Voter name is required.")))

	require.NoError(t, tester.Click("Voter Name"))
	tester.Type("Alice")
	tester.Frame()
	require.Equal(t, "Alice", value)
	require.Empty(t, message, "editing the control must clear its old validation")
	require.False(t, tester.HasText(tr("Voter name is required.")),
		"the stale error must leave the visual field with the input")
}

// TestFieldErrorIsAnnouncedOnce covers the part an accessible description
// cannot: focus stays on the submit button, so the new error needs a live
// announcement. Rebuilding the same error must stay quiet.
func TestFieldErrorIsAnnouncedOnce(t *testing.T) {
	value := ""
	message := "Candidate name is required."
	tester := ui.NewTester(func(c *ui.Context) {
		validatedField(c, "Candidate Name", &value, &message)
	}, 480, 180)
	tester.Frame()

	want := tr("Failed") + ": " + tr("Candidate Name") + ": " + tr("Candidate name is required.")
	require.Equal(t, []string{want}, tester.Announcements())
	tester.Frame()
	require.Empty(t, tester.Announcements(), "an unchanged error must not repeat")

	require.NoError(t, tester.Click("Candidate Name"))
	tester.Type("Alice")
	tester.Frame()
	require.Empty(t, tester.Announcements(), "clearing an error is not a failure to announce")

	message = "Candidate name is required."
	tester.Frame()
	require.Equal(t, []string{want}, tester.Announcements(),
		"the same error must be announced when it appears again")
}

// TestVisibleHintsDescribeTheirControls keeps help text in the accessibility
// tree after it moves below the control. The line must remain visible and also
// describe the specific input, grid or toggle it explains.
func TestVisibleHintsDescribeTheirControls(t *testing.T) {
	value, longValue := "", ""
	tester := ui.NewTester(func(c *ui.Context) {
		fieldHint(c, "Owner", "Owner hint.", &value)
		textAreaHint(c, "Headers", "Headers hint.", &longValue)
	}, 480, 260)
	tester.Frame()
	require.True(t, tester.HasText(tr("Owner hint.")))
	require.True(t, tester.HasText(tr("Headers hint.")))

	checks := map[string][]string{
		"controls.go": {
			"ui.TextInput(c, value).Description(tr(hintText))",
			"textAreaControl(c, value).Description(tr(hintText))",
		},
		"voting_setup.go": {
			`Description(tr("Local time. Ballots are rejected before it."))`,
			`Description(tr("Local time. Ballots are rejected after it."))`,
		},
		"lottery.go": {
			`Description(tr("How many winners to draw; must not exceed the participant count."))`,
		},
		"token.go": {
			`Description(tr("0 for a whole-unit token; 8 matches most asset conventions."))`,
		},
		"oracle_add.go": {
			`Description(tr("0 uses the default of 60; at most 30 days."))`,
		},
		"nft.go": {},
		"settings.go": {
			`Description(tr("Nothing is written to disk; quitting Aurora clears them."))`,
			`Description(tr("Applies to Dashboard, Oracle health and open readings; forms keep their text. Saved for the next launch."))`,
			`button.Description(tr(description))`,
		},
	}
	for name, fragments := range checks {
		source, err := os.ReadFile(name)
		require.NoError(t, err)
		for _, fragment := range fragments {
			require.Containsf(t, string(source), fragment,
				"%s must expose its visible hint as a control description", name)
		}
	}
}

// TestSearchFieldsAnnounceResultCounts covers the shared search interaction:
// a filtered repaint is visual, so every advertised search field must also
// report how many rows the new query left.
func TestSearchFieldsAnnounceResultCounts(t *testing.T) {
	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	searches, announcements := 0, 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") ||
			strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, err := os.ReadFile(name)
		require.NoError(t, err)
		// A page either builds its search field directly or leads with the
		// shared collection toolbar, which builds one for it. Both spellings
		// are views that must announce what their query selected.
		body := string(source)
		// controls.go defines both helpers; its own calls are not page-level
		// searches, so a page file is the one that counts.
		if name != "controls.go" {
			searches += strings.Count(body, "searchField(c,")
			searches += strings.Count(body, "collectionToolbar(c,")
			announcements += strings.Count(body, "announceSearchResults(c,")
		}
	}
	require.GreaterOrEqual(t, searches, 9)
	require.Equal(t, searches, announcements,
		"every SearchField must announce its result count")
}

// TestPagerMoveAnnouncesTheNewRange covers the other collection change that
// leaves focus on the initiating button: Next or Previous replaces every row,
// so the visible range must be spoken once when the successful page lands.
func TestPagerMoveAnnouncesTheNewRange(t *testing.T) {
	const pageSize = 100
	tokens := make([]TokenRow, pageSize+1)
	for i := range tokens {
		tokens[i] = TokenRow{
			ID: "token-" + strconv.Itoa(i), Name: "Token " + strconv.Itoa(i),
			Symbol: "TKN", TotalSupply: "1", Owner: "owner-pk",
		}
	}
	fake := &fakeServices{tokenRows: tokens}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 2200)
	app.ShowPage("Token")
	tester.Frame()
	tester.Announcements() // the page transition
	require.True(t, tester.HasText(tr("Token collection pager, Entries 1–100")))

	require.NoError(t, tester.Click("Next"))
	tester.Frame()
	require.Equal(t, []string{tr("Entries 101–101")}, tester.Announcements())
	require.True(t, tester.HasText(tr("Token collection pager, Entries 101–101")))
	tester.Frame()
	require.Empty(t, tester.Announcements(), "repainting the new page must stay silent")

	require.NoError(t, tester.Click("Previous"))
	tester.Frame()
	require.Equal(t, []string{tr("Entries 1–100")}, tester.Announcements())

	fake.loadErrs.tokens = errors.New("database is locked")
	require.NoError(t, tester.Click("Next"))
	tester.Frame()
	require.Zero(t, app.token.offset, "a failed pager move must keep the loaded page")
	require.NotContains(t, tester.Announcements(), tr("Entries 101–101"),
		"the pager must not claim a page that failed to load")
}

// TestVotingPagersNameTheirCollections distinguishes the three pagers that
// share the Voting setup page. Without the collection name, every group and
// both buttons sound identical while tabbing or navigating by controls.
func TestVotingPagersNameTheirCollections(t *testing.T) {
	fake := &fakeServices{
		voterRows:  []VoterRow{{PublicKey: "pk-voter", Name: "Alice"}},
		candidates: []CandidateRow{{ID: "cand-1", Name: "Alice"}},
		sessions: []SessionRow{{
			ID: "session-1", Title: "Budget", Status: "draft",
			StartTime: time.Now().Unix(), EndTime: time.Now().Add(time.Hour).Unix(),
		}},
	}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 7000)
	app.ShowPage("Voting")
	tester.Frame()

	for _, label := range []string{
		"Voters pager, Entries 1–1",
		"Candidates pager, Entries 1–1",
		"Sessions pager, Entries 1–1",
	} {
		require.Truef(t, tester.HasText(tr(label)), "missing accessible pager label %q", label)
	}
}

// TestLedgerPagerAnnouncesTheNewRange applies the shared pager behavior to the
// Blockchain page. Its pager has an exact total, so it proves the shared strip
// also announces ranges such as "11–20 of 150" rather than only "11–20".
func TestLedgerPagerAnnouncesTheNewRange(t *testing.T) {
	blocks := make([]BlockRow, 150)
	for i := range blocks {
		blocks[i] = BlockRow{
			Height: int64(len(blocks) - i), Module: "lottery",
			Summary: "row-" + strconv.Itoa(i),
		}
	}
	fake := &fakeServices{blocks: blocks}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Blockchain")
	tester.Frame()
	app.blockchain.blockLimit = 100
	tester.Frame()
	tester.Announcements()
	require.True(t, tester.HasText(tr("Blockchain ledger pager, Entries 1–100 of 150")))

	require.NoError(t, tester.Click("Next"))
	tester.Frame()
	require.Equal(t, []string{tr("Entries 101–150 of 150")}, tester.Announcements())
	require.True(t, tester.HasText(tr("Blockchain ledger pager, Entries 101–150 of 150")))
	tester.Frame()
	require.Empty(t, tester.Announcements(), "repainting the ledger page must stay silent")

	require.NoError(t, tester.Click("Previous"))
	tester.Frame()
	require.Equal(t, []string{tr("Entries 1–100 of 150")}, tester.Announcements())

	fake.loadErrs.blocks = errors.New("database is locked")
	require.NoError(t, tester.Click("Next"))
	tester.Frame()
	require.Zero(t, app.blockchain.blockOffset,
		"a failed ledger move must keep the page its rows came from")
	require.NotContains(t, tester.Announcements(), tr("Entries 101–150 of 150"),
		"the ledger pager must not claim a failed page")
}

// TestHeadingsCarrySemanticRanks keeps titles from being only visually bold.
// The toolkit exposes RoleHeading and Level to screen readers; retaining the
// exact call here protects the shared heading path.
func TestHeadingsCarrySemanticRanks(t *testing.T) {
	layout, err := os.ReadFile("layout.go")
	require.NoError(t, err)
	require.Contains(t, string(layout), "Role(ui.RoleHeading).Level(level)")

	app, err := os.ReadFile("chrome.go")
	require.NoError(t, err)
	require.Contains(t, string(app), "heading(c, tr(pageTitles[a.page]), pageTitleSize, 1)")
}
