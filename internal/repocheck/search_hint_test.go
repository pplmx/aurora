package repocheck

import (
	"io/fs"
	"os"
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
	// The token list joined the storage-side searches, so the page-local ones are
	// the voters, candidates, sessions, NFT collection and oracle sources. The
	// count guards against a list quietly switching to an in-memory filter while
	// its hint still promises a whole-store search.
	require.GreaterOrEqual(t, strings.Count(view, "containsFold("), 2,
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
	// The draws and the ledger always queried storage; the token, NFT and oracle
	// collections now do too, so five hints may promise it. A new one must come
	// with its own storage-side read, which is what this count is here to notice.
	assert.Equal(t, 5, wholeStore,
		"exactly the lists that query storage may promise a whole-store search")
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

// TestTheReferenceSaysJSONIsForSuccessOnly covers the half of the --json contract a script
// depends on and the reference did not state. The switch changes what a command prints when
// it succeeds; a failure still prints the human line on stderr and leaves stdout empty, so
// there is no error document to parse.
//
// Measured: `lottery verify --json` with no argument exits 1, writes nothing to stdout, and
// puts "❌ Error: accepts 1 arg(s), received 0" on stderr. A script treating unparseable
// stdout as "empty result" would read a failure as success.
func TestTheReferenceSaysJSONIsForSuccessOnly(t *testing.T) {
	doc := readRepoFile(t, "docs", "cli.md")
	require.Contains(t, doc, "`--json` 只改变**成功**时的输出",
		"the reference must scope the switch to the success path")
	require.Contains(t, doc, "失败时 stdout 为空",
		"and say what a caller sees instead")
	require.Contains(t, doc, "不要用「stdout 能否解析成 JSON」判断成败",
		"which is the mistake the sentence exists to prevent")

	// The code agrees: the switch is read on the printing path, and the failure path is
	// formatCLIError, which knows nothing about --json.
	jsonHelper := readRepoFile(t, "cmd", "aurora", "cmd", "json.go")
	require.Contains(t, jsonHelper, "func jsonRequested(",
		"the switch is a query a command makes when it prints")
	require.Contains(t, jsonHelper, "func printJSON(",
		"and printing is what it gates")

	root := readRepoFile(t, "cmd", "aurora", "cmd", "root.go")
	require.Contains(t, root, "func formatCLIError(",
		"failures have one formatter")
	require.NotContains(t, functionSource(t, root, "func formatCLIError("), "jsonRequested",
		"and it does not branch on --json, which is why stdout stays empty")
}

// TestTheCommandsWithoutJSONAreListedTogether holds the reference's "no JSON mode" list to
// the commands that actually have no flag. The list is the only place a script can learn that
// passing --json to one of them fails before the command does anything, so a command left off
// it is a command whose caller finds out the hard way.
func TestTheCommandsWithoutJSONAreListedTogether(t *testing.T) {
	doc := readRepoFile(t, "docs", "cli.md")

	// Each entry is named by the command a reader types, not by its Go variable.
	for _, want := range []struct {
		doc  string
		vars []string
	}{
		{"`token create`", []string{"tokenCreateCmd"}},
		{"`voter register`", []string{"voterRegisterCmd"}},
		{"`oracle source add`", []string{"sourceAddCmd"}},
		{"`oracle fetch`", []string{"fetchCmd"}},
		{"`lottery reset`", []string{"resetCmd"}},
	} {
		require.Containsf(t, doc, want.doc, "the reference must name %s", want.doc)
		for _, v := range want.vars {
			// Every command file, not a fixed shortlist: a flag registered from a different
			// file than the one that declares the command is exactly the case the list would
			// miss.
			registered := false
			for _, file := range commandFiles(t) {
				if strings.Contains(readRepoFile(t, "cmd", "aurora", "cmd", file), "addJSONFlag("+v+")") {
					registered = true
				}
			}
			require.Falsef(t, registered,
				"%s has a --json flag now, so the reference's no-JSON list is stale", v)
		}
	}

	// And the sentence about an unknown flag failing early, which is the cost of getting it
	// wrong.
	require.Contains(t, doc, "不认识的 flag 会让命令在写入前就失败退出",
		"the reference must say what happens when the flag is passed anyway")
}

// commandFiles lists the non-test sources of the CLI's command package, so a check about
// every command does not need a hand-kept shortlist.
func commandFiles(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(repoRoot(t), "cmd", "aurora", "cmd"))
	require.NoError(t, err)
	var names []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		names = append(names, name)
	}
	require.NotEmpty(t, names)
	return names
}

