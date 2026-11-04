package nativeui

import (
	"fmt"
	"time"
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

// OutboxState is what the sidebar footer needs to report a backlog, resolved
// once per frame. ok is false when the shell could not read it (no database yet,
// or a query error), which is different from an empty outbox: an empty one is
// healthy and says nothing.
type OutboxState struct {
	Pending int
	Oldest  time.Duration
	OK      bool
}

// readOutbox asks the shell for the current backlog. It is a method rather than a
// closure so a test can render a backlog without a database.
func (a *App) readOutbox() OutboxState {
	if a.actions.OutboxPending == nil {
		return OutboxState{}
	}
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
