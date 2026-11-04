package nativeui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Surfacing a stuck audit outbox.
//
// The outbox parks audit events whose delivery failed and retries them in the
// background. A backlog is therefore a real condition — the chain's own record is
// behind what the modules wrote — but it was reported in exactly one place: a line
// inside the copied diagnostics, which only appears if the operator opens
// Settings and clicks Copy. Nobody opens Settings to discover a problem.
//
// The reference apps built on MyGo put this kind of condition where it is seen
// without being looked for: the sidebar says which session needs you. The same
// rule applied here: a backlog appears at the foot of the sidebar, where the
// ledger's own health already lives, and disappears when it drains. It is not a
// dialog and not a toast, because a backlog can last for minutes and neither is
// the right shape for a state that persists.

// outboxWarnAge is when a pending event stops looking like normal retry traffic
// and starts looking stuck. The drainer retries on its own schedule, so a short
// backlog is expected during a network blip; an event still parked after a few
// minutes means the sink is unreachable rather than slow.
const outboxWarnAge = 5 * time.Minute

// PendingAuditLimit caps how many parked copies the ledger lists. The footer
// carries the true count, so a backlog longer than this is still reported
// accurately; the list exists to be read, not to be exhaustive.
const PendingAuditLimit = 20

// outboxReadInterval is how often the sidebar re-reads the backlog. It matches
// the live-refresh tick, which is already slow on purpose: a backlog is a
// condition that lasts minutes, so a value that is a few seconds old says the
// same thing as a fresh one.
const outboxReadInterval = liveRefreshInterval

// OutboxState is what the sidebar footer needs to report a backlog, resolved
// once per frame. ok is false when the shell could not read it (no database yet,
// or a query error), which is different from an empty outbox: an empty one is
// healthy and says nothing.
type OutboxState struct {
	Pending int
	Oldest  time.Duration
	OK      bool
}

// PendingEvent is one parked audit event as the ledger page lists it. The
// aggregate is the operation the copy belongs to; the attempts and next-try time
// are the drainer's own backoff, so a row whose attempts keep climbing is one the
// destination keeps refusing rather than one nobody has tried.
type PendingEvent struct {
	EventType string
	Module    string
	Aggregate string
	ParkedAt  time.Time
	Attempts  int
	NextTryAt time.Time
}

// readOutbox returns the backlog the sidebar draws. The footer renders on every
// frame, so the value is cached and re-read on the live-refresh tick rather than
// on each paint: three of the four facts it needs are storage reads, and an idle
// window repaints for a hover or a caret blink just as often as it does for new
// data.
func (a *App) readOutbox() OutboxState {
	if a.actions.OutboxPending == nil {
		return OutboxState{}
	}
	return a.outboxState
}

// refreshOutbox re-reads the backlog and caches it. It runs at the start of a
// frame when the cached value is due, so the first frame after a launch has a
// number rather than a blank.
func (a *App) refreshOutbox(now time.Time) {
	if a.actions.OutboxPending == nil {
		return
	}
	if !a.outboxRead.IsZero() && now.Sub(a.outboxRead) < outboxReadInterval {
		return
	}
	a.outboxRead = now
	a.outboxState = a.readOutboxStorage()
}

// readOutboxStorage is the actual read, isolated so refreshOutbox can cache it.
func (a *App) readOutboxStorage() OutboxState {
	pending, ok := a.actions.OutboxPending()
	if !ok {
		return OutboxState{}
	}
	state := OutboxState{Pending: pending, OK: true}
	// The age is only meaningful when something is parked; asking for it on an
	// empty outbox is a query whose answer cannot be shown.
	if pending > 0 && a.actions.OutboxOldest != nil {
		if age, ageOK := a.actions.OutboxOldest(); ageOK {
			state.Oldest = age
		}
	}
	return state
}

