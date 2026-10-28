package nativeui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/i18n"
)

// TestShowPageAcceptsStableIDs pins the contract deep links use: ids do not
// change when a title is translated, while the English titles stay accepted
// for older callers.
func TestShowPageAcceptsStableIDs(t *testing.T) {
	app := New(&fakeServices{})
	for p, id := range pageIDs {
		app.ShowPage(strings.ToUpper(id))
		require.Equal(t, p, app.page, "stable id %q must select its page", id)
	}
	app.ShowPage("token")
	require.Equal(t, pageToken, app.page)
}

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

// TestCardsStackWhenTheWindowIsNarrow pins the responsive layout: at 1280 the
// Token page's two work columns sit side by side, while at the 900px minimum
// the second column moves below the first instead of squeezing both into
// unreadable widths.
func TestCardsStackWhenTheWindowIsNarrow(t *testing.T) {
	wideApp := New(&fakeServices{})
	wide := ui.NewTester(wideApp.View, 1280, 3000)
	wideApp.ShowPage("Token")
	wide.Frame()
	create, ok := wide.Find("Create Token")
	require.True(t, ok)
	mint, ok := wide.Find("Only the token owner can mint, and only for a mintable token.")
	require.True(t, ok)
	require.NotEqual(t, create.X, mint.X, "a wide window keeps the two columns side by side")

	narrowApp := New(&fakeServices{})
	narrow := ui.NewTester(narrowApp.View, 900, 5000)
	narrowApp.ShowPage("Token")
	narrow.Frame()
	create, ok = narrow.Find("Create Token")
	require.True(t, ok)
	mint, ok = narrow.Find("Only the token owner can mint, and only for a mintable token.")
	require.True(t, ok)
	require.Equal(t, create.X, mint.X, "a narrow window puts both columns on the same left edge")
	require.Greater(t, mint.Y, create.Y, "the second column must follow the first")
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

// TestServiceErrorsUseTheDesktopLanguage covers the messages that come from
// the domain rather than the view: known errors are translated, a wrapped
// error keeps its prefix, and an unknown diagnostic stays verbatim so it still
// matches the log.
func TestServiceErrorsUseTheDesktopLanguage(t *testing.T) {
	tr := i18n.GetTranslator()
	before := tr.GetLocale()
	t.Cleanup(func() { tr.SetLocale(before) })

	tr.SetLocale("zh")
	require.Equal(t, "找不到代币", errStatus(errors.New("token not found")).text)
	require.Equal(t, "seed: 种子过短", errStatus(errors.New("seed: seed too short")).text)
	require.Equal(t, "database is locked", errStatus(errors.New("database is locked")).text)
	require.True(t, errStatus(errors.New("token not found")).isErr)
}

// TestChineseUIRendersTheWorkbench keeps the desktop's labels in the language
// switcher, not only its menus: page names, purposes, Quick start, metrics,
// tabs and the guide must come from the desktop table.
func TestChineseUIRendersTheWorkbench(t *testing.T) {
	tr := i18n.GetTranslator()
	before := tr.GetLocale()
	t.Cleanup(func() { tr.SetLocale(before) })

	tr.SetLocale("zh")
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 1600)
	tester.Frame()
	for _, want := range []string{"仪表盘", "快速开始", "运行抽奖", "抽奖总数", "账本完整性"} {
		require.Truef(t, tester.HasText(want), "the Chinese dashboard must show %q", want)
	}

	app.ShowPage("Token")
	tester.Frame()
	for _, want := range []string{"代币", "创建代币", "铸造", "转账", "查看", "历史"} {
		require.Truef(t, tester.HasText(want), "the Chinese Token page must show %q", want)
	}
	require.True(t, tester.HasText("高级"), "the Advanced disclosure must be translated")

	require.NoError(t, tester.Click("工作方式"))
	tester.Frame()
	require.True(t, tester.HasText("在左栏选择或创建一个代币。它的所有者公钥会自动填入账户字段。"))

	app.ShowPage("Settings")
	tester.Frame()
	for _, want := range []string{"设置", "语言", "数据", "外观", "关于 Aurora"} {
		require.Truef(t, tester.HasText(want), "the Chinese Settings page must show %q", want)
	}
}

