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
// Token page's two work columns sit side by side, while windows below 1240 put
// the second column below the first instead of squeezing controls past the
// card edge.
func TestCardsStackWhenTheWindowIsNarrow(t *testing.T) {
	wideApp := New(workbenchFakes())
	wide := ui.NewTester(wideApp.View, 1280, 3000)
	wideApp.ShowPage("Token")
	wide.Frame()
	create, ok := wide.Find("Define supply and attach the owner key that will authorize future writes.")
	require.True(t, ok)
	mint, ok := wide.Find("Only the token owner can mint, and only for a mintable token.")
	require.True(t, ok)
	require.NotEqual(t, create.X, mint.X, "a wide window keeps the two columns side by side")

	for _, width := range []int{900, 1239} {
		narrowApp := New(workbenchFakes())
		narrow := ui.NewTester(narrowApp.View, width, 5000)
		narrowApp.ShowPage("Token")
		narrow.Frame()
		create, ok := narrow.Find("Define supply and attach the owner key that will authorize future writes.")
		require.True(t, ok)
		mint, ok := narrow.Find("Only the token owner can mint, and only for a mintable token.")
		require.True(t, ok)
		require.Equalf(t, create.X, mint.X, "a %dpx window puts both columns on the same left edge", width)
		require.Greaterf(t, mint.Y, create.Y, "the second column must follow the first at %dpx", width)
	}
}

