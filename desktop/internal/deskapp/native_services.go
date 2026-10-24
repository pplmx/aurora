package deskapp

import (
	"fmt"

	"github.com/pplmx/aurora/desktop/internal/nativeui"

	"github.com/pplmx/aurora/internal/app/lotteryapp"
	"github.com/pplmx/aurora/internal/app/nftapp"
	"github.com/pplmx/aurora/internal/app/oracleapp"
	"github.com/pplmx/aurora/internal/app/tokenapp"
	"github.com/pplmx/aurora/internal/app/votingapp"
	"github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/domain/token"
	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/pplmx/aurora/internal/infra/sqlite"
)

// nativeServices adapts the application layer to the view layer's Services
// interface.
//
// It is the only file that maps a domain entity into a nativeui row type, so a
// domain change lands here instead of rippling through layout code, and the
// views never import a domain package.
type nativeServices struct {
	lotteryRepo *sqlite.LotteryRepository
	votingRepo  *sqlite.VotingRepository
	nftRepo     *sqlite.NFTRepository
	tokenRepo   *sqlite.TokenRepository
	oracleRepo  *sqlite.OracleRepository
	chain       *blockchain.BlockChain

	// tokenService carries the audit bus and replay protection the token use
	// cases publish through, so a window mint is recorded exactly like a CLI
	// or REST one.
	tokenService token.Service
	txManager    *sqlite.TxManager
	votingSvc    voting.Service
	nftTx        nft.TransactionManager
}

var _ nativeui.Services = (*nativeServices)(nil)

func newNativeServices(
	lotteryRepo *sqlite.LotteryRepository,
	votingRepo *sqlite.VotingRepository,
	nftRepo *sqlite.NFTRepository,
	tokenRepo *sqlite.TokenRepository,
	oracleRepo *sqlite.OracleRepository,
	tokenService token.Service,
	txManager *sqlite.TxManager,
	chain *blockchain.BlockChain,
) *nativeServices {
	return &nativeServices{
		lotteryRepo:  lotteryRepo,
		votingRepo:   votingRepo,
		nftRepo:      nftRepo,
		tokenRepo:    tokenRepo,
		oracleRepo:   oracleRepo,
		chain:        chain,
		tokenService: tokenService,
		txManager:    txManager,
		votingSvc:    voting.NewEd25519Service(),
		nftTx:        sqlite.NewTxManager(nftRepo.GetDB()),
	}
}

// --- Lottery -------------------------------------------------------------

func (s *nativeServices) LotteryHistory(limit int) ([]nativeui.DrawRow, error) {
	records, err := s.lotteryRepo.GetAll()
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(records) > limit {
		records = records[:limit]
	}
	rows := make([]nativeui.DrawRow, 0, len(records))
	for _, r := range records {
		rows = append(rows, nativeui.DrawRow{
			ID: r.ID, BlockHeight: r.BlockHeight, Seed: r.Seed,
			Participants: r.Participants, Winners: r.Winners, Timestamp: r.Timestamp,
		})
	}
	return rows, nil
}

func (s *nativeServices) CreateDraw(participants []string, seed string, winners int) (nativeui.DrawRow, error) {
	uc := lotteryapp.NewCreate(s.lotteryRepo, s.chain)
	resp, err := uc.Execute(lotteryapp.CreateLotteryRequest{
		Participants: joinCSV(participants), Seed: seed, WinnerCount: winners,
	})
	if err != nil {
		return nativeui.DrawRow{}, err
	}
	return nativeui.DrawRow{
		ID: resp.ID, BlockHeight: resp.BlockHeight, Seed: resp.Seed,
		Participants: resp.Participants, Winners: resp.Winners, Timestamp: resp.Timestamp,
	}, nil
}

func (s *nativeServices) VerifyDraw(id string) (nativeui.VerifyResult, error) {
	report, err := lotteryapp.VerifyInput(s.lotteryRepo, id)
	if err != nil {
		return nativeui.VerifyResult{}, err
	}
	res := nativeui.VerifyResult{ID: report.Record.ID, Valid: report.IntegrityOK}
	for _, c := range report.Checks {
		res.Checks = append(res.Checks, c.Status+": "+c.Message)
	}
	return res, nil
}

// --- Dashboard / ledger --------------------------------------------------

