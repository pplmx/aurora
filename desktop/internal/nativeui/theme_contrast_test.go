package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// TestPaletteTextMeetsTheContrastFloor guards both appearances, and their
// high-contrast variants, against a palette edit that makes text hard to read.
// The floor is WCAG AA for normal text: body and muted text, the module accent
// (tab labels and small headings use it), the status inks and the ink on a
// filled accent button all stay at 4.5:1 or better.
func TestPaletteTextMeetsTheContrastFloor(t *testing.T) {
	const floor = 4.5
	pages := []page{
		pageDashboard, pageLottery, pageVoting, pageToken,
		pageOracle, pageNFT, pageBlockchain, pageSettings,
	}
	modes := []struct {
		name   string
		mode   ThemeMode
		system bool
	}{
		{"dark", ThemeModeDark, true},
		{"light", ThemeModeLight, false},
	}
	for _, mode := range modes {
		for _, highContrast := range []bool{false, true} {
			prefs := ui.Preferences{HighContrast: highContrast}
			for _, p := range pages {
				theme := auroraThemeForState(prefs, mode.mode, mode.system, p)
				label := mode.name + "/hc=" + boolLabel(highContrast) + "/" + pageTitles[p]
				for _, surface := range []struct {
					name string
					bg   ui.Color
				}{
					{"background", theme.Background},
					{"surface", theme.Surface},
				} {
					require.GreaterOrEqualf(t, contrastRatio(theme.Text, surface.bg), floor,
						"text on %s (%s)", surface.name, label)
					require.GreaterOrEqualf(t, contrastRatio(theme.TextMuted, surface.bg), floor,
						"muted text on %s (%s)", surface.name, label)
					require.GreaterOrEqualf(t, contrastRatio(theme.Accent, surface.bg), floor,
						"module accent on %s (%s)", surface.name, label)
				}
				require.GreaterOrEqualf(t, contrastRatio(theme.AccentText, theme.Accent), floor,
					"text on the accent fill (%s)", label)
				require.GreaterOrEqualf(t, contrastRatio(theme.AccentText, theme.AccentHover), floor,
					"text on the hovered accent fill (%s)", label)
				require.GreaterOrEqualf(t, contrastRatio(theme.Success, theme.Surface), floor,
					"success text on a card (%s)", label)
				require.GreaterOrEqualf(t, contrastRatio(theme.Danger, theme.Surface), floor,
					"danger text on a card (%s)", label)
				require.GreaterOrEqualf(t, contrastRatio(theme.Warning, theme.Surface), floor,
					"warning text on a card (%s)", label)
			}
		}
	}
}

func boolLabel(v bool) string {
	if v {
		return "on"
	}
	return "off"
}
