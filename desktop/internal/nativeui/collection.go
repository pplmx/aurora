package nativeui

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

// The collection list.
//
// Every collection in the window — draws, tokens, NFTs, voters, candidates,
// sessions, oracle sources, readings, health, operation and transfer history —
// is a list of records. Each one used to carry two renderings of the same data:
// a `ui.Table` with named columns for normal text, and a hand-written record
// loop that took over at 1.75x text because the columns no longer fit. Twelve
// collections meant twelve pairs of functions that could disagree about which
// facts a record has, and several of them did: the table showed a truncated id
// while its record showed the full one, and a field added to one was not
// necessarily added to the other.
//
// The table was also carrying capability nobody used: no column in this window
// sets Sortable, Width/MaxWidth ordering or RowHeader, and `ListState` sorting
// and cell selection are never touched. What the tables actually provided was
// three shallow columns and a selection highlight.
//
// collectionList replaces both. The caller projects its row type into a
// collectionRecord — title, optional facts, optional fields, optional action —
// and the list draws the same card at every text size, selects through one
// path, and keeps one accessible label per record. What changes with enlarged
// text is only how the facts wrap, not which presentation is used.

// collectionFact is one labelled value shown beside a record's title. A fact
// with a tone is a short state ("Yes", "enabled"); one without is a value.
type collectionFact struct {
	label string
	value string
	// tone, when set, colours the value: a state word that carries a verdict
	// hue. It defaults to body ink.
	tone ui.Color
}

// collectionRecord is one row of a collection, as the list draws it.
type collectionRecord struct {
	// title is the record's identity: a token name, an id, a session title.
	// It is also what the selection and the accessible label name.
	title string
	// facts are short values drawn beside the title. They wrap onto the next
	// line rather than being clipped.
	facts []collectionFact
	// fields are labelled values stacked under the title, for anything longer
	// than a short fact: a public key, a window, a supply.
	fields []collectionField
	// action, when non-nil, draws a control at the record's foot. It is used by
	// rows that offer something other than selection, such as "use this draw".
	// It receives the build context because it draws while the record does.
	action func(*ui.Context)
}

// collectionField is one labelled value inside a record.
type collectionField struct {
	label string
	value string
}

// collectionList draws rows as records and reports which one was chosen.
//
// selected is the index of the chosen record, or -1. onChoose runs once with
// the chosen index; the list owns no state, so a caller keeps its existing
// "choose" function. A caller that does not want selection passes nil and the
// records are drawn without the button behaviour.
func collectionList(c *ui.Context, name string, rows []collectionRecord, selected int, onChoose func(int)) {
	ui.Column(c).FillWidth().Gap(space2).Role(ui.RoleList).Label(tr(name)).Children(func() {
		for i, row := range rows {
			i, row := i, row
			if onChoose == nil {
				ui.Column(c).FillWidth().Role(ui.RoleListItem).Children(func() {
					recordPanel(c, func() { collectionRecordBody(c, row) })
				})
				continue
			}
			isSelected := i == selected
			record := selectableRecordPanel(c, isSelected,
				collectionChoiceLabel(row, isSelected), func() {
					collectionRecordBody(c, row)
				})
			if record.Clicked() {
				// Choosing a record fills the page's forms and cards with its
				// values, which a screen reader does not otherwise notice: the
				// button keeps focus and only its own label changes. Say which
				// record became the page's subject.
				c.Announce(fmt.Sprintf(tr("Selected %s"), collectionRecordName(row)))
				onChoose(i)
			}
		}
	})
}

// collectionChoiceLabel is what assistive technology hears for a selectable
// record. It states the action and the record's facts rather than repeating the
// title the record draws itself, so choosing a row does not read as "Selected
// Gold, Gold".
func collectionChoiceLabel(row collectionRecord, selected bool) string {
	name := collectionRecordName(row)
	if selected {
		return fmt.Sprintf(tr("Selected %s"), name)
	}
	return fmt.Sprintf(tr("Select %s"), name)
}

