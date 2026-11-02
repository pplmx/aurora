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
