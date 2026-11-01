package nativeui

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/i18n"
)

// TestReadoutKeepsLabelsOnOneLine covers the fixed 72px label column: labels
// such as "Remembered keys" and "Command palette" wrapped onto two lines while
// the value stayed beside only the first one.
//
// The column is measured once for the whole panel, so the short label is given
// the same room as the long one. What matters is that the long label has as
// much room as its own text needs; equal widths are the point, not a defect.
func TestReadoutKeepsLabelsOnOneLine(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		var longWidth float32
		tester := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Width(360).Children(func() {
				readout(c, []readoutRow{
					{"Short", "value"},
					{"Remembered keys", "value"},
				})
			})
			longWidth, _ = c.MeasureText(0, ui.Span{
				Text: "Remembered keys", Size: textSize(c, labelSize),
			})
		}, 400, 160)
		tester.SetPreferences(ui.Preferences{TextScale: scale})
		tester.Frame()

		short, ok := tester.Find("Short")
		require.True(t, ok)
		long, ok := tester.Find("Remembered keys")
		require.True(t, ok)
		require.InDelta(t, short.H, long.H, 0.5, "readout labels must stay on one line at %.1fx", scale)
		require.InDelta(t, short.W, long.W, 0.5,
			"the panel's labels must share one column at %.1fx", scale)
		require.GreaterOrEqual(t, long.W, longWidth,
			"the shared column must fit its longest label at %.1fx", scale)
	}
}

// TestReadoutValuesShareOneColumn is the reason the column is a single
// measurement: each row used to size its own label, so the values began at a
// different x per row and a stack of facts read as a ragged list.
//
// Chinese is the case that exposed it. The old estimate multiplied the byte
// length by 7.5, which overstates 数据库路径 (15 bytes, 55px) by roughly three
// times, so the Chinese panels reserved a wide empty column and pushed their
// values right.
func TestReadoutValuesShareOneColumn(t *testing.T) {
	translator := i18n.GetTranslator()
	before := translator.GetLocale()
	t.Cleanup(func() { translator.SetLocale(before) })

	for _, locale := range []string{"en", "zh", "zh-Hant"} {
		t.Run(locale, func(t *testing.T) {
			translator.SetLocale(locale)
			var widest float32
			labels := []readoutRow{
				{"Version", "VALUE-ONE"},
				{"Build time", "VALUE-TWO"},
				{"Database path", "VALUE-THREE"},
			}
			tester := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Width(420).Children(func() {
					readout(c, labels)
				})
				for _, row := range labels {
					if w, _ := c.MeasureText(0, ui.Span{
						Text: tr(row.label), Size: textSize(c, labelSize),
					}); w > widest {
						widest = w
					}
				}
			}, 460, 220)
			tester.Frame()

			var first float32
			for i, value := range []string{"VALUE-ONE", "VALUE-TWO", "VALUE-THREE"} {
				rect, ok := tester.Find(value)
				require.Truef(t, ok, "%s: %q must be drawn", locale, value)
				if i == 0 {
					first = rect.X
					continue
				}
				require.InDeltaf(t, first, rect.X, 0.5,
					"%s: every value starts in the same column, %q began at %g instead of %g",
					locale, value, rect.X, first)
			}
			require.LessOrEqualf(t, first, widest+space3+40,
				"%s: the label column must fit the widest label (%gpx) instead of reserving the byte-length estimate",
				locale, widest)
		})
	}
}

// TestReadoutRevealsClippedValueOnHover covers the detail panel outside the
// tables: owner keys, descriptions and oracle payloads are routinely wider than
// the card. A single line is correct for the row, but the stored value must
// remain retrievable rather than ending at the card edge without explanation.
func TestReadoutRevealsClippedValueOnHover(t *testing.T) {
	const value = "A stored detail value long enough to be clipped by the card at the minimum window width"
	tester := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Width(360).Children(func() {
			readout(c, []readoutRow{{"Owner", value}})
		})
	}, 400, 160)
	tester.SetPreferences(ui.Preferences{TextScale: 1.5})
	tester.Frame()

	require.Equal(t, 1, countText(tester.Texts(), value), "the readout starts with one copy")
	rect, ok := tester.Find(value)
	require.True(t, ok)
	tester.Move(rect.X+rect.W/2, rect.Y+rect.H/2)
	tester.Advance(700 * time.Millisecond)
	tester.Frame()

	require.Equal(t, 2, countText(tester.Texts(), value),
		"hovering a clipped detail must show its full stored value")
}
