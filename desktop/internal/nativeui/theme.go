package nativeui

import "github.com/egoist/mygo/ui"

// The window's palette.
//
// Aurora is a ledger suite, so the interface reads as an instrument rather
// than a document: a deep polar-night background, one raised surface for
// anything the operator can act on, and a single accent that marks the
// current page and the one primary action per section. Colour is never the
// only carrier of meaning — every status line states its outcome in words,
// and every table column is labelled — because the theme is also what a
// reader with a colour-vision difference sees.
const (
	// Night is the window background: near-black with a blue cast, dark
	// enough that the surfaces above it separate without a border.
	night = "#070d15"
	// SidebarSurface is the navigation column: a tone between the window
	// background and a card, so the module list reads as chrome rather than
	// as another panel of content.
	sidebarSurface = "#0a121c"
	// Surface is a card: one step up from the background.
	surface = "#0f1a27"
	// SurfaceHover and SurfacePressed are that step while the pointer is on
	// a control, and while it is held.
	surfaceHover   = "#172635"
	surfacePressed = "#1e3144"
	// Border separates a surface from the background and from its neighbours.
	border = "#203245"
	// Ink is primary text; InkMuted is secondary text, placeholders and hints.
	ink      = "#e9f1f8"
	inkMuted = "#93a9bf"
	// Accent is Aurora's own light: the current page, primary buttons and the
	// focus ring. AuroraHover and AuroraPressed are its two states.
	aurora        = "#38bdf8"
	auroraHover   = "#67d3fb"
	auroraPressed = "#0ea5e9"
	// OnAccent is text drawn on the accent.
	onAccent = "#041c29"
	// AuroraGreen marks what succeeded, AuroraRed what failed and AuroraAmber
	// what needs attention. These are the same three hues every status line
	// and ledger verdict uses, so the colour never disagrees with the words.
	auroraGreen = "#4ade80"
	auroraRed   = "#f87171"
	auroraAmber = "#fbbf24"
	// Selection tints selected text; FocusRing is the keyboard focus ring.
	selection = "#16344c"
	focusRing = "#7dd3fc"
	// Scrollbar is the thumb of a scroll bar.
	scrollbar = "#31465c"
	// successTint and errorTint are the backgrounds of a status line: the
	// verdict's hue, dark enough that the text on it keeps its contrast.
	successTint = "#0f2a20"
	errorTint   = "#2a1318"
	warningTint = "#2a2111"
	// Module accents give each subsystem a recognisable mark. They paint the
	// page glyph and the dashboard's stat icons, never a status: the three
	// verdict hues above stay reserved for success, failure and attention.
	lotteryAccent    = "#fbbf24"
	votingAccent     = "#a78bfa"
	tokenAccent      = "#34d399"
	oracleAccent     = "#38bdf8"
	nftAccent        = "#f472b6"
	blockchainAccent = "#818cf8"
	settingsAccent   = "#94a3b8"
	// High-contrast variants: the same hues, with the borders, primary and
	// secondary text and focus ring pushed apart from the surfaces. The OS
	// asks for this through Preferences.HighContrast.
	hcBorder    = "#7c97ad"
	hcInk       = "#ffffff"
	hcInkMuted  = "#c9d8e5"
	hcFocusRing = "#ffffff"
	hcSelection = "#2c4f6d"
)

// auroraTheme returns the theme the window draws with: the dark theme with
// Aurora's palette in place of the toolkit's neutral one, and the metrics
// tuned for a form-heavy ledger.
//
// The window sets this on every frame instead of accepting the system's
// appearance, because the product is dark by design: the previous build set a
// dark window background and then drew MyGo's light theme on top of it, so
// every button, input and table read as a white panel on a black page.
func auroraTheme(highContrast bool) *ui.Theme {
	borderColor, textColor, mutedColor := border, ink, inkMuted
	focusColor, selectionColor := focusRing, selection
	if highContrast {
		borderColor, textColor, mutedColor = hcBorder, hcInk, hcInkMuted
		focusColor, selectionColor = hcFocusRing, hcSelection
	}
	t := ui.DarkTheme()
	t.Background = ui.Hex(night)
	t.Surface = ui.Hex(surface)
	t.SurfaceHover = ui.Hex(surfaceHover)
	t.SurfacePressed = ui.Hex(surfacePressed)
	t.Border = ui.Hex(borderColor)
	t.Text = ui.Hex(textColor)
	t.TextMuted = ui.Hex(mutedColor)
	t.Accent = ui.Hex(aurora)
	t.AccentHover = ui.Hex(auroraHover)
	t.AccentPressed = ui.Hex(auroraPressed)
	t.AccentText = ui.Hex(onAccent)
	t.Danger = ui.Hex(auroraRed)
	t.Warning = ui.Hex(auroraAmber)
	t.Success = ui.Hex(auroraGreen)
	t.Selection = ui.Hex(selectionColor)
	t.Focus = ui.Hex(focusColor)
	t.Scrollbar = ui.Hex(scrollbar)
	t.Radius = 10
	t.Spacing = 4
	return t
}

// pageAccent is the hue that marks a module: its sidebar glyph, its page tile
// and the icons of the counters that open it. It is deliberately not a verdict
// colour, so a module's identity can never read as a status.
func pageAccent(p page) ui.Color {
	switch p {
	case pageLottery:
		return ui.Hex(lotteryAccent)
	case pageVoting:
		return ui.Hex(votingAccent)
	case pageToken:
		return ui.Hex(tokenAccent)
	case pageOracle:
		return ui.Hex(oracleAccent)
	case pageNFT:
		return ui.Hex(nftAccent)
	case pageBlockchain:
		return ui.Hex(blockchainAccent)
	case pageSettings:
		return ui.Hex(settingsAccent)
	default:
		return ui.Hex(aurora)
	}
}

// auroraThemeFor applies the desktop preferences a custom theme does not
// inherit from MyGo's default theme. App text sets its explicit sizes through
// textSize; this scaled default covers controls drawn by the toolkit, such as
// alerts, inputs and tabs.
func auroraThemeFor(prefs ui.Preferences) *ui.Theme {
	t := auroraTheme(prefs.HighContrast)
	if prefs.TextScale > 0 {
		t.FontSize *= prefs.TextScale
	}
	return t
}
