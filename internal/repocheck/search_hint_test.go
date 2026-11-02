package repocheck

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSearchHintsMatchWhatTheSearchDoes covers the sentence beside every search
// field. Six lists filter the rows already loaded and say so ("Search narrows
// the current page"); two query storage across the whole set and say that
// instead. Swapping the two sentences would leave a reader looking for an old
// record in a page they cannot see, or trusting a filtered count as the whole
// store — the words are the only signal a user has.
//
// The check ties each sentence to the code path that backs it: the adapters that
// take a search string reach storage, and the view filters that call
// containsFold do not.
func TestSearchHintsMatchWhatTheSearchDoes(t *testing.T) {
	// Whole-store searches: each adapter method takes the term as a parameter
	// and hands it to a storage query. The parameter's name differs — one says
	// search, the other term — so the check matches either.
	for _, method := range []string{"LotteryHistory", "LedgerBlocksPage"} {
		found := false
		require.NoError(t, walkGoSource(filepath.Join(repoRoot(t), "desktop", "internal", "deskapp"), func(path, body string) {
			if regexp.MustCompile(`\) ` + method + `\((?:search|term) string`).MatchString(body) {
				found = true
			}
		}))
		assert.Truef(t, found, "%s must still take the search term and query storage", method)
	}

	// Page-local searches: the view filters what it loaded.
	view := readRepoFile(t, "desktop", "internal", "nativeui", "voting_lists.go") +
		readRepoFile(t, "desktop", "internal", "nativeui", "nft_collection.go") +
		readRepoFile(t, "desktop", "internal", "nativeui", "oracle_sources.go") +
		readRepoFile(t, "desktop", "internal", "nativeui", "token.go")
	require.GreaterOrEqual(t, strings.Count(view, "containsFold("), 5,
		"the page-local lists must still filter in memory")

	// Every hint is one of the two sentences, and the whole-store hint appears
	// only where storage is queried.
	hint := regexp.MustCompile(`"Search (?:narrows the current page|matches [^"]*across the whole [a-z]+)[^"]*"`)
	hints := hint.FindAllString(view+
		readRepoFile(t, "desktop", "internal", "nativeui", "lottery.go")+
		readRepoFile(t, "desktop", "internal", "nativeui", "blockchain.go"), -1)
	require.GreaterOrEqual(t, len(hints), 8, "every list must carry a search hint")

	wholeStore := 0
	for _, h := range hints {
		if strings.Contains(h, "across the whole") {
			wholeStore++
		}
	}
	assert.Equal(t, 2, wholeStore,
		"exactly the two lists that query storage may promise a whole-store search")
}

// TestAppliedQueryIsRecordedBeforeTheEmptyReturn covers the ordering that six
// window lists had wrong. The field records what the reader typed, so it has to
// be updated before a list returns early over an empty page: skipped there, a
// query cleared while the store was empty stayed applied, which hid each page's
// first-record guide and made the next read announce itself as a search result.
//
// The check is structural because the symptom is not: each guard must be
// preceded by a comparison of the search field against its applied copy, within
// the same function.
func TestAppliedQueryIsRecordedBeforeTheEmptyReturn(t *testing.T) {
	cases := []struct {
		file  string
		guard string
		sync  string
	}{
		{"voting_lists.go", "if len(a.voting.voters) == 0 {", "a.voting.voterSearch != a.voting.voterApplied"},
		{"voting_lists.go", "if len(a.voting.candidates) == 0 {", "a.voting.candSearch != a.voting.candApplied"},
		{"voting_lists.go", "if len(a.voting.sessions) == 0 {", "a.voting.sessionSearch != a.voting.sessionApplied"},
		{"token.go", "if len(a.token.tokens) == 0 {", "a.token.search != a.token.appliedSearch"},
		{"nft_collection.go", "if len(a.nft.list) == 0 {", "a.nft.search != a.nft.appliedSearch"},
		{"oracle_sources.go", "if len(a.oracle.sources) == 0 {", "a.oracle.search != a.oracle.appliedSearch"},
	}
	for _, c := range cases {
		body := readRepoFile(t, "desktop", "internal", "nativeui", c.file)
		guard := strings.Index(body, c.guard)
		require.NotEqualf(t, -1, guard, "%s must keep its empty-list guard", c.file)

		// The comparison has to run whether or not the list is empty, so it may
		// sit inside the guard's block or before the guard altogether — what it
		// must not do is sit only after it. Look to the end of the enclosing
		// function and require the first occurrence to come before the guard's
		// own body ends.
		start := strings.LastIndex(body[:guard], "\nfunc ")
		require.NotEqualf(t, -1, start, "%s: the guard must sit inside a function", c.file)
		// The guard's own block ends at the matching closing brace of its `if`,
		// which for these one-level guards is the first "\n\t}" after it.
		close := strings.Index(body[guard:], "\n\t\t}")
		require.NotEqualf(t, -1, close, "%s: the guard must close at its own indentation", c.file)
		block := body[guard : guard+close]
		if !strings.Contains(body[start:guard], c.sync) && !strings.Contains(block, c.sync) {
			t.Errorf("%s: %q is not compared before or inside the empty-list guard; "+
				"a query cleared over an empty page would not be recorded",
				c.file, c.sync)
		}
	}
}