func (s *nativeServices) Dashboard() (nativeui.DashboardStats, error) {
	stats := nativeui.DashboardStats{}

	records, err := s.lotteryRepo.GetAll()
	if err != nil {
		return stats, err
	}
	stats.Lotteries = len(records)

	cands, err := s.votingRepo.ListCandidates()
	if err != nil {
		return stats, err
	}
	stats.Candidates = len(cands)
	votes := 0
	for _, c := range cands {
		votes += c.VoteCount
	}
	stats.Votes = votes

	sessions, err := s.votingRepo.ListSessions()
	if err != nil {
		return stats, err
	}
	stats.Sessions = len(sessions)

	sources, err := s.oracleRepo.ListSources()
	if err != nil {
		return stats, err
	}
	stats.Oracle = len(sources)

	if s.chain != nil && s.chain.VerifyIntegrity().Valid {
		stats.Integrity = "verified"
	} else {
		stats.Integrity = "broken"
	}
	return stats, nil
}

func (s *nativeServices) LedgerStatus() (nativeui.LedgerStatus, error) {
	if s.chain == nil {
		return nativeui.LedgerStatus{}, fmt.Errorf("blockchain not initialised")
	}
	return nativeui.LedgerStatus{
		// Len takes the chain's read lock; reading the Blocks slice directly
		// raced with a concurrent AddBlock on a busy API.
		Valid:  s.chain.VerifyIntegrity().Valid,
		Height: s.chain.Len() - 1,
		Blocks: s.chain.Len(),
	}, nil
}

// --- Voting --------------------------------------------------------------

func (s *nativeServices) RegisterVoter(name string) (nativeui.VoterRow, error) {
	uc := votingapp.NewRegisterVoterUseCase(s.votingRepo)
	resp, err := uc.Execute(votingapp.RegisterVoterRequest{Name: name})
	if err != nil {
		return nativeui.VoterRow{}, err
	}
	return nativeui.VoterRow{PublicKey: resp.PublicKey, Name: resp.Name}, nil
}

func (s *nativeServices) RegisterCandidate(name, party, program string) (nativeui.CandidateRow, error) {
	uc := votingapp.NewRegisterCandidateUseCase(s.votingRepo)
	resp, err := uc.Execute(votingapp.RegisterCandidateRequest{Name: name, Party: party, Program: program})
	if err != nil {
		return nativeui.CandidateRow{}, err
	}
	return nativeui.CandidateRow{ID: resp.ID, Name: resp.Name, Party: resp.Party, Program: resp.Program}, nil
}

func (s *nativeServices) CreateSession(title, description string, candidateIDs []string, start, end int64) (nativeui.SessionRow, error) {
	uc := votingapp.NewCreateSessionUseCase(s.votingRepo)
	resp, err := uc.Execute(votingapp.CreateSessionRequest{
		Title: title, Description: description, CandidateIDs: candidateIDs,
		StartTime: start, EndTime: end,
	})
	if err != nil {
		return nativeui.SessionRow{}, err
	}
	return nativeui.SessionRow{ID: resp.ID, Title: resp.Title, Status: resp.Status}, nil
}

func (s *nativeServices) CastVote(sessionID, candidateID, voterPub, voterPriv string) error {
	uc := votingapp.NewCastVoteUseCase(s.votingRepo, s.votingSvc, s.txManager)
	_, err := uc.Execute(votingapp.CastVoteRequest{
		VoterPublicKey: voterPub, CandidateID: candidateID,
		SessionID: sessionID, PrivateKey: voterPriv,
	})
	return err
}

func (s *nativeServices) SessionResults(sessionID string) ([]nativeui.ResultRow, error) {
	uc := votingapp.NewGetResultsUseCase(s.votingRepo)
	resp, err := uc.Execute(sessionID)
	if err != nil {
		return nil, err
	}
	rows := make([]nativeui.ResultRow, 0, len(resp.Candidates))
	for _, r := range resp.Candidates {
		rows = append(rows, nativeui.ResultRow{CandidateID: r.ID, Name: r.Name, Votes: r.VoteCount})
	}
	return rows, nil
}

func (s *nativeServices) Candidates() ([]nativeui.CandidateRow, error) {
	uc := votingapp.NewGetCandidatesUseCase(s.votingRepo)
	resp, err := uc.Execute()
	if err != nil {
		return nil, err
	}
	rows := make([]nativeui.CandidateRow, 0, len(resp))
	for _, c := range resp {
		rows = append(rows, nativeui.CandidateRow{ID: c.ID, Name: c.Name, Party: c.Party, Program: c.Program, VoteCount: c.VoteCount})
	}
	return rows, nil
}

