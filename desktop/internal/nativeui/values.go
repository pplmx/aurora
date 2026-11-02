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

// ShortHash is the display form of a block hash: the first 12 hex characters
// two installations compare in a support thread. A shorter value (a test fake)
// is returned unchanged.
func ShortHash(hash string) string {
	const keep = 12
	if len(hash) <= keep {
		return hash
	}
	return hash[:keep]
}

// joinList renders a []string for a record fact, comma-space separated like the
// retired Web UI's participants.join(', '); a nil slice reads as an em dash so
// an empty value is visibly empty rather than blank.
func joinList(items []string) string {
	if len(items) == 0 {
		return "—"
	}
	return strings.Join(items, ", ")
}

// emptyNote draws the "nothing here yet" line a table shows instead of an empty
// body, distinct from a load failure.
func emptyNote(c *ui.Context, text string) {
	ui.Text(c, tr(text)).FontSize(textSize(c, 12)).TextColor(c.Theme().TextMuted)
}

// copyRow draws a labelled value with a Copy button and returns that button
// when a caller needs to move focus to it.
//
// Native text is not selectable, and some values exist nowhere else once the
// frame is gone — a voter registration returns the only copy of its signing
// key. Without the button the operator would have to retype an 88-character
// base64 key, or lose it.
func copyRow(c *ui.Context, label, value string, onCopied func()) ui.Element {
	if value == "" {
		return ui.Element{}
	}
	var copyButton ui.Element
	label = tr(label)
	ui.Row(c).FillWidth().Wrap().Gap(space2).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(textSize(c, labelSize)).Width(labelColumnWidth(c, label)).
			TextColor(c.Theme().TextMuted).SingleLine()
		ui.Text(c, abbreviate(value)).FontSize(textSize(c, 12)).TextColor(c.Theme().Text).
			Grow(1).Shrink(1).MinWidth(0).SingleLine().Ellipsis("…").Tooltip(value)
		copyButton = quietButton(c, copyActionLabel(label))
		if copyButton.Clicked() {
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
	return copyButton
}

// readoutRow is one label/value pair in a readout.
type readoutRow struct{ label, value string }

// labelSize is the one size a readout or copy-row label is drawn at. It is a
// constant rather than a literal at each call site because the column width is
// derived from it, and a label drawn at a different size than its own column
// was measured for wraps inside it.
const labelSize float32 = 11

// readout draws the result of a read as label/value rows on a recessed panel,
// so an answer is visibly an answer rather than another line of prose that
// looks like the hint above it.
//
// Every row shares one label column, taken as the widest label in the panel.
// Measuring each row on its own left the values starting at a different x per
// row — "Version" at one offset, "Database path" at another — so a stack of
// facts read as a ragged list instead of a table.
func readout(c *ui.Context, rows []readoutRow) {
	labels := make([]string, len(rows))
	for i, r := range rows {
		labels[i] = tr(r.label)
	}
	column := labelColumnWidth(c, labels...)
	ui.Column(c).FillWidth().Padding(space3).Gap(space1).Radius(radiusSmall).
		Background(c.Theme().Background.Mix(c.Theme().Surface, 0.5)).
		BorderWidth(1).BorderColor(c.Theme().Border).Children(func() {
		for i, r := range rows {
			label := labels[i]
			ui.Row(c).FillWidth().Gap(space2).AlignItems(ui.Start).Children(func() {
				ui.Text(c, label).FontSize(textSize(c, labelSize)).Width(column).
					TextColor(c.Theme().TextMuted).SingleLine().Ellipsis("…").Tooltip(label)
				value := ui.Text(c, r.value).FontSize(textSize(c, 12)).TextColor(c.Theme().Text).
					Grow(1).Shrink(1).MinWidth(0)
				if wideLayout(c) {
					value.SingleLine().Ellipsis("…").Tooltip(r.value)
				}
			})
		}
	})
}

// labelColumnWidth measures the labels it is given and returns the column they
// share, clamped so a short label still reads as a column and a long one cannot
// eat the values beside it.
//
// It measures instead of counting characters: the old estimate multiplied the
// byte length by 7.5, which overstates a Chinese label roughly threefold
// (数据库路径 is 15 bytes and 55px at this size). The Chinese About card then
// spent half its width on an empty label column and ellipsised the values.
func labelColumnWidth(c *ui.Context, labels ...string) float32 {
	scale := textSize(c, labelSize) / labelSize
	var widest float32
	for _, label := range labels {
		if w, _ := c.MeasureText(0, ui.Span{Text: label, Size: textSize(c, labelSize)}); w > widest {
			widest = w
		}
	}
	switch {
	case widest < 72*scale:
		return 72 * scale
	case widest > 160*scale:
		return 160 * scale
	default:
		return widest
	}
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

// conciseText bounds a user-supplied name in prose prompts. Dialog messages
// have a fixed width and no scroll, so a maximum-length NFT name at a large
// text scale could push the irreversible action's buttons below the window.
// The prompt still carries the unique shortened id beside this readable prefix.
func conciseText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 {
		return value
	}
	chars := []rune(value)
	if len(chars) <= limit {
		return value
	}
	return string(chars[:limit]) + "…"
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
	message := labelText(tr(what), localizeServiceError(err))
	// A caller that joins several reads may hand over the same cause more than
	// once, and the strip would then state it once per line. The heading already
	// names what failed, so a reason repeated inside the body adds length
	// without adding a fact.
	message = dedupeLines(message)
	stale := ""
	if hasContent {
		stale = tr("Showing the last successful load.")
	}
	spoken := message
	if stale != "" {
		spoken += " " + stale
	}
	warning := ui.Column(c).Key(what).FillWidth().Gap(2).Role(ui.RoleStatus).Label(spoken).
		Padding(space2+2, space3+2).Radius(radiusSmall).
		BorderWidth(0, 0, 0, 2).BorderColor(c.Theme().Danger)
	// The failure shares the status strip's shape: a tint that fades to the
	// right and a lit left edge. As plain red text it read as another hint
	// line inside the card, and the one state that must not be missed was the
	// quietest thing on the page.
	warning.Draw(func(p *ui.Painter, r ui.Rect) {
		p.FillGradient(r, statusTintFill(errorTint(c)), radiusSmall)
	})
	announceStatusChange(c, warning,
		statusLine{text: message + "\x00" + stale, isErr: true}, spoken)
	// The mark and the leading word are the strip's own signal, so the
	// sentence below them does not need the danger ink as well: red body copy
	// states the failure twice and made the retained-data line, the one that
	// explains why the page still has rows, look like part of the error.
	warning.Children(func() {
		ui.Row(c).FillWidth().Gap(space2 + 1).AlignItems(ui.Center).Children(func() {
			statusDot(c, c.Theme().Danger, 7)
			ui.Text(c, tr("Failed")).FontSize(textSize(c, 11.5)).FontWeight(700).
				TextColor(c.Theme().Danger).SingleLine().Shrink(0)
			ui.Text(c, message).FontSize(textSize(c, 12)).TextColor(c.Theme().Text).
				Grow(1).Shrink(1).MinWidth(0)
		})
		if stale != "" {
			ui.Text(c, stale).FontSize(textSize(c, 12)).TextColor(c.Theme().TextMuted).
				Grow(1).Shrink(1).MinWidth(0)
		}
	})
	return hasContent
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

// rejectIncomplete keeps a direct handler call from bypassing the disabled
// action that normally guards the same form. The inline readiness checks in
// views handle the normal pointer and keyboard paths; this is the defensive
// path for a stale frame or an in-process caller.
func rejectIncomplete(status *statusLine, message string, values ...string) bool {
	if allSet(values...) {
		return false
	}
	*status = errorStatus(message)
	return true
}

// dedupeLines drops repeated reasons from a message built out of several
// errors, keeping the first occurrence of each in order. errors.Join separates
// its errors with newlines and does not notice that three reads failed for the
// same reason; the first line of that message also carries the heading, so a
// plain line comparison would miss that the heading already contains the
// reason. A line is dropped when its text appears in what has been kept.
func dedupeLines(message string) string {
	if !strings.Contains(message, "\n") {
		return message
	}
	var kept []string
	for _, line := range strings.Split(message, "\n") {
		reason := strings.TrimSpace(line)
		if reason == "" {
			continue
		}
		if strings.Contains(strings.Join(kept, "\n"), reason) {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}