// TestLoadersCommitTheirOffsetOnlyAfterTheRead covers the rule the paging
// invariants state: a loader commits the rows, the offset and HasMore together,
// and only once the read succeeded. A commit moved above the error return would
// leave the page label describing rows that were never loaded — the failure the
// window reports as stale would point at the wrong page.
//
// The check is structural because every loader reads correctly today: the order
// is what a refactor can silently change.
func TestLoadersCommitTheirOffsetOnlyAfterTheRead(t *testing.T) {
	cases := []struct {
		file   string
		commit string
	}{
		{"token.go", "a.token.offset = offset"},
		{"nft.go", "a.nft.offset = offset"},
		{"nft_history.go", "a.nft.historyOffset = offset"},
		{"oracle.go", "a.oracle.offset = offset"},
		{"lottery.go", "a.lottery.offset = offset"},
		{"voting.go", "a.voting.voterOffset = offset"},
		{"voting.go", "a.voting.candOffset = offset"},
		{"voting.go", "a.voting.sessionOffset = offset"},
		{"blockchain.go", "a.blockchain.blockOffset = offset"},
	}
	for _, c := range cases {
		body := readRepoFile(t, "desktop", "internal", "nativeui", c.file)
		commit := strings.Index(body, c.commit)
		require.NotEqualf(t, -1, commit, "%s must still commit %s", c.file, c.commit)

		// The enclosing function's error return has to come first.
		start := strings.LastIndex(body[:commit], "\nfunc ")
		require.NotEqualf(t, -1, start, "%s: the commit must sit inside a function", c.file)
		before := body[start:commit]
		if !strings.Contains(before, "if err != nil {") {
			t.Errorf("%s: %s is committed without an error guard before it; "+
				"a failed read would leave the offset describing rows that were never loaded",
				c.file, c.commit)
		}
	}
}

// TestHighlightLookupsClearBeforeSearching covers the nine helpers that move a
// list's highlight to a named record. A lookup that misses must leave nothing
// highlighted: the write cards read their subject from that highlight, so a
// miss that keeps the old index points the next transfer, burn or ballot at a
// record the operator never chose.
//
// The check is structural because a miss is not reachable from the suite's
// happy paths, and because the same shape has been written wrong twice — the
// select helpers lacked the clear the restore helpers had.
func TestHighlightLookupsClearBeforeSearching(t *testing.T) {
	helpers := []struct {
		file string
		name string
	}{
		{"oracle.go", "restoreSourceChoice"},
		{"voting.go", "restoreVoterChoice"},
		{"voting.go", "restoreCandidateChoice"},
		{"voting.go", "restoreSessionChoice"},
		{"nft_collection.go", "selectNFTRow"},
		{"lottery.go", "selectLotteryByID"},
		{"oracle_add.go", "selectSourceByID"},
		{"token.go", "selectTokenByID"},
		{"voting_setup.go", "selectCandidateByID"},
		{"voting_setup.go", "selectSessionByID"},
		{"voting_setup.go", "selectVoterByPublicKey"},
	}
	for _, h := range helpers {
		body := readRepoFile(t, "desktop", "internal", "nativeui", h.file)
		start := strings.Index(body, "func (a *App) "+h.name+"(")
		require.NotEqualf(t, -1, start, "%s: %s must still exist", h.file, h.name)
		rest := body[start:]
		end := strings.Index(rest, "\nfunc ")
		if end < 0 {
			end = len(rest)
		}
		fn := rest[:end]

		// The clear is the first assignment in the function, before the loop
		// that may return early on a hit.
		loop := strings.Index(fn, "\tfor ")
		require.NotEqualf(t, -1, loop, "%s: %s must still search a list", h.file, h.name)
		if !strings.Contains(fn[:loop], "= -1") {
			t.Errorf("%s: %s searches without clearing the highlight first; "+
				"a miss would leave the previous row selected", h.file, h.name)
		}
	}
}

// TestEveryDestructiveActionAsksFirst covers the rule that a write which cannot
// be undone goes through a confirmation. Three of them exist — burning tokens,
// burning an asset, deleting a source — and the settings page's "forget keys" is
// the fourth. Each pairs a danger button with destructiveConfirm, and the pairing
// is what a refactor can drop: the button alone still works.
//
// The check is structural because a missing dialog is only visible by clicking,
// and because the confirmation is the one gate these actions have.
func TestEveryDestructiveActionAsksFirst(t *testing.T) {
	type pair struct {
		file   string
		button string
		ask    string
	}
	pairs := []pair{
		{"nft_write.go", `dangerButton(c, "Burn NFT…"`, "&a.nft.askBurn"},
		{"token_write.go", `dangerButton(c, "Burn…"`, "&a.token.askBurn"},
		{"oracle_sources.go", `dangerButton(c, "Delete"`, "&a.oracle.askDelete"},
		{"settings.go", `dangerButton(c, "Forget session keys"`, "&a.confirmForgetKeys"},
	}
	for _, p := range pairs {
		body := readRepoFile(t, "desktop", "internal", "nativeui", p.file)
		assert.Containsf(t, body, p.button, "%s must still offer the action", p.file)
		assert.Containsf(t, body, p.ask,
			"%s must open a confirmation for %s", p.file, p.button)
		assert.Containsf(t, body, "destructiveConfirm(c, "+p.ask,
			"%s must route that confirmation through destructiveConfirm", p.file)
	}

	// The dialog itself has to carry the question for assistive technology: a
	// modal that draws a title but exposes no role is silent to a screen reader.
	confirm := readRepoFile(t, "desktop", "internal", "nativeui", "confirmation.go")
	for _, want := range []string{"ui.RoleAlertDialog", ".Label(label)", ".Description(detail)"} {
		assert.Containsf(t, confirm, want, "the confirmation must expose %s", want)
	}
}

// TestTheDocumentedShortcutsAreRegistered covers the keyboard promises in
// docs/desktop.md. They are registered on the window rather than on buttons, so a
// refactor that moves the header can drop one without any page looking wrong —
// the action is still there, just no longer reachable by its chord.
func TestTheDocumentedShortcutsAreRegistered(t *testing.T) {
	pages := readRepoFile(t, "desktop", "internal", "nativeui", "pages.go")
	order := pageOrderEntry(pages)
	require.Len(t, order, 8,
		"the sidebar has eight pages, so 1…8 are the chords the document promises")
	assert.Equal(t, "pageSettings", order[7], "Settings is the eighth, as the document says")

	header := readRepoFile(t, "desktop", "internal", "nativeui", "chrome.go")
	for _, want := range []struct{ what, code string }{
		{"1…8 select pages", "for i, p := range pageOrder {"},
		{"the digits come from Key1", "ui.Key(int(ui.Key1)+i)"},
		{"Ctrl/Cmd+, opens Settings", "c.Shortcut(platformRefreshModifier(), ui.KeyComma)"},
		{"Ctrl/Cmd+K opens the palette", "c.Shortcut(platformRefreshModifier(), ui.KeyK)"},
	} {
		assert.Containsf(t, header, want.code, "chrome.go must keep the shortcut: %s", want.what)
	}
}

