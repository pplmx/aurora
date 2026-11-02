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
	// raw and rawArgs hold the message before translation, so the line can be said
	// again in another language. The window rebuilds every string it reads on each
	// frame, but a status line is a value: without this it kept the language it was
	// created in, and switching to Chinese left "Failed: token name is required" on
	// screen beside Chinese labels.
	//
	// raw stays a format with its arguments rather than a formatted string, because
	// the catalogs translate templates ("Registered %s"), not their results.
	raw     string
	rawArgs []any
}

func (s statusLine) empty() bool { return s.text == "" }

// spoken returns the line in the language in effect now. The stored text was
// translated when the line was created; this re-says the same message, which is
// what keeps one frame from mixing two languages after a switch.
func (s statusLine) spoken() string {
	if s.raw == "" {
		return s.text
	}
	if s.isErr {
		return localizeServiceErrorMessage(s.raw)
	}
	if len(s.rawArgs) == 0 {
		return tr(s.raw)
	}
	return fmt.Sprintf(tr(s.raw), s.rawArgs...)
}

var statusRevision atomic.Uint64

func nextStatusRevision() uint64 { return statusRevision.Add(1) }

func okStatus(format string, args ...any) statusLine {
	return statusLine{
		text:     fmt.Sprintf(tr(format), args...),
		revision: nextStatusRevision(),
		raw:      format,
		rawArgs:  args,
	}
}

func errStatus(err error) statusLine {
	if err == nil {
		return statusLine{}
	}
	// An error with an empty message would render as nothing at all: the line's
	// empty() test looks at the text, and a blank line is not drawn. A click that
	// fails silently is worse than one that fails oddly, so the sentence falls back
	// to a generic one — the only case where the message is not the error's own.
	if strings.TrimSpace(err.Error()) == "" {
		return statusLine{
			text:     tr("The operation failed without a message."),
			isErr:    true,
			revision: nextStatusRevision(),
			raw:      "The operation failed without a message.",
		}
	}
	return statusLine{
		text:     localizeServiceError(err),
		isErr:    true,
		revision: nextStatusRevision(),
		raw:      err.Error(),
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
		raw:      text,
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
	// The export destination checks carry a path, so a full-message lookup cannot
	// reach them; the sentence around the path is the translatable part.
	if dir, ok := strings.CutPrefix(message, "export directory "); ok {
		if dir, ok := strings.CutSuffix(dir, " does not exist"); ok {
			if translated := tr("export directory %s does not exist"); translated != "export directory %s does not exist" {
				return fmt.Sprintf(translated, dir)
			}
		}
	}
	// A wrapped error names its cause before the detail:
	// "http error response: status 503". The cause is the translatable part and
	// the detail stays as it is, so the head is tried before the tail — trying
	// the tail alone translates "status" on its own and leaves the sentence
	// half English.
	//
	// A cause can be wrapped twice: the fetcher reports
	// "failed to fetch data: request failed: Service Unavailable", where only
	// the middle segment has an entry. The head is therefore localized first
	// and the remainder localized again, so the inner cause is reached instead
	// of leaving the whole line English.
	if i := strings.Index(message, ": "); i >= 0 {
		head, rest := message[:i], message[i+2:]
		if translated := tr(head); translated != head {
			return translated + "：" + localizeServiceErrorMessage(rest)
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

// renderStatus draws a status line, if there is one, and reports whether the
// caller should take over focus after its Dismiss action removes the line.
//
// The line carries a word ("Failed"/"Done") as well as the colour, on a panel
// tinted by the outcome: colour alone is not a signal for a reader with a
// colour-vision difference, and a one-line sentence was easy to miss above a
// page of cards.
//
// It is drawn as a full-width strip with a lit left edge rather than a rounded
// badge over a solid border: the strip sits directly under the header band, so
// a heavy outline there fought the band's own hairline. The word, the tint and
// the edge say the same thing in three ways.
func renderStatus(c *ui.Context, scope page, s statusLine, dismiss func()) bool {
	if s.empty() {
		return false
	}
	dismissed := false
	label, tone, tint := "Done", c.Theme().Success, successTint(c)
	if s.isErr {
		label, tone, tint = "Failed", c.Theme().Danger, errorTint(c)
	}
	spoken := statusAnnouncementText(s)
	row := ui.Row(c).Key(scope).FillWidth().AlignItems(ui.Center).Gap(space3).
		Padding(space2+2, space3+2).Radius(radiusSmall).
		BorderWidth(0, 0, 0, 2).BorderColor(tone).
		Role(ui.RoleStatus).Label(spoken)
	row.Draw(func(p *ui.Painter, r ui.Rect) {
		p.FillGradient(r, statusTintFill(tint), radiusSmall)
	})
	announceStatusChange(c, row, s, spoken)
	row.Children(func() {
		statusDot(c, tone, 7)
		ui.Text(c, tr(label)).FontSize(textSize(c, 11.5)).FontWeight(700).
			TextColor(tone).SingleLine().Shrink(0).Role(ui.RoleNone)
		ui.Text(c, s.spoken()).FontSize(textSize(c, 12)).TextColor(c.Theme().Text).
			Grow(1).Shrink(1).MinWidth(0).Role(ui.RoleNone)
		if dismiss != nil {
			if quietButton(c, "Dismiss").Clicked() {
				dismiss()
				dismissed = true
			}
		}
	})
	return dismissed
}

// inlineStatus draws a card-local outcome. It is the compact sibling of
// renderStatus: no panel and no badge, because the result belongs to one card,
// but the colour still carries the outcome so a rejected write cannot read
// like a successful one when both are a line of muted text.
func inlineStatus(c *ui.Context, s statusLine) {
	if s.empty() {
		return
	}
	tone := c.Theme().Success
	if s.isErr {
		tone = c.Theme().Danger
	}
	spoken := statusAnnouncementText(s)
	status := ui.Text(c, s.spoken()).FontSize(textSize(c, 12)).TextColor(tone).
		Role(ui.RoleStatus).Label(spoken)
	announceStatusChange(c, status, s, spoken)
}

// statusAnnouncementText includes the outcome word because a screen reader
// cannot infer success or failure from the red and green tints.
func statusAnnouncementText(s statusLine) string {
	if s.isErr {
		return labelText(tr("Failed"), s.spoken())
	}
	return labelText(tr("Done"), s.spoken())
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
