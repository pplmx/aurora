// Package nativeui implements the desktop application's interface with MyGo's
// own widgets instead of a web page.
//
// The previous shell hosted the same HTML UI the REST gateway serves, which
// meant the desktop app depended on web/ and on a browser engine to show six
// pages of forms. A view here is a function of application state that MyGo
// draws directly: no HTML, no JavaScript, no Content-Security-Policy, and
// nothing that can fail silently the way an Alpine directive did when the CSP
// forbade its evaluator.
//
// Every view reads and writes the same application services the REST layer
// uses (see api.Services), so a record created in the window is the record the
// CLI prints and the gateway returns.
package nativeui

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// page identifies the module the window is showing.
type page int

const (
	pageDashboard page = iota
	pageLottery
	pageVoting
	pageToken
	pageOracle
	pageNFT
	pageBlockchain
)

// pageIDs are the stable identifiers the sidebar chooses by. They are
// separate from the titles because a title is shown text: translating it
// later must not change what a choice is remembered as.
var pageIDs = map[page]string{
	pageDashboard:  "dashboard",
	pageLottery:    "lottery",
	pageVoting:     "voting",
	pageToken:      "token",
	pageOracle:     "oracle",
	pageNFT:        "nft",
	pageBlockchain: "blockchain",
}

// pageByID is the reverse lookup the sidebar needs when a click arrives.
func pageByID(id string) (page, bool) {
	for p, want := range pageIDs {
		if want == id {
			return p, true
		}
	}
	return 0, false
}

var pageTitles = map[page]string{
	pageDashboard:  "Dashboard",
	pageLottery:    "Lottery",
	pageVoting:     "Voting",
	pageToken:      "Token",
	pageOracle:     "Oracle",
	pageNFT:        "NFT",
	pageBlockchain: "Blockchain",
}

// pageOrder is the sidebar order; it matches the module list the old Web UI
// navigation exposed, so muscle memory carries over.
var pageOrder = []page{
	pageDashboard, pageLottery, pageVoting, pageToken, pageOracle, pageNFT, pageBlockchain,
}

// statusLine is the transient result a view shows under its form: either the
// success text a service returned or the error it failed with. Views set it
// through ok()/fail() so the two cases render differently and an error can
// never look like a success.
type statusLine struct {
	text  string
	isErr bool
}

func (s statusLine) empty() bool { return s.text == "" }

func okStatus(format string, args ...any) statusLine {
	return statusLine{text: fmt.Sprintf(format, args...)}
}

func errStatus(err error) statusLine {
	if err == nil {
		return statusLine{}
	}
	return statusLine{text: err.Error(), isErr: true}
}

// renderStatus draws a status line, if there is one.
//
// The line carries a word ("Failed"/"Done") as well as the colour, on a panel
// tinted by the outcome: colour alone is not a signal for a reader with a
// colour-vision difference, and a one-line sentence was easy to miss above a
// page of cards.
func renderStatus(c *ui.Context, s statusLine) {
	if s.empty() {
		return
	}
	label, tone, tint := "Done", auroraGreen, successTint
	if s.isErr {
		label, tone, tint = "Failed", auroraRed, errorTint
	}
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(10).Padding(10).Radius(8).
		Background(ui.Hex(tint)).BorderWidth(1).BorderColor(ui.Hex(tone)).
		Children(func() {
			badge(c, label, tone, night)
			ui.Text(c, s.text).FontSize(textSize(c, 12)).TextColor(ui.Hex(ink))
		})
}

// inlineStatus draws a card-local outcome. It is the compact sibling of
// renderStatus: no panel and no badge, because the result belongs to one card,
// but the colour still carries the outcome so a rejected write cannot read
// like a successful one when both are a line of muted text.
func inlineStatus(c *ui.Context, s statusLine) {
	if s.empty() {
		return
	}
	tone := auroraGreen
	if s.isErr {
		tone = auroraRed
	}
	ui.Text(c, s.text).FontSize(textSize(c, 12)).TextColor(ui.Hex(tone))
}

// field builds a labelled input row. MyGo's Field draws the label above the
// control it wraps, which is the same relationship the HTML <label for> had.
func field(c *ui.Context, label string, value *string) {
	ui.Field(c, label, func() {
		ui.TextInput(c, value)
	})
}

// The page's three action styles. Every button comes from one of them, so the
// same job looks the same in every card: primary is the one action a form
// exists for, secondary is a read or a supporting write, and danger is
// destructive (still behind its confirmation dialog). They are ButtonBase with
// Aurora's own look rather than the toolkit's default: one height, radius and
// weight across cards, and the hover/press feedback the reference MyGo apps
// use, so a click is acknowledged before the write finishes.
const buttonRadius float32 = 8