// pageOrderEntry reads the pageOrder slice literal back out of pages.go.
func pageOrderEntry(src string) []string {
	start := strings.Index(src, "var pageOrder = []page{")
	if start < 0 {
		return nil
	}
	rest := src[start:]
	end := strings.Index(rest, "}")
	if end < 0 {
		return nil
	}
	var out []string
	for _, field := range strings.Split(rest[len("var pageOrder = []page{"):end], ",") {
		if name := strings.TrimSpace(field); name != "" {
			out = append(out, name)
		}
	}
	return out
}

// TestCrossFieldErrorsNameTheFieldTheyBlame covers the half of field validation
// that a single-field check cannot: a rule spanning two inputs still has to put
// its message on the one the reader must change. The session window is the case
// the reference names — an end before the start blames End, a missing start
// blames Start — and misplacing it sends the reader to edit a field that is fine.
func TestCrossFieldErrorsNameTheFieldTheyBlame(t *testing.T) {
	body := readRepoFile(t, "desktop", "internal", "nativeui", "voting_setup.go")
	start := strings.Index(body, "func (a *App) createVotingSession()")
	require.NotEqual(t, -1, start, "createVotingSession must still exist")
	fn := body[start:]
	if end := strings.Index(fn, "\nfunc "); end >= 0 {
		fn = fn[:end]
	}

	for _, want := range []struct{ rule, assign string }{
		{"a missing start blames Start", `a.voting.sessionStartErr = "Choose a start time."`},
		{"a missing end blames End", `a.voting.sessionEndErr = "Choose an end time."`},
		{"an end before the start blames End", `a.voting.sessionEndErr = "End time must be after the start time."`},
	} {
		assert.Containsf(t, fn, want.assign, "%s", want.rule)
	}
	// The reversed-window message must not blame the start: Start is the value the
	// reader typed correctly, and the sentence says to fix the end.
	assert.NotContainsf(t, fn, `a.voting.sessionStartErr = "End time must`,
		"the reversed-window error must blame End, not Start")
}

// TestEverySettingsWriteReportsItsFailure covers the three Settings choices that
// persist to desktop-settings.json: language, theme mode and live updates. Each
// applies in the window first and then reports a failed write, because a silent
// failure looks like success until the next launch reverts it — and the operator
// has no way to tell that from having misremembered.
func TestEverySettingsWriteReportsItsFailure(t *testing.T) {
	settings := readRepoFile(t, "desktop", "internal", "nativeui", "settings.go")
	for _, want := range []struct{ what, code string }{
		{"the language chooser reports through a handler", "reportLanguageSaveError(a)"},
		{"a failed theme write reports", "a.actions.SetThemeMode(mode); err != nil {"},
		{"a failed live-updates write reports", "a.actions.SetLiveUpdates(a.liveUpdates); err != nil {"},
	} {
		assert.Containsf(t, settings, want.code, "settings.go must keep the failure path: %s", want.what)
	}

	// The message has to name the write, not the value: "Couldn't save settings"
	// is what tells the reader the choice will not survive a restart.
	assert.Contains(t, settings, `tr("Couldn't save settings")`,
		"the failure line must say the settings could not be saved")
}

// TestTheBusyGateCoversEveryEntryPoint covers the four places a repository
// operation can be started from. Each checks the same `busy` flag, and each does
// it differently — the File menu button is disabled, the shortcuts return early,
// the palette omits the entries, the Refresh button is disabled — so a refactor
// that keeps three and drops one leaves a way to start a second operation against
// the same store.
func TestTheBusyGateCoversEveryEntryPoint(t *testing.T) {
	actions := readRepoFile(t, "desktop", "internal", "nativeui", "actions.go")
	for _, want := range []struct{ what, code string }{
		{"the File menu button is disabled", "Disabled(a.busy != \"\")"},
		{"the file shortcuts return before reading them", "func (a *App) actionShortcuts(c *ui.Context) {\n\tif a.busy != \"\" {"},
	} {
		assert.Containsf(t, actions, want.code, "actions.go must keep the gate: %s", want.what)
	}

	palette := readRepoFile(t, "desktop", "internal", "nativeui", "palette.go")
	assert.Contains(t, palette, "if a.busy == \"\" {",
		"palette.go must keep the reload and file entries out while a write runs")

	chrome := readRepoFile(t, "desktop", "internal", "nativeui", "chrome.go")
	assert.Contains(t, chrome, "refreshButton(c).Disabled(a.busy != \"\")",
		"chrome.go must disable Refresh while a write runs")

	// Navigation is deliberately outside the gate: a page switch does not touch
	// the store, and the palette test asserts the page entries survive.
	assert.Contains(t, palette, "for _, p := range pageOrder {",
		"the palette must keep listing pages regardless of the gate")
}

// TestThePagerDisablesItsEnds covers the two conditions a reader sees rather than
// a test can catch by clicking. The offset clamp already keeps the store from being
// asked for a page before the first or past the end, so moving the disabled
// condition changes nothing observable — it only offers a button that does nothing.
// A control that looks live and is not is the failure this pins.
func TestThePagerDisablesItsEnds(t *testing.T) {
	controls := readRepoFile(t, "desktop", "internal", "nativeui", "controls.go")

	body := functionSource(t, controls, "func pagerInline")
	assert.Contains(t, body, "Disabled(offset == 0)",
		"Previous must be disabled on the first page")
	assert.Contains(t, body, "Disabled(!hasMore)",
		"Next must be disabled when the store says there is no next page")
}

