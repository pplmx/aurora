package nativeui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// The ledger's record view.
//
// The Blockchain page and the dashboard's activity card both show the same
// thing — what the chain recorded — so they share one renderer. A record is
// drawn as a sentence: the module's mark and name, the action, the object it
// names, then the facts worth reading. The exact payload sits underneath the
// sentence, closed by default, because the decoded sentence must never become
// the only copy of what the chain holds.
//
// This replaced a three-column table whose middle-and-right cells were the
// raw payload: a scroll of JSON blobs and pipe-delimited base64 keys. An audit
// trail nobody can read is not an audit trail.

// recordModuleName renders the module a payload names. The stored value is the
// identifier the writer put on the chain ("lottery", "token"), which is not
// copy: the pill shows the module's own page name in the window's language, so
// a Chinese window says 抽奖 rather than "lottery" beside 抽奖 in the sidebar.
// An identifier with no page keeps its stored spelling.
func recordModuleName(module string) string {
	module = strings.TrimSpace(module)
	if module == "" {
		return "—"
	}
	if target, ok := pageForBlockModule(module); ok {
		return tr(pageTitles[target])
	}
	return module
}

// recordModuleTone gives a module its identity colour. It is the same hue the
// sidebar, the page tile and the dashboard counters use, so a record's mark
// says which module wrote it before a word is read.
func recordModuleTone(c *ui.Context, module string) ui.Color {
	if target, ok := pageForBlockModule(module); ok {
		return pageAccent(c, target)
	}
	return c.Theme().TextMuted
}

// blockRecord draws one chain entry as a record card. The exact payload opens
// and closes through the control at the card's right edge when onToggle is
// non-nil; the overview passes nil because it is a summary and leaves the
// evidence to the page that exists to hold it.
func blockRecord(c *ui.Context, entry BlockRow, expanded bool, onToggle func()) ui.Element {
	panel := recordSurface(c, recordModuleTone(c, entry.Module))
	panel.Label(blockRecordLabel(entry))
	panel.Children(func() {
		recordSentence(c, entry, expanded, onToggle)
		recordFacts(c, entry)
		if expanded {
			recordPayloadBody(c, entry.Payload)
		}
	})
	return panel
}

// recordSurface is the recessed card a record sits on. It is one step below
// the page's own cards: a record is an entry inside the ledger card, not a
// section of the page.
//
// A three-pixel bar of the module's hue runs down its left edge. The line of
// records then reads as a colour-coded stream — which module wrote what — but
// the bar never carries meaning alone: the module's name and coloured pill sit
// at the start of the same row.
func recordSurface(c *ui.Context, tone ui.Color) ui.Element {
	return ui.Column(c).FillWidth().Padding(space3+2, space4).Gap(space2 + 2).
		Radius(radiusSmall + 2).
		Background(c.Theme().Background.Mix(c.Theme().Surface, 0.35)).
		BorderWidth(1).BorderColor(c.Theme().Border).
		Draw(func(p *ui.Painter, r ui.Rect) {
			// The bar is drawn inset from the card's own rounded corners so it
			// never pokes past them, and it fades at both ends the way the
			// accent hairlines do: a flat block of colour read as a border of
			// its own rather than as a tint of the record's module.
			inset := radiusSmall + 2
			clear, solid := tone.Alpha(0), tone.Alpha(0.8)
			p.FillGradient(ui.Rect{X: r.X + 1, Y: r.Y + inset, W: 3, H: max(0, r.H-2*inset)},
				ui.LinearGradient{Angle: 180, Stops: []ui.GradientStop{
					{Color: clear, At: 0},
					{Color: solid, At: 0.18},
					{Color: solid, At: 0.82},
					{Color: clear, At: 1},
				}}, 1.5)
		})
}