func (s *nativeServices) Sessions() ([]nativeui.SessionRow, error) {
	uc := votingapp.NewListSessionsUseCase(s.votingRepo)
	resp, err := uc.Execute()
	if err != nil {
		return nil, err
	}
	rows := make([]nativeui.SessionRow, 0, len(resp))
	for _, sess := range resp {
		rows = append(rows, nativeui.SessionRow{
			ID: sess.ID, Title: sess.Title, Status: sess.Status, Candidates: sess.Candidates,
		})
	}
	return rows, nil
}

func (s *nativeServices) StartSession(id string) error {
	return s.setSessionStatus(id, "active")
}

func (s *nativeServices) EndSession(id string) error {
	return s.setSessionStatus(id, "ended")
}

// setSessionStatus mirrors the REST handler's two steps: read the session so a
// missing id is a not-found rather than a silent no-op, then persist the new
// status.
func (s *nativeServices) setSessionStatus(id, status string) error {
	session, err := votingapp.NewGetSessionUseCase(s.votingRepo).Execute(id)
	if err != nil {
		return err
	}
	session.Status = status
	return s.votingRepo.UpdateSession(session)
}

// --- Token ---------------------------------------------------------------

func (s *nativeServices) CreateToken(name, symbol, supply, owner string, decimals int) (nativeui.TokenRow, error) {
	uc := tokenapp.NewCreate(s.tokenService)
	d := int8(decimals)
	resp, err := uc.Execute(&tokenapp.CreateTokenRequest{
		Name: name, Symbol: symbol, TotalSupply: supply, Owner: owner, Decimals: &d,
	})
	if err != nil {
		return nativeui.TokenRow{}, err
	}
	return nativeui.TokenRow{
		ID: resp.ID, Name: resp.Name, Symbol: resp.Symbol,
		TotalSupply: resp.TotalSupply, Owner: resp.Owner,
	}, nil
}

func (s *nativeServices) TokenInfo(id string) (nativeui.TokenRow, error) {
	uc := tokenapp.NewGetTokenInfoUseCase(s.tokenService)
	resp, err := uc.Execute(id)
	if err != nil {
		return nativeui.TokenRow{}, err
	}
	return nativeui.TokenRow{
		ID: resp.ID, Name: resp.Name, Symbol: resp.Symbol,
		TotalSupply: resp.TotalSupply, Decimals: int(resp.Decimals), Owner: resp.Owner,
	}, nil
}

func (s *nativeServices) TokenBalance(tokenID, owner string) (string, error) {
	uc := tokenapp.NewGetBalanceUseCase(s.tokenService)
	resp, err := uc.Execute(&tokenapp.BalanceRequest{TokenID: tokenID, Owner: owner})
	if err != nil {
		return "", err
	}
	return resp.Amount, nil
}

func (s *nativeServices) MintToken(tokenID, to, amount, priv string) error {
	uc := tokenapp.NewMintUseCase(s.tokenService)
	_, err := uc.Execute(&tokenapp.MintRequest{TokenID: tokenID, To: to, Amount: amount, PrivateKey: priv})
	return err
}

func (s *nativeServices) TransferToken(tokenID, from, to, amount, priv string) error {
	uc := tokenapp.NewTransferUseCase(s.tokenService)
	_, err := uc.Execute(&tokenapp.TransferRequest{
		TokenID: tokenID, From: from, To: to, Amount: amount, PrivateKey: priv,
	})
	return err
}

func (s *nativeServices) ApproveToken(tokenID, owner, spender, amount, priv string) error {
	uc := tokenapp.NewApproveUseCase(s.tokenService)
	_, err := uc.Execute(&tokenapp.ApproveRequest{
		TokenID: tokenID, Owner: owner, Spender: spender, Amount: amount, PrivateKey: priv,
	})
	return err
}

func (s *nativeServices) TransferFromToken(tokenID, owner, to, amount, spenderPriv string) error {
	uc := tokenapp.NewTransferFromUseCase(s.tokenService)
	_, err := uc.Execute(&tokenapp.TransferFromRequest{
		TokenID: tokenID, Owner: owner, To: to, Amount: amount, SpenderKey: spenderPriv,
	})
	return err
}