// functionSource returns the source of the named function in src, from its
// declaration to the next one at the same level.
func functionSource(t *testing.T, src, declaration string) string {
	t.Helper()
	start := strings.Index(src, declaration)
	require.NotEqualf(t, -1, start, "%s must still exist", declaration)
	rest := src[start+len(declaration):]
	if end := strings.Index(rest, "\nfunc "); end >= 0 {
		rest = rest[:end]
	}
	return rest
}

// TestEveryRecordListIsNamed covers the promise that a table or record list says
// which records it holds before reading the first one. collectionList names its
// own list from the argument every caller passes; the lists built by hand have to
// set Role and Label themselves, which is where one was missing — the ledger's
// records, whose pager was named while the list under it was not.
func TestEveryRecordListIsNamed(t *testing.T) {
	// Files that build a list by hand rather than through collectionList.
	handBuilt := []struct{ file, label string }{
		{"blockchain.go", `tr("Blockchain ledger")`},
		{"voting_results.go", `tr("Voting results")`},
		{"layout.go", `tr("What opens next")`},
	}
	for _, want := range handBuilt {
		body := readRepoFile(t, "desktop", "internal", "nativeui", want.file)
		assert.Containsf(t, body, "Role(ui.RoleList)", "%s must mark its list as a list", want.file)
		assert.Containsf(t, body, want.label, "%s must name that list", want.file)
	}

	// The shared builder names from its argument; a caller passing an empty name
	// would produce an unnamed list that no assertion above would see.
	files := []string{
		"lottery.go", "nft_collection.go", "nft_history.go", "oracle_health.go",
		"oracle_readings.go", "oracle_sources.go", "token.go", "token_read.go",
		"voting_lists.go", "voting_results.go",
	}
	for _, file := range files {
		body := readRepoFile(t, "desktop", "internal", "nativeui", file)
		for _, call := range strings.Split(body, "collectionList(c, ")[1:] {
			if strings.HasPrefix(call, "\"\"") || strings.HasPrefix(call, "tr(\"\")") {
				t.Errorf("%s calls collectionList with an empty name", file)
			}
		}
	}
}

// TestEveryChoiceGroupIsNamed covers the labelled groups a screen reader announces
// before its options: radio groups, tab rows and checkboxes. Each renders a set of
// controls that are indistinguishable without the group's name — three radios
// reading "Follow System", "Light", "Dark" say nothing about what they choose.
//
// The two radio groups build their chain differently (one is a helper, one is
// inline), so the check is on the presence of a Label in the same file rather than
// on a shape both happen to share.
func TestEveryChoiceGroupIsNamed(t *testing.T) {
	settings := readRepoFile(t, "desktop", "internal", "nativeui", "settings.go")
	assert.Contains(t, settings, `.Label(tr("Theme mode"))`,
		"the theme radios must name what they choose")
	assert.Contains(t, settings, `.Label(tr("Display language"))`,
		"the language radios must name what they choose")

	// Checkboxes take their name as the second argument, so an empty one is the
	// failure shape rather than a missing Label.
	for _, want := range []string{
		`ui.Checkbox(c, &remember, tr("Remember private keys for this session"))`,
		`ui.Checkbox(c, &a.liveUpdates, tr("Every 10 seconds"))`,
	} {
		assert.Contains(t, settings, want, "a checkbox must carry its own name")
	}

	// tabs() names its row from the argument the caller passes; every call site
	// has to supply one.
	for _, file := range []string{
		"nft_write.go", "oracle.go", "token_write.go", "voting_setup.go",
	} {
		body := readRepoFile(t, "desktop", "internal", "nativeui", file)
		start := strings.Index(body, "tabs(c, ")
		require.NotEqualf(t, -1, start, "%s must still call tabs", file)
		call := body[start:]
		if end := strings.Index(call, ")\n"); end > 0 {
			call = call[:end]
		}
		assert.Containsf(t, call, `", tr(`,
			"%s must pass a group name before the tab labels", file)
	}
}

// TestFieldErrorsMarkTheControlInvalid covers the line that makes an error part of
// the field rather than a red sentence near it. MyGo's Error does three things:
// draws the message below the control, marks the control invalid so its border
// turns red and assistive technology reports AccessInvalid, and prepends the
// message to the control's description.
//
// Only the first is visible to Tester: the frame exposes text, not the invalid
// state, so the other two can only be pinned where they are asked for.
func TestFieldErrorsMarkTheControlInvalid(t *testing.T) {
	controls := readRepoFile(t, "desktop", "internal", "nativeui", "controls.go")
	body := functionSource(t, controls, "func attachFieldError")

	assert.Contains(t, body, "field.Error(localized)",
		"the message must reach the field, or the control is never marked invalid")
	// The text is drawn by Error, so drawing it again here would print it twice.
	assert.NotContains(t, body, "ui.Text(c, localized)",
		"the message is drawn by Field.Error, not by a second text element")
}

// TestOnlyTheDestructiveButtonFillsWithTheDangerColour covers the rule that makes
// the three status colours mean something. Danger is the page's "this will not
// come back" ink; a primary action filled with it would make an ordinary write
// look destructive, and a red fill anywhere but the destructive button would make
// the button itself less of a warning.
//
// Text and borders may use it freely — a failed load, a field error — because
// those are failures being reported. This is about the fill.
func TestOnlyTheDestructiveButtonFillsWithTheDangerColour(t *testing.T) {
	controls := readRepoFile(t, "desktop", "internal", "nativeui", "controls.go")
	destructive := functionSource(t, controls, "func dangerButton")
	require.Contains(t, destructive, "red := c.Theme().Danger",
		"the destructive button is the one control filled with Danger")
	require.Contains(t, destructive, "fill, textColor, glow := red",
		"and it uses it as the fill, not only as an edge")

	primary := functionSource(t, controls, "func primaryButton")
	require.NotContains(t, primary, "Danger",
		"an ordinary primary action must take the module accent, never the danger fill")
	require.NotContains(t, functionSource(t, controls, "func quietButton"), "Danger",
		"a quiet button must not be tinted as destructive")
}

