package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/stretchr/testify/require"
)

// TestChineseAtDoubleTextKeepsContentInsideTheWindow covers the widest
// combination the window can be asked for: Chinese doubles the width of every
// character that is not Latin, and the system text scale doubles the size
// again. Nothing in the suite combined the two, so a label that only fits in
// English would have been found by a user.
func TestChineseAtDoubleTextKeepsContentInsideTheWindow(t *testing.T) {
	translator := i18n.GetTranslator()
	before := translator.GetLocale()
	t.Cleanup(func() { translator.SetLocale(before) })
	translator.SetLocale("zh")

	const windowWidth, windowHeight = 1280, 900
	for _, page := range []string{"Dashboard", "Lottery", "Voting", "Token", "NFT", "Oracle", "Blockchain", "Settings"} {
		app := New(&fakeServices{})
		tester := ui.NewTester(app.View, windowWidth, windowHeight)
		app.ShowPage(page)
		tester.SetPreferences(ui.Preferences{TextScale: 2})
		tester.Frame()

		texts := tester.Texts()
		require.NotEmptyf(t, texts, "%s must draw something", page)
		// The widest right edge across every drawn string, reported once: a
		// per-string assertion through Find() measures the first match for a
		// repeated label, which is not the one that is widest.
		widest, worst := float32(0), ""
		for _, text := range texts {
			if rect, ok := tester.Find(text); ok && rect.X+rect.W > widest {
				widest, worst = rect.X+rect.W, text
			}
		}
		require.LessOrEqualf(t, widest, float32(windowWidth)+1,
			"%s: %q runs past the window at 2x in Chinese", page, worst)
	}
}
