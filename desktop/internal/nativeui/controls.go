package nativeui

import (
	"fmt"
	"strings"
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
	ui.Row(c).FillWidth().Padding(space3, space2).Gap(space2).AlignItems(ui.Start).
		Radius(radiusSmall).Background(c.Theme().Background.Mix(c.Theme().Surface, 0.4)).
		BorderWidth(1).BorderColor(c.Theme().Border).Children(func() {
		iconTile(c, infoIcon, ui.Hex(aurora), 26)
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Children(func() {
			ui.Collapsible(c, tr("How this works"), open, func() {
				ui.Column(c).Gap(6).Padding(0, space2, 0, 0).Children(func() {
					for i, step := range steps {
						ui.Row(c).Gap(space2).AlignItems(ui.Start).Children(func() {
							ui.Text(c, fmt.Sprintf(tr("%d."), i+1)).FontSize(textSize(c, 11.5)).FontWeight(600).
								Width(textSize(c, 18)).Shrink(0).TextColor(ui.Hex(aurora))
							ui.Text(c, tr(step)).FontSize(textSize(c, 12)).TextColor(c.Theme().TextMuted).
								Grow(1).Shrink(1).MinWidth(0)
						})
					}
				})
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

// fieldPair puts two short fields on one line inside a card. Labels and
// controls share the row so a form of short values does not become a column of
// full-width inputs; the two columns stay equal so the controls align.
func fieldPair(c *ui.Context, labelA string, valueA *string, labelB string, valueB *string) {
	ui.Row(c).FillWidth().Gap(space3).AlignItems(ui.Start).Children(func() {
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Children(func() {
			field(c, labelA, valueA)
		})
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Children(func() {
			field(c, labelB, valueB)
		})
	})
}

// fieldWithError keeps an error inside the field that caused it. MyGo marks
// the control invalid and exposes the message as its accessible description,
// so a screen reader does not have to infer which red line belongs where.
func fieldWithError(c *ui.Context, label, message string, fn func()) {
	f := ui.Field(c, tr(label), fn)
	attachFieldError(c, f, label, message)
}

// validatedField is the common text-input form of fieldWithError. Editing the
// input clears the validation result immediately, so a corrected value does
// not keep a stale red border until the operator submits the form again.
func validatedField(c *ui.Context, label string, value, message *string) {
	var input ui.Element
	f := ui.Field(c, tr(label), func() {
		input = ui.TextInput(c, value)
	})
	if input.Changed() {
		*message = ""
	}
	attachFieldError(c, f, label, *message)
}

// attachFieldError marks the field invalid and announces the message once.
// Field.Error exposes the text to assistive technology only when its control
// is revisited, while focus may still be on the submit button that started the
// validation.
func attachFieldError(c *ui.Context, field ui.Element, label, message string) {
	last := ui.Local(field, "field-error-announcement", func() string { return "" })
	if message == "" {
		*last = ""
		return
	}
	localized := localizeServiceErrorMessage(message)
	field.Error(localized)
	if *last == localized {
		return
	}
	*last = localized
	c.Announce(fmt.Sprintf("%s: %s: %s", tr("Failed"), tr(label), localized))
}

// The page's three action styles. Every button comes from one of them, so the
// same job looks the same in every card: primary is the one action a form
// exists for, secondary is a read or a supporting write, and danger is
// destructive (still behind its confirmation dialog). They are ButtonBase with
// Aurora's own look rather than the toolkit's default: one height, radius and
// weight across cards, and the hover/press feedback the reference MyGo apps
// use, so a click is acknowledged before the write finishes.
// maxListLimit is the largest page size a list form accepts; the REST surface
// applies the same ceiling to its ?limit.
const maxListLimit = 1000

// namedTable gives every data grid an accessible name. The table role alone
// says what kind of collection is present, not which collection it is.
func namedTable(c *ui.Context, name string, state *ui.ListState, columns []ui.TableColumn, n int, cell func(row, col int)) ui.Element {
	return ui.Table(c, state, columns, n, cell).Label(tr(name))
}

// announceSearchResults tells a screen reader what its new query selected.
// The list repaint alone is visual; the query's editor keeps its own value, so
// without this the operator cannot tell one match from no matches.
func announceSearchResults(c *ui.Context, changed bool, count int) {
	if !changed {
		return
	}
	if count == 1 {
		c.Announce(tr("1 result."))
		return
	}
	c.Announce(fmt.Sprintf(tr("%d results."), count))
}

// primaryButton is the one action a form exists for. It carries the accent as
// a light: a thin top-to-bottom gradient, a soft glow underneath and a dark
// label, so the submit control is the brightest thing in its card.
func primaryButton(c *ui.Context, label string) ui.Element {
	label = tr(label)
	b := ui.ButtonBase(c).Height(max(buttonHeight, textSize(c, buttonHeight))).
		PaddingX(max(space4+2, textSize(c, 18))).Radius(radiusSmall + 2).
		AlignItems(ui.Center).Label(label).TextColor(ui.Hex(onAccent))
	from, to := ui.Hex(auroraHover), ui.Hex(aurora)
	switch {
	case b.Pressed():
		from, to = ui.Hex(aurora), ui.Hex(auroraPressed)
	case b.Hovered():
		from, to = ui.Hex(auroraHover).Mix(ui.RGB(255, 255, 255), 0.16), ui.Hex(auroraHover)
	}
	b.Gradient(from, to, 180)
	b.Shadow(0, 8, 20, -8, ui.Hex(aurora).Alpha(0.5))
	b.InsetShadow(0, 1, 0, 0, ui.RGBA(255, 255, 255, 0.3))
	pose := ui.Pose{}
	if b.Pressed() {
		pose.Scale = 0.97
	}
	b.AnimatePose(pose, ui.Snappy)
	b.Children(func() {
		ui.Text(c, label).FontSize(textSize(c, 13)).FontWeight(600).SingleLine().
			TextColor(ui.Hex(onAccent))
	})
	return b
}

// secondaryButton is the supporting action: a read, a cancel, or the second
// write in a card. It keeps the surface and gains an accent-tinted outline on
// hover instead of filling with colour, so it never competes with the primary.
func secondaryButton(c *ui.Context, label string) ui.Element {
	return glassButton(c, label, c.Theme().Text)
}

func dangerButton(c *ui.Context, label string) ui.Element {
	red := ui.Hex(auroraRed)
	return filledButton(c, label, red,
		red.Mix(ui.RGB(255, 255, 255), 0.14), red.Mix(ui.RGB(0, 0, 0), 0.16),
		c.Theme().Background, red.Alpha(0.35))
}

// quietButton is the small, low-emphasis action used inside a card: copying a
// value, for instance, which should not compete with the card's main button.
func quietButton(c *ui.Context, label string) ui.Element {
	label = tr(label)
	b := ui.ButtonBase(c).Height(max(28, textSize(c, 28))).PaddingX(textSize(c, 10)).Radius(radiusSmall).
		AlignItems(ui.Center).Label(label).TextColor(c.Theme().TextMuted)
	fill := ui.Transparent
	if b.Hovered() || b.Pressed() {
		fill = c.Theme().SurfaceHover
	}
	b.Background(b.AnimateColor("fill", fill, ui.Animation{Duration: 100 * time.Millisecond}))
	b.Children(func() {
		ui.Text(c, label).FontSize(textSize(c, 12)).FontWeight(500).SingleLine()
	})
	return b
}

// actionSubject names the record the current action card acts on. Token's
// write column established this pattern; sharing it keeps a page from making
// the operator infer a subject from a form field that may have changed.
func actionSubject(c *ui.Context, id, emptyBadge, emptyHint, copyLabel, copyToast string) {
	id = strings.TrimSpace(id)
	if id == "" {
		ui.Row(c).FillWidth().Gap(space2).AlignItems(ui.Center).Padding(space2, space3).
			Radius(radiusSmall).Background(ui.Hex(warningTint)).
			BorderWidth(1).BorderColor(ui.Hex(auroraAmber).Alpha(0.45)).Children(func() {
			statusDot(c, ui.Hex(auroraAmber), 6)
			ui.Text(c, tr(emptyBadge)).FontSize(textSize(c, 11)).FontWeight(600).
				TextColor(ui.Hex(auroraAmber)).SingleLine().Shrink(0)
			ui.Text(c, tr(emptyHint)).FontSize(textSize(c, 11.5)).TextColor(c.Theme().Text).
				Grow(1).Shrink(1).MinWidth(0)
		})
		return
	}
	ui.Row(c).FillWidth().Gap(space2).AlignItems(ui.Center).Padding(space2, space3).
		Radius(radiusSmall).Background(c.Theme().Background.Mix(c.Theme().Surface, 0.45)).
		BorderWidth(1).BorderColor(c.Theme().Border).Children(func() {
		ui.Text(c, tr("Acting on")).FontSize(textSize(c, 11)).TextColor(c.Theme().TextMuted).
			SingleLine().Shrink(0)
		ui.Text(c, id).FontSize(textSize(c, 13)).Bold().TextColor(ui.Hex(aurora)).
			SingleLine().Ellipsis("…").Tooltip(id)
		if quietButton(c, copyLabel).Clicked() {
			c.Services().WriteClipboard(id)
			c.Toast(tr(copyToast))
		}
	})
}

// tabs draws a keyboard-operable tab row that wraps instead of pushing later
// tabs past the window edge. MyGo's stock Tabs is deliberately non-shrinking;
// the five Token write tabs already crossed the right edge at 2x text.
func tabs(c *ui.Context, selected *int, name string, labels ...string) {
	parts := ui.TabsBase(c, selected, len(labels))
	parts.List.Label(tr(name)).FillWidth().Wrap().Gap(2).Padding(4).Radius(radiusSmall + 2).
		Background(c.Theme().Background.Mix(c.Theme().Surface, 0.35)).
		BorderWidth(1).BorderColor(c.Theme().Border).Children(func() {
		for i, label := range labels {
			on := i == *selected
			tab := parts.Tab(i).Padding(textSize(c, 7), textSize(c, 14)).Radius(radiusSmall).FocusRing(false)
			fill := ui.Transparent
			switch {
			case on:
				tab.TextColor(c.Theme().Text)
				fill = c.Theme().Selection
				tab.BorderWidth(1).BorderColor(ui.Hex(aurora).Alpha(0.35))
			case tab.Hovered():
				tab.TextColor(c.Theme().Text)
				fill = c.Theme().SurfaceHover
			default:
				tab.TextColor(c.Theme().TextMuted)
			}
			tab.Background(tab.AnimateColor("fill", fill, ui.Animation{Duration: 100 * time.Millisecond}))
			tab.DrawOver(func(p *ui.Painter, r ui.Rect) {
				if on {
					// The active segment carries a 2px accent underline inside
					// its pill: the tint identifies the segment, the line gives
					// it the same "current tab" cue as the rest of the window.
					inset := textSize(c, 6)
					height := textSize(c, 2)
					p.Fill(ui.Rect{
						X: r.X + inset,
						Y: r.Y + r.H - height - 1,
						W: r.W - 2*inset,
						H: height,
					}, ui.Hex(aurora), height/2)
				}
				if tab.FocusVisible() {
					p.FocusRing(r, [4]float32{radiusSmall, radiusSmall, radiusSmall, radiusSmall})
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

// glassButton is the shared supporting-button shape: the card's own surface,
// a hairline border that turns accent on hover, and the button's native press
// and focus feedback. The caller supplies only the label colour.
func glassButton(c *ui.Context, label string, textColor ui.Color) ui.Element {
	label = tr(label)
	b := ui.ButtonBase(c).Height(max(buttonHeight, textSize(c, buttonHeight))).
		PaddingX(max(space4-2, textSize(c, 14))).Radius(radiusSmall + 2).
		AlignItems(ui.Center).Label(label).TextColor(textColor)
	from, to := c.Theme().Surface, c.Theme().Surface.Mix(c.Theme().Background, 0.2)
	outline := c.Theme().Border
	switch {
	case b.Pressed():
		from, to = c.Theme().SurfacePressed, c.Theme().SurfaceHover
	case b.Hovered():
		from, to = c.Theme().SurfaceHover, c.Theme().Surface
		outline = ui.Hex(aurora).Alpha(0.55)
	}
	b.Gradient(from, to, 180)
	b.BorderWidth(1).BorderColor(outline)
	pose := ui.Pose{}
	if b.Pressed() {
		pose.Scale = 0.97
	}
	b.AnimatePose(pose, ui.Snappy)
	b.Children(func() {
		ui.Text(c, label).FontSize(textSize(c, 13)).FontWeight(600).SingleLine().TextColor(textColor)
	})
	return b
}

// filledButton is a solid button with two states and an optional glow, used
// for the destructive action. The label colour is passed in because red needs
// the dark window ink while the accent needs its own on-accent tone.
func filledButton(c *ui.Context, label string, fill, hover, pressed, textColor ui.Color, glow ui.Color) ui.Element {
	label = tr(label)
	b := ui.ButtonBase(c).Height(max(buttonHeight, textSize(c, buttonHeight))).
		PaddingX(max(space4-2, textSize(c, 14))).Radius(radiusSmall + 2).
		AlignItems(ui.Center).Label(label).TextColor(textColor)
	state := fill
	switch {
	case b.Pressed():
		state = pressed
	case b.Hovered():
		state = hover
	}
	b.Background(b.AnimateColor("fill", state, ui.Animation{Duration: 120 * time.Millisecond}))
	b.Shadow(0, 8, 20, -10, glow)
	b.InsetShadow(0, 1, 0, 0, ui.RGBA(255, 255, 255, 0.18))
	pose := ui.Pose{}
	if b.Pressed() {
		pose.Scale = 0.97
	}
	b.AnimatePose(pose, ui.Snappy)
	b.Children(func() {
		ui.Text(c, label).FontSize(textSize(c, 13)).FontWeight(600).SingleLine().TextColor(textColor)
	})
	return b
}

// fieldHint builds a labelled input with a hint under it, for inputs whose
// accepted range is not obvious from the label.
func fieldHint(c *ui.Context, label, hintText string, value *string) {
	ui.Field(c, tr(label), func() {
		ui.Column(c).Gap(2).Children(func() {
			ui.TextInput(c, value).Description(tr(hintText))
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
			textAreaControl(c, value).Description(tr(hintText))
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
func listPager(c *ui.Context, name string, offset, count int, hasMore bool) (previous, next bool) {
	rangeLabel := fmt.Sprintf(tr("Entries %d–%d"), offset+1, offset+count)
	return pagerControls(c, name, offset, rangeLabel, offset == 0, !hasMore)
}

// pagerControls draws one named Previous/range/Next group. Most collections
// only know whether another row exists, while the ledger also knows its exact
// total, so the range text and disabled states are supplied by the caller.
func pagerControls(c *ui.Context, name string, offset int, rangeLabel string, previousDisabled, nextDisabled bool) (previous, next bool) {
	pagerLabel := fmt.Sprintf(tr("%s pager, %s"), tr(name), rangeLabel)
	pager := ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).
		Role(ui.RoleGroup).Label(pagerLabel)
	// A pager move replaces the rows while focus stays on its button. Announce
	// the range once the offset has changed, but not on the first frame, a
	// repaint, or a failed move that kept the old page.
	announcedOffset := ui.Local(pager, "announced-offset", func() int { return offset })
	if *announcedOffset != offset {
		*announcedOffset = offset
		c.Announce(rangeLabel)
	}
	pager.Children(func() {
		previous = pagerButton(c, "Previous").
			Description(fmt.Sprintf(tr("Show the previous %s page."), tr(name))).
			Disabled(previousDisabled).Clicked()
		ui.Spacer(c)
		ui.Text(c, rangeLabel).
			FontSize(textSize(c, 11)).TextColor(c.Theme().TextMuted)
		ui.Spacer(c)
		next = pagerButton(c, "Next").
			Description(fmt.Sprintf(tr("Show the next %s page."), tr(name))).
			Disabled(nextDisabled).Clicked()
	})
	return previous, next
}

// pagerButton is the compact control that moves a collection one page. It is
// quieter than a card action: the range between the two buttons is the fact
// the operator is reading, and two full-height buttons outweighed it.
func pagerButton(c *ui.Context, label string) ui.Element {
	label = tr(label)
	b := ui.ButtonBase(c).Height(max(28, textSize(c, 28))).PaddingX(max(space3, textSize(c, 12))).
		Radius(radiusSmall).AlignItems(ui.Center).Label(label).TextColor(c.Theme().TextMuted)
	fill, outline := ui.Transparent, c.Theme().Border
	switch {
	case b.Pressed():
		fill = c.Theme().SurfacePressed
	case b.Hovered():
		fill = c.Theme().Surface
		outline = ui.Hex(aurora).Alpha(0.45)
	}
	b.Background(b.AnimateColor("fill", fill, ui.Animation{Duration: 100 * time.Millisecond}))
	b.BorderWidth(1).BorderColor(outline)
	b.Children(func() {
		ui.Text(c, label).FontSize(textSize(c, 11.5)).FontWeight(500).SingleLine()
	})
	return b
}
