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

type Repository interface {
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
	// TryMarkVoted atomically claims a voter for voting. Implementations
	// MUST be concurrency-safe (e.g. via conditional UPDATE) so that
	// exactly one concurrent caller succeeds; the rest must receive a
	// sentinel error. This is the primitive that closes the TOCTOU
	// double-vote window in CastVoteUseCase.
	TryMarkVoted(publicKey, voteHash string) error
	ListVoters() ([]*Voter, error)

	SaveCandidate(candidate *Candidate) error
	GetCandidate(id string) (*Candidate, error)
	// IncrementCandidateVoteCount atomically adds one to the candidate's
	// vote_count. Implementations MUST be concurrency-safe (e.g. via a
	// conditional UPDATE) so concurrent CastVote calls to the same candidate
	// never lose an increment. The tally is what clients see, so a lost
	// increment is a silently under-counted election. Must return the
	// repository's not-found sentinel if the candidate no longer exists.
	IncrementCandidateVoteCount(candidateID string) error
	ListCandidates() ([]*Candidate, error)

	SaveSession(session *Session) error
	GetSession(id string) (*Session, error)
	UpdateSession(session *Session) error
	ListSessions() ([]*Session, error)
}