// maxListLimit is the largest page size a list form accepts; the REST surface
// applies the same ceiling to its ?limit.
const maxListLimit = 1000

func primaryButton(c *ui.Context, label string) ui.Element {
	return styledButton(c, label, ui.Hex(aurora), ui.Hex(auroraHover), ui.Hex(auroraPressed), onAccent)
}

func secondaryButton(c *ui.Context, label string) ui.Element {
	b := styledButton(c, label, ui.Hex(surface), ui.Hex(surfaceHover), ui.Hex(surfacePressed), ink)
	b.BorderWidth(1).BorderColor(ui.Hex(border))
	return b
}

func dangerButton(c *ui.Context, label string) ui.Element {
	red := ui.Hex(auroraRed)
	return styledButton(c, label, red,
		red.Mix(ui.RGB(255, 255, 255), 0.14), red.Mix(ui.RGB(0, 0, 0), 0.16), night)
}

// quietButton is the small, low-emphasis action used inside a card: copying a
// value, for instance, which should not compete with the card's main button.
func quietButton(c *ui.Context, label string) ui.Element {
	b := ui.ButtonBase(c).Height(26).PaddingX(10).Radius(6).
		AlignItems(ui.Center).Label(label).TextColor(ui.Hex(inkMuted))
	fill := ui.Transparent
	if b.Hovered() || b.Pressed() {
		fill = ui.Hex(surfaceHover)
	}
	b.Background(b.AnimateColor("fill", fill, ui.Animation{Duration: 100 * time.Millisecond}))
	b.Children(func() {
		ui.Text(c, label).FontSize(textSize(c, 12)).SingleLine()
	})
	return b
}

func styledButton(c *ui.Context, label string, fill, hover, pressed ui.Color, textColor string) ui.Element {
	b := ui.ButtonBase(c).Height(34).PaddingX(14).Radius(buttonRadius).
		AlignItems(ui.Center).Label(label).TextColor(ui.Hex(textColor))
	state := fill
	switch {
	case b.Pressed():
		state = pressed
	case b.Hovered():
		state = hover
	}
	b.Background(b.AnimateColor("fill", state, ui.Animation{Duration: 120 * time.Millisecond}))
	pose := ui.Pose{}
	if b.Pressed() {
		pose.Scale = 0.97
	}
	b.AnimatePose(pose, ui.Snappy)
	b.Children(func() {
		ui.Text(c, label).FontSize(textSize(c, 13)).FontWeight(600).SingleLine()
	})
	return b
}

// fieldHint builds a labelled input with a hint under it, for inputs whose
// accepted range is not obvious from the label.
func fieldHint(c *ui.Context, label, hintText string, value *string) {
	ui.Field(c, label, func() {
		ui.Column(c).Gap(2).Children(func() {
			ui.TextInput(c, value)
			hint(c, hintText)
		})
	})
}

// textArea builds a labelled multi-line input, for values that are naturally
// longer than one line (JSON headers, a description).
func textArea(c *ui.Context, label string, value *string) {
	ui.Field(c, label, func() {
		ui.TextArea(c, value).Height(64)
	})
}

// textAreaHint is textArea with a line under it, for a value whose format the
// label does not explain.
func textAreaHint(c *ui.Context, label, hintText string, value *string) {
	ui.Field(c, label, func() {
		ui.Column(c).Gap(2).Children(func() {
			ui.TextArea(c, value).Height(64)
			hint(c, hintText)
		})
	})
}

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

// hint draws a muted line under a control, explaining a value the label
// cannot.
func hint(c *ui.Context, text string) {
	ui.Text(c, text).FontSize(textSize(c, 11)).TextColor(ui.Hex(inkMuted))
}

// shortID shortens an identifier for a table cell, keeping enough of it to
// recognise and paste: the first 12 characters, with an ellipsis when there is
// more. Full ids are still reachable where a form takes one as input.
func shortID(id string) string {
	const keep = 12
	if len(id) <= keep {
		return id
	}
	return id[:keep] + "…"
}

// joinList renders a []string for a table cell. The Web UI used
// participants.join(', '); a nil slice reads as an em dash so an empty cell is
// visibly empty rather than blank.
func joinList(items []string) string {
	if len(items) == 0 {
		return "—"
	}
	return strings.Join(items, ", ")
}

// emptyNote draws the "nothing here yet" line a table shows instead of an empty
// body, distinct from a load failure.
func emptyNote(c *ui.Context, text string) {
	ui.Text(c, text).FontSize(textSize(c, 12)).TextColor(ui.Hex(inkMuted))
}

