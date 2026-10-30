package nativeui

import (
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

// textSize applies the desktop's text-scale preference to a fixed size.
// Explicit FontSize values are absolute, so without this the window ignored the
// OS text-size setting (the theme default only covered text with no explicit
// size). ReduceMotion needs no equivalent: MyGo sends animations to their
// target at once when it is on.
func textSize(c *ui.Context, size float32) float32 {
	return size * textScale(c)
}

func textScale(c *ui.Context) float32 {
	if scale := c.Preferences().TextScale; scale > 0 {
		return scale
	}
	return 1
}

// pageGlyph draws a page's icon in a tinted tile. It is the window's recurring
// mark for "which module is this": the same glyph sits in the sidebar row and
// beside the page title, so the page and its navigation entry are recognisably
// the same place. Each module has its own hue, which the dashboard's counters
// and the page tile repeat.
func pageGlyph(c *ui.Context, p page) ui.Element {
	return iconTile(c, pageIcons[p], pageAccent(c, p), iconTileMedium)
}

// twoColumns lays out two stacks side by side, each filling half the width,
// and stacks them in one column when the window is too narrow for two readable
// cards. A fixed two-column layout at the 900px minimum left each card about
// 300px wide, so labels, keys and table cells wrapped or clipped; the same
// content reads in one column there.
//
// A ui.Grid sizes every child to its own track, so cards of different content
// came out different widths — a three-field form was visibly narrower than the
// one beside it. Rows of two equal columns keep the cards aligned.
// sideBySide reports whether twoColumns places its columns beside each other
// at the current width and text scale, or stacks them.
func sideBySide(c *ui.Context) bool {
	width, _ := c.Size()
	return width >= twoColumnMinWidth*textScale(c)
}

func twoColumns(c *ui.Context, left, right func()) {
	if !sideBySide(c) {
		ui.Column(c).FillWidth().Gap(cardGap).Children(func() {
			left()
			right()
		})
		return
	}
	ui.Row(c).FillWidth().AlignItems(ui.Start).Gap(pageGap).Children(func() {
		ui.Column(c).Grow(1).Shrink(1).Gap(cardGap).Children(left)
		ui.Column(c).Grow(1).Shrink(1).Gap(cardGap).Children(right)
	})
}

// twoColumnMinWidth is the window width below which two cards would each be
// too narrow for their controls. At the default text size the sidebar and page
// padding leave about 480px per card at the 1240px threshold. The live
// threshold scales with the user's text size because tabs and labels grow with
// it while the window does not.
const twoColumnMinWidth float32 = 1240

// wideLayoutMinScale is where a side-by-side presentation gives way to a
// stacked one. The collections themselves no longer switch renderings — they
// are records at every text size (see collectionList) — but two surfaces still
// depend on a wide arrangement: a readout's value is drawn on one line only
// while it fits beside its label, and the tally's comparison bars need room
// for a name and a count on the same row.
const wideLayoutMinScale float32 = 1.75

// wideLayout reports whether the current text size still leaves room for a
// side-by-side arrangement. It is the one breakpoint those two surfaces share,
// so they change shape at the same moment rather than each picking its own.
func wideLayout(c *ui.Context) bool {
	return textScale(c) < wideLayoutMinScale
}

// heading draws a title with the rank assistive technology needs to navigate
// the page by sections. Bold text alone only communicates that hierarchy to
// sighted users.
func heading(c *ui.Context, text string, size float32, level int) ui.Element {
	return ui.Text(c, text).FontSize(textSize(c, size)).Bold().TextColor(c.Theme().Text).
		Role(ui.RoleHeading).Level(level)
}

// card draws a raised panel: the surface every section's content sits on.
// A card separates one concern from the next, and the dashboard repeats it
// for the composition and timeline rather than for a grid of equal counters.
func card(c *ui.Context, fn func()) ui.Element {
	// A card fills its column's width and is as tall as its content: growing
	// it would stretch a three-field form down the page. The soft shadow and
	// the hairline of light along the top edge lift it off the workspace;
	// a border alone read as a table cell rather than a card.
	panel := ui.Column(c).FillWidth().Padding(space5, space4+space1).Gap(space3).
		Radius(radiusPanel).LinearGradient(panelFill(c)).
		BorderWidth(1).BorderColor(c.Theme().Border).
		Shadow(0, 20, 46, -22, cardShadow(c)).
		InsetShadow(0, 1, 0, 0, cardHighlight(c))
	// A short accent rail on the left edge ties a card to the current
	// module without tinting its whole surface. The mark is deliberately
	// inset from both rounded corners so it reads as a binder tab, not a
	// second border.
	panel.DrawOver(func(p *ui.Painter, r ui.Rect) {
		rail := ui.Rect{X: r.X, Y: r.Y + radiusPanel, W: 3, H: max(0, r.H-2*radiusPanel)}
		tone := c.Theme().Accent
		p.FillGradient(rail, ui.LinearGradient{Angle: 180, Stops: []ui.GradientStop{
			{Color: tone.Alpha(0), At: 0},
			{Color: tone.Alpha(0.58), At: 0.18},
			{Color: tone.Alpha(0.58), At: 0.82},
			{Color: tone.Alpha(0), At: 1},
		}}, 1.5)
	})
	panel.Children(fn)
	return panel
}

// recordPanel is the compact surface used when a wide table becomes a list of
// records. It is recessed into a card so each record reads as one object, not
// as another form section.
func recordPanel(c *ui.Context, fn func()) {
	ui.Column(c).FillWidth().Padding(space3).Gap(space2).Radius(radiusSmall).
		Background(c.Theme().Background.Mix(c.Theme().Surface, 0.24)).
		BorderWidth(1).BorderColor(c.Theme().Border).
		InsetShadow(0, 1, 0, 0, cardHighlight(c)).Children(fn)
}

// selectableRecordPanel is the interactive form of recordPanel. It keeps the
// same recessed surface, adds the keyboard behavior of a button, and uses the
// selection tint plus accent border so a chosen record remains identifiable
// without relying on colour alone: the record's facts and the selected readout
// below the collection both change with it.
func selectableRecordPanel(c *ui.Context, selected bool, label string, fn func()) ui.Element {
	// The record's own padding stays at layout scale rather than growing with
	// the text setting: at 2x a scaled padding left a one-line title floating
	// in a card with more empty space than content. The text inside it still
	// scales through textSize.
	panel := ui.ButtonBase(c).FillWidth().Padding(space2, space2+2).
		Radius(radiusSmall + 2).AlignItems(ui.Start).Label(label)
	panel.BorderWidth(1)

	fill, outline := c.Theme().Background.Mix(c.Theme().Surface, 0.18), c.Theme().Border
	if selected {
		fill, outline = c.Theme().Selection, c.Theme().Accent
	} else if panel.Pressed() {
		fill = c.Theme().SurfacePressed
	} else if panel.Hovered() {
		fill = c.Theme().SurfaceHover
	}
	panel.BorderColor(outline)
	panel.Background(panel.AnimateColor("record-fill", fill,
		ui.Animation{Duration: 100 * time.Millisecond}))
	panel.Children(func() {
		ui.Column(c).FillWidth().Gap(space1 + 1).Children(fn)
	})
	return panel
}

// sectionHeading draws a section's title and its one-line purpose, the shape
// every page uses above its forms. It replaced a 22pt title that competed with
// the page header for the same attention.
func sectionHeading(c *ui.Context, title, purpose string) {
	ui.Column(c).Gap(space1).Children(func() {
		ui.Row(c).Gap(space2).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Width(4).Height(textSize(c, 15)).Radius(2).
				Background(c.Theme().Accent).Shrink(0)
			heading(c, tr(title), sectionSize, 2)
		})
		if purpose != "" {
			ui.Text(c, tr(purpose)).FontSize(textSize(c, 12)).TextColor(c.Theme().TextMuted)
		}
	})
}

