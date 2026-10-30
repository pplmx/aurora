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

// panelFill is the card's background: a quiet diagonal wash from the raised
// surface toward its resting tone. The card also carries a one-pixel top
// highlight and a mode-specific shadow; those three cues are what keep a
// card above the page without turning every panel into a coloured block.
func panelFill(c *ui.Context) ui.LinearGradient {
	return ui.LinearGradient{
		Angle: 155,
		From:  c.Theme().Surface.Mix(c.Theme().SurfaceHover, 0.32),
		To:    c.Theme().Surface,
	}
}

// workspaceFill is the page background. It carries the page's own hue as a
// very faint light behind the fixed header, then settles to the window tone.
// The wash stays under five percent so every text and border contrast ratio
// still describes the surface actually seen.
func workspaceFill(c *ui.Context) ui.LinearGradient {
	tone := c.Theme().Accent
	strength := float32(0.035)
	if !c.Theme().Dark {
		strength = 0.025
	}
	return ui.LinearGradient{
		Angle: 180,
		From:  c.Theme().Background.Mix(tone, strength),
		To:    c.Theme().Background,
		Start: 0,
		End:   0.55,
	}
}

// sidebarFill gives the navigation column its own paper/ink tone. The
// sidebar is chrome, not another content surface, so it is deliberately
// quieter than a card and separated from the workspace by one hairline.
func sidebarFill(c *ui.Context) ui.LinearGradient {
	tone := sidebarTone(c)
	return ui.LinearGradient{
		Angle: 180,
		From:  tone.Mix(c.Theme().Surface, 0.08),
		To:    tone.Mix(c.Theme().Background, 0.32),
	}
}

// panelGradient is panelFill tinted with a page or verdict hue, for the one
// panel a page leads with. The tint stays under a fifth of the way to the hue:
// it is a wash of light, not a coloured card.
func panelGradient(c *ui.Context, tone ui.Color, strength float32) ui.LinearGradient {
	settle := float32(0.24)
	if !c.Theme().Dark {
		settle = 0.14
	}
	return ui.LinearGradient{
		Angle: 150,
		From:  c.Theme().Surface.Mix(tone, strength),
		To:    c.Theme().Surface.Mix(c.Theme().Background, settle),
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
	tile := ui.Row(c).Size(size, size).Center().Radius(size*0.28).
		Gradient(tone.Mix(c.Theme().Surface, 0.72), tone.Mix(c.Theme().Surface, 0.9), 150).
		BorderWidth(1).BorderColor(tone.Alpha(0.42)).
		InsetShadow(0, 1, 0, 0, cardHighlight(c))
	tile.Children(func() {
		ui.Icon(c, icon).Size(size*0.48, size*0.48).TextColor(tone)
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
		Background(c.Theme().Background.Mix(c.Theme().Surface, 0.22)).
		BorderWidth(1).BorderColor(c.Theme().Border).Children(func() {
		eyebrow(c, label)
		ui.Text(c, value).FontSize(textSize(c, 17)).FontWeight(700).
			TextColor(c.Theme().Text).Font("monospace").FontFeatures("tnum").SingleLine()
	})
}

// verdictPill states a binary health result with both a dot and a word. The
// dot makes the state scannable; the word makes it readable without colour.
func verdictPill(c *ui.Context, ok bool, label string) ui.Element {
	tone, tint := c.Theme().Success, successTint(c)
	if !ok {
		tone, tint = c.Theme().Danger, errorTint(c)
	}
	return ui.Row(c).Padding(space2, space3+2).Gap(space2).Radius(999).
		Background(tint).BorderWidth(1).BorderColor(tone.Alpha(0.35)).
		Children(func() {
			statusDot(c, tone, 6)
			ui.Text(c, tr(label)).FontSize(textSize(c, 12)).FontWeight(600).
				TextColor(tone).SingleLine()
		})
}

// hoverRaise is the hover motion shared by the dashboard's quick actions: they grow a
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
	cap := ui.Row(c).Padding(2, space1+1).Radius(radiusSmall - 4).
		Background(c.Theme().Background.Mix(c.Theme().Surface, 0.7)).
		BorderWidth(1).BorderColor(c.Theme().Border).Shrink(0)
	cap.Children(func() {
		ui.Text(c, chord).FontSize(textSize(c, 9.5)).FontWeight(600).
			TextColor(c.Theme().TextMuted).Font("monospace").SingleLine()
	})
	return cap
}

// sectionRule opens a band of content with its name and a rule that fades out
// to the right. The rule replaces a full-width border: it groups what follows
// without drawing a line across the whole page, which read as a table edge
// above the counters.
func sectionRule(c *ui.Context, label string) {
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(space3).Padding(space3, 0, space1, 0).Children(func() {
		ui.Row(c).Gap(space2).AlignItems(ui.Center).Shrink(0).Children(func() {
			ui.Row(c).Width(3).Height(10).Radius(1.5).
				Background(c.Theme().Accent).Shrink(0)
			eyebrow(c, label)
		})
		line := ui.Row(c).Height(1).Grow(1)
		line.Draw(func(p *ui.Painter, r ui.Rect) {
			tone := c.Theme().Accent
			p.FillGradient(r, ui.LinearGradient{Angle: 90, Stops: []ui.GradientStop{
				{Color: tone.Alpha(0.55), At: 0},
				{Color: tone.Alpha(0), At: 1},
			}}, 0)
		})
	})
}
