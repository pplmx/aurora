package nativeui

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
)

// guideState remembers which page guides are open. They are per page so a
// reader can leave one explanation open while moving between modules.
type guideState struct {
	lottery    bool
	voting     bool
	token      bool
	oracle     bool
	nft        bool
	blockchain bool
}

// guide draws the page's "How this works" steps. The default is closed: the
// page itself should still be scannable, and the explanation is for the moment
// the operator asks "what is the order here?".
func guide(c *ui.Context, open *bool, steps ...string) {
	card(c, func() {
		ui.Collapsible(c, tr("How this works"), open, func() {
			ui.Column(c).Gap(6).Children(func() {
				for i, step := range steps {
					ui.Row(c).Gap(8).AlignItems(ui.Start).Children(func() {
						ui.Text(c, fmt.Sprintf(tr("%d."), i+1)).FontSize(textSize(c, 12)).Bold().
							Width(textSize(c, 18)).Shrink(0).TextColor(ui.Hex(aurora))
						ui.Text(c, tr(step)).FontSize(textSize(c, 12)).TextColor(c.Theme().TextMuted).
							Grow(1).Shrink(1).MinWidth(0)
					})
				}
			})
		})
	})
}

// field builds a labelled input row. MyGo's Field draws the label above the
// control it wraps, which is the same relationship the HTML <label for> had.
func field(c *ui.Context, label string, value *string) {
	fieldWithError(c, label, "", func() {
		ui.TextInput(c, value)
	})
}

// fieldWithError keeps an error inside the field that caused it. MyGo marks
// the control invalid and exposes the message as its accessible description,
// so a screen reader does not have to infer which red line belongs where.
func fieldWithError(c *ui.Context, label, message string, fn func()) {
	f := ui.Field(c, tr(label), fn)
	if message != "" {
		f.Error(localizeServiceErrorMessage(message))
	}
}

