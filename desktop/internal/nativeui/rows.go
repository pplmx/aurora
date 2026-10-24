package nativeui

// The row types a view renders. They are plain data: the adapter that
// implements Services maps domain entities into them, so the view layer never
// depends on a domain package's shape and a domain change cannot ripple into
// layout code.

// LedgerStatus is the audit chain's verified state.
type LedgerStatus struct {
	Valid  bool
	Height int
	Blocks int
}

// VoterRow is one registered voter.
type VoterRow struct {
	PublicKey string
	Name      string
	HasVoted  bool
}

// CandidateRow is one registered candidate.
type CandidateRow struct {
	ID        string
	Name      string
	Party     string
	Program   string
	VoteCount int
}

// SessionRow is one voting session.
type SessionRow struct {
	ID         string
	Title      string
	Status     string
	Candidates []string
	StartTime  int64
	EndTime    int64
}

// ResultRow is one candidate's tally in one session.
type ResultRow struct {
	CandidateID string
	Name        string
	Votes       int
}

// TokenRow is a token's metadata.
type TokenRow struct {
	ID          string
	Name        string
	Symbol      string
	TotalSupply string
	Decimals    int
	Owner       string
}

// TransferRow is one transfer in a token's history.
type TransferRow struct {
	From   string
	To     string
	Amount string
	Nonce  uint64
}

// SourceRow is one oracle data source.
type SourceRow struct {
	ID         string
	Name       string
	URL        string
	Type       string
	Enabled    bool
	Interval   int
	HasHeaders bool
}

// SourceInput is the form's payload for a new source.
type SourceInput struct {
	Name     string
	URL      string
	Type     string
	Method   string
	Path     string
	Headers  string
	Interval int
}

// TemplateRow is a built-in oracle preset.
type TemplateRow struct {
	ID   string
	Name string
}

// DataRow is one stored oracle reading.
type DataRow struct {
	ID          string
	Value       string
	Timestamp   int64
	BlockHeight int64
}

// HealthRow is one source's fetch health.
type HealthRow struct {
	SourceID    string
	Name        string
	Attempts    int
	Successes   int
	Failures    int
	LastError   string
	LastSuccess int64
}

// NFTRow is an NFT's metadata.
type NFTRow struct {
	ID          string
	Name        string
	Description string
	Owner       string
	Creator     string
	ImageURL    string
	TokenURI    string
	BlockHeight int64
}

// NFTOpRow is one operation in an NFT's history.
type NFTOpRow struct {
	Type        string
	From        string
	To          string
	BlockHeight int64
}
