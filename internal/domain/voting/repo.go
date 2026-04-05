package voting

type Repository interface {
	SaveVote(vote *Vote) error
	GetVote(id string) (*Vote, error)
	GetVotesByCandidate(candidateID string) ([]*Vote, error)
	GetVotesByVoter(voterPK string) ([]*Vote, error)

	SaveVoter(voter *Voter) error
	GetVoter(pk string) (*Voter, error)
	UpdateVoter(voter *Voter) error
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