// collectionRecordBody draws the inside of one record: its title, the short
// facts beside it, the fields under it, and the row's own action if it has one.
//
// The title and the facts share a baseline rather than a bounding box: a
// label-over-value fact is taller than a single-line title, and aligning their
// bottoms left the title floating at the label's height instead of beside the
// value it belongs to.
func collectionRecordBody(c *ui.Context, row collectionRecord) {
	ui.Row(c).FillWidth().Wrap().Gap(space4).AlignItems(ui.Baseline).Children(func() {
		// The title wraps rather than clipping: it is the record's identity,
		// and the values that reach it (a draw id, a public key) are exactly
		// the long ones a one-line cell used to cut to a prefix.
		ui.Text(c, row.title).FontSize(textSize(c, 15)).Bold().
			TextColor(c.Theme().Text).Grow(1).Shrink(1).MinWidth(0)
		for _, fact := range row.facts {
			if fact.value == "" || fact.value == "—" {
				// A fact with nothing in it is noise: it takes a slot, shows a
				// label, and says the store has no value for it, which the
				// record looks better without.
				continue
			}
			collectionFactView(c, fact)
		}
	})
	if len(row.fields) > 0 {
		// Fields sit tighter than the title row above them: they are one block
		// of supporting detail, not another section of the record. The gap is
		// a fixed pixel value rather than a layout constant, because these
		// constants scale with the text setting and would open a visible hole
		// between a title and its own id at 2x.
		ui.Column(c).FillWidth().Gap(2).Children(func() {
			for _, field := range row.fields {
				collectionFieldView(c, field)
			}
		})
	}
	if row.action != nil {
		ui.Row(c).Padding(space1, 0, 0, 0).Children(func() { row.action(c) })
	}
}

// collectionFieldView draws one labelled value inside a record, with the label
// and its value on the same line when both fit. Stacking them cost a line per
// field: a token record held two empty-looking gaps for "ID" and its value.
//
// The value wraps rather than clipping. A field is where the long values live —
// public keys, windows, a full id — and those are exactly what a one-line cell
// used to cut to a prefix, which is the defect this list replaced.
func collectionFieldView(c *ui.Context, field collectionField) {
	label := tr(field.label)
	ui.Row(c).FillWidth().Wrap().Gap(space2).AlignItems(ui.Start).Children(func() {
		ui.Text(c, label).FontSize(textSize(c, 10)).
			TextColor(c.Theme().TextMuted).Shrink(0).SingleLine()
		ui.Text(c, field.value).FontSize(textSize(c, 11.5)).
			TextColor(c.Theme().Text).Grow(1).Shrink(1).MinWidth(0)
	})
}

// collectionFactView draws one short fact: its label above its value, so a row
// of facts scans by label and reads by value.
func collectionFactView(c *ui.Context, fact collectionFact) {
	tone := fact.tone
	if tone == (ui.Color{}) {
		tone = c.Theme().Text
	}
	ui.Column(c).Gap(1).Shrink(0).Children(func() {
		ui.Text(c, tr(fact.label)).FontSize(textSize(c, 10)).
			TextColor(c.Theme().TextMuted).SingleLine()
		ui.Text(c, fact.value).FontSize(textSize(c, 14)).FontWeight(600).
			TextColor(tone).SingleLine().Tooltip(fact.value)
	})
}

// collectionRecordName is what the list calls a record: its title, or the
// module's word for an untitled row so the selection is never anonymous.
func collectionRecordName(row collectionRecord) string {
	if row.title != "" {
		return row.title
	}
	return "—"
}

// collectionRecords builds the display records for a typed slice. It is the
// one place that turns a row into a record, so a caller only supplies the
// projection and cannot accidentally give the table and the records different
// facts.
func collectionRecords[T any](rows []T, project func(T) collectionRecord) []collectionRecord {
	out := make([]collectionRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, project(row))
	}
	return out
}