// TestEveryConfirmationGateHasItsWarning holds the family the translation scan cannot reach.
// addConfirmFlag takes its key as a literal, but the renderer wraps it at run time
// (cli.confirm.required formats "⚠️ this %s"), so the sentence a user reads is assembled from
// two keys and the scan for literal keys sees only the second one. A new gate with no entry
// would print the key itself on the line that is supposed to explain what is about to be
// destroyed.
func TestEveryConfirmationGateHasItsWarning(t *testing.T) {
	gate := regexp.MustCompile(`addConfirmFlag\(\w+,\s*"(cli\.confirm\.[a-z_]+)"\)`)

	seen := map[string]string{}
	for _, file := range commandFiles(t) {
		body := readRepoFile(t, "cmd", "aurora", "cmd", file)
		for _, m := range gate.FindAllStringSubmatch(body, -1) {
			seen[m[1]] = file
		}
	}
	require.GreaterOrEqual(t, len(seen), 7, "the destructive commands must still register gates")

	catalogs := []struct {
		name string
		path []string
	}{
		{"English", []string{"internal", "i18n", "messages_en.go"}},
		{"Simplified Chinese", []string{"internal", "i18n", "messages_zh.go"}},
		{"Traditional Chinese", []string{"internal", "i18n", "messages_zh_hant.go"}},
	}
	for key, file := range seen {
		for _, catalog := range catalogs {
			source := readRepoFile(t, catalog.path...)
			require.Containsf(t, source, `"`+key+`"`,
				"%s catalog is missing %s, registered in %s", catalog.name, key, file)
		}
		// And the wrapper that reads it, which is what the user actually sees.
		require.Contains(t, readRepoFile(t, "cmd", "aurora", "cmd", "root.go"), "cli.confirm.required",
			"the gate sentence is assembled from this key")
	}
}

// TestCommentsNameFunctionsThatExist covers a class of reference that fails quietly: a
// comment saying "views set it through ok()/fail()" names functions that were never written,
// and the reader either looks for them or trusts a name that does not exist. Neither breaks a
// build, so nothing surfaces it.
//
// The scan is narrow on purpose: only `name()` written in a comment, only lowercase names, and
// only against the function declarations in the same package tree. That keeps it free of false
// positives — prose like "the check()" is rare, and a name that resolves is evidence enough.
func TestCommentsNameFunctionsThatExist(t *testing.T) {
	roots := []string{
		"desktop/internal/nativeui",
		"internal/app",
		"internal/api",
		"internal/domain",
		"internal/infra",
		"internal/records",
		"internal/ledger",
	}
	root := repoRoot(t)
	decl := regexp.MustCompile(`(?m)^func (?:\([^)]*\) )?(\w+)\(`)
	call := regexp.MustCompile(`\b([a-z][a-zA-Z0-9_]{2,})\(\)`)

	checked := 0
	for _, tree := range roots {
		declared := map[string]bool{}
		files := []string{}
		err := filepath.WalkDir(filepath.Join(root, tree), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			files = append(files, path)
			body, err := os.ReadFile(path) //nolint:gosec // path comes from the repo walk
			if err != nil {
				return err
			}
			for _, m := range decl.FindAllStringSubmatch(string(body), -1) {
				declared[m[1]] = true
			}
			return nil
		})
		require.NoError(t, err)
		require.NotEmptyf(t, files, "%s must hold Go sources", tree)

		for _, path := range files {
			body, err := os.ReadFile(path) //nolint:gosec // path comes from the repo walk
			require.NoError(t, err)
			rel, err := filepath.Rel(root, path)
			require.NoError(t, err)
			for _, line := range strings.Split(string(body), "\n") {
				trimmed := strings.TrimSpace(line)
				if !strings.HasPrefix(trimmed, "//") {
					continue
				}
				for _, m := range call.FindAllStringSubmatch(trimmed, -1) {
					checked++
					require.Truef(t, declared[m[1]],
						"%s names %s() in a comment, and this package has no such function",
						rel, m[1])
				}
			}
		}
	}
	require.Greaterf(t, checked, 0, "the scan must find at least one comment reference")
}

