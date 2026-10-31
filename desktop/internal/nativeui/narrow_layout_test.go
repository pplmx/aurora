package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestEveryPageStackedKeepsContentInsideTheWindow covers the layout the window
// switches to below twoColumnMinWidth, where the two columns become one. Only
// the token page had ever been rendered at that width; the other seven shared
// the branch by construction but nothing checked it, and a card that assumed a
// side-by-side row would run past the edge on every narrow window.
func TestEveryPageStackedKeepsContentInsideTheWindow(t *testing.T) {
	const width, height = 900, 700
	require.False(t, width >= twoColumnMinWidth,
		"the probe width must be below the stacking threshold, or it tests nothing")

	for _, page := range []string{
		"dashboard", "lottery", "voting", "token", "oracle", "nft", "blockchain", "settings",
	} {
		app := New(&fakeServices{})
		app.ShowPage(page)
		tester := ui.NewTester(app.View, width, height)
		tester.Frame()

		texts := tester.Texts()
		require.NotEmptyf(t, texts, "%s must draw something when stacked", page)
		widest, worst := float32(0), ""
		for _, text := range texts {
			if rect, ok := tester.Find(text); ok && rect.X+rect.W > widest {
				widest, worst = rect.X+rect.W, text
			}
		}
		require.LessOrEqualf(t, widest, float32(width)+1,
			"%s: %q runs past the window when stacked", page, worst)
	}
}