// TestTheTimelineSpineRunsThroughItsDots covers a geometry the frame cannot show:
// the dashboard timeline paints its spine with a Painter call, and a Painter call
// leaves no element for Tester to measure. The numbers are what keep the spine
// centred under each dot — the dot column is 18 wide, so its centre is 9, and the
// 1px spine at 8.5 spans 8.5..9.5, whose centre is also 9.
//
// Changing either number alone puts the spine beside the dots, which reads as two
// vertical rules rather than one line through the events.
func TestTheTimelineSpineRunsThroughItsDots(t *testing.T) {
	dashboard := readRepoFile(t, "desktop", "internal", "nativeui", "dashboard.go")

	// The dot column's width and the spine's offset, as written.
	require.Contains(t, dashboard, ".Width(18)",
		"the timeline's dot column must keep its stated width")
	require.Contains(t, dashboard, "X: r.X + 8.5",
		"the spine must stay centred in that column")

	// And the dot that sits in it.
	require.Contains(t, dashboard, "statusDot(c, tone, 9)",
		"the timeline node must keep the size the spine is centred for")
}

// TestTheSpacingScaleIsOneArithmeticRun keeps the layout scale a scale. Six of
// its steps run 4 apart, so a card's gap, its padding and the space above a
// heading are drawn from measurements a reader can compare; a step nudged by two
// would make two cards that should look equally spaced look unevenly spaced.
//
// space7 is 32 rather than the next step of 28, and deliberately so: it is the one
// empty-state inset, a jump out of the rhythm for a panel that is alone on the
// page. It is not part of the run this test measures.
//
// The check also fails if a step is dropped: callers spell these names, so a
// missing constant is a build error, but a scale with a hole in it — four steps
// where the layout uses six — is not.
func TestTheSpacingScaleIsOneArithmeticRun(t *testing.T) {
	design := readRepoFile(t, "desktop", "internal", "nativeui", "design.go")

	steps := []struct {
		name string
		want string
	}{
		{"space1", "4"}, {"space2", "8"}, {"space3", "12"},
		{"space4", "16"}, {"space5", "20"}, {"space6", "24"},
	}
	for _, step := range steps {
		require.Containsf(t, design, step.name+" float32 = "+step.want,
			"%s must be %s", step.name, step.want)
	}

	// The radii are their own short scale, not derived from the spacing: a card's
	// corner and the space inside it answer different questions.
	for _, radius := range []string{"radiusSmall float32 = 8", "radiusPanel float32 = 14", "radiusLarge float32 = 18"} {
		require.Containsf(t, design, radius, "%q must stay on the scale", radius)
	}
}

// TestWarningIconIsOnlyForConfirmations keeps the icon's contract closed. Its
// comment used to claim two uses — an irreversible confirmation and a load that
// kept stale rows — and only the first was true: a failed load draws the same
// status strip as success and busy, with a dot and the word "Failed".
//
// A comment that lists a use the code does not have is worse than no comment: it
// tells the next reader that this shape already covers a case it does not.
func TestWarningIconIsOnlyForConfirmations(t *testing.T) {
	confirm := readRepoFile(t, "desktop", "internal", "nativeui", "confirmation.go")
	require.Contains(t, confirm, "iconTile(c, warningIcon, danger, iconTileMedium)",
		"the destructive dialog is what the triangle marks")

	values := readRepoFile(t, "desktop", "internal", "nativeui", "values.go")
	require.NotContains(t, values, "warningIcon",
		"a failed load shares the status strip's dot and word, not this triangle")

	icons := readRepoFile(t, "desktop", "internal", "nativeui", "icons.go")
	require.Contains(t, icons, "warningIcon marks a confirmation whose action cannot be undone",
		"the comment must name the one use")
	require.NotContains(t, icons, "and a\n\t// load that kept stale rows",
		"the comment must not claim a use the code dropped")
}

// TestEveryActionButtonLooksDisabledBeforeItsFieldsAreReady covers the promise in
// the reference: a primary action is disabled until its fields are filled, rather
// than submitting and reading back an error. The button is the visible half; the
// handlers check the same fields again, which is the half already covered.
//
// The shapes differ by page — a form uses allSet, the Oracle card computes one
// ready flag and passes it to four buttons, Voting passes its own — so the check is
// per shape rather than one pattern.
func TestEveryActionButtonLooksDisabledBeforeItsFieldsAreReady(t *testing.T) {
	controls := readRepoFile(t, "desktop", "internal", "nativeui", "controls.go")
	primary := functionSource(t, controls, "func primaryButton")
	require.Contains(t, primary, "Disabled(!enabled)",
		"a primary button must be disabled when it is not ready")

	// Every call site has to hand it a readiness value, or it is always enabled.
	files := []string{
		"lottery.go", "token.go", "token_write.go", "nft_mint.go", "nft_write.go",
		"voting_setup.go", "voting_ballot.go", "oracle.go", "oracle_add.go",
		"oracle_sources.go",
	}
	unguarded := 0
	for _, file := range files {
		body := readRepoFile(t, "desktop", "internal", "nativeui", file)
		for _, call := range strings.Split(body, "primaryButton(c, ")[1:] {
			if !strings.Contains(call[:min(len(call), 120)], ",") {
				unguarded++
				t.Errorf("%s calls primaryButton without a readiness value", file)
			}
		}
	}
	require.Zero(t, unguarded, "every primary action must state when it is ready")

	// The two cards that act on a chosen record compute one flag for a group.
	oracle := readRepoFile(t, "desktop", "internal", "nativeui", "oracle_sources.go")
	require.Contains(t, oracle, `sourceReady := strings.TrimSpace(a.oracle.fetchID) != ""`,
		"the Oracle actions need a chosen source or a typed id")
	for _, label := range []string{"Enable", "Disable", "Fetch now", "Delete"} {
		require.Containsf(t, oracle, `(c, "`+label+`", sourceReady)`,
			"%s must use the source readiness flag", label)
	}
	ballot := readRepoFile(t, "desktop", "internal", "nativeui", "voting_ballot.go")
	for _, label := range []string{"Start Session", "End Session"} {
		require.Containsf(t, ballot, `(c, "`+label+`", sessionReady)`,
			"%s must use the session readiness flag", label)
	}
}

