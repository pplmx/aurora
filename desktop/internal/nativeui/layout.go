package nativeui

import (
	"fmt"
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

// tableColumnWidth scales a column's text-driven minimum with the desktop's
// text-size preference. Table Width and MinWidth are layout values in DIPs,
// while FontSize is already scaled by textSize; without this, larger text had
// less room for the same number of glyphs and long table values were clipped.
func tableColumnWidth(c *ui.Context, width float32) float32 {
	return width * textScale(c)
}

// pageGlyph draws a page's icon in a tinted tile. It is the window's recurring
// mark for "which module is this": the same glyph sits in the sidebar row and
// beside the page title, so the page and its navigation entry are recognisably
// the same place. Each module has its own hue, which the dashboard's counters
// and the page tile repeat.
func pageGlyph(c *ui.Context, p page) ui.Element {
	return iconTile(c, pageIcons[p], pageAccent(p), iconTileMedium)
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
func twoColumns(c *ui.Context, left, right func()) {
	width, _ := c.Size()
	if width < twoColumnMinWidth*textScale(c) {
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

// stackedRecordMinScale is where multi-column tables become record cards.
// At 2x the Lottery history and Oracle health tables each had columns beyond
// the minimum window; below this point they still fit and remain scannable as
// tables.
const stackedRecordMinScale float32 = 1.75

func stackedRecords(c *ui.Context) bool {
	return textScale(c) >= stackedRecordMinScale
}

// heading draws a title with the rank assistive technology needs to navigate
// the page by sections. Bold text alone only communicates that hierarchy to
// sighted users.
func heading(c *ui.Context, text string, size float32, level int) ui.Element {
	return ui.Text(c, text).FontSize(textSize(c, size)).Bold().TextColor(c.Theme().Text).
		Role(ui.RoleHeading).Level(level)
}

// card draws a raised panel: the surface every section's content sits on.
// A card is what separates one concern from the next now that the sidebar no
// longer does it with rules, and it is the unit the dashboard grid repeats.
func card(c *ui.Context, fn func()) ui.Element {
	// A card fills its column's width and is as tall as its content: growing
	// it would stretch a three-field form down the page. The soft shadow and
	// the hairline of light along the top edge lift it off the polar-night
	// background; a border alone read as a table cell rather than a card.
	return ui.Column(c).FillWidth().Padding(space5, space4+space1).Gap(space3).
		Radius(radiusPanel).LinearGradient(panelFill(c)).
		BorderWidth(1).BorderColor(c.Theme().Border).
		Shadow(0, 22, 48, -20, ui.RGBA(0, 0, 0, 0.55)).
		InsetShadow(0, 1, 0, 0, ui.RGBA(255, 255, 255, 0.045)).
		Children(fn)
}

// recordPanel is the compact surface used when a wide table becomes a list of
// records. It is recessed into a card so each record reads as one object, not
// as another form section.
func recordPanel(c *ui.Context, fn func()) {
	ui.Column(c).FillWidth().Padding(space3).Gap(space2).Radius(radiusSmall).
		Background(c.Theme().Background.Mix(c.Theme().Surface, 0.35)).
		BorderWidth(1).BorderColor(c.Theme().Border).Children(fn)
}

// selectableRecordPanel is the interactive form of recordPanel. It keeps the
// same recessed surface, adds the keyboard behavior of a button, and uses the
// selection tint plus accent border so a chosen record remains identifiable
// without relying on colour alone: the record's facts and the selected readout
// below the collection both change with it.
func selectableRecordPanel(c *ui.Context, selected bool, label string, fn func()) ui.Element {
	panel := ui.ButtonBase(c).FillWidth().Padding(12).Radius(10).
		AlignItems(ui.Start).Label(label)
	panel.BorderWidth(1)

	fill, outline := c.Theme().Background, c.Theme().Border
	if selected {
		fill, outline = c.Theme().Selection, ui.Hex(aurora)
	} else if panel.Pressed() {
		fill = c.Theme().SurfacePressed
	} else if panel.Hovered() {
		fill = c.Theme().SurfaceHover
	}
	panel.BorderColor(outline)
	panel.Background(panel.AnimateColor("record-fill", fill,
		ui.Animation{Duration: 100 * time.Millisecond}))
	panel.Children(func() {
		ui.Column(c).FillWidth().Gap(6).Children(fn)
	})
	return panel
}

// recordChoiceLabel tells assistive technology which record a panel selects,
// and changes once the panel is chosen so the state is not left to colour.
func recordChoiceLabel(name string, selected bool) string {
	if selected {
		return fmt.Sprintf(tr("Selected %s"), name)
	}
	return fmt.Sprintf(tr("Select %s"), name)
}

// recordField stacks a label over a value that may wrap. Unlike a table cell,
// it does not need an ellipsis to keep the next column visible.
func recordField(c *ui.Context, label, value string) {
	ui.Column(c).FillWidth().Gap(2).Children(func() {
		ui.Text(c, tr(label)).FontSize(textSize(c, 10)).TextColor(c.Theme().TextMuted)
		ui.Text(c, value).FontSize(textSize(c, 12)).TextColor(c.Theme().Text)
	})
}

// recordMetric draws one compact fact inside a selectable record. It is the
// shared shape behind health counters, token symbols and exact supplies.
func recordMetric(c *ui.Context, label, value string, tone ui.Color) {
	ui.Column(c).Gap(1).Children(func() {
		ui.Text(c, tr(label)).FontSize(textSize(c, 10)).TextColor(c.Theme().TextMuted)
		ui.Text(c, value).FontSize(textSize(c, 15)).Bold().TextColor(tone)
	})
}

// metricCard draws one counter: a label, the value, and what it counts. A
// numeric value is the largest type on the page so the eye lands on it first;
// a word verdict ("verified") is drawn smaller, because 30pt of lowercase
// reads as an error rather than a summary. The module's glyph sits in the
// corner so the four counters that open the same page are still told apart,
// and a hairline of the module's hue runs along the card's top edge.
func metricCard(c *ui.Context, label, value, note string, tone ui.Color, numeric bool, target page) bool {
	body := func() {
		ui.Row(c).FillWidth().Gap(space2).AlignItems(ui.Center).Children(func() {
			eyebrow(c, label).Grow(1).Shrink(1).MinWidth(0)
			iconTile(c, pageIcons[target], pageAccent(target), iconTileSmall)
		})
		// The value sits in a fixed-height slot so a word verdict and a
		// one-digit number put their note on the same line across the row.
		size := float32(20)
		if numeric {
			size = statValueSize
		}
		ui.Column(c).Height(textSize(c, 40)).Justify(ui.Center).Children(func() {
			ui.Text(c, tr(value)).FontSize(textSize(c, size)).FontWeight(700).TextColor(tone).
				FontFeatures("tnum").SingleLine()
		})
		ui.Text(c, tr(note)).FontSize(textSize(c, 11.5)).TextColor(c.Theme().TextMuted)
	}
	if target == pageDashboard {
		card(c, body)
		return false
	}
	return cardAccent(actionCard(c, label, body), pageAccent(target)).Clicked()
}

// actionCard is a whole card that opens another page. It keeps the card's
// surface, border and lift, but adds the button's hover/press state so the
// dashboard counters read as shortcuts rather than decoration.
func actionCard(c *ui.Context, label string, body func()) ui.Element {
	b := ui.ButtonBase(c).FillWidth().Label(tr(label)).
		Padding(space5, space4+space1).Radius(radiusPanel).AlignItems(ui.Start).
		LinearGradient(panelFill(c)).
		BorderWidth(1).BorderColor(c.Theme().Border).
		Shadow(0, 22, 48, -20, ui.RGBA(0, 0, 0, 0.55)).
		InsetShadow(0, 1, 0, 0, ui.RGBA(255, 255, 255, 0.045))
	switch {
	case b.Pressed():
		b.LinearGradient(ui.LinearGradient{
			Angle: 165,
			From:  c.Theme().SurfacePressed,
			To:    c.Theme().Surface,
		})
	case b.Hovered():
		b.LinearGradient(ui.LinearGradient{
			Angle: 150,
			From:  c.Theme().SurfaceHover,
			To:    c.Theme().Surface,
		})
		b.BorderColor(ui.Hex(aurora).Alpha(0.45))
	}
	hoverRaise(c, b, b.Hovered())
	b.Children(func() { ui.Column(c).FillWidth().Gap(space3).Children(body) })
	return b
}

// metricRow lays out cards in equal columns. A ui.Grid sizes its tracks from
// its children, which let the dashboard's six cards run past the window's
// right edge; rows of growing columns stay inside the content width.
func metricRow(c *ui.Context, n int, fn func(i int)) {
	ui.Row(c).FillWidth().AlignItems(ui.Start).Gap(14).Children(func() {
		for i := range n {
			i := i
			ui.Column(c).Grow(1).Shrink(1).Children(func() { fn(i) })
		}
	})
}

// sectionHeading draws a section's title and its one-line purpose, the shape
// every page uses above its forms. It replaced a 22pt title that competed with
// the page header for the same attention.
func sectionHeading(c *ui.Context, title, purpose string) {
	ui.Column(c).Gap(space1).Children(func() {
		ui.Row(c).Gap(space2).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Width(3).Height(textSize(c, 15)).Radius(1.5).
				Background(ui.Hex(aurora)).Shrink(0)
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
	emptyStateIcon(c, pageIcons[p], pageAccent(p), title, body)
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