// TestChineseUIRendersBusyAndLedgerPagerCopy covers two labels that bypassed
// the desktop table: the long-running write banner and the ledger's "of N"
// pager count used to fall back to English in an otherwise Chinese window.
func TestChineseUIRendersBusyAndLedgerPagerCopy(t *testing.T) {
	tr := i18n.GetTranslator()
	before := tr.GetLocale()
	t.Cleanup(func() { tr.SetLocale(before) })

	tr.SetLocale("zh")
	blocks := make([]BlockRow, 12)
	for i := range blocks {
		blocks[i] = BlockRow{Height: int64(len(blocks) - i), Module: "lottery"}
	}
	app := New(&fakeServices{blocks: blocks})
	tester := ui.NewTester(app.View, 1280, 4000)
	app.ShowPage("Blockchain")
	app.blockchain.blockLimit = 10
	tester.Frame()
	require.True(t, tester.HasText("第 1–10 条，共 12 条"))

	app.busy = "Casting vote…"
	app.busySince = time.Now().Add(-21 * time.Second)
	tester.Frame()
	require.True(t, tester.HasText("仍在处理… 这可能需要一些时间；操作会自行完成。"))
}

// TestChineseUIRendersStoredLifecycleLabels keeps values that come from the
// domain, rather than from a label in the view, inside the language switch:
// session states and NFT operation types used to remain lowercase English in
// an otherwise translated table.
func TestChineseUIRendersStoredLifecycleLabels(t *testing.T) {
	translator := i18n.GetTranslator()
	before := translator.GetLocale()
	t.Cleanup(func() { translator.SetLocale(before) })

	translator.SetLocale("zh")
	fake := &fakeServices{sessions: []SessionRow{
		{ID: "s1", Title: "Draft", Status: "draft"},
		{ID: "s2", Title: "Live", Status: "active"},
		{ID: "s3", Title: "Finished", Status: "ended"},
	}}
	app := New(fake)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Voting")
	tester.Frame()
	for _, want := range []string{"草稿", "进行中", "已结束"} {
		require.Truef(t, tester.HasText(want), "the session table must show %q", want)
	}
	require.False(t, tester.HasText("active"), "a stored status must not remain English")

	app.ShowPage("NFT")
	app.nft.opTab = 2
	app.nft.historyID = "n1"
	app.nft.historyLoadedID = "n1"
	app.nft.history = []NFTOpRow{
		{Type: "mint", To: "alice"},
		{Type: "transfer", From: "alice", To: "bob"},
		{Type: "burn", From: "bob"},
	}
	tester.Frame()
	for _, want := range []string{"铸造", "转让", "销毁"} {
		require.Truef(t, tester.HasText(want), "the NFT history must show %q", want)
	}
	require.False(t, tester.HasText("transfer"), "a stored operation type must not remain English")
}

// TestEveryContentPageHasAHowThisWorksGuide keeps the plain-language steps on
// every module page: the guide is closed by default, but one click must show
// the order of operations. Dashboard uses Quick start instead.
func TestEveryContentPageHasAHowThisWorksGuide(t *testing.T) {
	cases := []struct {
		page string
		step string
	}{
		{"Lottery", "Create the draw: the result and its VRF proof are recorded on the ledger."},
		{"Voting", "Create a session with a roster and a start/end window."},
		{"Token", "Select a token in the left column, or create one. Its owner public key fills the account fields for you."},
		{"Oracle", "Select a source in Sources to fetch it now, enable/disable it, or delete it."},
		{"NFT", "Transfer or Burn in the right column; the current holder's private key signs. A generated creator key can be reused while the creator still holds the asset."},
		{"Blockchain", "The badge verifies the chain. Search matches module, payload or height across the whole chain; Show sets the page size."},
	}
	for _, tc := range cases {
		t.Run(tc.page, func(t *testing.T) {
			app := New(&fakeServices{})
			tester := ui.NewTester(app.View, 1280, 1200)
			app.ShowPage(tc.page)
			tester.Frame()
			require.True(t, tester.HasText("How this works"))
			require.False(t, tester.HasText(tc.step), "the steps start collapsed")
			require.NoError(t, tester.Click("How this works"))
			tester.Frame()
			require.True(t, tester.HasText(tc.step), "the clicked guide must show its steps")
		})
	}
}