// TestEveryCopyConfirmsItselfNearby covers the feedback rule in the reference: a
// copy button writes the clipboard and also raises a toast. The status line is
// pinned to the page header, which can be a card away from the value just copied,
// so the toast is what confirms the copy where the pointer already is.
//
// The check is that every clipboard write is followed by a toast in the same
// function. One that only updated the status line would look, from the pointer, like
// a copy that did nothing.
func TestEveryCopyConfirmsItselfNearby(t *testing.T) {
	files := []string{
		"controls.go", "lottery.go", "records.go", "settings.go", "values.go",
	}
	writes := 0
	for _, file := range files {
		body := readRepoFile(t, "desktop", "internal", "nativeui", file)
		for _, after := range strings.Split(body, "WriteClipboard(")[1:] {
			writes++
			// The toast is the next statement in every case; look far enough for
			// a message built across a line or two.
			window := after
			if end := strings.Index(window, "\n\t}\n"); end > 0 {
				window = window[:end]
			}
			require.Containsf(t, window, "c.Toast(",
				"%s: a clipboard write must also raise a toast", file)
		}
	}
	require.GreaterOrEqual(t, writes, 5,
		"the window's copy buttons: the generic row, the acting-on strip, the payload, "+
			"the verdict and the diagnostics")
}

// TestTheTwoSurfacesDriveSessionStatusTheSameWay covers a rule stated in one file
// and implemented in two: the window's session controls "mirror the REST handler's
// two steps". Two implementations of one rule drift, and the drift is invisible —
// both surfaces keep working, they just stop agreeing about what a missing session
// does or whether the read happens before the write.
//
// The check holds the shape both need: read first so a missing id is a not-found
// rather than a write of nothing, then persist the new status.
func TestTheTwoSurfacesDriveSessionStatusTheSameWay(t *testing.T) {
	rest := readRepoFile(t, "internal", "api", "handler", "voting.go")
	restBody := functionSource(t, rest, "func (h *VotingHandler) updateSessionStatus")
	require.Contains(t, restBody, "NewGetSessionUseCase(h.repo)",
		"the REST handler must read the session before writing its status")
	require.Contains(t, restBody, "h.repo.UpdateSession(session)",
		"and then persist it")
	// Both transitions go through this one helper, so neither can grow a second
	// path with different rules.
	require.Contains(t, functionSource(t, rest, "func (h *VotingHandler) StartSession"),
		"updateSessionStatus(w, r, voting.SessionStatusActive)", "Start is a status write")
	require.Contains(t, functionSource(t, rest, "func (h *VotingHandler) EndSession"),
		"updateSessionStatus(w, r, voting.SessionStatusEnded)", "End is the same write")

	desktop := readRepoFile(t, "desktop", "internal", "deskapp", "native_voting.go")
	desktopBody := functionSource(t, desktop, "func (s *nativeServices) setSessionStatus")
	require.Contains(t, desktopBody, "NewGetSessionUseCase(s.votingRepo)",
		"the window must read the session before writing its status")
	require.Contains(t, desktopBody, "s.votingRepo.UpdateSession(session)",
		"and then persist it")
}

// TestEveryCreateReturnsItsListToTheFirstPage covers the four create paths — a voter,
// a candidate, a session, a token, an NFT — and the one step they share that a test
// cannot see: after a create the list is read from page zero.
//
// Without it the new record is somewhere the reader is not looking. If they had paged
// back before creating, the highlight lands on nothing and the page reads as a create
// that stored nothing; the row exists one page up.
//
// The check reads each create path for the reload, the highlight and the field clear,
// which are one habit written five times.
func TestEveryCreateReturnsItsListToTheFirstPage(t *testing.T) {
	cases := []struct {
		name    string
		file    string
		reload  string
		select_ string
	}{
		{"voter", "voting_setup.go", "a.loadVoterPage(0)", "a.selectVoterByPublicKey("},
		{"candidate", "voting_setup.go", "a.loadCandidatePage(0)", "a.selectCandidateByID("},
		{"session", "voting_setup.go", "a.loadSessionPage(0)", "a.selectSessionByID("},
		{"token", "token.go", "a.refreshTokenAt(0)", "a.selectTokenByID("},
		{"nft", "nft_mint.go", "a.loadNFTPage(0)", "a.selectNFTRow("},
	}
	for _, want := range cases {
		body := readRepoFile(t, "desktop", "internal", "nativeui", want.file)
		require.Containsf(t, body, want.reload,
			"a new %s must be shown from the first page", want.name)
		require.Containsf(t, body, want.select_,
			"and the new %s must be highlighted in it", want.name)
	}
}

