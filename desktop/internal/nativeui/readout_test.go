package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestReadoutKeepsLabelsOnOneLine covers the fixed 72px label column: labels
// such as "Remembered keys" and "Command palette" wrapped onto two lines while
// the value stayed beside only the first one.
func TestReadoutKeepsLabelsOnOneLine(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		tester := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Width(360).Children(func() {
				readout(c, []readoutRow{
					{"Short", "value"},
					{"Remembered keys", "value"},
				})
			})
		}, 400, 160)
		tester.SetPreferences(ui.Preferences{TextScale: scale})
		tester.Frame()

		short, ok := tester.Find("Short")
		require.True(t, ok)
		long, ok := tester.Find("Remembered keys")
		require.True(t, ok)
		require.InDelta(t, short.H, long.H, 0.5, "readout labels must stay on one line at %.1fx", scale)
		require.Greater(t, long.W, short.W, "the label column must grow at %.1fx", scale)
		require.GreaterOrEqual(t, long.W, 108*scale, "the long label must have room at %.1fx", scale)
	}
}