// TestDashboardQuickStartNavigates pins every first-move button: the sidebar
// lists modules, so the dashboard is where a task name leads to its page.
func TestDashboardQuickStartNavigates(t *testing.T) {
	for _, tc := range []struct {
		label string
		want  page
	}{
		{"Run a draw", pageLottery},
		{"Create a token", pageToken},
		{"Register a voter", pageVoting},
		{"Add a data source", pageOracle},
		{"Mint an NFT", pageNFT},
	} {
		app := New(&fakeServices{})
		tester := ui.NewTester(app.View, 1280, 1000)
		tester.Frame()
		require.Equalf(t, pageDashboard, app.page, "%s must start from the dashboard", tc.label)
		require.NoErrorf(t, tester.Click(tc.label), "click %q", tc.label)
		tester.Frame()
		require.Equalf(t, tc.want, app.page, "%q must open its page", tc.label)
	}
}

// TestFreshDashboardCanOpenTheLottery pins the quick-start card's first move:
// the empty dashboard names "Run a draw", and the button takes it instead of
// leaving the operator to hunt the sidebar.
type emptyDashboard struct{ *fakeServices }

func (emptyDashboard) Dashboard() (DashboardStats, error) {
	return DashboardStats{Integrity: "verified"}, nil
}

type candidateOnlyDashboard struct{ *fakeServices }

func (candidateOnlyDashboard) Dashboard() (DashboardStats, error) {
	return DashboardStats{
		Candidates: 1,
		Integrity:  "verified",
	}, nil
}

func TestFreshDashboardCanOpenTheLottery(t *testing.T) {
	app := New(emptyDashboard{fakeServices: &fakeServices{}})
	tester := ui.NewTester(app.View, 1280, 860)
	tester.Frame()
	require.Equal(t, pageDashboard, app.page)

	require.NoError(t, tester.Click("Run a draw"))
	tester.Frame()
	require.Equal(t, pageLottery, app.page)
}

// TestDashboardDoesNotCallCandidateOnlyStoreEmpty guards the empty-state
// decision: it originally looked only at lotteries, votes and oracle sources,
// so a store with a registered candidate but no ballots claimed that nothing
// had been recorded yet.
func TestDashboardDoesNotCallCandidateOnlyStoreEmpty(t *testing.T) {
	app := New(candidateOnlyDashboard{fakeServices: &fakeServices{}})
	tester := ui.NewTester(app.View, 1280, 1200)
	tester.Frame()

	require.True(t, tester.HasText("Candidates"))
	require.False(t, tester.HasText("Nothing recorded yet — run a draw to create the first ledger entry."))
}

// TestDashboardDoesNotCallBlockOnlyStoreEmpty covers modules without a
// dashboard counter, such as a token or NFT operation: a non-empty audit chain
// is still evidence that the store is in use.
func TestDashboardDoesNotCallBlockOnlyStoreEmpty(t *testing.T) {
	app := New(emptyDashboard{fakeServices: &fakeServices{
		blocks: []BlockRow{{Height: 1, Module: "token", Summary: "created"}},
	}})
	tester := ui.NewTester(app.View, 1280, 1200)
	tester.Frame()

	require.True(t, tester.HasText("created"))
	require.False(t, tester.HasText("Nothing recorded yet — run a draw to create the first ledger entry."))
}

// TestDashboardDoesNotCallUnknownChainEmpty covers the other side of the chain
// check: when the recent-activity read fails, zero module counters cannot prove
// the audit chain is empty, so the page must show the warning without claiming
// this is a fresh install.
func TestDashboardDoesNotCallUnknownChainEmpty(t *testing.T) {
	services := &fakeServices{}
	services.loadErrs.blocks = errors.New("ledger unavailable")
	app := New(emptyDashboard{fakeServices: services})
	tester := ui.NewTester(app.View, 1280, 1200)
	tester.Frame()

	require.True(t, tester.HasText("Couldn't read recent activity"))
	require.False(t, tester.HasText("Nothing recorded yet — run a draw to create the first ledger entry."))
}

