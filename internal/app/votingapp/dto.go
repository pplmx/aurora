package votingapp

type CastVoteRequest struct {
	VoterPublicKey string
	CandidateID    string
	PrivateKey     string
	SessionID      string
}

type RegisterVoterRequest struct {
	Name string
}

type ListVotersRequest struct {
	Limit  int
	Offset int
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

type ListCandidatesRequest struct {
	Limit  int
	Offset int
}

type ListSessionsRequest struct {
	Limit  int
	Offset int
}

type VoteResponse struct {
	ID          string `json:"id"`
	BlockHeight int64  `json:"block_height"`
}

type VoterResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

type VoterSummaryResponse struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
	HasVoted  bool   `json:"has_voted"` // lifetime history, not current-session eligibility
}

type CandidateResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Party     string `json:"party"`
	Program   string `json:"program"`
	VoteCount int    `json:"vote_count"`
}

type SessionResponse struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Candidates  []string `json:"candidates"`
	// StartTime and EndTime are the window during which a ballot is
	// accepted, in unix seconds. They are the reason a vote is refused, so a
	// caller that lists sessions needs them to explain the window without a
	// second request.
	StartTime int64 `json:"start_time"`
	EndTime   int64 `json:"end_time"`
}

type ResultsRequest struct {
	SessionID string
}

// CandidateResult is one candidate's tally within a session's results.
type CandidateResult struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Party     string `json:"party"`
	VoteCount int    `json:"vote_count"`
}

// ResultsResponse is the per-session tally returned by the results API.
type ResultsResponse struct {
	SessionID  string            `json:"session_id"`
	TotalVotes int               `json:"total_votes"`
	Candidates []CandidateResult `json:"candidates"`
}
