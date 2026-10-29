package nativeui

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// The window's visual scale. These are layout values, not text: every size
// that carries text goes through textSize so the OS text preference still
// applies. Keeping the numbers here is what makes a card, a button and a page
// header agree on rhythm instead of each file inventing its own spacing.
const (
	space1 float32 = 4
	space2 float32 = 8
	space3 float32 = 12
	space4 float32 = 16
	space5 float32 = 20
	space6 float32 = 24
	space7 float32 = 32

	radiusSmall float32 = 8
	radiusPanel float32 = 14
	radiusLarge float32 = 18

	pageTitleSize  float32 = 28
	sectionSize    float32 = 16
	statValueSize  float32 = 30
	eyebrowSize    float32 = 10
	buttonHeight   float32 = 36
	iconTileSmall  float32 = 30
	iconTileMedium float32 = 38
	iconTileLarge  float32 = 48

	// pageGap is the space between the cards of one page body. Without it the
	// cards abut and their rounded corners merge into one slab: the page read
	// as a single panel with seams rather than as a list of jobs.
	pageGap float32 = 18
	// cardGap is the space between the cards of one column, slightly tighter
	// than pageGap so a column still reads as one group of related cards.
	cardGap float32 = 14
)

// localTimeLayout is the one spelling of a local timestamp in the window. A
// session's acceptance window and a recorded ledger entry both use it, so the
// time a module stored and the time the chain recorded read identically.
const localTimeLayout = "2006-01-02 15:04"

// panelFill is the card's background: a vertical wash from one step above the
// surface to the surface itself. It is mixed from the theme rather than fixed,
// so the high-contrast preference changes the card with the rest of the
// window, and a card still reads as one object instead of a flat rectangle.
func panelFill(c *ui.Context) ui.LinearGradient {
	return ui.LinearGradient{
		Angle: 165,
		From:  c.Theme().Surface.Mix(c.Theme().SurfaceHover, 0.45),
		To:    c.Theme().Surface,
	}
}

// workspaceFill is the page background: the window tone with the faintest
// aurora light gathering at the top, as if the content sat under a night sky
// rather than on a flat sheet. It stays within a few points of the background
// so every contrast ratio in theme.go still describes it.
func workspaceFill(c *ui.Context) ui.LinearGradient {
	return ui.LinearGradient{
		Angle: 180,
		From:  c.Theme().Background.Mix(ui.Hex(aurora), 0.045),
		To:    c.Theme().Background,
		Start: 0,
		End:   0.7,
	}
}

// sidebarFill gives the navigation column the same subtle depth: lighter at
// the masthead, settling to the window tone at the bottom.
func sidebarFill(c *ui.Context) ui.LinearGradient {
	return ui.LinearGradient{
		Angle: 180,
		From:  ui.Hex(sidebarSurface).Mix(c.Theme().Background, 0.05),
		To:    ui.Hex(sidebarSurface).Mix(c.Theme().Background, 0.45),
	}
}

// panelGradient is panelFill tinted with a page or verdict hue, for the one
// panel a page leads with. The tint stays under a fifth of the way to the hue:
// it is a wash of light, not a coloured card.
func panelGradient(c *ui.Context, tone ui.Color, strength float32) ui.LinearGradient {
	return ui.LinearGradient{
		Angle: 150,
		From:  c.Theme().Surface.Mix(tone, strength),
		To:    c.Theme().Surface.Mix(c.Theme().Background, 0.35),
	}
}

// eyebrow draws the small label above a value or a section title. It is the
// quietest type on the page: caps-height weight, muted colour, a little
// tracking, so a column of facts can be scanned by its labels alone.
func eyebrow(c *ui.Context, text string) ui.Element {
	return ui.Text(c, tr(text)).FontSize(textSize(c, eyebrowSize)).FontWeight(600).
		LetterSpacing(0.6).TextColor(c.Theme().TextMuted).SingleLine()
}

// iconTile is the recurring mark beside a page title, a counter and an action:
// a rounded square of the module's hue with its glyph inside. The fill is the
// hue mixed into the surface, so the tile belongs to the theme instead of
// sitting on top of it.
func iconTile(c *ui.Context, icon *ui.SVG, tone ui.Color, size float32) ui.Element {
	tile := ui.Row(c).Size(size, size).Center().Radius(size*0.3).
		Gradient(tone.Mix(c.Theme().Surface, 0.8), tone.Mix(c.Theme().Surface, 0.92), 150).
		BorderWidth(1).BorderColor(tone.Alpha(0.35))
	tile.Children(func() {
		ui.Icon(c, icon).Size(size*0.5, size*0.5).TextColor(tone)
	})
	return tile
}

// accentHairline draws a short gradient rule along an element's top edge. It
// gives the hero panel a lit edge without a second border or a coloured block.
func accentHairline(c *ui.Context, e ui.Element, tone ui.Color) ui.Element {
	e.DrawOver(func(p *ui.Painter, r ui.Rect) {
		from, to := tone.Alpha(0), tone.Alpha(0.55)
		p.FillGradient(ui.Rect{X: r.X + radiusLarge, Y: r.Y, W: r.W - 2*radiusLarge, H: 1},
			ui.LinearGradient{Angle: 90, From: from, Stops: []ui.GradientStop{
				{Color: from, At: 0},
				{Color: to, At: 0.5},
				{Color: from, At: 1},
			}}, 0)
	})
	return e
}