// TestTextScaleIsHonoured pins the OS text-size setting. A theme's default
// font follows it, but every explicit FontSize is absolute; before the
// textSize helper the window ignored the setting for all of its own text.
func TestTextScaleIsHonoured(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 860)
	tester.Frame()
	before, ok := tester.Find("Aurora")
	require.True(t, ok)

	tester.SetPreferences(ui.Preferences{TextScale: 1.5})
	tester.Frame()
	after, ok := tester.Find("Aurora")
	require.True(t, ok)
	require.Greater(t, after.H, before.H*1.3, "1.5x text must render visibly larger")
}

// TestDashboardLinksNavigateFromAnotherPage pins the dashboard as a real
// destination. The deep-link parser used to spell "aurora://" and
// "aurora://index" as an empty page, which ShowIntent treats as "leave the
// window alone" — so a dashboard link from another page did nothing.
func TestDashboardLinksNavigateFromAnotherPage(t *testing.T) {
	app := New(&fakeServices{})

	app.ShowPage("Token")
	app.ShowIntent("dashboard", "")
	require.Equal(t, pageDashboard, app.page)

	app.ShowPage("Token")
	app.ShowPage("index")
	require.Equal(t, pageDashboard, app.page, "the old index spelling names the dashboard")
}

// TestSettingsShortcut pins Ctrl/Cmd+, (the desktop settings convention)
// alongside the numbered sidebar shortcuts.
func TestSettingsShortcut(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 860)
	tester.Frame()

	tester.Key(platformRefreshModifier(), ui.KeyComma)
	tester.Frame()
	require.Equal(t, pageSettings, app.page)
}

// TestDashboardPrioritisesStateOverActions keeps the returning dashboard's
// hierarchy: counters and recent activity come before the quick-start card.
func TestDashboardPrioritisesStateOverActions(t *testing.T) {
	app := New(&fakeServices{blocks: []BlockRow{{Height: 1, Module: "lottery", Summary: "draw"}}})
	tester := ui.NewTester(app.View, 1280, 2200)
	tester.Frame()

	metrics, ok := tester.Find("Total Lotteries")
	require.True(t, ok)
	quick, ok := tester.Find("Quick start")
	require.True(t, ok)
	require.Less(t, metrics.Y, quick.Y, "counters must precede the action card")
}

// TestDashboardMetricCardNavigates makes the overview cards real shortcuts.
func TestDashboardMetricCardNavigates(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 1600)
	tester.Frame()

	require.NoError(t, tester.Click("Total Lotteries"))
	tester.Frame()
	require.Equal(t, pageLottery, app.page)
}

// TestDashboardActivityOpensItsModule turns a recent-activity row into the
// page that wrote it instead of leaving a dead highlight.
func TestDashboardActivityOpensItsModule(t *testing.T) {
	app := New(&fakeServices{blocks: []BlockRow{{Height: 1, Module: "voting", Summary: "ballot"}}})
	tester := ui.NewTester(app.View, 1280, 2200)
	tester.Frame()

	require.NoError(t, tester.Click("voting"))
	tester.Frame()
	require.Equal(t, pageVoting, app.page)
}

// TestLiveIndicatorOnlyOnLivePages keeps the opt-in timer visible without
// adding chrome to pages it does not refresh.
func TestLiveIndicatorOnlyOnLivePages(t *testing.T) {
	app := New(&fakeServices{})
	app.setLiveUpdates(true)
	tester := ui.NewTester(app.View, 1280, 1600)
	app.ShowPage("Dashboard")
	tester.Frame()
	require.True(t, tester.HasText("Live · 10s"))

	app.ShowPage("Token")
	tester.Frame()
	require.False(t, tester.HasText("Live · 10s"), "only Dashboard/Oracle are live")
}

// TestSettingsDocumentsTheWindowShortcuts makes the keyboard contract
// discoverable where the other app configuration lives.
func TestSettingsDocumentsTheWindowShortcuts(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Settings")
	tester.Frame()
	require.True(t, tester.HasText("Shortcuts"))
	require.True(t, tester.HasText(shortcutChord("R")))
	require.True(t, tester.HasText(shortcutChord(",")))
}