// TestTheThreeSurfacesShareOneDrawSearch covers the promise in the reference: a
// lottery search is one read, whether it comes from the window, the API's `q` or the
// CLI's `-s`. The three agree because they call the same repository method, and the
// method matches the same four fields.
//
// A surface that filtered in memory instead would answer a different question — it
// would report "no matches" for a draw on another page — and nothing would look
// wrong. This pins the shared call rather than the result.
func TestTheThreeSurfacesShareOneDrawSearch(t *testing.T) {
	for _, surface := range []struct {
		name string
		path []string
	}{
		{"the API handler", []string{"internal", "api", "handler", "lottery.go"}},
		{"the window's adapter", []string{"desktop", "internal", "deskapp", "native_lottery.go"}},
		{"the CLI", []string{"cmd", "aurora", "cmd", "lottery.go"}},
	} {
		body := readRepoFile(t, surface.path...)
		require.Containsf(t, body, "ListMatching(",
			"%s must search through the repository, not in memory", surface.name)
	}

	// And the method itself matches all four fields the reference names.
	repo := readRepoFile(t, "internal", "infra", "sqlite", "lottery.go")
	search := functionSource(t, repo, "func (r *LotteryRepository) ListMatching")
	for _, field := range []string{"l.id", "l.seed", "l.participants", "l.winners"} {
		require.Containsf(t, search, field, "the search must match %s", field)
	}
	// A term carrying LIKE's own wildcards must match literally.
	require.Contains(t, search, `%`, "and escape the wildcards a seed can contain")
}

// TestTheExportPathChecksBothHalvesOfARecord covers the two-layer validation an
// imported draw goes through. Validate answers "is this record well-formed"; VerifyRecord
// answers "is it intact", which needs the address derivation and is therefore in the app
// layer. The reference promises both guards on import, and the split is easy to mistake
// for a gap: Validate alone accepts a record whose WinnerAddresses is one short.
func TestTheExportPathChecksBothHalvesOfARecord(t *testing.T) {
	portability := readRepoFile(t, "internal", "app", "lotteryapp", "portability.go")
	importBody := functionSource(t, portability, "func ImportRecords")
	require.Contains(t, importBody, "record.Validate()",
		"an imported record must be checked for shape")
	require.Contains(t, importBody, "VerifyRecord(&record)",
		"and for integrity, which is the half that catches a short address list")
	require.Contains(t, importBody, "report.IntegrityOK",
		"the integrity verdict is what decides the record's fate")

	// VerifyRecord owns the address relation; Validate must not be mistaken for it.
	verify := readRepoFile(t, "internal", "app", "lotteryapp", "verify.go")
	require.Contains(t, functionSource(t, verify, "func verifyWinnerAddresses"),
		"len(record.Winners) != len(record.WinnerAddresses)",
		"the address count is compared where the derivation lives")
}

// TestTheIntervalCeilingIsTheDomainConstant covers a number that has to agree across
// two layers. The form's number input stops at the domain's ceiling; the store refuses
// anything above it. Two copies of "30 days" agree the day they are written and drift
// the moment one moves — and the drift's symptom is a form that offers a value the save
// button then rejects, which reads as a bug in neither place.
//
// The check is that the window reads the constant rather than restating it.
func TestTheIntervalCeilingIsTheDomainConstant(t *testing.T) {
	desktop := readRepoFile(t, "desktop", "internal", "nativeui", "oracle.go")
	require.Contains(t, desktop, "const maxOracleIntervalSeconds = oracle.MaxSourceIntervalSeconds",
		"the form's ceiling must be the domain's ceiling, not a second copy of the number")
	require.NotContains(t, desktop, "const maxOracleIntervalSeconds = 30 * 24 * 60 * 60",
		"a restated number is what drifts")

	domain := readRepoFile(t, "internal", "domain", "oracle", "service.go")
	require.Contains(t, domain, "MaxSourceIntervalSeconds = 30 * 24 * 60 * 60",
		"and the domain keeps the one definition")
}

// TestOwnershipChangedStaysASingleton covers a sentinel that is a pointer, not a value.
// ErrOwnershipChanged is &OwnershipChangedError{}, and its callers classify it with
// errors.Is — which, for a type with no Is or Unwrap method, compares pointers. Returning
// a fresh &OwnershipChangedError{} from either repository would therefore still print the
// same sentence and stop matching the sentinel, so a concurrent-transfer rejection would
// surface as an unclassified 500.
//
// Two things keep it safe: the var stays a package-level value, and both repositories
// return that value rather than building one.
func TestOwnershipChangedStaysASingleton(t *testing.T) {
	repo := readRepoFile(t, "internal", "domain", "nft", "repo.go")
	require.Contains(t, repo, "var ErrOwnershipChanged = &OwnershipChangedError{}",
		"the sentinel must stay one package-level value")

	// No method-based matching exists, so pointer identity is the only thing errors.Is can
	// use; if that changes, this check should be replaced rather than deleted.
	require.NotContains(t, repo, "func (*OwnershipChangedError) Is(",
		"if the type gains an Is method, errors.Is no longer depends on the singleton")

	for _, surface := range []struct {
		name string
		path []string
		ctor string
	}{
		{"the SQLite repository", []string{"internal", "infra", "sqlite", "nft.go"}, "OwnershipChangedError{}"},
		{"the in-memory repository", []string{"internal", "domain", "nft", "inmem_repo.go"}, "OwnershipChangedError{}"},
	} {
		body := readRepoFile(t, surface.path...)
		require.NotContainsf(t, body, surface.ctor,
			"%s must return the sentinel, not build a new one", surface.name)
		require.Containsf(t, body, "ErrOwnershipChanged",
			"%s must classify the rejection with the sentinel", surface.name)
	}
}

