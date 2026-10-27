package nativeui

import (
	"github.com/egoist/mygo/ui"
)

// textSize applies the desktop's text-scale preference to a fixed size.
// Explicit FontSize values are absolute, so without this the window ignored the
// OS text-size setting (the theme default only covered text with no explicit
// size). ReduceMotion needs no equivalent: MyGo sends animations to their
// target at once when it is on.
func textSize(c *ui.Context, size float32) float32 {
	scale := c.Preferences().TextScale
	if scale <= 0 {
		scale = 1
	}
	return size * scale
}

// pageGlyph draws a page's icon in a tinted tile. It is the window's recurring
// mark for "which module is this": the same glyph sits in the sidebar row and
// beside the page title, so the page and its navigation entry are recognisably
// the same place.
func pageGlyph(c *ui.Context, p page) {
	ui.Row(c).Size(36, 36).Center().Radius(10).Background(ui.Hex(selection)).
		BorderWidth(1).BorderColor(ui.Hex(border)).Children(func() {
		ui.Icon(c, pageIcons[p]).Size(18, 18).TextColor(ui.Hex(aurora))
	})
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
	if width < twoColumnMinWidth {
		ui.Column(c).FillWidth().Gap(16).Children(func() {
			left()
			right()
		})
		return
	}
	ui.Row(c).FillWidth().AlignItems(ui.Start).Gap(16).Children(func() {
		ui.Column(c).Grow(1).Shrink(1).Gap(16).Children(left)
		ui.Column(c).Grow(1).Shrink(1).Gap(16).Children(right)
	})
}

// twoColumnMinWidth is the window width below which two cards would each be
// too narrow for a label and a base64 key. The sidebar and page padding take
// 264px, so the cards get about 360px each at the threshold.
const twoColumnMinWidth float32 = 1000

// card draws a raised panel: the surface every section's content sits on.
// A card is what separates one concern from the next now that the sidebar no
// longer does it with rules, and it is the unit the dashboard grid repeats.
func card(c *ui.Context, fn func()) ui.Element {
	// A card fills its column's width and is as tall as its content: growing
	// it would stretch a three-field form down the page. The soft shadow and
	// the hairline of light along the top edge lift it off the polar-night
	// background; a border alone read as a table cell rather than a card.
	return ui.Column(c).FillWidth().Padding(16).Gap(10).Radius(12).
		Background(ui.Hex(surface)).BorderWidth(1).BorderColor(ui.Hex(border)).
		Shadow(0, 12, 30, -6, ui.RGBA(0, 0, 0, 0.35)).
		InsetShadow(0, 1, 0, 0, ui.RGBA(255, 255, 255, 0.04)).
		Children(fn)
}

// metricCard draws one counter: a label, the value, and what it counts. A
// numeric value is the largest type on the page so the eye lands on it first;
// a word verdict ("verified") is drawn smaller, because 30pt of lowercase
// reads as an error rather than a summary.
func metricCard(c *ui.Context, label, value, note string, tone string, numeric bool, target page) bool {
	body := func() {
		ui.Text(c, tr(label)).FontSize(textSize(c, 11)).TextColor(ui.Hex(inkMuted)).SingleLine()
		// The value sits in a fixed-height slot so a word verdict and a
		// one-digit number put their note on the same line across the row.
		size := float32(20)
		if numeric {
			size = 30
		}
		ui.Column(c).Height(38).Justify(ui.Center).Children(func() {
			ui.Text(c, tr(value)).FontSize(textSize(c, size)).Bold().TextColor(ui.Hex(tone)).SingleLine()
		})
		ui.Text(c, tr(note)).FontSize(textSize(c, 11)).TextColor(ui.Hex(inkMuted))
	}
	if target == pageDashboard {
		card(c, body)
		return false
	}
	return actionCard(c, label, body).Clicked()
}

// actionCard is a whole card that opens another page. It keeps the card's
// surface, border and lift, but adds the button's hover/press state so the
// dashboard counters read as shortcuts rather than decoration.
func actionCard(c *ui.Context, label string, body func()) ui.Element {
	b := ui.ButtonBase(c).FillWidth().Label(tr(label)).Padding(16).Radius(12).AlignItems(ui.Start).
		Background(ui.Hex(surface)).BorderWidth(1).BorderColor(ui.Hex(border)).
		Shadow(0, 12, 30, -6, ui.RGBA(0, 0, 0, 0.35)).
		InsetShadow(0, 1, 0, 0, ui.RGBA(255, 255, 255, 0.04))
	if b.Hovered() {
		b.Background(ui.Hex(surfaceHover))
	}
	if b.Pressed() {
		b.Background(ui.Hex(surfacePressed))
	}
	b.Children(func() { ui.Column(c).FillWidth().Gap(10).Children(body) })
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
	ui.Column(c).Gap(2).Children(func() {
		ui.Text(c, tr(title)).FontSize(textSize(c, 16)).Bold().TextColor(ui.Hex(ink))
		if purpose != "" {
			ui.Text(c, tr(purpose)).FontSize(textSize(c, 12)).TextColor(ui.Hex(inkMuted))
		}
	})
}

// badge draws a small pill of text. It must sit in a row: a badge drawn
// straight into a card's column takes the whole width and reads as a banner.
func badge(c *ui.Context, text, background, foreground string) {
	ui.Badge(c, tr(text)).Background(ui.Hex(background)).TextColor(ui.Hex(foreground))
}

// verdictBadge draws a good/bad pill: green for a verdict that holds, red for
// one that does not.
func verdictBadge(c *ui.Context, text string, ok bool) {
	if ok {
		badge(c, text, auroraGreen, night)
		return
	}
	badge(c, text, auroraRed, night)
}

// formCard draws a section as a card: its heading, its fields, and whatever
// the section reports. Cards are what give the scrolling page structure, so a
// long form reads as a list of jobs rather than one wall of inputs.
func formCard(c *ui.Context, title, purpose string, fn func()) {
	card(c, func() {
		sectionHeading(c, title, purpose)
		ui.Column(c).Gap(8).Children(fn)
	})
}

// emptyState is the visual for a store that is genuinely empty. It keeps the
// page's identity (the same icon as the sidebar) instead of a lone muted line,
// and separates the title from the explanation so a first-time reader knows
// whether the store is empty or the read failed.
func emptyState(c *ui.Context, p page, title, body string) {
	ui.Column(c).FillWidth().Padding(24, 20).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Size(42, 42).Center().Radius(12).Background(ui.Hex(selection)).
			BorderWidth(1).BorderColor(ui.Hex(border)).Children(func() {
			ui.Icon(c, pageIcons[p]).Size(22, 22).TextColor(ui.Hex(aurora))
		})
		ui.Text(c, tr(title)).FontSize(textSize(c, 14)).Bold().TextColor(ui.Hex(ink)).TextAlign(ui.Center)
		ui.Text(c, tr(body)).FontSize(textSize(c, 12)).TextColor(ui.Hex(inkMuted)).TextAlign(ui.Center)
	})
}
