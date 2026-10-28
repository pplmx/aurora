package nativeui

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestLongTableValueKeepsItsFullTooltip pins the disclosure added for columns
// that cannot show every stored value at once. The visible cell may be cut by
// a narrow column or enlarged text, but resting on it must still reveal the
// exact value instead of leaving an ambiguous prefix.
func TestLongTableValueKeepsItsFullTooltip(t *testing.T) {
	const supply = "9223372036854775807"
	const name = "Aurora Governance Reserve for Long-Term Protocol Maintenance"
	app := New(&fakeServices{tokenRows: []TokenRow{{
		ID: "AUR", Name: name, Symbol: "AUR", TotalSupply: supply,
	}}})
	tester := ui.NewTester(app.View, 900, 2400)
	tester.SetPreferences(ui.Preferences{TextScale: 1.5})
	app.ShowPage("Token")
	tester.Frame()

	// The numeric column grows with text size so the largest valid int64
	// amount is not silently cut at 1.5x.
	supplyRect, ok := tester.Find(supply)
	require.True(t, ok)
	require.GreaterOrEqual(t, supplyRect.W, float32(170))

	require.Equal(t, 1, countText(tester.Texts(), name), "the cell starts with one copy")
	rect, ok := tester.Find(name)
	require.True(t, ok)
	tester.Move(rect.X+rect.W/2, rect.Y+rect.H/2)
	tester.Advance(700 * time.Millisecond)
	tester.Frame()

	require.Equal(t, 2, countText(tester.Texts(), name),
		"hovering the clipped cell must show its full value in a tooltip")
}

func countText(texts []string, want string) int {
	n := 0
	for _, text := range texts {
		if strings.Contains(text, want) {
			n++
		}
	}
	return n
}
