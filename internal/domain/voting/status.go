package voting

// Session lifecycle states. Sessions are created as draft (NewSession), moved
// to active by `voting session start` / POST .../start, and permanently closed
// to ended by `session end`. Voting itself is gated by the session's time
// window plus the ended lock (see CastVoteUseCase); `start` is the
// lifecycle/reporting action and is deliberately not a hard pre-vote gate —
// that would contradict the application-layer contract (votes on in-window
// sessions without an explicit start are exercised throughout the app tests).
const (
	StatusDraft  = "draft"
	StatusActive = "active"
	StatusEnded  = "ended"
)

// ValidateSessionTransition enforces the session lifecycle's one hard rule
// (TASK-096, ISS-088): an ENDED session must never be re-activated. The REST
// StartSession handler and the CLI `session start` command both route their
// transitions through this predicate, so a reopened ballot cannot silently
// start accepting votes again after the operator closed it. Every other
// transition is idempotent and allowed (starting an already-active session,
// ending a draft/active/ended session).
func ValidateSessionTransition(from, to string) error {
	if to == StatusActive && from == StatusEnded {
		return ErrSessionAlreadyEnded
	}
	return nil
}
