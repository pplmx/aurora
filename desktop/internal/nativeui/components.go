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
		text.TextColor(ui.Hex("#c62828"))
	} else {
		text.TextColor(ui.Hex("#2e7d32"))
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
			ui.Text(c, hint).FontSize(11).TextColor(ui.Hex("#6b7c7f"))
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

// sectionTitle draws a page heading with the same eyebrow/purpose shape every
// module page used, so the pages stay recognisable across the rewrite.
func sectionTitle(c *ui.Context, title, purpose string) {
	ui.Text(c, title).FontSize(22).Bold()
	if purpose != "" {
		ui.Text(c, purpose).FontSize(12).TextColor(ui.Hex("#6b7c7f"))
	}
}

// emptyNote draws the "nothing here yet" line a table shows instead of an empty
// body, distinct from a load failure.
func emptyNote(c *ui.Context, text string) {
	ui.Text(c, text).FontSize(12).TextColor(ui.Hex("#6b7c7f"))
}
