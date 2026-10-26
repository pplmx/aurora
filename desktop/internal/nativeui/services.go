package nativeui

// Services is the subset of the application container a view needs. The
// concrete implementation lives in desktop/internal/deskapp, so this package
// does not import a domain, handler or storage implementation.
type Services interface {
	// LotteryHistory returns the stored draws, newest first.
	LotteryHistory(limit int) ([]DrawRow, error)
	// CreateDraw runs a VRF draw and stores it.
	CreateDraw(participants []string, seed string, winners int) (DrawRow, error)
	// VerifyDraw re-checks a stored draw against its VRF proof.
	VerifyDraw(id string) (VerifyResult, error)

	// Dashboard returns the overview counters.
	Dashboard() (DashboardStats, error)
	// LedgerStatus verifies the audit chain.
	LedgerStatus() (LedgerStatus, error)
	// LedgerBlocks returns the payload of every recorded block, newest
	// first, so the page can show what the chain holds rather than only how
	// much of it there is.
	LedgerBlocks(limit int) ([]BlockRow, error)

	// Voting surface.
	RegisterVoter(name string) (VoterRow, error)
	RegisterCandidate(name, party, program string) (CandidateRow, error)
	CreateSession(title, description string, candidateIDs []string, start, end int64) (SessionRow, error)
	CastVote(sessionID, candidateID, voterPub, voterPriv string) error
	SessionResults(sessionID string) ([]ResultRow, error)
	Candidates() ([]CandidateRow, error)
	Sessions() ([]SessionRow, error)
	StartSession(id string) error
	EndSession(id string) error

	// GenerateKeyPair makes an Ed25519 key pair (base64 public and private) for a
	// form that needs an owner or creator key; the CLI generates one when the
	// user omits it.
	GenerateKeyPair() (publicKey, privateKey string, err error)

	// Token surface.
	CreateToken(name, symbol, supply, owner string, decimals int) (TokenRow, error)
	TokenInfo(id string) (TokenRow, error)
	// ListTokens returns every stored token, newest first.
	ListTokens() ([]TokenRow, error)
	TokenBalance(tokenID, owner string) (string, error)
	MintToken(tokenID, to, amount, priv string) error
	TransferToken(tokenID, from, to, amount, priv string) error
	ApproveToken(tokenID, owner, spender, amount, priv string) error
	TransferFromToken(tokenID, owner, to, amount, spenderPriv string) error
	BurnToken(tokenID, from, amount, priv string) error
	TokenAllowance(tokenID, owner, spender string) (string, error)
	TokenHistory(tokenID, owner string, limit, offset int) ([]TransferRow, error)

	// Oracle surface.
	OracleSources() ([]SourceRow, error)
	OracleTemplates() ([]TemplateRow, error)
	AddOracleSource(in SourceInput) (SourceRow, error)
	DeleteOracleSource(id string) error
	SetOracleSourceEnabled(id string, enabled bool) error
	FetchOracleSource(id string) error
	OracleData(sourceID string, limit int) ([]DataRow, error)
	OracleLatest(sourceID string) (DataRow, error)
	OracleHealth() ([]HealthRow, error)

	// NFT surface.
	MintNFT(name, description, imageURL, tokenURI, creator string) (NFTRow, error)
	GetNFT(id string) (NFTRow, error)
	ListNFTs(owner string) ([]NFTRow, error)
	// ListAllNFTs returns every NFT, newest first. Without it the page could
	// only reach an asset whose owner key the operator already had.
	ListAllNFTs() ([]NFTRow, error)
	TransferNFT(id, from, to, priv string) error
	BurnNFT(id, owner, priv string) error
	NFTHistory(id string) ([]NFTOpRow, error)
}