// badge draws a small pill of text. It must sit in a row: a badge drawn
// straight into a card's column takes the whole width and reads as a banner.
func badge(c *ui.Context, text string, background, foreground ui.Color) ui.Element {
	return ui.Badge(c, tr(text)).Background(background).TextColor(foreground)
}

// formCard draws a section as a card: its heading, its fields, and whatever
// the section reports. Cards are what give the scrolling page structure, so a
// long form reads as a list of jobs rather than one wall of inputs.
func formCard(c *ui.Context, title, purpose string, fn func()) {
	panel := card(c, func() {
		sectionHeading(c, title, purpose)
		ui.Column(c).Gap(8).Children(fn)
	})
	panel.Label(tr(title))
}

// emptyState is the visual for a store that is genuinely empty. It keeps the
// page's identity (the same icon as the sidebar) instead of a lone muted line,
// and separates the title from the explanation so a first-time reader knows
// whether the store is empty or the read failed.
func emptyState(c *ui.Context, p page, title, body string) {
	emptyStateIcon(c, pageIcons[p], pageAccent(c, p), title, body)
}

// emptyPanel is the compact empty state for a collection that is already
// inside a named card: the module's mark and one sentence, without repeating a
// title the card above already gives. It replaces a lone grey line where the
// list would be, so an empty store reads as a state rather than as a failed
// load.
func emptyPanel(c *ui.Context, icon *ui.SVG, tone ui.Color, text string) {
	ui.Column(c).FillWidth().Padding(space6, space5).Gap(space2).
		AlignItems(ui.Center).Children(func() {
		iconTile(c, icon, tone, iconTileMedium)
		ui.Text(c, tr(text)).FontSize(textSize(c, 12)).TextColor(c.Theme().TextMuted).
			TextAlign(ui.Center).MaxWidth(textSize(c, 420))
	})
}