// statusDot is the small round mark beside a state word. It is always drawn
// next to text that names the state, so the colour is reinforcement, not the
// only signal.
func statusDot(c *ui.Context, tone ui.Color, size float32) ui.Element {
	return ui.Row(c).Size(size, size).Radius(size / 2).Background(tone).Shrink(0)
}

// statChip is one compact fact about the current collection, drawn as a
// recessed tile beside the controls that act on it. It replaces the two-row
// readout on the ledger page, where a full-width box held two short numbers.
func statChip(c *ui.Context, label, value string) {
	ui.Column(c).Padding(space2+2, space3+2).Gap(1).Radius(radiusSmall + 2).
		Background(c.Theme().Background.Mix(c.Theme().Surface, 0.35)).
		BorderWidth(1).BorderColor(c.Theme().Border).Children(func() {
		eyebrow(c, label)
		ui.Text(c, value).FontSize(textSize(c, 17)).FontWeight(700).
			TextColor(c.Theme().Text).FontFeatures("tnum").SingleLine()
	})
}

// verdictPill states a binary health result with both a dot and a word. The
// dot makes the state scannable; the word makes it readable without colour.
func verdictPill(c *ui.Context, ok bool, label string) ui.Element {
	tone, tint := ui.Hex(auroraGreen), ui.Hex(successTint)
	if !ok {
		tone, tint = ui.Hex(auroraRed), ui.Hex(errorTint)
	}
	return ui.Row(c).Padding(space2, space3+2).Gap(space2).Radius(999).
		Background(tint).BorderWidth(1).BorderColor(tone.Alpha(0.35)).
		Children(func() {
			statusDot(c, tone, 6)
			ui.Text(c, tr(label)).FontSize(textSize(c, 12)).FontWeight(600).
				TextColor(tone).SingleLine()
		})
}

// hoverRaise is the hover motion shared by the dashboard's cards: they grow a
// fraction and warm their border instead of changing colour abruptly. The
// motion is a scale rather than an upward shift because a shifted element moves
// out from under a pointer resting on its last pixel, which flickers the hover
// state; scaling only ever grows the target. MyGo turns the motion into an
// instant state change when the desktop asks for reduced motion, so the
// feedback survives that preference.
func hoverRaise(c *ui.Context, b ui.Element, hovered bool) ui.Element {
	pose := ui.Pose{}
	if hovered {
		pose = ui.Pose{Scale: 1.006}
	}
	b.AnimatePose(pose, ui.Animation{Duration: 120 * time.Millisecond})
	return b
}

// shortcutCap is the muted key cap shown inside a button or a footer row. It
// documents a chord where the action is, so the shortcut is discoverable
// without opening Settings.
func shortcutCap(c *ui.Context, chord string) ui.Element {
	cap := ui.Row(c).Padding(2, space1+1).Radius(radiusSmall - 4).Background(c.Theme().Surface).
		BorderWidth(1).BorderColor(c.Theme().Border).Shrink(0)
	cap.Children(func() {
		ui.Text(c, chord).FontSize(textSize(c, 9.5)).FontWeight(600).
			TextColor(c.Theme().TextMuted).SingleLine()
	})
	return cap
}

// sectionRule opens a band of content with its name and a rule that fades out
// to the right. The rule replaces a full-width border: it groups what follows
// without drawing a line across the whole page, which read as a table edge
// above the counters.
func sectionRule(c *ui.Context, label string) {
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(space3).Padding(space3, 0, space1, 0).Children(func() {
		eyebrow(c, label)
		line := ui.Row(c).Height(1).Grow(1)
		line.Draw(func(p *ui.Painter, r ui.Rect) {
			tone := c.Theme().Border
			p.FillGradient(r, ui.LinearGradient{Angle: 90, Stops: []ui.GradientStop{
				{Color: tone, At: 0},
				{Color: tone.Alpha(0), At: 1},
			}}, 0)
		})
	})
}

// cardAccent lights the top edge of a card with a module's hue. It is the
// quietest possible statement of "this card belongs to that module": two
// pixels of light along the rim, gone before the title, so six counters in one
// grid still scan as one family instead of six coloured blocks.
func cardAccent(e ui.Element, tone ui.Color) ui.Element {
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		inset := radiusPanel + space2
		p.FillGradient(ui.Rect{X: r.X + inset, Y: r.Y, W: r.W - 2*inset, H: 2},
			ui.LinearGradient{Angle: 90, Stops: []ui.GradientStop{
				{Color: tone.Alpha(0), At: 0},
				{Color: tone.Alpha(0.55), At: 0.5},
				{Color: tone.Alpha(0), At: 1},
			}}, 1)
	})
	return e
}
