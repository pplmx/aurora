package voting

type CastVoteRequest struct {
	VoterPublicKey string
	CandidateID    string
	PrivateKey     string
}

type RegisterVoterRequest struct {
	Name string
}

type RegisterCandidateRequest struct {
	Name    string
	Party   string
	Program string
}

type CreateSessionRequest struct {
	Title        string
	Description  string
	CandidateIDs []string
	StartTime    int64
	EndTime      int64
}

type VoteResponse struct {
	ID          string
	BlockHeight int64
}

type VoterResponse struct {
	ID         string
	Name       string
	PublicKey  string
	PrivateKey string
}

type CandidateResponse struct {
	ID        string
	Name      string
	Party     string
	Program   string
	VoteCount int
}

type SessionResponse struct {
	ID          string
	Title       string
	Description string
	Status      string
	Candidates  []string
}