// emptyStateIcon is emptyState with the mark chosen by the caller: a list of
// voters, candidates or sessions shares the Voting page's hue but not its
// glyph, so three empty cards on one page do not repeat the same picture.
func emptyStateIcon(c *ui.Context, icon *ui.SVG, tone ui.Color, title, body string) {
	ui.Column(c).FillWidth().Padding(space7, space6).Gap(space2).AlignItems(ui.Center).Children(func() {
		iconTile(c, icon, tone, iconTileLarge)
		heading(c, tr(title), 14.5, 3).TextAlign(ui.Center)
		ui.Text(c, tr(body)).FontSize(textSize(c, 12)).TextColor(c.Theme().TextMuted).
			TextAlign(ui.Center).MaxWidth(420)
	})
}

// firstRecordOnly is the gate shared by the modules whose page opens on a
// create form. It is true only after a successful read returned no rows and
// nothing else on the page has been touched. A load failure keeps the full
// page so its warning and any retained rows stay visible; a filter, a typed id
// or an open read belongs to a collection that already exists.
func firstRecordOnly(loadErr error, rows int, entered bool) bool {
	return loadErr == nil && rows == 0 && !entered
}

// firstRunPath is the companion to a module's create form when the store has
// no records yet. It names the state, then shows the capabilities that open
// after the first write. Dependent action cards stay off the page until that
// happens: a row of disabled controls does not teach a first-time operator
// what to do, and it makes the one useful form compete for attention.
func firstRunPath(c *ui.Context, p page, title, body string, steps ...string) {
	tone := pageAccent(c, p)
	panel := card(c, func() {
		ui.Row(c).FillWidth().Gap(space3).AlignItems(ui.Start).Children(func() {
			iconTile(c, pageIcons[p], tone, iconTileMedium)
			ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(space1).Children(func() {
				eyebrow(c, "First record")
				heading(c, tr(title), sectionSize, 2)
				ui.Text(c, tr(body)).FontSize(textSize(c, 12)).TextColor(c.Theme().TextMuted)
			})
		})
		ui.Column(c).FillWidth().Gap(space2).Children(func() {
			eyebrow(c, "What opens next")
			ui.Column(c).FillWidth().Gap(0).Role(ui.RoleList).Label(tr("What opens next")).Children(func() {
				for i, step := range steps {
					i, step := i, step
					last := i == len(steps)-1
					row := ui.Row(c).FillWidth().Padding(space2, 0).Gap(space3).AlignItems(ui.Start)
					row.Draw(func(painter *ui.Painter, r ui.Rect) {
						if last {
							return
						}
						painter.Fill(ui.Rect{X: r.X + 10.5, Y: r.Y + 22, W: 1, H: max(0, r.H-22)},
							c.Theme().Border, 0)
					})
					row.Role(ui.RoleListItem).Children(func() {
						mark := ui.Row(c).Size(22, 22).Center().Radius(999).
							Background(tone.Alpha(0.13)).BorderWidth(1).BorderColor(tone.Alpha(0.34)).
							Shrink(0)
						mark.Children(func() {
							ui.Text(c, strconv.Itoa(i+1)).FontSize(textSize(c, 10.5)).FontWeight(700).
								TextColor(tone).Font("monospace").SingleLine()
						})
						ui.Text(c, tr(step)).FontSize(textSize(c, 11.5)).TextColor(c.Theme().TextMuted).
							Grow(1).Shrink(1).MinWidth(0)
					})
				}
			})
		})
	})
	panel.LinearGradient(panelGradient(c, tone, 0.1))
}
