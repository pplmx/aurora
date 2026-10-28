package nativeui

import (
	"fmt"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/i18n"
)

// TestMetricCardKeepsValueAndNoteApartAtLargeText covers the dashboard's
// fixed-height value slot. The value font scales with the OS text setting but
// the slot used not to, so at 2x the large number painted over its label and
// the sentence explaining what it counts.
func TestMetricCardKeepsValueAndNoteApartAtLargeText(t *testing.T) {
	tester := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Width(360).Padding(24).Children(func() {
			metricCard(c, "Records", "123", "Exact stored rows", ink, true, pageDashboard)
		})
	}, 420, 300)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	label, ok := tester.Find("Records")
	require.True(t, ok)
	value, ok := tester.Find("123")
	require.True(t, ok)
	note, ok := tester.Find("Exact stored rows")
	require.True(t, ok)

	require.LessOrEqual(t, label.Y+label.H, value.Y,
		"the enlarged value must start below its label")
	require.LessOrEqual(t, value.Y+value.H, note.Y,
		"the note must start below the enlarged value")
}

// TestSidebarScrollsAtLargeText keeps every module reachable in the shortest
// supported window. At 2x the navigation is taller than the fixed sidebar, so
// the Settings row and File action used to paint below the window edge with no
// way to scroll them into view.
func TestSidebarScrollsAtLargeText(t *testing.T) {
	app := New(&fakeServices{})
	app.SetActions(Actions{BackUpDatabase: func() {}})
	tester := ui.NewTester(app.View, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	file, ok := tester.Find(i18n.GetText("desktop.menu.file"))
	require.True(t, ok)
	require.LessOrEqual(t, file.Y+file.H, float32(600),
		"the fixed File action must stay inside the window")

	tester.Scroll(90, 300, 0, 600)
	tester.Frame()
	settings, ok := tester.Find(tr(pageTitles[pageSettings]))
	require.True(t, ok)
	require.GreaterOrEqual(t, settings.Y, float32(0))
	require.LessOrEqual(t, settings.Y+settings.H, float32(600),
		"scrolling the module list must reveal Settings inside the sidebar")

	require.NoError(t, tester.Click(tr(pageTitles[pageSettings])))
	tester.Frame()
	require.Equal(t, pageSettings, app.page)
}

// TestSidebarFileActionScalesWithLargeText keeps the pinned File control from
// squeezing its scaled label into fixed chrome. It shares the action button's
// 34px base height, so at 2x it must grow just as the card actions do.
func TestSidebarFileActionScalesWithLargeText(t *testing.T) {
	app := New(&fakeServices{})
	app.SetActions(Actions{BackUpDatabase: func() {}})
	tester := ui.NewTester(app.View, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	file, ok := tester.Find(i18n.GetText("desktop.menu.file"))
	require.True(t, ok)
	require.GreaterOrEqual(t, file.H, float32(68),
		"the pinned File action must scale its full 34px height at 2x")
}

// TestSidebarRevealsKeyboardPageChoiceAtLargeText covers keyboard navigation:
// changing the page must reveal the newly selected row instead of changing an
// item that is still below the visible scroll area.
func TestSidebarRevealsKeyboardPageChoiceAtLargeText(t *testing.T) {
	app := New(&fakeServices{})
	tester := ui.NewTester(app.View, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	tester.Key(platformRefreshModifier(), ui.KeyComma)
	tester.Frame()
	require.Equal(t, pageSettings, app.page)

	settings, ok := tester.Find(tr(pageTitles[pageSettings]))
	require.True(t, ok)
	require.Less(t, settings.X, sidebarWidthFor(900),
		"the assertion must inspect the sidebar row, not the page heading")
	require.GreaterOrEqual(t, settings.Y, float32(0))
	require.LessOrEqual(t, settings.Y+settings.H, float32(600),
		"the selected module must be revealed in the sidebar")
}

// TestSidebarGivesLargeEnglishLabelsEnoughRoom keeps the scaled navigation
// from growing its type inside a fixed-width column. At 1.5x and 2x the brand
// subtitle and module names used to be cut to prefixes even though the window
// had room to widen the sidebar.
func TestSidebarGivesLargeEnglishLabelsEnoughRoom(t *testing.T) {
	for _, scale := range []float32{1.5, 2} {
		t.Run(fmt.Sprintf("scale-%g", scale), func(t *testing.T) {
			app := New(&fakeServices{})
			var brandWidth, moduleWidth float32
			tester := ui.NewTester(func(c *ui.Context) {
				app.View(c)
				brandWidth, _ = c.MeasureText(0, ui.Span{
					Text: "Verifiable ledger suite",
					Size: textSize(c, 11),
				})
				moduleWidth, _ = c.MeasureText(0, ui.Span{
					Text: "Blockchain",
					Size: c.Theme().FontSize,
				})
			}, 900, 600)
			tester.SetPreferences(ui.Preferences{TextScale: scale})
			tester.Frame()

			brand, ok := tester.Find("Verifiable ledger suite")
			require.True(t, ok)
			module, ok := tester.Find("Blockchain")
			require.True(t, ok)
			require.GreaterOrEqual(t, brand.W, brandWidth,
				"the brand subtitle needs its measured width at %.1fx", scale)
			require.GreaterOrEqual(t, module.W, moduleWidth,
				"the longest module label needs its measured width at %.1fx", scale)
		})
	}
}

// TestOracleSourceActionsWrapAtLargeText keeps the four per-source actions
// inside their card. At 2x in the minimum window, the unwrapped row pushed the
// destructive Delete button past the right edge, leaving only part of it on
// screen.
func TestOracleSourceActionsWrapAtLargeText(t *testing.T) {
	app := New(&fakeServices{sources: []SourceRow{{
		ID: "source-1", Name: "BTC feed", URL: "https://example.com/price",
		Type: "http", Enabled: true, Interval: 60,
	}}})
	tester := ui.NewTester(app.View, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	app.ShowPage("Oracle")
	tester.Frame()
	app.scroll.Y = math.MaxFloat32
	tester.Frame()

	enable, ok := tester.Find("Enable")
	require.True(t, ok)
	for _, label := range []string{"Enable", "Disable", "Fetch now", "Delete"} {
		rect, ok := tester.Find(label)
		require.Truef(t, ok, "%q action must be built", label)
		require.GreaterOrEqualf(t, rect.X, float32(0), "%q action starts in the window", label)
		require.LessOrEqualf(t, rect.X+rect.W, float32(900), "%q action stays in the window", label)
	}
	deleteAction, ok := tester.Find("Delete")
	require.True(t, ok)
	require.Greater(t, deleteAction.Y, enable.Y,
		"the action row must wrap instead of clipping its destructive action")
}

// TestOracleSourcesUseRecordsAtLargeText replaces the five-column source table
// when enlarged text no longer fits it. At 2x the fixed type, id, status and
// headers columns pushed values outside the card, while the shortened id left
// an operator unable to identify the source before loading it into the actions.
func TestOracleSourcesUseRecordsAtLargeText(t *testing.T) {
	const (
		sourceID = "20261025233845.347259429-29ee362515b7c1fc06313345ef6be416"
		name     = "A weather feed whose full display name must remain readable"
	)
	fake := &fakeServices{sources: []SourceRow{{
		ID: sourceID, Name: name, URL: "https://example.com/weather",
		Type: "http-json", Enabled: true, Interval: 60, HasHeaders: true,
	}}}
	app := New(fake)
	app.ShowPage("Oracle")

	var nameWidth, nameHeight float32
	tester := ui.NewTester(func(c *ui.Context) {
		app.View(c)
		nameWidth, nameHeight = c.MeasureText(0, ui.Span{
			Text: name, Size: textSize(c, 13),
		})
	}, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	first, ok := tester.Find(name)
	require.True(t, ok)
	app.scroll.Y = max(0, first.Y-250)
	tester.Frame()

	nameRect, ok := tester.Find(name)
	require.True(t, ok)
	require.True(t, nameRect.W >= nameWidth || nameRect.H >= 1.5*nameHeight,
		"the full source name must wrap instead of being clipped: rect=%+v measured=(%g,%g)",
		nameRect, nameWidth, nameHeight)
	require.True(t, tester.HasText(sourceID),
		"the source record must show the complete id before selection")
	require.True(t, tester.HasText("http-json"))
	require.True(t, tester.HasText("configured"), "the header state must remain visible")

	app.oracle.data = []DataRow{{Value: "18"}}
	app.oracle.latest = &DataRow{Value: "18"}
	require.NoError(t, tester.Click(name))
	tester.Frame()
	require.Equal(t, sourceID, app.oracle.fetchID)
	require.Equal(t, sourceID, app.oracle.queryID)
	require.Equal(t, sourceID, app.oracle.latestID)
	require.Equal(t, sourceID, fake.lastOracleDataSource)
	require.Equal(t, sourceID, fake.lastOracleLatestSource)
	_, ok = tester.Find(fmt.Sprintf(tr("Selected %s"), name))
	require.True(t, ok, "the chosen source must announce its selected state")
}

// TestOracleReadingsUseRecordsAtLargeText keeps a stored value readable after
// the system enlarges its type. The value column is deliberately variable
// width, while the table cell is single-line; at 2x a nested JSON reading or
// long text is reduced to a prefix even though the card has room to stack it
// beneath its block.
func TestOracleReadingsUseRecordsAtLargeText(t *testing.T) {
	const (
		sourceID = "source-0123456789abcdef"
		value    = `{"temperature":21.5,"humidity":46,"station":"north","observed_at":"2026-10-28T08:15:00+08:00"}`
	)
	app := New(&fakeServices{})
	app.ShowPage("Oracle")
	app.oracle.tab = 2
	app.oracle.queryID = sourceID
	app.oracle.data = []DataRow{{Value: value, BlockHeight: 42}}
	app.oracle.dataLoadedSource = sourceID
	app.oracle.dataLoadedLimit = app.oracleQueryLimit()

	var valueWidth, valueHeight float32
	tester := ui.NewTester(func(c *ui.Context) {
		app.View(c)
		valueWidth, valueHeight = c.MeasureText(0, ui.Span{
			Text: value, Size: textSize(c, 12),
		})
	}, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	first, ok := tester.Find(value)
	require.True(t, ok)
	app.scroll.Y = max(0, first.Y-250)
	tester.Frame()

	valueRect, ok := tester.Find(value)
	require.True(t, ok)
	require.True(t, valueRect.W >= valueWidth || valueRect.H >= 1.5*valueHeight,
		"the complete reading must wrap instead of being clipped: rect=%+v measured=(%g,%g)",
		valueRect, valueWidth, valueHeight)
	require.True(t, tester.HasText("42"), "the block height must remain beside the value")
}

// TestOracleLatestValueWrapsAtLargeText covers the single-value readout next
// to Query Data. It is not a table, but its value shared the same one-line
// treatment, so the latest nested JSON at 2x was cut to a prefix as well.
func TestOracleLatestValueWrapsAtLargeText(t *testing.T) {
	const (
		sourceID = "source-0123456789abcdef"
		value    = `{"temperature":21.5,"humidity":46,"station":"north","observed_at":"2026-10-28T08:15:00+08:00"}`
	)
	app := New(&fakeServices{})
	app.ShowPage("Oracle")
	app.oracle.tab = 2
	app.oracle.latestID = sourceID
	app.oracle.latest = &DataRow{Value: value, BlockHeight: 42}
	app.oracle.latestLoadedSource = sourceID

	var valueWidth, valueHeight float32
	tester := ui.NewTester(func(c *ui.Context) {
		app.View(c)
		valueWidth, valueHeight = c.MeasureText(0, ui.Span{
			Text: value, Size: textSize(c, 12),
		})
	}, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	first, ok := tester.Find(value)
	require.True(t, ok)
	app.scroll.Y = max(0, first.Y-250)
	tester.Frame()

	valueRect, ok := tester.Find(value)
	require.True(t, ok)
	require.True(t, valueRect.W >= valueWidth || valueRect.H >= 1.5*valueHeight,
		"the latest value must wrap instead of being clipped: rect=%+v measured=(%g,%g)",
		valueRect, valueWidth, valueHeight)
}

// TestCopyRowKeepsValueVisibleAtLargeText covers one-time secrets such as the
// generated token owner key. At 2x the fixed label and full-width copy action
// left the value with no width at all, so the row offered a copy button but
// showed no trace of what it would copy.
func TestCopyRowKeepsValueVisibleAtLargeText(t *testing.T) {
	app := New(&fakeServices{})
	app.token.ownerPriv = "secret"
	app.ShowPage("Token")
	app.scroll.Y = 900
	tester := ui.NewTester(app.View, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	value, ok := tester.Find("secret")
	require.True(t, ok)
	require.Greater(t, value.W, float32(0),
		"the value must keep visible width beside its label and copy action")
	copyAction, ok := tester.Find("Copy Owner private key")
	require.True(t, ok)
	require.LessOrEqual(t, copyAction.X+copyAction.W, float32(900),
		"the copy action must stay inside the window")
}

// TestTokenWriteTabsWrapAtLargeText covers the five write tabs at the minimum
// window width. At 2x the built-in single-line tab row placed Burn beyond the
// right edge, so the destructive form had no visible entry point.
func TestTokenWriteTabsWrapAtLargeText(t *testing.T) {
	app := New(&fakeServices{})
	app.ShowPage("Token")
	tester := ui.NewTester(app.View, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	rects := make(map[string]ui.Rect)
	for _, label := range []string{"Mint", "Transfer", "Approve", "Transfer From", "Burn"} {
		rect, ok := tester.Find(label)
		require.Truef(t, ok, "%q tab must be built", label)
		require.LessOrEqualf(t, rect.X+rect.W, float32(900),
			"%q tab must stay inside the window", label)
		rects[label] = rect
	}
	require.Greater(t, rects["Burn"].Y, rects["Mint"].Y,
		"the tab row must wrap instead of pushing Burn off screen")

	app.scroll.Y = rects["Burn"].Y - 300
	tester.Frame()
	require.NoError(t, tester.Click("Burn"))
	tester.Frame()
	require.Equal(t, 4, app.token.opTab, "the wrapped Burn tab must remain operable")
}

// TestLotteryHistoryStacksAtLargeText keeps every draw fact readable without
// sideways scrolling. At 2x the four-column table put Winners at x=901 in the
// 900px window, and the row itself had no visible action for loading its full
// id into Verify.
func TestLotteryHistoryStacksAtLargeText(t *testing.T) {
	drawID := "draw-0123456789abcdef"
	app := New(&fakeServices{draws: []DrawRow{{
		ID: drawID, BlockHeight: 42,
		Participants: []string{"A", "B", "C"}, Winners: []string{"A"},
	}}})
	app.ShowPage("Lottery")
	tester := ui.NewTester(app.View, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	for _, label := range []string{"Participants", "Winners"} {
		rect, ok := tester.Find(label)
		require.Truef(t, ok, "%q field must be shown", label)
		require.LessOrEqualf(t, rect.X+rect.W, float32(900),
			"%q must stay inside the window", label)
	}
	action, ok := tester.Find("Use this draw for Verify")
	require.True(t, ok)
	app.scroll.Y = action.Y - 300
	tester.Frame()
	require.NoError(t, tester.Click("Use this draw for Verify"))
	tester.Frame()
	require.Equal(t, drawID, app.lottery.verifyID,
		"the stacked record must still load the full id into Verify")
}

// TestOracleHealthStacksAtLargeText replaces the five-column health table when
// enlarged type no longer fits it. At 2x the Last error header began at x=931,
// outside the 900px window, so the most actionable symptom was inaccessible.
func TestOracleHealthStacksAtLargeText(t *testing.T) {
	app := New(&fakeServices{
		sources: []SourceRow{{
			ID: "source-0123456789abcdef", Name: "BTC feed",
		}},
		healthRows: []HealthRow{{
			SourceID: "source-0123456789abcdef", Name: "BTC feed",
			Attempts: 10, Successes: 9, Failures: 1,
			LastError: "temporary upstream error",
		}},
	})
	app.ShowPage("Oracle")
	app.oracle.tab = 1
	tester := ui.NewTester(app.View, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	for _, text := range []string{"Last error", "temporary upstream error"} {
		rect, ok := tester.Find(text)
		require.Truef(t, ok, "%q must be shown", text)
		require.LessOrEqualf(t, rect.X+rect.W, float32(900),
			"%q must stay inside the window", text)
	}
}

// TestQuietButtonsGrowWithLargeText covers copy, dismiss and other low-chrome
// actions. They kept a 26px height while their 12pt label measured 33px high
// at 2x, so the text was vertically clipped inside the hit target.
func TestQuietButtonsGrowWithLargeText(t *testing.T) {
	var textHeight float32
	tester := ui.NewTester(func(c *ui.Context) {
		quietButton(c, "Dismiss key")
		_, textHeight = c.MeasureText(0, ui.Span{
			Text: "Dismiss key", Size: textSize(c, 12),
		})
	}, 320, 120)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	button, ok := tester.Find("Dismiss key")
	require.True(t, ok)
	require.GreaterOrEqual(t, button.H, textHeight,
		"the quiet button must be at least as tall as its scaled label")
}

// TestActionButtonsScaleWithLargeText keeps the three button styles from
// packing a 33px label into a 34px box at 2x. The base control is 34px tall,
// so scaling the whole action preserves its normal breathing room instead of
// only preventing a one-pixel clip.
func TestActionButtonsScaleWithLargeText(t *testing.T) {
	tester := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Gap(8).Children(func() {
			primaryButton(c, "Create")
			secondaryButton(c, "Refresh")
			dangerButton(c, "Delete")
		})
	}, 320, 260)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	for _, label := range []string{"Create", "Refresh", "Delete"} {
		button, ok := tester.Find(label)
		require.Truef(t, ok, "%q button must be built", label)
		require.GreaterOrEqualf(t, button.H, float32(68),
			"%q must scale its full 34px height at 2x", label)
	}
}

// TestTextAreasScaleWithLargeText keeps multiline fields showing several lines
// after the system enlarges their type. The fixed 64px box held barely one
// scaled line at 2x, so editing JSON headers or a long description meant
// scrolling through a viewport the height of the caret.
func TestTextAreasScaleWithLargeText(t *testing.T) {
	var area ui.Handle
	var ctx *ui.Context
	value := "{}"
	tester := ui.NewTester(func(c *ui.Context) {
		ctx = c
		textAreaControl(c, &value).Ref(&area)
	}, 420, 360)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	bounds := area.Bounds(ctx)
	require.GreaterOrEqual(t, bounds.H, float32(128),
		"a multiline field must scale its full 64px height at 2x")
}

// TestTokenCollectionUsesRecordsAtLargeText replaces the four-column token
// table when enlarged text no longer fits it. At 2x in the minimum window the
// Supply header was clipped to 16px, a long token name to 100px, and even the
// shortened id lost characters, so the collection could not identify its own
// rows without horizontal scrolling.
func TestTokenCollectionUsesRecordsAtLargeText(t *testing.T) {
	const (
		tokenID = "token-0123456789abcdef"
		name    = "A treasury token whose full name must remain readable"
		supply  = "9223372036854775807"
	)
	app := New(&fakeServices{tokenRows: []TokenRow{{
		ID: tokenID, Name: name, Symbol: "TREASURY", TotalSupply: supply,
	}}})
	app.ShowPage("Token")

	var nameWidth, nameHeight, supplyLabelWidth float32
	tester := ui.NewTester(func(c *ui.Context) {
		app.View(c)
		nameWidth, nameHeight = c.MeasureText(0, ui.Span{
			Text: name, Size: textSize(c, 13),
		})
		supplyLabelWidth, _ = c.MeasureText(0, ui.Span{
			Text: "Supply", Size: textSize(c, 10),
		})
	}, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	first, ok := tester.Find(name)
	require.True(t, ok)
	app.scroll.Y = max(0, first.Y-250)
	tester.Frame()

	nameRect, ok := tester.Find(name)
	require.True(t, ok)
	require.True(t, nameRect.W >= nameWidth || nameRect.H >= 1.5*nameHeight,
		"the full token name must wrap instead of being clipped: rect=%+v measured=(%g,%g)",
		nameRect, nameWidth, nameHeight)

	supplyLabel, ok := tester.Find("Supply")
	require.True(t, ok)
	require.GreaterOrEqual(t, supplyLabel.W, supplyLabelWidth,
		"the Supply label must keep its full width in a record")
	require.True(t, tester.HasText(tokenID),
		"the record must show the full token id, not only the table's abbreviation")

	require.NoError(t, tester.Click(name))
	tester.Frame()
	require.Equal(t, tokenID, app.token.tokenID,
		"clicking the record must make the full token id the page's subject")
	require.Equal(t, 0, app.token.chosen)
	_, ok = tester.Find(fmt.Sprintf(tr("Selected %s"), name))
	require.True(t, ok, "the chosen record must announce its selected state")
	require.True(t, tester.HasText("Selected supply"))
}

// TestNFTCollectionUsesRecordsAtLargeText applies the same large-text rule to
// the asset collection. Its name cell was clipped to a short prefix while the
// id column showed only a fragment, so the row could not identify the asset
// before the operator made it the subject of four write cards.
func TestNFTCollectionUsesRecordsAtLargeText(t *testing.T) {
	const (
		nftID = "nft-0123456789abcdef"
		name  = "A generative artwork whose full name remains readable"
	)
	app := New(&fakeServices{nftRows: []NFTRow{{
		ID: nftID, Name: name, BlockHeight: 42,
	}}})
	app.ShowPage("NFT")

	var nameWidth, nameHeight float32
	tester := ui.NewTester(func(c *ui.Context) {
		app.View(c)
		nameWidth, nameHeight = c.MeasureText(0, ui.Span{
			Text: name, Size: textSize(c, 13),
		})
	}, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	first, ok := tester.Find(name)
	require.True(t, ok)
	app.scroll.Y = max(0, first.Y-250)
	tester.Frame()

	nameRect, ok := tester.Find(name)
	require.True(t, ok)
	require.True(t, nameRect.W >= nameWidth || nameRect.H >= 1.5*nameHeight,
		"the full NFT name must wrap instead of being clipped: rect=%+v measured=(%g,%g)",
		nameRect, nameWidth, nameHeight)
	require.True(t, tester.HasText(nftID),
		"the record must show the full NFT id before it becomes the subject")

	require.NoError(t, tester.Click(name))
	tester.Frame()
	for _, id := range []string{
		app.nft.id, app.nft.transferID, app.nft.burnID, app.nft.historyID,
	} {
		require.Equal(t, nftID, id)
	}
	require.Equal(t, 0, app.nft.selected)
	_, ok = tester.Find(fmt.Sprintf(tr("Selected %s"), name))
	require.True(t, ok, "the chosen record must announce its selected state")
}

// TestTokenHistoryUsesRecordsAtLargeText keeps both sides of a transfer and
// its exact amount readable at 2x. The four fixed-width columns reduced public
// keys to short prefixes and pushed the amount and nonce together, even though
// a transfer history is read as one event rather than four unrelated cells.
func TestTokenHistoryUsesRecordsAtLargeText(t *testing.T) {
	const (
		tokenID = "token-0123456789abcdef"
		from    = "RnJvbS1hY2NvdW50LXdpdGgtYS1sb25nLWJhc2U2NC1wdWJsaWMta2V5"
		to      = "VG8tYWNjb3VudC13aXRoLWEtbG9uZy1iYXNlNjQtcHVibGljLWtleQ=="
		amount  = "9223372036854775807"
	)
	app := New(&fakeServices{})
	app.ShowPage("Token")
	app.token.tokenID = tokenID
	app.token.history = []TransferRow{{
		From: from, To: to, Amount: amount, Nonce: 18446744073709551615,
	}}
	app.token.historyLoadedToken, app.token.historyLoadedOwner = tokenID, ""
	app.token.historyLoadedLimit = 50
	app.token.historyLoadedOffset = 0

	var fromWidth, fromHeight float32
	tester := ui.NewTester(func(c *ui.Context) {
		app.View(c)
		fromWidth, fromHeight = c.MeasureText(0, ui.Span{
			Text: from, Size: textSize(c, 12),
		})
	}, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	first, ok := tester.Find(from)
	require.True(t, ok)
	app.scroll.Y = max(0, first.Y-250)
	tester.Frame()

	fromRect, ok := tester.Find(from)
	require.True(t, ok)
	require.True(t, fromRect.W >= fromWidth || fromRect.H >= 1.5*fromHeight,
		"the complete sender key must wrap instead of being clipped: rect=%+v measured=(%g,%g)",
		fromRect, fromWidth, fromHeight)
	require.True(t, tester.HasText(to), "the complete recipient key must remain visible")
	require.True(t, tester.HasText(amount), "the exact amount must remain visible")
	require.True(t, tester.HasText("18446744073709551615"), "the nonce must remain visible")
}

// TestNFTHistoryUsesRecordsAtLargeText applies the same rule to the operation
// trail. Type, both account keys and the block height used to compete for a
// width that could not hold them at 2x; one record now keeps the operation and
// provenance together without truncating the parties.
func TestNFTHistoryUsesRecordsAtLargeText(t *testing.T) {
	const (
		nftID = "nft-0123456789abcdef"
		from  = "QWxpY2UtYWNjb3VudC13aXRoLWEtbG9uZy1iYXNlNjQtcHVibGljLWtleQ=="
		to    = "Qm9iLWFjY291bnQtd2l0aC1hLWxvbmctYmFzZTY0LXB1YmxpYy1rZXk="
		block = int64(9223372036854775807)
	)
	app := New(&fakeServices{})
	app.ShowPage("NFT")
	app.nft.opTab = 2
	app.nft.historyID = nftID
	app.nft.history = []NFTOpRow{{
		Type: "transfer", From: from, To: to, BlockHeight: block,
	}}
	app.nft.historyLoadedID = nftID

	var fromWidth, fromHeight float32
	tester := ui.NewTester(func(c *ui.Context) {
		app.View(c)
		fromWidth, fromHeight = c.MeasureText(0, ui.Span{
			Text: from, Size: textSize(c, 12),
		})
	}, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	first, ok := tester.Find(from)
	require.True(t, ok)
	app.scroll.Y = max(0, first.Y-250)
	tester.Frame()

	fromRect, ok := tester.Find(from)
	require.True(t, ok)
	require.True(t, fromRect.W >= fromWidth || fromRect.H >= 1.5*fromHeight,
		"the complete previous owner key must wrap instead of being clipped: rect=%+v measured=(%g,%g)",
		fromRect, fromWidth, fromHeight)
	require.True(t, tester.HasText(to), "the complete next owner key must remain visible")
	require.True(t, tester.HasText("transfer"), "the operation type must remain visible")
	require.True(t, tester.HasText("9223372036854775807"), "the exact block height must remain visible")
}

// TestDashboardRecentUsesRecordsAtLargeText keeps the latest audit event
// readable at 2x. The three-column table reduced its payload to a prefix, and
// the row's navigation target was the only way to learn what the event said.
func TestDashboardRecentUsesRecordsAtLargeText(t *testing.T) {
	const (
		summary = `{"module":"token","description":"A long token event whose complete audit payload must remain readable"}`
		block   = int64(9223372036854775807)
	)
	app := New(&fakeServices{blocks: []BlockRow{{
		Height: block, Module: "token", Summary: summary,
	}}})

	var summaryWidth, summaryHeight float32
	tester := ui.NewTester(func(c *ui.Context) {
		app.View(c)
		summaryWidth, summaryHeight = c.MeasureText(0, ui.Span{
			Text: summary, Size: textSize(c, 12),
		})
	}, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	first, ok := tester.Find(summary)
	require.True(t, ok)
	app.scroll.Y = max(0, first.Y-250)
	tester.Frame()

	summaryRect, ok := tester.Find(summary)
	require.True(t, ok)
	require.True(t, summaryRect.W >= summaryWidth || summaryRect.H >= 1.5*summaryHeight,
		"the full recent payload must wrap instead of being clipped: rect=%+v measured=(%g,%g)",
		summaryRect, summaryWidth, summaryHeight)
	require.True(t, tester.HasText("9223372036854775807"), "the exact block height must remain visible")
	require.True(t, tester.HasText("token"), "the source module must remain visible")

	require.NoError(t, tester.Click(summary))
	tester.Frame()
	require.Equal(t, pageToken, app.page,
		"choosing the complete recent record must still open its module")
}

// TestBlockchainUsesRecordsAtLargeText keeps each ledger payload inside the
// audit page at 2x. The height and module columns left the one-line payload
// cell with a narrow strip, so an auditor saw only the first part of the event
// that the chain is meant to preserve.
func TestBlockchainUsesRecordsAtLargeText(t *testing.T) {
	const (
		summary = `{"module":"oracle","source":"btc","price":12345,"observed_at":"2026-10-28T08:15:00+08:00"}`
		block   = int64(9223372036854775807)
	)
	app := New(&fakeServices{blocks: []BlockRow{{
		Height: block, Module: "oracle", Summary: summary,
	}}})
	app.ShowPage("Blockchain")

	var summaryWidth, summaryHeight float32
	tester := ui.NewTester(func(c *ui.Context) {
		app.View(c)
		summaryWidth, summaryHeight = c.MeasureText(0, ui.Span{
			Text: summary, Size: textSize(c, 12),
		})
	}, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	first, ok := tester.Find(summary)
	require.True(t, ok)
	app.scroll.Y = max(0, first.Y-250)
	tester.Frame()

	summaryRect, ok := tester.Find(summary)
	require.True(t, ok)
	require.True(t, summaryRect.W >= summaryWidth || summaryRect.H >= 1.5*summaryHeight,
		"the complete ledger payload must wrap instead of being clipped: rect=%+v measured=(%g,%g)",
		summaryRect, summaryWidth, summaryHeight)
	require.True(t, tester.HasText("9223372036854775807"), "the exact height must remain visible")
	require.True(t, tester.HasText("oracle"), "the source module must remain visible")
}

// TestVotingResultsUseRecordsAtLargeText keeps a long candidate name intact
// beside its tally at 2x. The two-column table made Votes readable but reduced
// the candidate to a single-line prefix, leaving the count unattached to a
// full identity.
func TestVotingResultsUseRecordsAtLargeText(t *testing.T) {
	const (
		sessionID = "session-0123456789abcdef"
		name      = "A coalition candidate whose complete ballot name must remain readable"
	)
	app := New(&fakeServices{})
	app.ShowPage("Voting")
	app.voting.tab = 2
	app.voting.resultsSession = sessionID
	app.voting.results = []ResultRow{{CandidateID: "candidate-1", Name: name, Votes: 123}}
	app.voting.resultsLoadedID = sessionID

	var nameWidth, nameHeight float32
	tester := ui.NewTester(func(c *ui.Context) {
		app.View(c)
		nameWidth, nameHeight = c.MeasureText(0, ui.Span{
			Text: name, Size: textSize(c, 13),
		})
	}, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 2})
	tester.Frame()

	first, ok := tester.Find(name)
	require.True(t, ok)
	app.scroll.Y = max(0, first.Y-250)
	tester.Frame()

	nameRect, ok := tester.Find(name)
	require.True(t, ok)
	require.True(t, nameRect.W >= nameWidth || nameRect.H >= 1.5*nameHeight,
		"the complete candidate name must wrap instead of being clipped: rect=%+v measured=(%g,%g)",
		nameRect, nameWidth, nameHeight)
	require.True(t, tester.HasText("123"), "the tally must remain beside the candidate")
}

// TestVotingCollectionsUseRecordsAtLargeText applies the collection rule to
// the three Setup lists. At 2x the voter table pushed the public key off the
// card, candidate names and party labels were clipped, and the session table's
// four columns could not identify a row before the operator loaded it into the
// ballot and results cards.
func TestVotingCollectionsUseRecordsAtLargeText(t *testing.T) {
	const (
		publicKey   = "VGVzdCBwdWJsaWMga2V5IHRoYXQgaXMgbG9uZyBlbm91Z2ggdG8gbmVlZCBtdWx0aXBsZSBsaW5lcw=="
		candidateID = "candidate-0123456789abcdef"
		sessionID   = "session-0123456789abcdef"
	)
	tests := []struct {
		name     string
		itemName string
		fullID   string
		services *fakeServices
		check    func(*testing.T, *App)
	}{
		{
			name:     "voter",
			itemName: "A registered voter whose full display name must remain readable",
			fullID:   publicKey,
			services: &fakeServices{voterRows: []VoterRow{{
				PublicKey: publicKey, Name: "A registered voter whose full display name must remain readable",
			}}},
			check: func(t *testing.T, app *App) {
				require.Equal(t, publicKey, app.voting.voteVoterPub)
				require.Equal(t, 0, app.voting.voterChoice)
			},
		},
		{
			name:     "candidate",
			itemName: "A candidate whose full display name must remain readable",
			fullID:   candidateID,
			services: &fakeServices{candidates: []CandidateRow{{
				ID: candidateID, Name: "A candidate whose full display name must remain readable",
				Party: "Independent",
			}}},
			check: func(t *testing.T, app *App) {
				require.Equal(t, candidateID, app.voting.voteCandidate)
				require.Equal(t, 0, app.voting.candChoice)
			},
		},
		{
			name:     "session",
			itemName: "A governance session whose full title must remain readable",
			fullID:   sessionID,
			services: &fakeServices{sessions: []SessionRow{{
				ID: sessionID, Title: "A governance session whose full title must remain readable",
				Status: "active", StartTime: 1_800_000_000, EndTime: 1_800_086_400,
			}}},
			check: func(t *testing.T, app *App) {
				require.Equal(t, sessionID, app.voting.controlID)
				require.Equal(t, sessionID, app.voting.voteSession)
				require.Equal(t, sessionID, app.voting.resultsSession)
				require.Equal(t, 0, app.voting.sessionChoice)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := New(tt.services)
			app.ShowPage("Voting")

			var nameWidth, nameHeight float32
			tester := ui.NewTester(func(c *ui.Context) {
				app.View(c)
				nameWidth, nameHeight = c.MeasureText(0, ui.Span{
					Text: tt.itemName, Size: textSize(c, 13),
				})
			}, 900, 600)
			tester.SetPreferences(ui.Preferences{TextScale: 2})
			tester.Frame()

			first, ok := tester.Find(tt.itemName)
			require.True(t, ok)
			app.scroll.Y = max(0, first.Y-250)
			tester.Frame()

			itemRect, ok := tester.Find(tt.itemName)
			require.True(t, ok)
			require.True(t, itemRect.W >= nameWidth || itemRect.H >= 1.5*nameHeight,
				"the full %s name must wrap instead of being clipped: rect=%+v measured=(%g,%g)",
				tt.name, itemRect, nameWidth, nameHeight)
			require.True(t, tester.HasText(tt.fullID),
				"the %s record must show the full id before selection", tt.name)

			require.NoError(t, tester.Click(tt.itemName))
			tester.Frame()
			tt.check(t, app)
			_, ok = tester.Find(fmt.Sprintf(tr("Selected %s"), tt.itemName))
			require.True(t, ok, "the chosen record must announce its selected state")
		})
	}
}
