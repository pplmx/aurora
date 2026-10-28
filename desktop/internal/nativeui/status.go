package nativeui

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

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
	return statusLine{text: fmt.Sprintf(tr(format), args...)}
}

func errStatus(err error) statusLine {
	if err == nil {
		return statusLine{}
	}
	return statusLine{text: localizeServiceError(err), isErr: true}
}

// localizeServiceError translates a known service error message. Unknown or
// dynamic messages stay verbatim, so a diagnostic still matches the log; for a
// wrapped sentinel such as "seed: seed too short", the last segment is
// translated while its context is kept.
func localizeServiceError(err error) string {
	message := err.Error()
	if translated := tr(message); translated != message {
		return translated
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
func renderStatus(c *ui.Context, s statusLine, dismiss func()) {
	if s.empty() {
		return
	}
	label, tone, tint := "Done", ui.Hex(auroraGreen), ui.Hex(successTint)
	if s.isErr {
		label, tone, tint = "Failed", ui.Hex(auroraRed), ui.Hex(errorTint)
	}
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(10).Padding(10).Radius(8).
		Background(tint).BorderWidth(1).BorderColor(tone).
		Children(func() {
			badge(c, label, tone, c.Theme().Background)
			ui.Text(c, s.text).FontSize(textSize(c, 12)).TextColor(c.Theme().Text)
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
	ui.Text(c, s.text).FontSize(textSize(c, 12)).TextColor(tone)
}