// TestTheSnapshotReportSaysItIsASnapshot keeps a dated report from reading as current fact.
// ablation.md names symbols that existed at its baseline commit and do not exist now
// (Token.AddToSupply, Migrator.DB, LotteryRecord.FromJSON), which is fine for a snapshot and
// misleading for anything else — a reader who finds a name there and greps for it today gets
// nothing, with no clue that the document was describing a different tree.
func TestTheSnapshotReportSaysItIsASnapshot(t *testing.T) {
	doc := readRepoFile(t, "docs", "ablation.md")

	require.Contains(t, doc, "基线：",
		"the report must keep naming the commit it analysed")
	require.Contains(t, doc, "这是一份带基线的快照报告",
		"and say that its symbol references belong to that commit")
	require.Contains(t, doc, "不是指它们现在还存在",
		"the consequence a reader needs")
	require.Contains(t, doc, "docs/architecture.md",
		"and where the current facts live")

	// The claim is true: at least one symbol it names is gone from the tree, which is why the
	// note is needed rather than optional.
	gone := false
	for _, member := range []string{"AddToSupply", "FromJSON", "DB"} {
		found := false
		for _, tree := range []string{"internal/domain", "internal/app", "internal/infra"} {
			walkErr := filepath.WalkDir(filepath.Join(repoRoot(t), tree), func(path string, entry fs.DirEntry, err error) error {
				if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
					return err
				}
				body, readErr := os.ReadFile(path) //nolint:gosec // path comes from the repo walk
				if readErr != nil {
					return readErr
				}
				if strings.Contains(string(body), ") "+member+"(") {
					found = true
				}
				return nil
			})
			require.NoError(t, walkErr)
		}
		if !found {
			gone = true
		}
	}
	require.True(t, gone,
		"if every symbol it names still exists, the note can be dropped rather than kept")
}

// TestTheDesktopDataDirectoryMatchesItsDoc covers the one path a user is told to look in. The
// reference says the default is MyGo's PathUserData, spelled $XDG_CONFIG_HOME/Aurora/aurora.db,
// and adds that SetName runs before Start or a machine ends up with both MyGo/ and Aurora/.
//
// Both halves are checkable and both matter: MyGo builds that directory from App.Name(), so the
// capital A is SetName's doing, and removing the call would move the database without changing a
// line of the sentence — leaving the doc describing a directory that no longer exists.
func TestTheDesktopDataDirectoryMatchesItsDoc(t *testing.T) {
	doc := readRepoFile(t, "docs", "desktop.md")
	require.Contains(t, doc, "App.Path(PathUserData)",
		"the reference must name the call that resolves the directory")
	require.Contains(t, doc, "Aurora/aurora.db",
		"and spell the result, since App.Name() is what capitalises it")
	require.Contains(t, doc, "logs/", "the log directory sits beside the database")

	// The call is real: main resolves the default through MyGo, not through the fallback.
	// Asserted with a short message rather than require.Contains, which would print the whole
	// file on failure — main.go is long and the missing sentence is one line of it.
	main := readRepoFile(t, "desktop", "cmd", "aurora-desktop", "main.go")
	require.True(t, strings.Contains(main, "mygo.App.Path(mygo.PathUserData)"),
		"the packaged default must come from MyGo's own resolution")
	require.True(t, strings.Contains(main, "mygo.App.SetName(deskapp.AppName)"),
		"and the name must be set, because PathUserData joins App.Name()")

	// The name is what makes the capital A in the documented path.
	shell := readRepoFile(t, "desktop", "internal", "deskapp", "deskapp.go")
	require.Contains(t, shell, `const AppName = "Aurora"`,
		"the documented directory name is AppName")

	// SetName must come before Start creates anything.
	setAt := strings.Index(main, "mygo.App.SetName(deskapp.AppName)")
	startAt := strings.Index(main, "deskapp.Start(ctx, deskapp.Options{")
	require.Greater(t, setAt, 0)
	require.Greater(t, startAt, setAt,
		"SetName runs first; the other order creates MyGo/ alongside Aurora/")
}