// copyRow draws a labelled value with a Copy button.
//
// Native text is not selectable, and some values exist nowhere else once the
// frame is gone — a voter registration returns the only copy of its signing
// key. Without the button the operator would have to retype an 88-character
// base64 key, or lose it.
func copyRow(c *ui.Context, label, value string, onCopied func()) {
	if value == "" {
		return
	}
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(textSize(c, 11)).Width(copyLabelWidth(label)).TextColor(ui.Hex(inkMuted))
		ui.Text(c, abbreviate(value)).FontSize(textSize(c, 12)).TextColor(ui.Hex(ink)).SingleLine()
		if quietButton(c, "Copy "+label).Clicked() {
			// The whole value is copied, not the shortened cell.
			c.Services().WriteClipboard(value)
			// The status line lives above the cards and may be scrolled out of
			// sight; the toast confirms the copy where the pointer is.
			c.Toast(label + " is on the clipboard")
			if onCopied != nil {
				onCopied()
			}
		}
	})
}

// copyLabelWidth keeps a copy row's label readable when it is longer than the
// old fixed 72px (for example "Selected candidate ID" or "Owner private key").
// The width is an estimate at the 11pt label size, clamped so the value still
// has room beside it.
func copyLabelWidth(label string) float32 {
	width := float32(len(label)) * 6.4
	if width < 72 {
		return 72
	}
	if width > 140 {
		return 140
	}
	return width
}

// abbreviate keeps both ends of a long value so the row shows what it holds
// without stretching across the card.
func abbreviate(value string) string {
	const head, tail = 10, 6
	if len(value) <= head+tail+1 {
		return value
	}
	return value[:head] + "…" + value[len(value)-tail:]
}

// loadWarning reports a failed refresh above whatever the page already showed.
//
// Every loader keeps its previous rows when a read fails, but the views used to
// return on the error line alone: a transient database failure blanked a list
// the operator was reading (Voting did not even draw the error, so the failure
// was invisible). It draws the reason and, when there is still content from the
// last successful load, says so and returns true so the caller keeps drawing it.
// With nothing to show it returns false: a page that never loaded must not fall
// through to its "nothing here yet" note, which would claim the store is empty
// when the truth is that it could not be read.
func loadWarning(c *ui.Context, what string, err error, hasContent bool) bool {
	ui.Text(c, what+": "+err.Error()).FontSize(textSize(c, 12)).TextColor(ui.Hex(auroraRed))
	if !hasContent {
		return false
	}
	ui.Text(c, "Showing the last successful load.").FontSize(textSize(c, 12)).TextColor(ui.Hex(inkMuted))
	return true
}

// valueOr renders an empty form value as an em dash, for prose that names what
// the operator typed.
func valueOr(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return strings.TrimSpace(value)
}

// containsFold reports whether any field contains query, ignoring case: the
// filter behind a list's SearchField. An empty query matches everything.
func containsFold(query string, fields ...string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return true
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), q) {
			return true
		}
	}
	return false
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
func metricCard(c *ui.Context, label, value, note string, tone string, numeric bool) {
	card(c, func() {
		ui.Text(c, label).FontSize(textSize(c, 11)).TextColor(ui.Hex(inkMuted)).SingleLine()
		// The value sits in a fixed-height slot so a word verdict and a
		// one-digit number put their note on the same line across the row.
		size := float32(20)
		if numeric {
			size = 30
		}
		ui.Column(c).Height(38).Justify(ui.Center).Children(func() {
			ui.Text(c, value).FontSize(textSize(c, size)).Bold().TextColor(ui.Hex(tone)).SingleLine()
		})
		ui.Text(c, note).FontSize(textSize(c, 11)).TextColor(ui.Hex(inkMuted))
	})
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
		ui.Text(c, title).FontSize(textSize(c, 16)).Bold().TextColor(ui.Hex(ink))
		if purpose != "" {
			ui.Text(c, purpose).FontSize(textSize(c, 12)).TextColor(ui.Hex(inkMuted))
		}
	})
}

// badge draws a small pill of text. It must sit in a row: a badge drawn
// straight into a card's column takes the whole width and reads as a banner.
func badge(c *ui.Context, text, background, foreground string) {
	ui.Badge(c, text).Background(ui.Hex(background)).TextColor(ui.Hex(foreground))
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

// fieldError draws a field's complaint directly under the control that caused
// it, in the same red the status lines use.
func fieldError(c *ui.Context, message string) {
	if message == "" {
		return
	}
	ui.Text(c, message).FontSize(textSize(c, 11)).TextColor(ui.Hex(auroraRed))
}