// TestTheCLIFailureOutputIsDocumentedAsMultiline holds the reference to what a failing
// command actually writes. It said "失败只打印一行错误前缀后退出", and that is only true
// when a log path is configured: without one the logger falls back to stderr and the
// same failure prints a structured line as well. A script written against the promised
// single line would compare the wrong one.
func TestTheCLIFailureOutputIsDocumentedAsMultiline(t *testing.T) {
	doc := readRepoFile(t, "docs", "cli.md")
	require.Contains(t, doc, "日志回退到 stderr",
		"the reference must say where the structured line goes when no log path is set")
	require.Contains(t, doc, "ERR Application error",
		"and what it looks like, since that is the line a script must skip")
	require.Contains(t, doc, "Using config file",
		"the config notice is the other unannounced line")
	require.NotContains(t, doc, "失败只打印一行错误前缀后退出",
		"the single-line promise is what this check replaced")

	// The fallback is real: the logger reaches for stderr only when no log file opened.
	logger := readRepoFile(t, "internal", "logger", "zero.go")
	require.Contains(t, logger, "fallbackToConsole",
		"the stderr path must still be the fallback the reference describes")
	require.Contains(t, logger, "zerolog.ConsoleWriter{Out: os.Stderr}",
		"and it must be the console writer on stderr, not stdout")
}

// TestAStartupFailureReachesTheLog covers the record an operator looks for after a server
// that would not start. The API initialises the logger, then creates the server; when that
// fails the only trace was a line on stderr, so a deployment whose log.path is set had a log
// file with nothing in it about the failure — the one place the operator reads afterwards.
//
// The config-load failure above it is different and stays as it is: that error happens
// before the logger exists, so stderr is all there is.
func TestAStartupFailureReachesTheLog(t *testing.T) {
	main := readRepoFile(t, "cmd", "api", "main.go")

	initAt := strings.Index(main, "logger.Init()")
	require.GreaterOrEqual(t, initAt, 0, "the API must initialise the logger")

	createAt := strings.Index(main, "srv, err := api.NewServer()")
	require.Greater(t, createAt, initAt, "the server is created after the logger is up")

	// Everything between the two is the start-up path that can fail with a logger present.
	startup := main[initAt:strings.Index(main, "srv.RegisterBuildInfo")]
	require.Contains(t, startup, "logger.Error()",
		"a failure with the logger up must reach it, not only the terminal")
	require.Contains(t, startup, "fmt.Fprintf(os.Stderr",
		"and the terminal stays informed, since the log may be a file nobody is watching")

	// The earlier failure is before Init and legitimately uses stderr alone.
	configAt := strings.Index(main, "Failed to load config")
	require.Less(t, configAt, initAt,
		"the config failure happens before the logger exists; that is why it is stderr-only")
}

// TestADesktopStartFailureReachesTheLog covers the same gap the API had, one layer down and
// harder to notice: the window's start-up path initialises the logger, then opens the
// server. When either of the two calls after that fails, the window never opens — so the
// dialog that reports it is the user's only view and disappears when dismissed. The log file
// is what remains to be read, and it used to say nothing.
//
// The second failure is the sharper case: it closes the logger on its way out, so a line
// written after the close would be lost rather than merely absent.
func TestADesktopStartFailureReachesTheLog(t *testing.T) {
	body := readRepoFile(t, "desktop", "internal", "deskapp", "deskapp.go")

	initAt := strings.Index(body, "logger.Init()")
	require.GreaterOrEqual(t, initAt, 0, "the window must initialise the logger")

	createAt := strings.Index(body, "srv, err := newAPIServer()")
	require.Greater(t, createAt, initAt, "the server is created after the logger is up")

	// The server-creation failure must be logged before it is returned.
	createFail := body[createAt:]
	end := strings.Index(createFail, "keepApplicationFilesPrivate(dbPath")
	require.Greater(t, end, 0, "the private-files check follows the server creation")
	require.Contains(t, createFail[:end], "logger.Error()",
		"a failed server creation must reach the log, not only the dialog")

	// And the data-directory failure must log before closing the logger.
	privAt := strings.Index(body, "if err := keepApplicationFilesPrivate(dbPath")
	require.Greater(t, privAt, 0)
	privBlock := body[privAt:]
	closeAt := strings.Index(privBlock, "logger.Close()")
	require.Greater(t, closeAt, 0, "that path closes the logger")
	require.Contains(t, privBlock[:closeAt], "logger.Error()",
		"the failure must be written before the handle it writes through is closed")
}

// TestEveryExitPathStopsTheRuntime covers the resource the deferred Close cannot reach.
// os.Exit does not run defers, so a process that exits from inside a failure path leaves
// the SQLite handle, the oracle scheduler and the audit outbox to be torn down by process
// teardown instead of being stopped in order — the WAL is left as it was and whatever the
// outbox had not drained is lost.
//
// Both mains have such a path and both now close first. The check reads each exit for the
// stop that has to come before it, which is the part a later edit removes without noticing.
func TestEveryExitPathStopsTheRuntime(t *testing.T) {
	api := readRepoFile(t, "cmd", "api", "main.go")
	// The listener goroutine cannot use a deferred close, and says so.
	fatalAt := strings.Index(api, "logger.Fatal()")
	require.Greater(t, fatalAt, 0)
	require.Contains(t, api[:fatalAt], "srv.Close()",
		"the API must stop the server before the Fatal that skips defers")

	desktop := readRepoFile(t, "desktop", "cmd", "aurora-desktop", "main.go")
	// The stderr sentence appears in three branches, so the anchor is the branch itself:
	// everything up to the closing of that block is what the check reads.
	// The indentation is the discriminator: main's call is at one tab, the generation
	// branch's is nested one level deeper and has no runtime to close.
	branchStart := strings.Index(desktop, "\n\tif err := mygo.App.Run(); err != nil {")
	require.Greater(t, branchStart, 0, "the window loop's failure branch must exist")
	branch := desktop[branchStart:]
	if end := strings.Index(branch, "\n\t}\n"); end > 0 {
		branch = branch[:end]
	}
	require.Contains(t, branch, "rt.Close()",
		"the window must stop its runtime before exiting, since the defer is skipped")
	require.Contains(t, branch, "os.Exit(1)",
		"and the branch is the one that exits")

	// And the normal path still defers its close, so the added call is the exception.
	require.Contains(t, desktop, "defer func() { _ = rt.Close() }()",
		"the ordinary shutdown must stay deferred")
}
