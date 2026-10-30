package nativeui

import (
	"image"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestTabsMarkTheCurrentSegmentOnce pins the tab row's look: an active
// segment is an accent label over an accent underline, with no filled pill and
// no second box around the row. It used to carry a wash, a border and an
// underline at once, which said "current" three times in three shapes.
func TestTabsMarkTheCurrentSegmentOnce(t *testing.T) {
	selected := 0
	var listWidth float32
	tester := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(auroraThemeFor(c.Preferences()))
		ui.Column(c).Padding(20).Children(func() {
			listWidth = 420
			tabs(c, &selected, "Token actions", "Mint", "Transfer", "Approve")
		})
	}, 480, 120)
	tester.Frame()

	active, ok := tester.Find("Mint")
	require.True(t, ok)
	require.LessOrEqual(t, active.W, listWidth,
		"the tab row must not draw a second box wider than its content")
	inactive, ok := tester.Find("Transfer")
	require.True(t, ok)
	require.InDelta(t, float64(active.Y), float64(inactive.Y), 1,
		"both tabs share the row")

	img := tester.Image()
	labelPixel := img.At(int(active.X)+6, int(active.Y)+int(active.H)/2)
	r, _, b, _ := labelPixel.RGBA()
	require.Greater(t, int(b>>8), int(r>>8),
		"the current segment's label must be drawn in the accent hue")

	// The underline sits at the bottom of the segment, not under the label.
	underline := img.At(int(active.X)+int(active.W)/2, int(active.Y+active.H)-2)
	ur, _, ub, _ := underline.RGBA()
	require.Greater(t, int(ub>>8), int(ur>>8),
		"the current segment must carry the accent underline")
}

// TestTabSelectionFollowsTheKeyboard pins that the restyled row still reports
// a new choice to the page, which is what swaps the write form below it.
func TestTabSelectionFollowsTheKeyboard(t *testing.T) {
	selected := 0
	tester := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(auroraThemeFor(c.Preferences()))
		ui.Column(c).Padding(20).Children(func() {
			tabs(c, &selected, "Token actions", "Mint", "Transfer", "Approve")
		})
	}, 480, 120)
	tester.Frame()

	tester.Key(0, ui.KeyTab)
	tester.Key(0, ui.KeyRight)
	tester.Frame()
	require.Equal(t, 1, selected, "Right must select the next segment")
	require.True(t, tester.Focused(tr("Transfer")))
}

// TestActionSubjectLeadsWithTheRecord pins the two states of the subject
// strip. A chosen record puts its id on the accent; no record turns the strip
// amber and names what to do first, so the two states differ before the hint
// is read.
func TestActionSubjectLeadsWithTheRecord(t *testing.T) {
	render := func(t *testing.T, id string) *ui.Tester {
		t.Helper()
		tester := ui.NewTester(func(c *ui.Context) {
			c.SetTheme(auroraThemeFor(c.Preferences()))
			ui.Column(c).Width(420).Padding(20).Children(func() {
				actionSubject(c, id, "No token selected", "Create or inspect one first.",
					"Copy token ID", "Token ID is on the clipboard")
			})
		}, 480, 160)
		tester.Frame()
		return tester
	}

	chosen := render(t, "token-42")
	require.True(t, chosen.HasText("Acting on"))
	id, ok := chosen.Find("token-42")
	require.True(t, ok)
	label, ok := chosen.Find("Acting on")
	require.True(t, ok)
	require.Less(t, label.X, id.X, "the quiet label comes before the id")

	chosenImg := chosen.Image()
	r, _, b, _ := chosenImg.At(int(id.X)+3, int(id.Y)+int(id.H)/2).RGBA()
	require.Greater(t, int(b>>8), int(r>>8),
		"a chosen subject's id must be drawn in the accent hue")

	empty := render(t, "")
	require.True(t, empty.HasText("No token selected"))
	require.False(t, empty.HasText("Acting on"))
	hint, ok := empty.Find("No token selected")
	require.True(t, ok)
	emptyImg := empty.Image()
	er, _, eb, _ := emptyImg.At(int(hint.X)-16, int(hint.Y)+int(hint.H)/2).RGBA()
	require.Greater(t, int(er>>8), int(eb>>8),
		"the empty state must carry the warning hue")
}

// TestDashboardCountersCarryTheirModuleHue ties each counter's number to the
// module it opens: the glyph beside it already had that hue, and leaving the
// value in body ink made the pair look unrelated.
func TestDashboardCountersCarryTheirModuleHue(t *testing.T) {
	app := New(&fakeServices{blocks: []BlockRow{{Height: 1, Module: "lottery", Summary: "draw"}}})
	tester := ui.NewTester(app.View, 1280, 1800)
	tester.Frame()

	img := tester.Image()
	// The glyph is anti-aliased against the card, so the hue is read from its
	// most saturated pixel rather than one fixed offset.
	hue := func(t *testing.T, text string) (uint32, uint32) {
		t.Helper()
		value, ok := tester.Find(text)
		require.Truef(t, ok, "the counter %q must be drawn", text)
		var bestR, bestB, bestDiff uint32
		for y := int(value.Y); y < int(value.Y+value.H); y++ {
			for x := int(value.X); x < int(value.X+value.W); x++ {
				r, _, b, _ := img.At(x, y).RGBA()
				r, b = r>>8, b>>8
				diff := uint32(absInt(int(r) - int(b)))
				if diff > bestDiff {
					bestDiff, bestR, bestB = diff, r, b
				}
			}
		}
		return bestR, bestB
	}

	// The lottery counter is amber (more red than blue) and the vote counter
	// is violet (more red than green); both used to be body ink, which is the
	// same near-white for every card.
	lotteryR, lotteryB := hue(t, "3")
	require.Greater(t, lotteryR, lotteryB,
		"the lottery counter must take its amber module hue")
	votes, ok := tester.Find("9")
	require.True(t, ok)
	votesR, votesG := hueRG(t, img, votes)
	require.Greater(t, votesR, votesG,
		"the vote counter must take its violet module hue")
}

// hueRG reads the most saturated red/green pair in a text rect, for a violet
// whose blue and red channels are close enough that blue alone is ambiguous.
func hueRG(t *testing.T, img *image.RGBA, value ui.Rect) (uint32, uint32) {
	t.Helper()
	var bestR, bestG, bestDiff uint32
	for y := int(value.Y); y < int(value.Y+value.H); y++ {
		for x := int(value.X); x < int(value.X+value.W); x++ {
			r, g, _, _ := img.At(x, y).RGBA()
			r, g = r>>8, g>>8
			diff := uint32(absInt(int(r) - int(g)))
			if diff > bestDiff {
				bestDiff, bestR, bestG = diff, r, g
			}
		}
	}
	return bestR, bestG
}
