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
	night = "#0b1620"
	// Surface is a card: one step up from the background.
	surface = "#131f2b"
	// SurfaceHover and SurfacePressed are that step while the pointer is on
	// a control, and while it is held.
	surfaceHover   = "#1b2937"
	surfacePressed = "#243343"
	// Border separates a surface from the background and from its neighbours.
	border = "#243544"
	// Ink is primary text; InkMuted is secondary text, placeholders and hints.
	ink      = "#e6edf3"
	inkMuted = "#8ba3b8"
	// Accent is Aurora's own light: the current page, primary buttons and the
	// focus ring. AuroraHover and AuroraPressed are its two states.
	aurora        = "#38bdf8"
	auroraHover   = "#5ecdfb"
	auroraPressed = "#0ea5e9"
	// OnAccent is text drawn on the accent.
	onAccent = "#04222f"
	// AuroraGreen marks what succeeded, AuroraRed what failed and AuroraAmber
	// what needs attention. These are the same three hues every status line
	// and ledger verdict uses, so the colour never disagrees with the words.
	auroraGreen = "#4ade80"
	auroraRed   = "#f87171"
	auroraAmber = "#fbbf24"
	// Selection tints selected text; FocusRing is the keyboard focus ring.
	selection = "#1e3a52"
	focusRing = "#7dd3fc"
	// Scrollbar is the thumb of a scroll bar.
	scrollbar = "#3b5064"
	// successTint and errorTint are the backgrounds of a status line: the
	// verdict's hue, dark enough that the text on it keeps its contrast.
	successTint = "#12291f"
	errorTint   = "#2a1417"
)

// auroraTheme returns the theme the window draws with: the dark theme with
// Aurora's palette in place of the toolkit's neutral one, and the metrics
// tuned for a form-heavy ledger.
//
// The window sets this on every frame instead of accepting the system's
// appearance, because the product is dark by design: the previous build set a
// dark window background and then drew MyGo's light theme on top of it, so
// every button, input and table read as a white panel on a black page.
func auroraTheme() *ui.Theme {
	t := ui.DarkTheme()
	t.Background = ui.Hex(night)
	t.Surface = ui.Hex(surface)
	t.SurfaceHover = ui.Hex(surfaceHover)
	t.SurfacePressed = ui.Hex(surfacePressed)
	t.Border = ui.Hex(border)
	t.Text = ui.Hex(ink)
	t.TextMuted = ui.Hex(inkMuted)
	t.Accent = ui.Hex(aurora)
	t.AccentHover = ui.Hex(auroraHover)
	t.AccentPressed = ui.Hex(auroraPressed)
	t.AccentText = ui.Hex(onAccent)
	t.Danger = ui.Hex(auroraRed)
	t.Warning = ui.Hex(auroraAmber)
	t.Success = ui.Hex(auroraGreen)
	t.Selection = ui.Hex(selection)
	t.Focus = ui.Hex(focusRing)
	t.Scrollbar = ui.Hex(scrollbar)
	t.Radius = 8
	t.Spacing = 4
	return t
}
