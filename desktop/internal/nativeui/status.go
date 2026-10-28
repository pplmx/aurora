package nativeui

import (
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/egoist/mygo/ui"
)

// statusLine is the transient result a view shows under its form: either the
// success text a service returned or the error it failed with. Views set it
// through ok()/fail() so the two cases render differently and an error can
// never look like a success.
type statusLine struct {
	text     string
	isErr    bool
	revision uint64
}

func (s statusLine) empty() bool { return s.text == "" }

var statusRevision atomic.Uint64

func nextStatusRevision() uint64 { return statusRevision.Add(1) }

func okStatus(format string, args ...any) statusLine {
	return statusLine{
		text:     fmt.Sprintf(tr(format), args...),
		revision: nextStatusRevision(),
	}
}

func errStatus(err error) statusLine {
	if err == nil {
		return statusLine{}
	}
	return statusLine{
		text:     localizeServiceError(err),
		isErr:    true,
		revision: nextStatusRevision(),
	}
}

func errorStatus(format string, args ...any) statusLine {
	return errorStatusText(fmt.Sprintf(tr(format), args...))
}

func errorStatusText(text string) statusLine {
	return statusLine{
		text:     text,
		isErr:    true,
		revision: nextStatusRevision(),
	}
}

// LocalizeError translates the same service errors for desktop dialogs that
// live outside a view, such as File > Import and files opened by the shell.
// Unknown diagnostics stay verbatim.
func LocalizeError(err error) string {
	return localizeServiceError(err)
}

// localizeServiceError translates a known service error message. Unknown or
// dynamic messages stay verbatim, so a diagnostic still matches the log; for a
// wrapped sentinel such as "seed: seed too short", the last segment is
// translated while its context is kept.
func localizeServiceError(err error) string {
	if err == nil {
		return ""
	}
	return localizeServiceErrorMessage(err.Error())
}

func localizeServiceErrorMessage(message string) string {
	if translated := tr(message); translated != message {
		return translated
	}
	if detail, ok := strings.CutPrefix(message, "parse document: "); ok {
		document := tr("invalid lottery import document")
		if document != "invalid lottery import document" {
			return document + "：" + tr("parse document") + "：" + detail
		}
	}
	if i := strings.LastIndex(message, ": "); i >= 0 {
		tail := message[i+2:]
		if translated := tr(tail); translated != tail {
			return message[:i+2] + translated
		}
	}
	return message
}

// renderStatus draws a status line, if there is one.
//
// The line carries a word ("Failed"/"Done") as well as the colour, on a panel
// tinted by the outcome: colour alone is not a signal for a reader with a
// colour-vision difference, and a one-line sentence was easy to miss above a
// page of cards.
func renderStatus(c *ui.Context, scope page, s statusLine, dismiss func()) {
	if s.empty() {
		return
	}
	label, tone, tint := "Done", ui.Hex(auroraGreen), ui.Hex(successTint)
	if s.isErr {
		label, tone, tint = "Failed", ui.Hex(auroraRed), ui.Hex(errorTint)
	}
	spoken := statusAnnouncementText(s)
	row := ui.Row(c).Key(scope).FillWidth().AlignItems(ui.Center).Gap(10).Padding(10).Radius(8).
		Background(tint).BorderWidth(1).BorderColor(tone).
		Role(ui.RoleStatus).Label(spoken)
	announceStatusChange(c, row, s, spoken)
	row.Children(func() {
		badge(c, label, tone, c.Theme().Background).Role(ui.RoleNone)
		ui.Text(c, s.text).FontSize(textSize(c, 12)).TextColor(c.Theme().Text).Role(ui.RoleNone)
		if dismiss != nil {
			ui.Spacer(c)
			if quietButton(c, "Dismiss").Clicked() {
				dismiss()
			}
		}
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
	tone := ui.Hex(auroraGreen)
	if s.isErr {
		tone = ui.Hex(auroraRed)
	}
	spoken := statusAnnouncementText(s)
	status := ui.Text(c, s.text).FontSize(textSize(c, 12)).TextColor(tone).
		Role(ui.RoleStatus).Label(spoken)
	announceStatusChange(c, status, s, spoken)
}

// statusAnnouncementText includes the outcome word because a screen reader
// cannot infer success or failure from the red and green tints.
func statusAnnouncementText(s statusLine) string {
	if s.isErr {
		return tr("Failed") + ": " + s.text
	}
	return tr("Done") + ": " + s.text
}

type announcedStatus struct {
	text     string
	isErr    bool
	revision uint64
}

// announceStatusChange reports a new outcome once for the element that shows
// it. Frames are rebuilt for unrelated input, so announcing the visible line
// on every frame would repeat the same result until it was dismissed.
func announceStatusChange(c *ui.Context, owner ui.Element, s statusLine, spoken string) {
	last := ui.Local(owner, "status-announcement", func() announcedStatus {
		return announcedStatus{}
	})
	changed := s.text != last.text || s.isErr != last.isErr
	if s.revision != 0 && s.revision != last.revision {
		changed = true
	}
	if !changed {
		return
	}
	last.text, last.isErr, last.revision = s.text, s.isErr, s.revision
	c.Announce(spoken)
}
