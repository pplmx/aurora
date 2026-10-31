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
//
// Both Chinese catalogs run: they are separate tables, and a line can be short
// in one and long in the other.
func TestChineseAtDoubleTextKeepsContentInsideTheWindow(t *testing.T) {
	translator := i18n.GetTranslator()
	before := translator.GetLocale()
	t.Cleanup(func() { translator.SetLocale(before) })

	const windowWidth, windowHeight = 1280, 900
	pages := []string{"Dashboard", "Lottery", "Voting", "Token", "NFT", "Oracle", "Blockchain", "Settings"}

	for _, locale := range []string{"zh", "zh-Hant"} {
		t.Run(locale, func(t *testing.T) {
			translator.SetLocale(locale)
			for _, page := range pages {
				app := New(&fakeServices{})
				tester := ui.NewTester(app.View, windowWidth, windowHeight)
				app.ShowPage(page)
				tester.SetPreferences(ui.Preferences{TextScale: 2})
				tester.Frame()

				texts := tester.Texts()
				require.NotEmptyf(t, texts, "%s must draw something", page)
				// Both edges, reported once each: a per-string assertion through
				// Find() measures the first match for a repeated label, which is
				// not the one that is widest. A left edge below zero means the
				// content was pushed off the other side.
				widest, worst := float32(0), ""
				leftmost, leftText := float32(windowWidth), ""
				for _, text := range texts {
					rect, ok := tester.Find(text)
					if !ok {
						continue
					}
					if right := rect.X + rect.W; right > widest {
						widest, worst = right, text
					}
					if rect.X < leftmost {
						leftmost, leftText = rect.X, text
					}
				}
				require.GreaterOrEqualf(t, leftmost, float32(-1),
					"%s %s: %q starts left of the window at 2x", locale, page, leftText)
				require.LessOrEqualf(t, widest, float32(windowWidth)+1,
					"%s %s: %q runs past the window at 2x", locale, page, worst)
			}
		})
	}
}