// TestTheBuildTimeChainIsWhole holds the three links the packaging section promises. A packaged
// binary cannot take main.BuildTime from the justfile — `mygo build` supplies its own -ldflags —
// so the timestamp travels through a file: a buildCommand writes it under resources/, MyGo ships
// that directory with the app, and the binary reads it back from PathResources.
//
// Any link breaking leaves `Build Time: unknown`, which the same section says CI refuses. That
// promise is only worth something if all three exist, so the check reads each one.
func TestTheBuildTimeChainIsWhole(t *testing.T) {
	// Link 1: the config runs the stamping script.
	config := readRepoFile(t, "desktop", "mygo.config.ts")
	require.Contains(t, config, "node scripts/stamp-build-time.mjs",
		"mygo build must run the stamping script")
	require.Contains(t, config, "buildCommand",
		"and it must be the build command, which runs before resources are copied")

	// Link 2: the script writes inside resources/, the directory the config ships.
	script := readRepoFile(t, "desktop", "scripts", "stamp-build-time.mjs")
	require.Contains(t, script, `"resources/build-time.txt"`,
		"the stamp must land in the directory the packaging copies")
	require.Contains(t, script, "new Date().toISOString()",
		"and carry a real timestamp rather than a placeholder")

	// Link 3: the binary reads it back from PathResources, which is that same directory.
	main := readRepoFile(t, "desktop", "cmd", "aurora-desktop", "main.go")
	require.True(t, strings.Contains(main, "mygo.App.Path(mygo.PathResources)"),
		"the runtime must ask MyGo where the shipped resources are")
	require.True(t, strings.Contains(main, `"build-time.txt"`),
		"and read the file the script wrote")

	// The consequence the section promises: CI refuses a package that lost the stamp.
	found := false
	for _, workflow := range []string{"ci.yml", "desktop-build.yml", "release.yml"} {
		body := readRepoFile(t, ".github", "workflows", workflow)
		if strings.Contains(body, "Build Time: unknown") || strings.Contains(body, "unknown\"") {
			found = true
		}
	}
	require.True(t, found,
		"the reference says CI rejects Build Time: unknown, so some workflow must check it")

	doc := readRepoFile(t, "docs", "desktop.md")
	require.Contains(t, doc, "resources/build-time.txt",
		"and the packaging section must describe this chain")
}

// TestNothingGoLivesInShippedResources holds the boundary three comments describe. Every file in
// desktop/resources is copied next to the executable and into the .deb/tarball by `mygo build`, so
// a Go package placed there would ship its sources and tests with the app. The icon lives in
// desktop/appicon for exactly that reason, and the packaging config names it from there.
//
// The other direction matters too: resources/ holds generated files only, and the one it holds is
// ignored — committing it would freeze a build timestamp into the repository, and every later
// build would ship the old one unless the file changed.
func TestNothingGoLivesInShippedResources(t *testing.T) {
	// No Go sources under resources/, and it holds at most the generated stamp.
	found := 0
	err := filepath.WalkDir(filepath.Join(repoRoot(t), "desktop", "resources"),
		func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return filepath.SkipAll // a fresh clone has no generated stamp yet
				}
				return err
			}
			if entry.IsDir() {
				return nil
			}
			found++
			require.Falsef(t, strings.HasSuffix(path, ".go"),
				"%s would be copied into the shipped app", path)
			return nil
		})
	require.NoError(t, err)
	require.LessOrEqual(t, found, 1, "resources/ holds the generated stamp, not shipped assets")

	// The packaging names the icon from appicon/, which is what makes its location load-bearing.
	config := readRepoFile(t, "desktop", "mygo.config.ts")
	require.Contains(t, config, `icon: "./appicon/icon.png"`,
		"the packaging icon must come from outside resources/")
	require.Contains(t, config, "It is deliberately outside resources/",
		"and the reason must stay next to it")

	// The embed keeps the same file as the window icon for an unpackaged build.
	icon := readRepoFile(t, "desktop", "appicon", "icon.go")
	require.Contains(t, icon, "//go:embed icon.png",
		"the runtime icon is the same bytes, embedded")
	require.True(t, strings.Contains(icon, "It deliberately lives outside desktop/resources"),
		"with the boundary stated where the package begins")

	// The generated stamp stays untracked.
	require.Contains(t, readRepoFile(t, ".gitignore"), "/desktop/resources/build-time.txt",
		"the stamp is generated per build, so it must not be committed")
}

