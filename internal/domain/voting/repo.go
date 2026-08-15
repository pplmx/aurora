package voting

import "database/sql"

// TransactionManager abstracts the infra transaction runner so use cases can
// group several repository writes (voter claim + vote row + tally increment)
// into one atomic unit. Implementations commit on nil and roll back on error.
type TransactionManager interface {
	WithTransaction(fn func(tx *sql.Tx) error) error
}

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
	GetVote(id string) (*Vote, error)
	GetVotesByCandidate(candidateID string) ([]*Vote, error)
	GetVotesByVoter(voterPK string) ([]*Vote, error)
	DeleteVote(id string) error

	SaveVoter(voter *Voter) error
	GetVoter(pk string) (*Voter, error)
	UpdateVoter(voter *Voter) error
	// TryMarkVoted atomically claims a voter for voting. Implementations
	// MUST be concurrency-safe (e.g. via conditional UPDATE) so that
	// exactly one concurrent caller succeeds; the rest must receive a
	// sentinel error. This is the primitive that closes the TOCTOU
	// double-vote window in CastVoteUseCase.
	TryMarkVoted(publicKey, voteHash string) error
	// UnmarkVoted resets the voter's has_voted flag. Historically the
	// best-effort rollback for a failed CastVote; CastVote now runs in a
	// real transaction, so this primitive is retained for administrative /
	// corrective flows only.
	UnmarkVoted(publicKey string) error
	ListVoters() ([]*Voter, error)

	SaveCandidate(candidate *Candidate) error
	GetCandidate(id string) (*Candidate, error)
	UpdateCandidate(candidate *Candidate) error
	// IncrementCandidateVoteCount atomically adds one to the candidate's
	// vote_count. Implementations MUST be concurrency-safe (e.g. via a
	// conditional UPDATE) so concurrent CastVote calls to the same candidate
	// never lose an increment. The tally is what clients see, so a lost
	// increment is a silently under-counted election. Must return the
	// repository's not-found sentinel if the candidate no longer exists.
	IncrementCandidateVoteCount(candidateID string) error
	DeleteCandidate(id string) error
	ListCandidates() ([]*Candidate, error)

	SaveSession(session *Session) error
	GetSession(id string) (*Session, error)
	UpdateSession(session *Session) error
	ListSessions() ([]*Session, error)
}