// TestDashboardMetricGridUsesTwoColumnsAtMinimumWidth keeps the counters
// readable at the 900px window minimum instead of squeezing three cards.
func TestDashboardMetricGridUsesTwoColumnsAtMinimumWidth(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 900, 2200)
	tester.Frame()

	first, ok := tester.Find("Total Lotteries")
	require.True(t, ok)
	second, ok := tester.Find("Total Votes")
	require.True(t, ok)
	third, ok := tester.Find("Candidates")
	require.True(t, ok)
	require.InDelta(t, float64(first.Y), float64(second.Y), 1, "two cards share the first row")
	require.Greater(t, third.Y, second.Y, "the third card wraps to the next row")
}

// TestCommandPaletteOpensAndNavigates pins the keyboard-first page switcher:
// Ctrl/Cmd+K opens it, typing narrows the list, and Enter takes the first match.
func TestCommandPaletteOpensAndNavigates(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 900)
	tester.Frame()

	tester.Key(platformRefreshModifier(), ui.KeyK)
	tester.Frame()
	require.True(t, tester.HasText("Jump to"))
	require.True(t, app.paletteOpen)

	app.paletteQuery = "Token"
	tester.Frame()
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	require.Equal(t, pageToken, app.page)
	require.False(t, app.paletteOpen)
}

// TestChineseCommandPaletteSearchesWhatItDisplays keeps the keyboard switcher
// usable in a translated window. The entries were drawn in Chinese but matched
// against their English source strings, so typing a visible Chinese title found
// nothing.
func TestChineseCommandPaletteSearchesWhatItDisplays(t *testing.T) {
	translator := i18n.GetTranslator()
	before := translator.GetLocale()
	t.Cleanup(func() { translator.SetLocale(before) })

	translator.SetLocale("zh")
	app := New(&fakeServices{})
	app.SetActions(Actions{
		BackUpDatabase:  func() {},
		ExportLotteries: func() {},
		ImportLotteries: func() {},
	})
	tester := ui.NewTester(app.View, 1280, 900)
	tester.Frame()
	tester.Key(platformRefreshModifier(), ui.KeyK)
	tester.Frame()

	require.True(t, tester.HasText("刷新当前页面"))
	require.True(t, tester.HasText("备份数据库…"))

	app.paletteQuery = "设置"
	tester.Frame()
	require.False(t, tester.HasText(fmt.Sprintf(tr("No pages match “%s”."), "设置")),
		"a visible Chinese page title must match the palette query")
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	require.Equal(t, pageSettings, app.page)
	require.False(t, app.paletteOpen)
}

// TestCommandPaletteClosesWithEscape keeps the modal dismissible without a
// navigation side effect.
func TestCommandPaletteClosesWithEscape(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 900)
	tester.Frame()
	tester.Key(platformRefreshModifier(), ui.KeyK)
	tester.Frame()
	require.True(t, app.paletteOpen)

	tester.Key(0, ui.KeyEscape)
	tester.Frame()
	require.False(t, app.paletteOpen)
}

// TestBackgroundWriteShowsBusyThenApplies covers the desktop write path: the
// worker leaves the frame thread, the busy banner is painted, and its apply
// closure updates the page once the worker returns.
func TestBackgroundWriteShowsBusyThenApplies(t *testing.T) {
	app := New(&fakeServices{})
	app.SetBackgroundWrites(true)
	tester := ui.NewTester(app.View, 1280, 900)
	app.ShowPage("Token")
	tester.Frame()

	started := make(chan struct{})
	release := make(chan struct{})
	app.runWrite("Minting…", func() func() {
		close(started)
		<-release
		return func() { app.token.status = okStatus("done") }
	})
	<-started
	tester.Frame()
	require.True(t, tester.HasText("Minting…"), "the busy banner must be painted before the worker finishes")
	require.True(t, app.asyncCh != nil)

	close(release)
	tester.Advance(60 * time.Millisecond)
	tester.Frame()
	require.False(t, tester.HasText("Minting…"))
	require.Equal(t, "done", app.token.status.text)
}