// recordSentence is the record's first line: which module, what it did, and
// what it did it to, with the block's own time, its height and the payload
// toggle at the right edge.
func recordSentence(c *ui.Context, entry BlockRow, expanded bool, onToggle func()) {
	tone := recordModuleTone(c, entry.Module)
	ui.Row(c).FillWidth().Wrap().Gap(space2 + 2).AlignItems(ui.Center).Children(func() {
		// The module is a lit pill rather than plain text: it is the record's
		// provenance, and the one field an auditor scans first when reading a
		// long chain. The name stays beside the dot, so the colour is
		// reinforcement rather than the only signal.
		ui.Row(c).Gap(space2).AlignItems(ui.Center).Padding(1, space2+2).Radius(999).
			Background(tone.Alpha(0.12)).BorderWidth(1).BorderColor(tone.Alpha(0.32)).
			Shrink(0).Children(func() {
			statusDot(c, tone, 5)
			ui.Text(c, recordModuleName(entry.Module)).FontSize(textSize(c, 10.5)).FontWeight(600).
				TextColor(tone).SingleLine()
		})
		// The action is the sentence's verb. When it says the same word as the
		// module pill beside it (a Chinese "抽奖 抽奖"), one of the two is
		// dropped: the pill already names the module, and repeating it reads
		// as a rendering bug rather than as a sentence.
		if action := tr(entry.Action); entry.Action != "" && action != recordModuleName(entry.Module) {
			ui.Text(c, action).FontSize(textSize(c, 12.5)).FontWeight(600).
				TextColor(c.Theme().Text).Shrink(0)
		}
		recordObject(c, entry)
		recordStamp(c, entry, expanded, onToggle)
	})
}

// recordObject draws what the action was done to. A decoded record has a
// subject; an entry the decoder does not understand falls back to its stored
// text, so an unfamiliar payload still reads as something rather than as an
// empty line under its module.
func recordObject(c *ui.Context, entry BlockRow) {
	if entry.Subject != "" {
		ui.Text(c, entry.Subject).FontSize(textSize(c, 12.5)).
			TextColor(c.Theme().Text).Grow(1).Shrink(1).MinWidth(0).
			SingleLine().Ellipsis("…").Tooltip(entry.Subject)
		return
	}
	text := entry.Summary
	if text == "" {
		text = entry.Payload
	}
	if text == "" {
		ui.Spacer(c)
		return
	}
	// The stored text wraps instead of clipping: it is all the rendering an
	// unfamiliar payload gets, so cutting it would lose the entry's only
	// readable form.
	ui.Text(c, text).FontSize(textSize(c, 12)).TextColor(c.Theme().TextMuted).
		Grow(1).Shrink(1).MinWidth(0)
}

// recordStamp is the record's provenance at the right edge: what the block
// recorded, the payload control that opens the exact bytes, and the block's
// own time and height.
func recordStamp(c *ui.Context, entry BlockRow, expanded bool, onToggle func()) {
	ui.Row(c).Gap(space2).AlignItems(ui.Center).Shrink(0).Children(func() {
		if stamp := recordTimeLabel(entry.Timestamp); stamp != "" {
			ui.Text(c, stamp).FontSize(textSize(c, 10.5)).
				TextColor(c.Theme().TextMuted).SingleLine()
		}
		blockHeightChip(c, entry.Height)
		if onToggle != nil {
			recordPayloadToggle(c, expanded, onToggle)
		}
	})
}

// recordFacts draws the labelled values the decoder read out of the payload.
// Facts read left to right in the order the payload named them, so an auditor
// comparing two records finds the same field in the same place.
func recordFacts(c *ui.Context, entry BlockRow) {
	if len(entry.Facts) == 0 {
		return
	}
	ui.Row(c).FillWidth().Wrap().Gap(space4).AlignItems(ui.End).Children(func() {
		for _, fact := range entry.Facts {
			recordFactView(c, fact)
		}
	})
}

// recordFactView draws one decoded fact on a record.
func recordFactView(c *ui.Context, fact BlockFact) {
	value := fact.Value
	if value == "" {
		value = "—"
	}
	ui.Column(c).Gap(1).Shrink(0).MaxWidth(textSize(c, 260)).Children(func() {
		ui.Text(c, tr(fact.Label)).FontSize(textSize(c, 9.5)).FontWeight(600).
			LetterSpacing(0.4).TextColor(c.Theme().TextMuted).SingleLine()
		ui.Text(c, value).FontSize(textSize(c, 12)).TextColor(c.Theme().Text).
			SingleLine().Ellipsis("…").Tooltip(value)
	})
}