// TestTextScaleStacksCardsBeforeTheirControlsCompress ties the responsive
// threshold to the same text-size preference as the labels, and pins the
// stacked order: the column carrying the records comes first, the write column
// follows. At 1.5x text the default width no longer holds two columns.
func TestTextScaleStacksCardsBeforeTheirControlsCompress(t *testing.T) {
	cases := []struct {
		page   string
		object string
		action string
	}{
		{
			page:   "Token",
			object: "Define supply and attach the owner key that will authorize future writes.",
			action: "Only the token owner can mint, and only for a mintable token.",
		},
		{
			page:   "NFT",
			object: "Every stored asset, newest first. Choose a row to load its id into Transfer, Burn and History.",
			action: "Move an NFT to a new owner; the transfer is signed by the current one.",
		},
	}
	for _, tc := range cases {
		t.Run(tc.page, func(t *testing.T) {
			app := New(workbenchFakes())
			tester := ui.NewTester(app.View, 1280, 5000)
			tester.SetPreferences(ui.Preferences{TextScale: 1.5})
			app.ShowPage(tc.page)
			tester.Frame()

			object, ok := tester.Find(tc.object)
			require.True(t, ok)
			action, ok := tester.Find(tc.action)
			require.True(t, ok)
			require.Equal(t, object.X, action.X,
				"1.5x text needs the stacked layout at the default width")
			require.Greater(t, action.Y, object.Y,
				"the write column must follow the column carrying the records")
		})
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
	require.Equal(t,
		"抽奖导入文件无效：解析导入文件失败：invalid character 'n' looking for beginning of object key string",
		LocalizeError(errors.New("parse document: invalid character 'n' looking for beginning of object key string")))
	require.Equal(t,
		"抽奖导入文件无效：解析导入文件失败：应为抽奖记录 JSON 数组，但收到 null",
		LocalizeError(errors.New("parse document: expected a JSON array of lottery records, got null")))

	tr.SetLocale("en")
	const english = "parse document: invalid character 'n' looking for beginning of object key string"
	require.Equal(t, english, LocalizeError(errors.New(english)))
}

// TestChinesePagesShowNoEnglishChrome guards the translation table where it
// actually renders: a label that forgets tr(), or a toolkit string the view
// could override, shows as English in a Chinese window. MyGo's number-stepper
// labels ("Increase"/"Decrease") are its own and stay English, so they are
// deliberately not in this list.
func TestChinesePagesShowNoEnglishChrome(t *testing.T) {
	tr := i18n.GetTranslator()
	before := tr.GetLocale()
	t.Cleanup(func() { tr.SetLocale(before) })
	tr.SetLocale("zh")

	english := []string{
		"Refresh", "Search", "Show", "Next", "Previous",
		"Cancel", "Delete", "Copy", "Save", "Clear", "Loading",
	}
	app := New(workbenchFakes())
	tester := ui.NewTester(app.View, 1280, 4000)
	tester.Frame()
	for _, p := range pageOrder {
		app.ShowPage(pageTitles[p])
		tester.Frame()
		for _, word := range english {
			require.Falsef(t, tester.HasText(word),
				"%s still shows English %q in a Chinese window", pageTitles[p], word)
		}
	}
}

// TestChineseUIRendersTheWorkbench keeps the desktop's labels in the language
// switcher, not only its menus: page names, purposes, Quick start, metrics,
// tabs and the guide must come from the desktop table.
func TestChineseUIRendersTheWorkbench(t *testing.T) {
	tr := i18n.GetTranslator()
	before := tr.GetLocale()
	t.Cleanup(func() { tr.SetLocale(before) })

	tr.SetLocale("zh")
	app := New(workbenchFakes())
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
	require.True(t, tester.HasText("在“代币”列表选择或创建一个代币。它的所有者公钥会自动填入账户字段。"))

	app.ShowPage("Settings")
	tester.Frame()
	for _, want := range []string{
		"设置", "语言", "数据", "外观", "关于 Aurora",
		"主题", "深色", "字号", "对比度", "动效",
		"100%", "标准", "已启用",
		"备份数据库", "导出抽奖记录", "导入抽奖记录",
	} {
		require.Truef(t, tester.HasText(want), "the Chinese Settings page must show %q", want)
	}
	require.False(t, tester.HasText("Dark"), "the theme value must not remain English")
	require.False(t, tester.HasText("Follows system"), "appearance values must follow the language switch")
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
		{"Token", "Select a token in Tokens, or create one. Its owner public key fills the account fields for you."},
		{"Oracle", "Select a source in Sources to fetch it now, enable/disable it, or delete it."},
		{"NFT", "Transfer or Burn in the action tabs; the current holder's private key signs. A generated creator key can be reused while the creator still holds the asset."},
		{"Blockchain", "The badge verifies the chain. Search matches module, payload or height across the whole chain; Show sets the page size."},
	}
	for _, tc := range cases {
		t.Run(tc.page, func(t *testing.T) {
			app := New(workbenchFakes())
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
		tester := ui.NewTester(app.View, 1280, 860)
		tester.Frame()
		require.Equalf(t, pageDashboard, app.page, "%s must start from the dashboard", tc.label)
		quick, ok := tester.Find("Quick start")
		require.Truef(t, ok, "the dashboard must offer Quick start")
		app.scroll.Y = max(0, quick.Y-160)
		tester.Frame()
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
// destination. A bare "aurora://" used to parse as an empty page, which
// ShowIntent treats as "leave the window alone" — so a dashboard link from
// another page did nothing.
func TestDashboardLinksNavigateFromAnotherPage(t *testing.T) {
	app := New(&fakeServices{})

	app.ShowPage("Token")
	app.ShowIntent("dashboard", "")
	require.Equal(t, pageDashboard, app.page)

	app.ShowPage("Token")
	app.ShowPage("Dashboard")
	require.Equal(t, pageDashboard, app.page, "the page id and its title both name the dashboard")
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
// hierarchy: composition and recent activity come before the quick-start card.
func TestDashboardPrioritisesStateOverActions(t *testing.T) {
	app := New(&fakeServices{blocks: []BlockRow{{Height: 1, Module: "lottery", Summary: "draw"}}})
	tester := ui.NewTester(app.View, 1280, 2200)
	tester.Frame()

	composition, ok := tester.Find("What the ledger is made of")
	require.True(t, ok)
	quick, ok := tester.Find("Quick start")
	require.True(t, ok)
	require.Less(t, composition.Y, quick.Y, "the composition must precede the action card")
}

// TestDashboardCompositionRailNavigates makes the overview rails real
// shortcuts rather than decoration.
func TestDashboardCompositionRailNavigates(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 1600)
	tester.Frame()

	require.NoError(t, tester.Click("Total Lotteries"))
	tester.Frame()
	require.Equal(t, pageLottery, app.page)
}

// TestDashboardActivityOpensItsModule turns a recent-activity record into the
// page that wrote it instead of leaving a dead row. The record is one button,
// so the click lands on its rendered sentence rather than on a link beneath.
func TestDashboardActivityOpensItsModule(t *testing.T) {
	app := New(&fakeServices{blocks: []BlockRow{{
		Height: 1, Module: "voting", Action: "Ballot", Subject: "cand-1",
		Facts: []BlockFact{{Label: "Candidate", Value: "cand-1"}},
	}}})
	tester := ui.NewTester(app.View, 1280, 2200)
	tester.Frame()

	require.True(t, tester.HasText("cand-1"), "the record must name what it recorded")
	require.NoError(t, tester.Click("cand-1"))
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
	require.True(t, tester.HasText(
		"Navigation and window commands; Reload applies to pages that read stored data."))
	require.True(t, tester.HasText("Reload (data pages)"))
	require.True(t, tester.HasText(shortcutChord("R")))
	require.True(t, tester.HasText("Back up database"))
	require.True(t, tester.HasText(shortcutChord("B")))
	require.True(t, tester.HasText("Export lotteries"))
	require.True(t, tester.HasText(shortcutChord("E")))
	require.True(t, tester.HasText("Import lotteries"))
	require.True(t, tester.HasText(shortcutChord("I")))
	require.True(t, tester.HasText(shortcutChord(",")))
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
	require.True(t, tester.Focused("Navigation"),
		"the disposed palette search must hand focus to the sidebar")
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
	require.False(t, tester.HasText(fmt.Sprintf(tr("No matches for “%s”."), "设置")),
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

// TestEmptyCollectionKeepsItsEmptyStateWhenAQueryIsOpen pins the one state
// where the empty collection card is still reachable: a search left over from
// an earlier session keeps the full workbench, and the card must then say the
// store itself is empty rather than showing an empty list with no explanation.
// The untouched store opens on the first-record path instead.
func TestEmptyCollectionKeepsItsEmptyStateWhenAQueryIsOpen(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 5000)
	app.token.search, app.token.appliedSearch = "gold", "gold"
	app.ShowPage("Token")
	tester.Frame()
	require.True(t, tester.HasText("No tokens yet"))
	require.True(t, tester.HasText("Create the first token below; the list, forms and history follow it."))
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

// TestCommandPaletteRanksTitlePrefixesFirst pins the match order: a page whose
// title starts with the query must outrank one that only contains the letter.
// Without the ranking, typing "t" and pressing Enter opened Lottery because it
// comes first on the sidebar.
func TestCommandPaletteRanksTitlePrefixesFirst(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 1280, 900)
	tester.Frame()
	tester.Key(platformRefreshModifier(), ui.KeyK)
	tester.Frame()
	tester.Type("t")
	tester.Frame()

	matches := app.paletteMatches()
	require.NotEmpty(t, matches)
	require.Equal(t, pageToken, matches[0].page,
		"a title prefix must outrank a later sidebar page that contains the letter")

	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	require.Equal(t, pageToken, app.page)
}

// TestCommandPaletteKeepsLongListReachableAtMinimumHeight covers the
// keyboard-first switcher in the shortest supported window. Its page and file
// commands used to extend below the viewport with no scroll, so moving the
// selection to a lower command changed an invisible row.
func TestCommandPaletteKeepsLongListReachableAtMinimumHeight(t *testing.T) {
	for _, scale := range []float32{1, 1.5, 2} {
		t.Run(fmt.Sprintf("scale-%g", scale), func(t *testing.T) {
			app := New(&fakeServices{})
			shown := false
			app.SetActions(Actions{
				BackUpDatabase:  func() {},
				ExportLotteries: func() {},
				ImportLotteries: func() {},
				ShowDataFolder:  func() { shown = true },
			})
			tester := ui.NewTester(app.View, 900, 600)
			if scale != 1 {
				tester.SetPreferences(ui.Preferences{TextScale: scale})
			}
			tester.Frame()
			tester.Key(platformRefreshModifier(), ui.KeyK)
			tester.Frame()

			matches := app.paletteMatches()
			require.Greater(t, len(matches), 10)
			last := tr(matches[len(matches)-1].title)
			require.Equal(t, tr("Show data folder"), last)

			for range matches[1:] {
				tester.Key(0, ui.KeyDown)
				tester.Frame()
			}
			rect, ok := tester.Find(last)
			require.True(t, ok, "the selected command must stay in the built list")
			require.Greater(t, rect.W, float32(0))
			require.Greater(t, rect.H, float32(0))
			require.GreaterOrEqual(t, rect.Y, float32(0))
			require.LessOrEqual(t, rect.Y+rect.H, float32(600),
				"the selected command must be revealed inside the window")

			tester.Key(0, ui.KeyEnter)
			tester.Frame()
			require.True(t, shown, "the visible selection must still run its command")
			require.False(t, app.paletteOpen)
		})
	}
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
	for range 30 {
		if tester.Focused(tr("Dismiss")) {
			break
		}
		tester.Key(0, ui.KeyTab)
		tester.Frame()
	}
	require.True(t, tester.Focused(tr("Dismiss")), "the dismiss action must be keyboard reachable")
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	require.True(t, app.token.status.empty())
	require.False(t, tester.HasText("done"))
	require.True(t, tester.Focused("Navigation"),
		"the disappearing dismiss action must leave focus on the sidebar")
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

// TestTraditionalChineseUIRendersTheWorkbench: choosing 繁體中文 used to render
// the Simplified table, so the option named a script the window never showed.
// The Hant catalog keeps the chrome, lists and forms Traditional.
func TestTraditionalChineseUIRendersTheWorkbench(t *testing.T) {
	tr := i18n.GetTranslator()
	before := tr.GetLocale()
	t.Cleanup(func() { tr.SetLocale(before) })

	tr.SetLocale("zh-Hant")
	app := New(workbenchFakes())
	tester := ui.NewTester(app.View, 1280, 1600)
	tester.Frame()
	for _, want := range []string{"儀表盤", "快速開始", "運行抽獎", "抽獎總數", "賬本完整性"} {
		require.Truef(t, tester.HasText(want), "the Traditional dashboard must show %q", want)
	}

	app.ShowPage("Token")
	tester.Frame()
	for _, want := range []string{"代幣", "創建代幣", "鑄造", "轉賬", "查看", "歷史"} {
		require.Truef(t, tester.HasText(want), "the Traditional Token page must show %q", want)
	}
	require.False(t, tester.HasText("仪表盘"), "no Simplified chrome may slip through")
}

// TestTraditionalDesktopCatalogCoversEveryKey: the Hant table is generated from
// the Simplified one, so a key added on one side only would fall back to
// English (or Simplified) at run time.
func TestTraditionalDesktopCatalogCoversEveryKey(t *testing.T) {
	for key := range desktopMessages {
		require.Containsf(t, desktopMessagesHant, key, "%q has no Traditional entry", key)
	}
	require.Len(t, desktopMessagesHant, len(desktopMessages),
		"the Traditional table must not carry keys the Simplified table lacks")
}

// TestSettingsShowsTheDatabaseSize: the About readout asks the shell for the
// ledger's current size, so the number matches the path above it. A window
// built without the callback omits the row instead of showing a zero.
func TestSettingsShowsTheDatabaseSize(t *testing.T) {
	app := New(&fakeServices{})
	app.SetActions(Actions{DatabaseSize: func() string { return "1.5 MiB" }})
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Settings")
	tester.Frame()
	require.True(t, tester.HasText("Database size"))
	require.True(t, tester.HasText("1.5 MiB"))

	app.SetActions(Actions{})
	tester.Frame()
	require.False(t, tester.HasText("Database size"))
}
