package nativeui

// The row types a view renders. They are plain data: the adapter that
// implements Services maps domain entities into them, so the view layer never
// depends on a domain package's shape and a domain change cannot ripple into
// layout code.

// DrawRow is one draw as a view shows it.
type DrawRow struct {
	ID           string
	BlockHeight  int64
	Seed         string
	Participants []string
	Winners      []string
	Timestamp    int64
}

// BlockRow is one audit block as the ledger page shows it. It is the ledger's
// record: the module that wrote it, what it did, the object it names, the
// facts worth reading, and the exact payload underneath for an auditor who
// needs the bytes rather than the sentence.
//
// Summary is kept beside these fields because a page that wants one line still
// needs a compact form, while the ledger page reads Action, Subject and Facts.
type BlockRow struct {
	// Height is the block's position in the chain.
	Height int64
	// Timestamp is the block's own creation time, in Unix seconds. It is 0
	// for an entry read without one.
	Timestamp int64
	// Module names the subsystem that recorded it, taken from the payload.
	Module string
	// Action is the decoded verb: Draw, Mint, Transfer, Ballot, Observation.
	Action string
	// Subject is the object the action names: the winners of a draw, the
	// token of a mint, the candidate of a ballot, the value of an observation.
	Subject string
	// Facts are the labelled values worth reading beside the subject. Labels
	// are English keys the view translates, like every other visible label.
	Facts []BlockFact
	// Summary is the payload, shortened for a compact row.
	Summary string
	// Payload is the exact recorded text, unmodified. The ledger page shows it
	// on request so the decoded sentence never becomes the only copy.
	Payload string
}

// BlockFact is one labelled value read out of a recorded payload.
type BlockFact struct {
	Label string
	Value string
}

// VerifyResult is the outcome of re-checking a draw.
type VerifyResult struct {
	ID      string
	Valid   bool
	Summary string
	Checks  []string
}

// New builds the window's state and loads the first page. A nil Services is a
// programming error, not a user-visible condition: the shell only constructs
// an App after it has wired the database.
//
// The first page is loaded here rather than on the first render because a view
// only draws what the state already holds: a window that opened on the
// dashboard and never switched pages showed "Loading…" forever, since nothing
// else asked the services for the counters.

// LedgerStatus is the audit chain's verified state.
type LedgerStatus struct {
	Valid  bool
	Height int
	Blocks int
	// Head is the newest block's hash prefix: the chain's identity when two
	// installations are compared.
	Head string
}

// LedgerBlockPage is one newest-first page of the audit chain plus the total
// number of entries matching the same query.
type LedgerBlockPage struct {
	Rows  []BlockRow
	Total int
}

// VoterRow is one registered voter.
type VoterRow struct {
	PublicKey string
	Name      string
	HasVoted  bool // lifetime history; ballot eligibility is per session
	// PrivateKey is the ballot-signing key the registration generated. It is
	// returned once, by RegisterVoter only: the stored voter keeps just the
	// public half, so a window that dropped it left the operator with a voter
	// that can never cast a signed ballot. Empty when the row came from a list
	// read, where the key is not available.
	PrivateKey string
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
	ID    string
	Title string
	// Description is the optional text entered with the session; the table is
	// too narrow to show it, but the selected-session readout does.
	Description string
	Status      string
	Candidates  []string
	StartTime   int64
	EndTime     int64
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
	ID string
	// Name, URL, Method, Path and Interval are the preset's values: choosing
	// one fills the form with all of them, the way `oracle source template`
	// does. Carrying only the name and type produced a form missing the URL,
	// so Add Source refused a preset the CLI would have accepted.
	Name     string
	URL      string
	Method   string
	Path     string
	Interval int
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
