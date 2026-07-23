package voting

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
	ListVoters() ([]*Voter, error)

	SaveCandidate(candidate *Candidate) error
	GetCandidate(id string) (*Candidate, error)
	UpdateCandidate(candidate *Candidate) error
	DeleteCandidate(id string) error
	ListCandidates() ([]*Candidate, error)

	SaveSession(session *Session) error
	GetSession(id string) (*Session, error)
	UpdateSession(session *Session) error
	ListSessions() ([]*Session, error)
}