// recordPayloadToggle is the control that opens the exact recorded text. It is
// a quiet chip rather than a full-width button: the evidence is one click
// away, not the first thing read, and a button here made every record look
// like it existed to be opened.
func recordPayloadToggle(c *ui.Context, expanded bool, onToggle func()) {
	label := "Payload"
	if expanded {
		label = "Hide payload"
	}
	b := ui.ButtonBase(c).Height(max(20, textSize(c, 20))).PaddingX(space2).
		Radius(radiusSmall - 2).AlignItems(ui.Center).Label(tr(label))
	fill := c.Theme().Surface
	if b.Hovered() || b.Pressed() {
		fill = c.Theme().SurfaceHover
	}
	b.Background(b.AnimateColor("payload-fill", fill, ui.Animation{Duration: 100 * time.Millisecond}))
	b.BorderWidth(1).BorderColor(c.Theme().Border)
	b.Children(func() {
		ui.Text(c, tr(label)).FontSize(textSize(c, 9.5)).FontWeight(600).
			TextColor(c.Theme().TextMuted).SingleLine()
	})
	if b.Clicked() {
		onToggle()
	}
}

// blockHeightChip is the block number, drawn in tabular figures so heights
// line up when records stack. The chip carries the bare number as its tooltip:
// a record is read as a sentence, but the exact height is what an auditor
// quotes, and a wrapped line must never be the only place it appears.
func blockHeightChip(c *ui.Context, height int64) ui.Element {
	exact := strconv.FormatInt(height, 10)
	return ui.Row(c).Padding(1, space2).Radius(radiusSmall - 2).
		Background(c.Theme().Surface).BorderWidth(1).BorderColor(c.Theme().Border).
		Shrink(0).Children(func() {
		ui.Text(c, "#"+exact).FontSize(textSize(c, 10)).
			FontWeight(600).TextColor(c.Theme().TextMuted).FontFeatures("tnum").
			SingleLine().Ellipsis("…").Tooltip(exact)
	})
}

// recordPayloadBody shows the exact recorded text. It is drawn in the recessed
// tone the readouts use rather than as a card of its own: it is evidence
// attached to the record above it, not a second record.
func recordPayloadBody(c *ui.Context, payload string) {
	if payload == "" {
		emptyNote(c, "This entry has no stored payload.")
		return
	}
	ui.Column(c).FillWidth().Padding(space2+2, space3).Gap(space1).Radius(radiusSmall).
		Background(c.Theme().Background).BorderWidth(1).BorderColor(c.Theme().Border).
		Children(func() {
			ui.Text(c, payload).FontSize(textSize(c, 11.5)).
				TextColor(c.Theme().TextMuted).FontFeatures("tnum")
		})
}

// recordTimeLabel renders a block's own timestamp in local time. It returns ""
// when the entry carries no clock, so a record read without one does not show
// a 1970 date.
func recordTimeLabel(timestamp int64) string {
	if timestamp <= 0 {
		return ""
	}
	return time.Unix(timestamp, 0).Format(localTimeLayout)
}

// blockRecordLabel is the sentence assistive technology reads for a chain
// record. A table cell exposed only its value, which left a screen-reader user
// to assemble the meaning from unrelated fragments.
func blockRecordLabel(entry BlockRow) string {
	parts := make([]string, 0, 4)
	if entry.Module != "" {
		parts = append(parts, recordModuleName(entry.Module))
	}
	if entry.Action != "" {
		parts = append(parts, tr(entry.Action))
	}
	if entry.Subject != "" {
		parts = append(parts, entry.Subject)
	} else if entry.Summary != "" {
		parts = append(parts, entry.Summary)
	}
	if entry.Height > 0 {
		parts = append(parts, fmt.Sprintf(tr("block %d"), entry.Height))
	}
	return joinWith(parts, " · ")
}

// joinWith joins non-empty parts with a separator.
func joinWith(parts []string, sep string) string {
	out := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		if out != "" {
			out += sep
		}
		out += part
	}
	return out
}
