package nativeui

import "github.com/egoist/mygo/ui"

// ThemeMode is the appearance choice saved by the desktop. An empty or
// unknown value from an older preferences file resolves to Dark, which was
// the only appearance Aurora shipped before this choice existed.
type ThemeMode string

const (
	ThemeModeSystem ThemeMode = "system"
	ThemeModeLight  ThemeMode = "light"
	ThemeModeDark   ThemeMode = "dark"
)

// normalizeThemeMode keeps preferences and callers on the three supported
// values. Legacy files have no theme field, so their zero value stays Dark.
func normalizeThemeMode(mode ThemeMode) ThemeMode {
	switch mode {
	case ThemeModeSystem, ThemeModeLight:
		return mode
	default:
		return ThemeModeDark
	}
}

// themeModeDark resolves the appearance actually drawn this frame. System
// mode receives the desktop's effective appearance from the shell.
func themeModeDark(mode ThemeMode, systemDark bool) bool {
	switch normalizeThemeMode(mode) {
	case ThemeModeLight:
		return false
	case ThemeModeSystem:
		return systemDark
	default:
		return true
	}
}

// palette is one complete semantic surface scale. Renderers resolve every
// colour through ui.Theme; this struct is the single source behind those
// fields, and exists twice so light and dark are deliberately different
// designs rather than one design with inverted pixels.
type palette struct {
	background     string
	sidebar        string
	surface        string
	surfaceHover   string
	surfacePressed string
	border         string
	ink            string
	inkMuted       string
	success        string
	danger         string
	warning        string
	scrollbar      string

	hcBorder    string
	hcInk       string
	hcInkMuted  string
	hcFocusRing string
}

var lightPalette = palette{
	background:     "#F4F0E6",
	sidebar:        "#ECE6D9",
	surface:        "#FFFCF5",
	surfaceHover:   "#F0E9DC",
	surfacePressed: "#E4DACB",
	border:         "#D2C6B4",
	ink:            "#152622",
	inkMuted:       "#52645D",
	success:        "#106B45",
	danger:         "#B42318",
	warning:        "#9A5B00",
	scrollbar:      "#A99C88",
	hcBorder:       "#5B5043",
	hcInk:          "#07110E",
	hcInkMuted:     "#34453E",
	hcFocusRing:    "#0A5148",
}

var darkPalette = palette{
	background:     "#0A1211",
	sidebar:        "#0D1816",
	surface:        "#14211E",
	surfaceHover:   "#1C2C28",
	surfacePressed: "#263A34",
	border:         "#30433D",
	ink:            "#ECF3EC",
	inkMuted:       "#9BAEA5",
	success:        "#5EE0A0",
	danger:         "#FF7B83",
	warning:        "#F4C15D",
	scrollbar:      "#526A61",
	hcBorder:       "#91A89E",
	hcInk:          "#FFFFFF",
	hcInkMuted:     "#D7E1DB",
	hcFocusRing:    "#FFFFFF",
}

// modulePalette gives each module an ink that is legible on that
// appearance's card surface. The hues keep their identity across modes;
// their luminance changes so a light card does not need a neon mark.
var modulePalette = map[page]struct {
	light string
	dark  string
}{
	pageLottery:    {"#9A5C00", "#F2C14E"},
	pageVoting:     {"#814A8F", "#F0A6D2"},
	pageToken:      {"#08705A", "#55D6AE"},
	pageOracle:     {"#0B6690", "#69BDF0"},
	pageNFT:        {"#9B3E74", "#F08AC0"},
	pageBlockchain: {"#5146A8", "#B0AAFF"},
	pageSettings:   {"#55655F", "#AAB9B2"},
	pageDashboard:  {"#0B6D78", "#54D6CE"},
}

// pageAccent returns the module hue for the appearance currently in effect.
// It is intentionally not one of the verdict colours, so a module's identity
// can never read as success or failure.
func pageAccent(c *ui.Context, p page) ui.Color {
	return pageAccentForPalette(c.Theme().Dark, p)
}

// brandAccent is fixed product identity. It is deliberately independent of
// the open module: the wordmark must remain Aurora even while every section
// and action takes the current page's hue.
func brandAccent(c *ui.Context) ui.Color {
	if c.Theme().Dark {
		return ui.Hex("#54D6CE")
	}
	return ui.Hex("#0B6D78")
}

func brandWarm(c *ui.Context) ui.Color {
	if c.Theme().Dark {
		return ui.Hex("#E8B85A")
	}
	return ui.Hex("#8B5200")
}

// themeAccentText returns the text on a filled accent control. Every light
// module hue is dark enough for the light surface, while every dark module
// hue is bright enough for near-black.
func themeAccentText(dark bool) ui.Color {
	if dark {
		return ui.Hex("#07110E")
	}
	return ui.Hex("#FFFCF5")
}