// validatedField is the common text-input form of fieldWithError.
func validatedField(c *ui.Context, label string, value *string, message string) {
	fieldWithError(c, label, message, func() {
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

// namedTable gives every data grid an accessible name. The table role alone
// says what kind of collection is present, not which collection it is.
func namedTable(c *ui.Context, name string, state *ui.ListState, columns []ui.TableColumn, n int, cell func(row, col int)) ui.Element {
	return ui.Table(c, state, columns, n, cell).Label(tr(name))
}

func primaryButton(c *ui.Context, label string) ui.Element {
	return styledButton(c, label, ui.Hex(aurora), ui.Hex(auroraHover), ui.Hex(auroraPressed), ui.Hex(onAccent))
}

func secondaryButton(c *ui.Context, label string) ui.Element {
	b := styledButton(c, label, c.Theme().Surface, c.Theme().SurfaceHover, c.Theme().SurfacePressed, c.Theme().Text)
	b.BorderWidth(1).BorderColor(c.Theme().Border)
	return b
}

func dangerButton(c *ui.Context, label string) ui.Element {
	red := ui.Hex(auroraRed)
	return styledButton(c, label, red,
		red.Mix(ui.RGB(255, 255, 255), 0.14), red.Mix(ui.RGB(0, 0, 0), 0.16), c.Theme().Background)
}

// quietButton is the small, low-emphasis action used inside a card: copying a
// value, for instance, which should not compete with the card's main button.
func quietButton(c *ui.Context, label string) ui.Element {
	label = tr(label)
	b := ui.ButtonBase(c).Height(max(26, textSize(c, 26))).PaddingX(textSize(c, 10)).Radius(6).
		AlignItems(ui.Center).Label(label).TextColor(c.Theme().TextMuted)
	fill := ui.Transparent
	if b.Hovered() || b.Pressed() {
		fill = c.Theme().SurfaceHover
	}
	b.Background(b.AnimateColor("fill", fill, ui.Animation{Duration: 100 * time.Millisecond}))
	b.Children(func() {
		ui.Text(c, label).FontSize(textSize(c, 12)).SingleLine()
	})
	return b
}

// tabs draws a keyboard-operable tab row that wraps instead of pushing later
// tabs past the window edge. MyGo's stock Tabs is deliberately non-shrinking;
// the five Token write tabs already crossed the right edge at 2x text.
func tabs(c *ui.Context, selected *int, name string, labels ...string) {
	parts := ui.TabsBase(c, selected, len(labels))
	parts.List.Label(tr(name)).FillWidth().Wrap().Gap(4).Children(func() {
		for i, label := range labels {
			on := i == *selected
			tab := parts.Tab(i).Padding(textSize(c, 8), textSize(c, 12)).Radius(6).FocusRing(false)
			if on {
				tab.TextColor(c.Theme().Text)
			} else {
				tab.TextColor(c.Theme().TextMuted)
				if tab.Hovered() {
					tab.TextColor(c.Theme().Text)
				}
			}
			tab.DrawOver(func(p *ui.Painter, r ui.Rect) {
				if on {
					inset := textSize(c, 6)
					height := textSize(c, 2)
					p.Fill(ui.Rect{
						X: r.X + inset,
						Y: r.Y + r.H - height,
						W: r.W - 2*inset,
						H: height,
					}, ui.Hex(aurora), height/2)
				}
				if tab.FocusVisible() {
					p.FocusRing(r, [4]float32{6, 6, 6, 6})
				}
			})
			tab.Children(func() {
				text := ui.Text(c, label).FontSize(textSize(c, 13)).SingleLine()
				if on {
					text.FontWeight(600)
				}
			})
		}
	})
}

func styledButton(c *ui.Context, label string, fill, hover, pressed, textColor ui.Color) ui.Element {
	label = tr(label)
	b := ui.ButtonBase(c).Height(max(34, textSize(c, 34))).PaddingX(max(14, textSize(c, 14))).Radius(buttonRadius).
		AlignItems(ui.Center).Label(label).TextColor(textColor)
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
	ui.Field(c, tr(label), func() {
		ui.Column(c).Gap(2).Children(func() {
			ui.TextInput(c, value)
			hint(c, hintText)
		})
	})
}

// textArea builds a labelled multi-line input, for values that are naturally
// longer than one line (JSON headers, a description).
func textArea(c *ui.Context, label string, value *string) {
	ui.Field(c, tr(label), func() {
		textAreaControl(c, value)
	})
}

// textAreaHint is textArea with a line under it, for a value whose format the
// label does not explain.
func textAreaHint(c *ui.Context, label, hintText string, value *string) {
	ui.Field(c, tr(label), func() {
		ui.Column(c).Gap(2).Children(func() {
			textAreaControl(c, value)
			hint(c, hintText)
		})
	})
}

func textAreaControl(c *ui.Context, value *string) ui.Element {
	return ui.TextArea(c, value).Height(max(64, textSize(c, 64)))
}

// hint draws a muted line under a control, explaining a value the label
// cannot.
func hint(c *ui.Context, text string) {
	ui.Text(c, tr(text)).FontSize(textSize(c, 11)).TextColor(c.Theme().TextMuted)
}

// listPager draws the Previous/Entries/Next strip shared by paged
// collections. hasMore comes from fetching one row beyond the page, so an
// exact multiple of the page size does not enable a dead Next button.
func listPager(c *ui.Context, offset, count int, hasMore bool) (previous, next bool) {
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Children(func() {
		previous = secondaryButton(c, "Previous").Disabled(offset == 0).Clicked()
		ui.Spacer(c)
		ui.Text(c, fmt.Sprintf(tr("Entries %d–%d"), offset+1, offset+count)).
			FontSize(textSize(c, 11)).TextColor(c.Theme().TextMuted)
		ui.Spacer(c)
		next = secondaryButton(c, "Next").Disabled(!hasMore).Clicked()
	})
	return previous, next
}