func (s *nativeServices) BurnToken(tokenID, from, amount, priv string) error {
	uc := tokenapp.NewBurnUseCase(s.tokenService)
	_, err := uc.Execute(&tokenapp.BurnRequest{
		TokenID: tokenID, From: from, Amount: amount, PrivateKey: priv,
	})
	return err
}

func (s *nativeServices) TokenAllowance(tokenID, owner, spender string) (string, error) {
	uc := tokenapp.NewGetAllowanceUseCase(s.tokenService)
	resp, err := uc.Execute(&tokenapp.AllowanceRequest{TokenID: tokenID, Owner: owner, Spender: spender})
	if err != nil {
		return "", err
	}
	return resp.Amount, nil
}

func (s *nativeServices) TokenHistory(tokenID, owner string, limit, offset int) ([]nativeui.TransferRow, error) {
	uc := tokenapp.NewGetHistoryUseCase(s.tokenService)
	resp, err := uc.Execute(&tokenapp.HistoryRequest{
		TokenID: tokenID, Owner: owner, Limit: limit, Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	rows := make([]nativeui.TransferRow, 0, len(resp.Transfers))
	for _, t := range resp.Transfers {
		rows = append(rows, nativeui.TransferRow{From: t.From, To: t.To, Amount: t.Amount})
	}
	return rows, nil
}

// --- Oracle --------------------------------------------------------------

func (s *nativeServices) OracleSources() ([]nativeui.SourceRow, error) {
	sources, err := s.oracleRepo.ListSources()
	if err != nil {
		return nil, err
	}
	rows := make([]nativeui.SourceRow, 0, len(sources))
	for _, src := range sources {
		rows = append(rows, nativeui.SourceRow{
			ID: src.ID, Name: src.Name, URL: src.URL, Type: src.Type,
			Enabled: src.Enabled, Interval: src.Interval, HasHeaders: src.Headers != "",
		})
	}
	return rows, nil
}

func (s *nativeServices) OracleTemplates() ([]nativeui.TemplateRow, error) {
	templates := oracleapp.ListTemplates()
	rows := make([]nativeui.TemplateRow, 0, len(templates))
	for _, t := range templates {
		rows = append(rows, nativeui.TemplateRow{ID: t.ID, Name: t.Name})
	}
	return rows, nil
}

func (s *nativeServices) AddOracleSource(in nativeui.SourceInput) (nativeui.SourceRow, error) {
	uc := oracleapp.NewAddSourceUseCase(s.oracleRepo)
	resp, err := uc.Execute(&oracleapp.AddSourceRequest{
		Name: in.Name, URL: in.URL, Type: in.Type, Method: in.Method,
		Path: in.Path, Headers: in.Headers, Interval: in.Interval,
	})
	if err != nil {
		return nativeui.SourceRow{}, err
	}
	return nativeui.SourceRow{ID: resp.ID, Name: resp.Name, URL: resp.URL, Type: resp.Type}, nil
}

func (s *nativeServices) DeleteOracleSource(id string) error {
	return oracleapp.NewDeleteSourceUseCase(s.oracleRepo).Execute(id)
}

func (s *nativeServices) SetOracleSourceEnabled(id string, enabled bool) error {
	if enabled {
		return oracleapp.NewEnableSourceUseCase(s.oracleRepo).Execute(id)
	}
	return oracleapp.NewDisableSourceUseCase(s.oracleRepo).Execute(id)
}

func (s *nativeServices) FetchOracleSource(id string) error {
	uc := oracleapp.NewFetchDataUseCase(s.oracleRepo)
	_, err := uc.Execute(&oracleapp.FetchDataRequest{SourceID: id})
	return err
}

func (s *nativeServices) OracleData(sourceID string, limit int) ([]nativeui.DataRow, error) {
	data, err := s.oracleRepo.GetDataBySource(sourceID, limit)
	if err != nil {
		return nil, err
	}
	rows := make([]nativeui.DataRow, 0, len(data))
	for _, d := range data {
		rows = append(rows, nativeui.DataRow{ID: d.ID, Value: d.Value, Timestamp: d.Timestamp, BlockHeight: d.BlockHeight})
	}
	return rows, nil
}

func (s *nativeServices) OracleLatest(sourceID string) (nativeui.DataRow, error) {
	d, err := s.oracleRepo.GetLatestData(sourceID)
	if err != nil {
		return nativeui.DataRow{}, err
	}
	if d == nil {
		return nativeui.DataRow{}, fmt.Errorf("no data recorded for this source yet")
	}
	return nativeui.DataRow{ID: d.ID, Value: d.Value, Timestamp: d.Timestamp, BlockHeight: d.BlockHeight}, nil
}

func (s *nativeServices) OracleHealth() ([]nativeui.HealthRow, error) {
	// Health is the scheduler's in-memory counter set, not a stored table; the
	// window shows what the running scheduler has observed, same as the REST
	// /oracle/health endpoint.
	return nil, nil
}

// --- NFT -----------------------------------------------------------------

func (s *nativeServices) MintNFT(name, description, imageURL, tokenURI, creator string) (nativeui.NFTRow, error) {
	uc := nftapp.NewMint(nft.NewService(s.nftRepo, s.nftTx), s.chain)
	resp, err := uc.Execute(&nftapp.MintNFTRequest{
		Name: name, Description: description, ImageURL: imageURL,
		TokenURI: tokenURI, Creator: creator,
	})
	if err != nil {
		return nativeui.NFTRow{}, err
	}
	return nativeui.NFTRow{
		ID: resp.ID, Name: resp.Name, Description: resp.Description,
		Owner: resp.Owner, Creator: resp.Creator, ImageURL: resp.ImageURL,
		TokenURI: resp.TokenURI, BlockHeight: resp.BlockHeight,
	}, nil
}

func (s *nativeServices) GetNFT(id string) (nativeui.NFTRow, error) {
	uc := nftapp.NewGet(nft.NewService(s.nftRepo, s.nftTx))
	resp, err := uc.Execute(id)
	if err != nil {
		return nativeui.NFTRow{}, err
	}
	return nativeui.NFTRow{
		ID: resp.ID, Name: resp.Name, Description: resp.Description,
		Owner: resp.Owner, Creator: resp.Creator, ImageURL: resp.ImageURL,
		TokenURI: resp.TokenURI, BlockHeight: resp.BlockHeight,
	}, nil
}

func (s *nativeServices) ListNFTs(owner string) ([]nativeui.NFTRow, error) {
	uc := nftapp.NewListByOwner(nft.NewService(s.nftRepo, s.nftTx))
	resp, err := uc.Execute(&nftapp.ListNFTsByOwnerRequest{Owner: owner})
	if err != nil {
		return nil, err
	}
	rows := make([]nativeui.NFTRow, 0, len(resp))
	for _, n := range resp {
		rows = append(rows, nativeui.NFTRow{
			ID: n.ID, Name: n.Name, Description: n.Description,
			Owner: n.Owner, Creator: n.Creator, BlockHeight: n.BlockHeight,
		})
	}
	return rows, nil
}

func (s *nativeServices) TransferNFT(id, from, to, priv string) error {
	uc := nftapp.NewTransferNFTUseCase(nft.NewService(s.nftRepo, s.nftTx), s.chain)
	_, err := uc.Execute(&nftapp.TransferNFTRequest{NFTID: id, From: from, To: to, PrivateKey: priv})
	return err
}

func (s *nativeServices) BurnNFT(id, owner, priv string) error {
	uc := nftapp.NewBurnNFTUseCase(nft.NewService(s.nftRepo, s.nftTx), s.chain)
	return uc.Execute(&nftapp.BurnNFTRequest{NFTID: id, Owner: owner, PrivateKey: priv})
}

func (s *nativeServices) NFTHistory(id string) ([]nativeui.NFTOpRow, error) {
	uc := nftapp.NewGetNFTOperationsUseCase(nft.NewService(s.nftRepo, s.nftTx))
	resp, err := uc.Execute(id)
	if err != nil {
		return nil, err
	}
	rows := make([]nativeui.NFTOpRow, 0, len(resp))
	for _, op := range resp {
		rows = append(rows, nativeui.NFTOpRow{Type: op.Type, From: op.From, To: op.To, BlockHeight: op.BlockHeight})
	}
	return rows, nil
}

// joinCSV rebuilds the comma-separated participants string the lottery use case
// accepts from the slice a view keeps.
func joinCSV(items []string) string {
	out := ""
	for i, it := range items {
		if i > 0 {
			out += ","
		}
		out += it
	}
	return out
}