// followOutbox opens the ledger narrowed to the module a backlog belongs to.
//
// What the page can and cannot show is worth being precise about. A parked event
// is an audit *copy* of an operation whose block was already written: the module
// wrote its record to the chain, and the separate delivery of that record to the
// audit store failed. So the ledger holds the operation, and it does not hold the
// backlog — pending_events is its own table. Opening the ledger therefore answers
// "what was being written when the sink started refusing", not "which rows are
// stuck"; the footer's count remains the only place the backlog itself is
// reported, and the copied diagnostics carries the age a support thread needs.
//
// One parked module becomes the ledger's search term because that is the useful
// question the page can answer. Several are left unfiltered: no single term names
// them all, and searching for one would hide the rest.
func (a *App) followOutbox() {
	search := ""
	if a.actions.OutboxModules != nil {
		if modules := a.actions.OutboxModules(); len(modules) == 1 {
			// The search field shows a word the reader is looking at, so it
			// carries the module's own name in the window's language rather than
			// the id the chain stores. Both find the same records — the storage
			// search matches the localized vocabulary as well as the raw ids —
			// but "代币" in a Chinese window is the word the record beside it
			// says, and "token" there would look like a term from another app.
			search = recordModuleName(modules[0])
		}
	}
	a.showLedgerSearch(search)
}

// outboxLine is the sentence the footer shows, and its severity. A backlog that
// the drainer is still moving through is a warning; one whose oldest event has
// outlived the retry window is a failure, because the audit chain is then
// behind the modules with nobody addressing it.
func outboxLine(state OutboxState) (string, bool) {
	if !state.OK || state.Pending <= 0 {
		return "", false
	}
	stuck := state.Oldest >= outboxWarnAge
	switch {
	case stuck:
		return fmt.Sprintf(tr("Audit outbox: %d pending, oldest %s — not draining"),
			state.Pending, state.Oldest.Round(time.Second)), true
	case state.Oldest > 0:
		return fmt.Sprintf(tr("Audit outbox: %d pending, oldest %s"),
			state.Pending, state.Oldest.Round(time.Second)), false
	default:
		return fmt.Sprintf(tr("Audit outbox: %d pending"), state.Pending), false
	}
}

// pendingEventRow draws one parked audit event. It reads as a record like the
// blocks above it: the module and verb first, the operation it belongs to, then
// the facts that say whether the drainer is making progress. The payload is
// deliberately absent — the block for the same operation is on the page with its
// exact bytes, and this list exists to say *which* copies are stuck.
func pendingEventRow(c *ui.Context, event PendingEvent, now time.Time) {
	tone := pendingEventTone(c, event, now)
	ui.Column(c).FillWidth().Padding(space2+1, space3+2).Gap(2).Radius(radiusSmall).
		Background(c.Theme().Background.Mix(c.Theme().Surface, 0.3)).
		BorderWidth(0, 0, 0, 2).BorderColor(tone.Alpha(0.7)).Children(func() {
		ui.Row(c).FillWidth().Wrap().Gap(space3).AlignItems(ui.Baseline).Children(func() {
			ui.Text(c, pendingEventTitle(event)).FontSize(textSize(c, 12)).FontWeight(600).
				TextColor(c.Theme().Text).Grow(1).Shrink(1).MinWidth(0)
			ui.Text(c, event.ParkedAt.Local().Format(localTimeLayout)).FontSize(textSize(c, 10.5)).
				TextColor(c.Theme().TextMuted).Shrink(0).SingleLine()
		})
		ui.Row(c).FillWidth().Wrap().Gap(space4).AlignItems(ui.Start).Children(func() {
			pendingFact(c, "Operation", event.Aggregate, c.Theme().TextMuted)
			pendingFact(c, "Attempts", strconv.Itoa(event.Attempts), tone)
			if !event.NextTryAt.IsZero() {
				// "Due in" rather than an absolute time: the reader's question is
				// whether the loop is still trying, and a future clock time makes
				// them do the subtraction against now.
				pendingFact(c, "Next try", nextTryText(event.NextTryAt, now), c.Theme().TextMuted)
			}
		})
	})
}