// auroraThemeForState builds the complete palette for one frame. The accent
// is the open module's colour, so section marks, tabs, selected records and
// primary actions belong to the page rather than to a global blue layer.
func auroraThemeForState(prefs ui.Preferences, mode ThemeMode, systemDark bool, p page) *ui.Theme {
	dark := themeModeDark(mode, systemDark)
	pal := darkPalette
	if !dark {
		pal = lightPalette
	}

	borderColor, textColor, mutedColor := pal.border, pal.ink, pal.inkMuted
	focusColor := ui.Hex(pal.inkMuted)
	if prefs.HighContrast {
		borderColor, textColor, mutedColor = pal.hcBorder, pal.hcInk, pal.hcInkMuted
		focusColor = ui.Hex(pal.hcFocusRing)
	}

	base := ui.LightTheme()
	if dark {
		base = ui.DarkTheme()
	}
	accent := pageAccentForPalette(dark, p)
	hover := accent.Mix(ui.RGB(255, 255, 255), 0.14)
	pressed := accent.Mix(ui.RGB(0, 0, 0), 0.16)
	if !dark {
		hover = accent.Mix(ui.RGB(0, 0, 0), 0.12)
		pressed = accent.Mix(ui.RGB(0, 0, 0), 0.24)
		focusColor = accent
	}
	if prefs.HighContrast {
		focusColor = ui.Hex(pal.hcFocusRing)
	}

	t := base
	t.Dark = dark
	t.Background = ui.Hex(pal.background)
	t.Surface = ui.Hex(pal.surface)
	t.SurfaceHover = ui.Hex(pal.surfaceHover)
	t.SurfacePressed = ui.Hex(pal.surfacePressed)
	t.Border = ui.Hex(borderColor)
	t.Text = ui.Hex(textColor)
	t.TextMuted = ui.Hex(mutedColor)
	t.Accent = accent
	t.AccentHover = hover
	t.AccentPressed = pressed
	t.AccentText = themeAccentText(dark)
	t.Danger = ui.Hex(pal.danger)
	t.Warning = ui.Hex(pal.warning)
	t.Success = ui.Hex(pal.success)
	// Selection is a quiet wash of the page accent. Mixing toward the window
	// rather than the bright accent keeps body text readable on a selected row.
	strength := float32(0.20)
	if !dark {
		strength = 0.16
	}
	t.Selection = t.Background.Mix(accent, strength)
	t.Focus = focusColor
	t.Scrollbar = ui.Hex(pal.scrollbar)
	t.Radius = 10
	t.Spacing = 4
	if prefs.TextScale > 0 {
		t.FontSize *= prefs.TextScale
	}
	return t
}

func pageAccentForPalette(dark bool, p page) ui.Color {
	colors, ok := modulePalette[p]
	if !ok {
		colors = modulePalette[pageDashboard]
	}
	if dark {
		return ui.Hex(colors.dark)
	}
	return ui.Hex(colors.light)
}

// auroraTheme and auroraThemeFor are the dark defaults used by isolated
// component tests and by callers that do not have a mode yet. The window's
// real frame goes through auroraThemeForState.
func auroraTheme(highContrast bool) *ui.Theme {
	return auroraThemeForState(
		ui.Preferences{HighContrast: highContrast},
		ThemeModeDark, true, pageDashboard,
	)
}

func auroraThemeFor(prefs ui.Preferences) *ui.Theme {
	return auroraThemeForState(prefs, ThemeModeDark, true, pageDashboard)
}

// tintedSurface is the shared derivation for a status, warning or record
// wash. It stays opaque so contrast tests and screenshots see the colour the
// renderer actually paints, not a blend whose result depends on what is
// underneath.
func tintedSurface(c *ui.Context, tone ui.Color, strength float32) ui.Color {
	return c.Theme().Background.Mix(tone, strength)
}

func successTint(c *ui.Context) ui.Color {
	return tintedSurface(c, c.Theme().Success, statusTintStrength(c))
}

func errorTint(c *ui.Context) ui.Color {
	return tintedSurface(c, c.Theme().Danger, statusTintStrength(c))
}

func warningTint(c *ui.Context) ui.Color {
	return tintedSurface(c, c.Theme().Warning, statusTintStrength(c))
}

// statusTintStrength keeps status text above WCAG AA on its own wash. The
// light palette has less headroom than the dark one, so its tint stays faint;
// this is the one place that adjusts it for all three verdict tones.
func statusTintStrength(c *ui.Context) float32 {
	return statusTintStrengthFor(c.Theme())
}

func statusTintStrengthFor(theme *ui.Theme) float32 {
	if theme.Dark {
		return 0.14
	}
	return 0.02
}

func sidebarTone(c *ui.Context) ui.Color {
	pal := darkPalette
	if !c.Theme().Dark {
		pal = lightPalette
	}
	return ui.Hex(pal.sidebar)
}

func cardShadow(c *ui.Context) ui.Color {
	if c.Theme().Dark {
		return ui.RGBA(0, 0, 0, 0.46)
	}
	return ui.RGBA(74, 61, 42, 0.14)
}

func cardHighlight(c *ui.Context) ui.Color {
	if c.Theme().Dark {
		return ui.RGBA(255, 255, 255, 0.055)
	}
	return ui.RGBA(255, 255, 255, 0.8)
}

func disabledFill(c *ui.Context) ui.Color {
	return c.Theme().Background.Mix(c.Theme().Surface, 0.55)
}