// TestSidebarNarrowsWithTheWindow keeps the navigation from taking a quarter
// of the minimum-width window.
func TestSidebarNarrowsWithTheWindow(t *testing.T) {
	require.Equal(t, float32(184), sidebarWidthFor(900))
	require.Equal(t, float32(216), sidebarWidthFor(1200))
}

// TestEmptyListsUseTheEmptyState checks the richer first-run state still names
// the missing object and the next move.
func TestEmptyListsUseTheEmptyState(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Token")
	tester.Frame()
	require.True(t, tester.HasText("No tokens yet"))
	require.True(t, tester.HasText("Create the first token above; the list, forms and history follow it."))
}

// TestCommandPaletteArrowSelection moves the highlight before Enter instead of
// always taking the first page.
func TestCommandPaletteArrowSelection(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 900)
	tester.Frame()
	tester.Key(platformRefreshModifier(), ui.KeyK)
	tester.Frame()

	tester.Key(0, ui.KeyDown)
	tester.Frame()
	tester.Key(0, ui.KeyDown)
	tester.Frame()
	require.Equal(t, 2, app.paletteChoice)
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	require.Equal(t, pageVoting, app.page)
}

// TestStatusLineCanBeDismissed keeps a stale success from describing the next
// action; the fixed strip is dismissible without waiting for a page switch.
func TestStatusLineCanBeDismissed(t *testing.T) {
	app := New(&fakeServices{})
	app.token.status = okStatus("done")
	tester := ui.NewTester(app.View, 1280, 1200)
	app.ShowPage("Token")
	tester.Frame()

	require.True(t, tester.HasText("done"))
	require.NoError(t, tester.Click("Dismiss"))
	tester.Frame()
	require.True(t, app.token.status.empty())
	require.False(t, tester.HasText("done"))
}

// TestBusyWriteDisablesTheFrame prevents edits and a second action from racing
// the in-flight write's result.
func TestBusyWriteDisablesTheFrame(t *testing.T) {
	app := New(&fakeServices{})
	app.SetBackgroundWrites(true)
	tester := ui.NewTester(app.View, 1280, 1600)
	tester.Frame()

	started := make(chan struct{})
	release := make(chan struct{})
	app.runWrite("Working…", func() func() {
		close(started)
		<-release
		return func() {}
	})
	<-started
	tester.Frame()
	require.True(t, app.busy != "")
	require.NoError(t, tester.Click("Total Lotteries"))
	tester.Frame()
	require.Equal(t, pageDashboard, app.page, "the dashboard must not navigate while a write owns the window")
	close(release)
	tester.Advance(60 * time.Millisecond)
	tester.Frame()
	require.Equal(t, "", app.busy)
}

// TestAllSetIsTheFormReadinessContract keeps the disabled-action helper
// honest: whitespace is not a filled field.
func TestAllSetIsTheFormReadinessContract(t *testing.T) {
	require.False(t, allSet("", "x"))
	require.False(t, allSet("  ", "x"))
	require.True(t, allSet("x", "y"))
}

// TestCommandPaletteRunsAnAction covers the second kind of palette entry:
// file actions use the same search/Enter path as pages.
func TestCommandPaletteRunsAnAction(t *testing.T) {
	app := New(&fakeServices{})
	backedUp := false
	app.SetActions(Actions{BackUpDatabase: func() { backedUp = true }})
	tester := ui.NewTester(app.View, 1280, 900)
	tester.Frame()
	tester.Key(platformRefreshModifier(), ui.KeyK)
	tester.Frame()

	app.paletteQuery = "Back up"
	tester.Frame()
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	require.True(t, backedUp)
	require.False(t, app.paletteOpen)
}

// TestSlowWriteShowsTimeoutNotice tells the operator that a long operation is
// still alive instead of leaving an unchanged spinner with no reassurance.
func TestSlowWriteShowsTimeoutNotice(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 1200)
	app.busy = "Minting…"
	app.busySince = time.Now().Add(-21 * time.Second)
	tester.Frame()
	require.True(t, tester.HasText("Still working…"))
}