// TestIgnoreRulesForBinariesAreAnchored holds the lesson .gitignore already records: a bare
// `aurora` pattern matches the `cmd/aurora/` source directory at any depth, and git then ignores
// every file under it without a word. The rule is anchored to the root now, and the comment beside
// it says why.
//
// The check is narrow on purpose — it asks only about patterns whose name is also a directory under
// cmd/ or internal/, since those are the ones a missing slash silently swallows.
func TestIgnoreRulesForBinariesAreAnchored(t *testing.T) {
	// Directory names a stray ignore pattern could swallow.
	shadows := map[string]bool{}
	for _, tree := range []string{"cmd", "internal"} {
		entries, err := os.ReadDir(filepath.Join(repoRoot(t), tree))
		require.NoError(t, err)
		for _, e := range entries {
			if e.IsDir() {
				shadows[e.Name()] = true
			}
		}
	}
	require.NotEmpty(t, shadows, "the source tree must have package directories")

	for _, line := range strings.Split(readRepoFile(t, ".gitignore"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		pattern := strings.TrimSuffix(trimmed, "/")
		require.Falsef(t, shadows[pattern],
			".gitignore has a bare %q, which would also match the source directory of the same name; anchor it as /%s",
			pattern, pattern)
	}

	// The build outputs are still ignored, which is what the anchored rules are for.
	ignore := readRepoFile(t, ".gitignore")
	for _, want := range []string{"/aurora", "/aurora-*", "/bin"} {
		require.Containsf(t, ignore, want,
			"%s is where just build and a bare go build put their output", want)
	}
}

// TestCICoversWhatTheLocalGateCannot holds the split AGENTS.md describes: `just verify` runs the
// gates a laptop can run, while the race detector needs cgo and packaging needs a toolchain — so
// CI is where those two live. If CI stopped running either, the local gate would still pass and
// nobody would notice until a data race reached a release.
func TestCICoversWhatTheLocalGateCannot(t *testing.T) {
	ci := readRepoFile(t, ".github", "workflows", "ci.yml")

	// The race detector runs for both modules. It is the one gate deliberately outside
	// `just verify`, so it has no other home.
	require.Contains(t, ci, "-race", "CI must run the race detector, since the local gate cannot")
	require.GreaterOrEqual(t, strings.Count(ci, "-race"), 2,
		"both the core module and the desktop run it; one job alone leaves half the tree unchecked")

	// Packaging is exercised too, because a build that only compiles proves nothing about a
	// package's metadata.
	require.Contains(t, ci, "Package", "CI must package the desktop app")
	require.Contains(t, ci, "Verify pure-Go build metadata",
		"and check that the packaged binaries stayed pure Go")

	// Order independence has no other home: a fixed-order run cannot detect a test that only
	// passes because another ran first, and the local gate does not run shuffle either.
	require.GreaterOrEqual(t, strings.Count(ci, "-shuffle=on"), 2,
		"both modules must run with shuffled order; one job alone leaves half the tree unchecked")

	// `just verify` is what the AGENTS table calls the local equivalent. It is a dependency-only
	// recipe with no commands of its own, so the declaration line is what to read: it must not
	// name race.
	just := readRepoFile(t, "justfile")
	var verifyLine string
	for _, line := range strings.Split(just, "\n") {
		if strings.HasPrefix(line, "verify:") {
			verifyLine = line
			break
		}
	}
	require.NotEmpty(t, verifyLine, "the justfile must define a verify recipe")
	require.NotContains(t, verifyLine, "race",
		"a local gate that needs cgo is why the race run lives in CI instead")
	require.Contains(t, verifyLine, "desktop-coverage-floor",
		"and the local gate must cover both modules' floors")
}

// TestTheThreeVerificationLayersStayDistinct keeps the split between what each layer can see. The
// offscreen tester renders without a window backend, CI starts the packaged artifact under Xvfb and
// inspects the state it writes, and the screenshot script is the only one that shows what the
// window drew.
//
// The script's header used to say CI "only proves the binary starts", which undersold the smoke
// test and told a reader the gap was startup rather than appearance. All three comments are cheap
// to drift, so the check reads each one.
func TestTheThreeVerificationLayersStayDistinct(t *testing.T) {
	// The header wraps, so the phrases are checked against a joined copy. A failure here prints
	// one line, not the 500-line script.
	script := strings.ReplaceAll(readRepoFile(t, "desktop", "scripts", "e2e-window.sh"), "\n# ", " ")
	require.True(t, strings.Contains(script, "inspects what it writes"),
		"the header must describe what the CI smoke test actually checks")
	require.True(t, strings.Contains(script, "Neither shows what the window drew"),
		"and name the gap this script exists to close")
	require.False(t, strings.Contains(script, "only proves the binary starts"),
		"the older, inaccurate description must stay gone")

	// The smoke test really does more than start the app: it waits for the state and checks modes.
	ci := readRepoFile(t, ".github", "workflows", "ci.yml")
	require.Contains(t, ci, "Smoke-test the packaged app",
		"the packaging job must keep starting the real artifact")
	require.Contains(t, ci, `stat -c '%a' "$data/aurora.db"`,
		"and check that what it wrote is owner-only, not merely present")
	require.Contains(t, ci, "kill -0 \"$pid\"",
		"and that the window was still alive once the state appeared")
}
