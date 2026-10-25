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

// renderStatus draws a status line, if there is one. The colour is the only
// difference between success and failure, so the text always states which it
// is — a reader who cannot see the colour still learns the outcome.
func renderStatus(c *ui.Context, s statusLine) {
	if s.empty() {
		return
	}
	text := ui.Text(c, s.text)
	if s.isErr {
		text.TextColor(ui.Hex(auroraRed))
	} else {
		text.TextColor(ui.Hex(auroraGreen))
	}
}

// field builds a labelled input row. MyGo's Field draws the label above the
// control it wraps, which is the same relationship the HTML <label for> had.
func field(c *ui.Context, label string, value *string) {
	ui.Field(c, label, func() {
		ui.TextInput(c, value)
	})
}

// fieldHint builds a labelled input with a hint under it, for inputs whose
// accepted range is not obvious from the label.
func fieldHint(c *ui.Context, label, hint string, value *string) {
	ui.Field(c, label, func() {
		ui.Column(c).Gap(2).Children(func() {
			ui.TextInput(c, value)
			ui.Text(c, hint).FontSize(11).TextColor(ui.Hex(inkMuted))
		})
	})
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
	ui.Text(c, text).FontSize(12).TextColor(ui.Hex(inkMuted))
}

// twoColumns lays out two stacks side by side, each filling half the width.
//
// A ui.Grid sizes every child to its own track, so cards of different content
// came out different widths — a three-field form was visibly narrower than the
// one beside it. Rows of two equal columns keep the cards aligned.
func twoColumns(c *ui.Context, left, right func()) {
	ui.Row(c).FillWidth().AlignItems(ui.Start).Gap(16).Children(func() {
		ui.Column(c).Grow(1).Shrink(1).Gap(16).Children(left)
		ui.Column(c).Grow(1).Shrink(1).Gap(16).Children(right)
	})
}

// card draws a raised panel: the surface every section's content sits on.
// A card is what separates one concern from the next now that the sidebar no
// longer does it with rules, and it is the unit the dashboard grid repeats.
func card(c *ui.Context, fn func()) ui.Element {
	// A card fills its column's width and is as tall as its content: growing
	// it would stretch a three-field form down the page.
	return ui.Column(c).FillWidth().Padding(16).Gap(10).Radius(12).
		Background(ui.Hex(surface)).BorderWidth(1).BorderColor(ui.Hex(border)).
		Children(fn)
}

// metricCard draws one counter: a label, the value, and what it counts. A
// numeric value is the largest type on the page so the eye lands on it first;
// a word verdict ("verified") is drawn smaller, because 30pt of lowercase
// reads as an error rather than a summary.
func metricCard(c *ui.Context, label, value, note string, tone string, numeric bool) {
	card(c, func() {
		ui.Text(c, label).FontSize(11).TextColor(ui.Hex(inkMuted)).SingleLine()
		size := float32(20)
		if numeric {
			size = 30
		}
		ui.Text(c, value).FontSize(size).Bold().TextColor(ui.Hex(tone)).SingleLine()
		ui.Text(c, note).FontSize(11).TextColor(ui.Hex(inkMuted))
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
		ui.Text(c, title).FontSize(16).Bold().TextColor(ui.Hex(ink))
		if purpose != "" {
			ui.Text(c, purpose).FontSize(12).TextColor(ui.Hex(inkMuted))
		}
	})
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
	ui.Text(c, message).FontSize(11).TextColor(ui.Hex(auroraRed))
}
