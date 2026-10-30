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
//
// Closed, it is one quiet line rather than a boxed panel: a bordered card with
// a 26px tile and a single sentence weighed as much as the page's first real
// card, and pushed that card down by 68px on every page. The accent bar at its
// left keeps it recognisable as guidance when the text is all a reader has.
func guide(c *ui.Context, open *bool, steps ...string) {
	ui.Column(c).FillWidth().Padding(space2, space3).Gap(space1).Radius(radiusSmall).
		Background(c.Theme().Surface.Alpha(0.55)).
		BorderWidth(0, 0, 0, 2).BorderColor(c.Theme().Accent.Alpha(0.62)).Children(func() {
		ui.Collapsible(c, tr("How this works"), open, func() {
			ui.Column(c).Gap(6).Padding(0, space2, space1, 0).Children(func() {
				for i, step := range steps {
					ui.Row(c).Gap(space2).AlignItems(ui.Start).Children(func() {
						ui.Text(c, fmt.Sprintf(tr("%d."), i+1)).FontSize(textSize(c, 11.5)).FontWeight(600).
							Width(textSize(c, 18)).Shrink(0).TextColor(c.Theme().Accent).
							Font("monospace")
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

// searchField is the one place a page builds its filter control. Every page
// gets the same magnified input, and the accessible label names the collection
// it narrows. MyGo's own placeholder is the untranslated word "Search", so the
// field carries the collection's name instead: the hint says what the query
// narrows, and follows the window's language with the rest of the copy.
func searchField(c *ui.Context, value *string, label string) ui.Element {
	return ui.SearchField(c, value).Label(tr(label)).Placeholder(tr(label))
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
func primaryButton(c *ui.Context, label string, ready ...bool) ui.Element {
	enabled := true
	if len(ready) > 0 {
		enabled = ready[0]
	}
	label = tr(label)
	b := ui.ButtonBase(c).Height(max(buttonHeight, textSize(c, buttonHeight))).
		PaddingX(max(space4+2, textSize(c, 18))).Radius(radiusSmall + 2).
		AlignItems(ui.Center).Label(label).Disabled(!enabled)
	from, to := c.Theme().AccentHover, c.Theme().Accent
	textColor := c.Theme().AccentText
	if !enabled {
		from, to = disabledFill(c), c.Theme().Background.Mix(c.Theme().Surface, 0.42)
		textColor = c.Theme().TextMuted
		b.BorderWidth(1).BorderColor(c.Theme().Border).Shadow(0, 0, 0, 0, ui.Transparent)
	} else {
		switch {
		case b.Pressed():
			from, to = c.Theme().Accent, c.Theme().AccentPressed
		case b.Hovered():
			from, to = c.Theme().AccentHover.Mix(ui.RGB(255, 255, 255), 0.16), c.Theme().AccentHover
		}
		b.Shadow(0, 8, 20, -8, c.Theme().Accent.Alpha(0.46))
		b.InsetShadow(0, 1, 0, 0, ui.RGBA(255, 255, 255, 0.28))
	}
	b.Gradient(from, to, 180)
	pose := ui.Pose{}
	if b.Pressed() {
		pose.Scale = 0.97
	}
	b.AnimatePose(pose, ui.Snappy)
	b.Children(func() {
		ui.Row(c).Gap(space2).AlignItems(ui.Center).Children(func() {
			ui.Text(c, label).FontSize(textSize(c, 13)).FontWeight(600).SingleLine().
				TextColor(textColor)
			if !enabled {
				ui.Text(c, tr("Complete the required fields")).FontSize(textSize(c, 10.5)).
					TextColor(c.Theme().TextMuted).SingleLine()
			}
		})
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
	red := c.Theme().Danger
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
//
// With a subject, the strip leads with the record's own id in the accent and
// keeps the quiet "Acting on" label before it, so the eye lands on what the
// next click will touch. Without one it turns amber and says what to do first:
// the two states must be distinguishable without reading a word of the hint.
func actionSubject(c *ui.Context, id, emptyBadge, emptyHint, copyLabel, copyToast string) {
	id = strings.TrimSpace(id)
	if id == "" {
		ui.Row(c).FillWidth().Gap(space2).AlignItems(ui.Center).Padding(space2, space3).
			Radius(radiusSmall).Background(warningTint(c)).
			BorderWidth(0, 0, 0, 2).BorderColor(c.Theme().Warning).Children(func() {
			statusDot(c, c.Theme().Warning, 6)
			ui.Text(c, tr(emptyBadge)).FontSize(textSize(c, 11)).FontWeight(600).
				TextColor(c.Theme().Warning).SingleLine().Shrink(0)
			ui.Text(c, tr(emptyHint)).FontSize(textSize(c, 11.5)).TextColor(c.Theme().Text).
				Grow(1).Shrink(1).MinWidth(0)
		})
		return
	}
	ui.Row(c).FillWidth().Gap(space2).AlignItems(ui.Center).Padding(space2, space3).
		Radius(radiusSmall).Background(c.Theme().Background.Mix(c.Theme().Surface, 0.35)).
		BorderWidth(0, 0, 0, 2).BorderColor(c.Theme().Accent.Alpha(0.82)).Children(func() {
		ui.Text(c, tr("Acting on")).FontSize(textSize(c, 11)).TextColor(c.Theme().TextMuted).
			SingleLine().Shrink(0)
		ui.Text(c, id).FontSize(textSize(c, 13)).Bold().TextColor(c.Theme().Accent).
			Font("monospace").
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
//
// The row is an underline bar, not a row of pills inside a bordered box: the
// box drew a second frame inside the card, and the active segment carried both
// a tint and an underline, which said "current" twice in two different shapes.
// One accent underline under an accent label is enough, and it matches the
// section headings' own lit marks.
func tabs(c *ui.Context, selected *int, name string, labels ...string) {
	parts := ui.TabsBase(c, selected, len(labels))
	parts.List.Label(tr(name)).FillWidth().Wrap().Gap(2).Padding(0, 0, 0, 0).
		BorderWidth(0, 0, 1, 0).BorderColor(c.Theme().Border).Children(func() {
		for i, label := range labels {
			on := i == *selected
			tab := parts.Tab(i).Padding(textSize(c, 8), textSize(c, 12)).Radius(0).FocusRing(false)
			// The active segment carries no fill at all: the accent label and
			// the underline below it are the mark, and a wash behind them left
			// a square block sitting on a separate rule. Only the pointer gets
			// a hover tint, so the row reads as text on the page's own surface.
			fill := ui.Transparent
			switch {
			case on:
				tab.TextColor(c.Theme().Accent)
			case tab.Hovered():
				tab.TextColor(c.Theme().Text)
				fill = c.Theme().SurfaceHover.Alpha(0.6)
			default:
				tab.TextColor(c.Theme().TextMuted)
			}
			tab.Background(tab.AnimateColor("fill", fill, ui.Animation{Duration: 100 * time.Millisecond}))
			tab.DrawOver(func(p *ui.Painter, r ui.Rect) {
				if on {
					// The underline sits on the row's own hairline, so the
					// active segment reads as joined to the content below it.
					height := textSize(c, 2)
					p.Fill(ui.Rect{
						X: r.X + textSize(c, 4),
						Y: r.Y + r.H - height,
						W: max(0, r.W-2*textSize(c, 4)),
						H: height,
					}, c.Theme().Accent, height/2)
				}
				if tab.FocusVisible() {
					p.FocusRing(r, [4]float32{radiusSmall, radiusSmall, 0, 0})
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
		outline = c.Theme().Accent.Alpha(0.62)
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

// collectionToolbar is the one strip every paged collection leads with.
//
// Before it, a card stacked four separate bands: the search field, a muted
// sentence repeating that search only narrowed the page, a Previous/range/Next
// pager, and then the rows. Measured on the Token page, a one-row collection
// spent 247px on that stack before a single record was read, mostly on the
// repeated sentence and the gap between bands.
//
// The strip puts the three working controls on one line — query, range,
// paging — so the only thing between the card's title and its content is the
// row an operator actually uses. The sentence about paging is kept as the
// search field's accessible description instead of a second visible line; it
// is help, not a fact somebody needs to read on every frame.
//
// It reports the same previous/next a caller acts on, so a collection keeps
// its paging handler unchanged.
func collectionToolbar(c *ui.Context, query *string, searchLabel, hintText, name string,
	offset, count int, hasMore bool,
) (previous, next bool) {
	search := searchField(c, query, searchLabel)
	if hintText != "" {
		// The paging sentence is help, not a fact: it describes what the field
		// does without taking a second visible line on every frame.
		search.Description(tr(hintText))
	}
	// The strip is two lines: the query, then the range with its two buttons.
	//
	// Trying to fit both on one line does not survive the toolkit's sizing: a
	// row is measured from its children's content, and a search field's content
	// is a text editor with a generous minimum width, so the pager was pushed
	// onto its own line at every window size no matter how the field was grown,
	// shrunk or capped. Two deliberate lines replace four stacked bands — the
	// field, the repeated sentence, and a full-width pager — and the field is
	// never crushed to make them share a row.
	// The field and its pager are one control group: the pager is right-aligned
	// directly under the field, in the same column, so the two read as a single
	// block rather than as two sections of the card.
	//
	// The pager stays even on a single page. Its range is how an operator
	// learns that a collection has exactly these N rows rather than more behind
	// a failed read, and its accessible name is the collection's own label;
	// both are content a hidden strip would take away.
	ui.Column(c).FillWidth().Gap(space1 / 2).Children(func() {
		search.FillWidth()
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Spacer(c)
			previous, next = pagerInline(c, name, offset, count, hasMore)
		})
	})
	return previous, next
}

// collectionPager is the pager strip for a collection that has no search field
// of its own: an operation history or a reading list is bounded by its own
// controls, so it only needs the range and the two buttons. It uses the same
// compact shape the toolbar gives a paged, searchable collection, so a page
// does not mix two pager heights.
func collectionPager(c *ui.Context, name string, offset, count int, hasMore bool) (previous, next bool) {
	ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
		ui.Spacer(c)
		previous, next = pagerInline(c, name, offset, count, hasMore)
	})
	return previous, next
}

// pagerInline is pagerControls without its own full-width band: the range and
// the two buttons sit beside the search field, and the accessible group name
// and announcement are unchanged.
func pagerInline(c *ui.Context, name string, offset, count int, hasMore bool) (previous, next bool) {
	rangeLabel := fmt.Sprintf(tr("Entries %d–%d"), offset+1, offset+count)
	pagerLabel := fmt.Sprintf(tr("%s pager, %s"), tr(name), rangeLabel)
	pager := ui.Row(c).AlignItems(ui.Center).Gap(space2).
		Role(ui.RoleGroup).Label(pagerLabel).Shrink(0)
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
			Disabled(offset == 0).Clicked()
		ui.Text(c, rangeLabel).FontSize(textSize(c, 11)).
			TextColor(c.Theme().TextMuted).Shrink(0)
		next = pagerButton(c, "Next").
			Description(fmt.Sprintf(tr("Show the next %s page."), tr(name))).
			Disabled(!hasMore).Clicked()
		_ = pagerLabel
	})
	return previous, next
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
	b := ui.ButtonBase(c).Height(max(24, textSize(c, 24))).PaddingX(max(space2+2, textSize(c, 10))).
		Radius(radiusSmall - 2).AlignItems(ui.Center).Label(label).TextColor(c.Theme().TextMuted)
	// No outline at rest: the buttons sit inside the collection's control
	// group, and a bordered pair there competed with the card's real actions.
	fill, outline := ui.Transparent, ui.Transparent
	switch {
	case b.Pressed():
		fill = c.Theme().SurfacePressed
	case b.Hovered():
		fill = c.Theme().SurfaceHover
		outline = c.Theme().Accent.Alpha(0.5)
	}
	b.Background(b.AnimateColor("fill", fill, ui.Animation{Duration: 100 * time.Millisecond}))
	b.BorderWidth(1).BorderColor(outline)
	b.Children(func() {
		ui.Text(c, label).FontSize(textSize(c, 11)).FontWeight(500).SingleLine()
	})
	return b
}
