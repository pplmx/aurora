package voting

import (
	"database/sql"

	"github.com/pplmx/aurora/internal/ledger"
)

// TransactionManager is the shared transaction port. The alias keeps voting
// use cases and mocks source-compatible after the port moved to ledger.
type TransactionManager = ledger.TransactionManager

// TransactableRepository is a Repository that can additionally scope its
// operations to an open transaction. CastVoteUseCase drives all of its
// mutations through a tx-scoped repository so a failure in any step rolls
// back the whole ballot instead of relying on best-effort compensation
// (UnmarkVoted/DeleteVote). Follows the token module's
// decision-token-tx-scoped-repos pattern.
type TransactableRepository interface {
	Repository
	// WithTx returns a Repository whose operations participate in tx.
	// Implementations without real transaction support (in-memory fakes)
	// return the receiver unchanged.
	WithTx(tx *sql.Tx) Repository
}

// Counts is the dashboard's exact stored voting totals. Votes is the sum of
// the candidates' lifetime tallies, matching the number the desktop has always
// shown.
type Counts struct {
	Candidates int
	Votes      int
	Sessions   int
}

type Repository interface {
	// SaveVote persists a ballot. When SessionID is non-empty, it also claims
	// (session_id, voter_pk) in the same transaction and returns
	// ErrAlreadyVoted if that voter already has a ballot in the session. The
	// voters.has_voted flag remains a lifetime "has ever voted" attribute.
	SaveVote(vote *Vote) error
	GetVotesByCandidate(candidateID string) ([]*Vote, error)
	GetVotesByVoter(voterPK string) ([]*Vote, error)
	// CountVotesBySession returns the number of votes for one candidate in one
	// session. Results must be session-scoped: a candidate can appear in
	// multiple elections, and candidates.vote_count is its global lifetime
	// tally (ISS-299).
	CountVotesBySession(sessionID, candidateID string) (int, error)

	SaveVoter(voter *Voter) error
	GetVoter(pk string) (*Voter, error)
	UpdateVoter(voter *Voter) error
	// ListVoters returns one newest-first page. limit <= 0 returns the
	// remainder from offset, preserving the CLI's full-list form.
	ListVoters(limit, offset int) ([]*Voter, error)

	SaveCandidate(candidate *Candidate) error
	GetCandidate(id string) (*Candidate, error)
	// IncrementCandidateVoteCount atomically adds one to the candidate's
	// vote_count. Implementations MUST be concurrency-safe (e.g. via a
	// conditional UPDATE) so concurrent CastVote calls to the same candidate
	// never lose an increment. The tally is what clients see, so a lost
	// increment is a silently under-counted election. Must return the
	// repository's not-found sentinel if the candidate no longer exists.
	IncrementCandidateVoteCount(candidateID string) error
	// ListCandidates returns one page newest first. limit <= 0 returns the
	// remainder from offset, preserving the CLI's unbounded form.
	ListCandidates(limit, offset int) ([]*Candidate, error)

	SaveSession(session *Session) error
	GetSession(id string) (*Session, error)
	UpdateSession(session *Session) error
	// ListSessions returns one page newest first. limit <= 0 returns the
	// remainder from offset, preserving the CLI's unbounded form.
	ListSessions(limit, offset int) ([]*Session, error)
	// Counts returns exact stored totals without materialising candidates or
	// sessions.
	Counts() (Counts, error)
}