// pendingFact is one labelled value under a parked event. It reads like the facts
// on a block record above it: a small label over a value, so a column of events
// scans by label.
func pendingFact(c *ui.Context, label, value string, tone ui.Color) {
	ui.Column(c).Gap(0).Shrink(0).MaxWidth(textSize(c, 220)).Children(func() {
		ui.Text(c, tr(label)).FontSize(textSize(c, 9.5)).FontWeight(600).
			LetterSpacing(0.4).TextColor(c.Theme().TextMuted).SingleLine()
		ui.Text(c, value).FontSize(textSize(c, 11.5)).TextColor(tone).Tooltip(value)
	})
}

// pendingEventTitle is the line's headline: the module in the window's language
// and the verb the event type names, which is what distinguishes two parked
// copies of the same module.
func pendingEventTitle(event PendingEvent) string {
	module := pendingModuleName(event.Module)
	if verb := pendingEventVerb(event.EventType); verb != "" {
		return module + " · " + verb
	}
	return module
}

// pendingModuleName resolves the module word the way the ledger records do, so
// the list and the blocks above it call the same subsystem the same thing.
func pendingModuleName(module string) string {
	if strings.TrimSpace(module) == "" {
		return "—"
	}
	return recordModuleName(module)
}

// pendingEventVerb takes the verb out of an event type like "token.mint". The
// event type is a storage identifier, not copy, so the verb is resolved through
// the same record vocabulary the blocks use.
func pendingEventVerb(eventType string) string {
	_, verb, found := strings.Cut(eventType, ".")
	if !found || verb == "" {
		return ""
	}
	return recordActionName("", verb)
}

// pendingEventTone says whether this parked copy is moving. A row still inside
// the retry window is a warning; one whose attempts have climbed past the point
// where backoff keeps doubling means the destination keeps refusing it.
func pendingEventTone(c *ui.Context, event PendingEvent, now time.Time) ui.Color {
	if event.Attempts >= pendingAttemptsStuck || (!event.ParkedAt.IsZero() && now.Sub(event.ParkedAt) >= outboxWarnAge) {
		return c.Theme().Danger
	}
	return c.Theme().Warning
}

// pendingAttemptsStuck is where a parked copy stops looking like a transient
// failure. The drainer backs off geometrically, so five failed deliveries span
// minutes of trying rather than a single blip.
const pendingAttemptsStuck = 5

// nextTryText renders when the drainer may try again, as a quantity the reader
// can act on.
func nextTryText(at, now time.Time) string {
	wait := at.Sub(now)
	if wait <= 0 {
		return tr("due")
	}
	return fmt.Sprintf(tr("in %s"), wait.Round(time.Second))
}

// pendingAuditCard lists the parked audit copies. The footer says how many are
// waiting; this says which, because "four pending" is not something an operator
// can act on — a module, a verb and an attempt count are.
//
// The card appears only when there is something to list, and it sits under the
// ledger rather than above it: the ledger is the page's subject, and these copies
// belong to operations the blocks already show.
func (a *App) pendingAuditCard(c *ui.Context) {
	if a.actions.OutboxList == nil {
		return
	}
	events := a.blockchain.pending
	if len(events) == 0 {
		return
	}
	now := time.Now()
	if a.liveNow != nil {
		now = a.liveNow()
	}
	panel := card(c, func() {
		sectionHeading(c, "Waiting to be delivered",
			"Audit copies of blocks that are already on the chain; the drainer retries them until they land.")
		ui.Column(c).FillWidth().Gap(space2).Role(ui.RoleList).Label(tr("Pending audit events")).Children(func() {
			for _, event := range events {
				pendingEventRow(c, event, now)
			}
		})
	})
	panel.Label(tr("Waiting to be delivered"))
}
