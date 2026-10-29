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

// TestLongLedgerHeightStaysReadable covers the numeric values outside the
// token and NFT collections. A ledger height is an int64; at 1.5x text a fixed
// 90px column cut it to a prefix, and plain text offered no way to see the
// recorded block. The record's height chip now draws the block number at its
// natural width and carries the bare number as its tooltip.
func TestLongLedgerHeightStaysReadable(t *testing.T) {
	const height = "9223372036854775807"
	app := New(&fakeServices{blocks: []BlockRow{{
		Height: 9223372036854775807, Module: "oracle", Action: "Observation",
		Subject: "68421.50", Payload: `{"source":"btc","price":12345}`,
	}}})
	tester := ui.NewTester(app.View, 900, 2000)
	tester.SetPreferences(ui.Preferences{TextScale: 1.5})
	app.ShowPage("Blockchain")
	tester.Frame()

	chip, ok := tester.Find("#" + height)
	require.True(t, ok, "the record must show the block number at 1.5x text")
	require.GreaterOrEqual(t, chip.W, float32(170),
		"the height chip must grow with text size instead of cutting the value")
	tester.Move(chip.X+chip.W/2, chip.Y+chip.H/2)
	tester.Advance(700 * time.Millisecond)
	tester.Frame()

	// The number appears in the drawn chip, in its tooltip and in the record's
	// accessible label; the point is that the drawn form is not the only one.
	require.GreaterOrEqual(t, countText(tester.Texts(), height), 2,
		"the chip must expose the exact height, both drawn and on hover")
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
