package nativeui

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

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
	ui.Text(c, tr(text)).FontSize(textSize(c, 12)).TextColor(ui.Hex(inkMuted))
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
	label = tr(label)
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(textSize(c, 11)).Width(labelColumnWidth(c, label)).
			TextColor(ui.Hex(inkMuted)).SingleLine()
		ui.Text(c, abbreviate(value)).FontSize(textSize(c, 12)).TextColor(ui.Hex(ink)).SingleLine()
		if quietButton(c, tr("Copy")+" "+label).Clicked() {
			// The whole value is copied, not the shortened cell.
			c.Services().WriteClipboard(value)
			// The status line lives above the cards and may be scrolled out of
			// sight; the toast confirms the copy where the pointer is.
			c.Toast(fmt.Sprintf(tr("%s is on the clipboard"), label))
			if onCopied != nil {
				onCopied()
			}
		}
	})
}

// readoutRow is one label/value pair in a readout.
type readoutRow struct{ label, value string }

// readout draws the result of a read as label/value rows on a recessed panel,
// so an answer is visibly an answer rather than another line of prose that
// looks like the hint above it.
func readout(c *ui.Context, rows []readoutRow) {
	ui.Column(c).FillWidth().Padding(10).Gap(4).Radius(8).
		Background(ui.Hex(night)).Children(func() {
		for _, r := range rows {
			label := tr(r.label)
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				ui.Text(c, label).FontSize(textSize(c, 11)).Width(labelColumnWidth(c, label)).
					TextColor(ui.Hex(inkMuted)).SingleLine()
				ui.Text(c, r.value).FontSize(textSize(c, 12)).TextColor(ui.Hex(ink)).
					Grow(1).Shrink(1).MinWidth(0).SingleLine()
			})
		}
	})
}

// labelColumnWidth keeps a label readable when it is longer than the old fixed
// 72px, without letting it consume the value column. The estimate follows the
// 11pt label size and the user's text-size preference.
func labelColumnWidth(c *ui.Context, label string) float32 {
	scale := textSize(c, 11) / 11
	width := float32(len(label)) * 7.5 * scale
	if width < 72*scale {
		return 72 * scale
	}
	if width > 160*scale {
		return 160 * scale
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
	ui.Text(c, tr(what)+": "+err.Error()).FontSize(textSize(c, 12)).TextColor(ui.Hex(auroraRed))
	if !hasContent {
		return false
	}
	ui.Text(c, tr("Showing the last successful load.")).FontSize(textSize(c, 12)).TextColor(ui.Hex(inkMuted))
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

// allSet reports whether every value has non-whitespace content. It is the
// readiness test for a form's primary action: a button that cannot submit yet
// is clearer than a click that returns a service error for an empty field.
func allSet(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return false
		}
	}
	return true
}
