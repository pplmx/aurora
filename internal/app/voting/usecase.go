package voting

import (
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/pplmx/aurora/internal/infra/sqlite"
)

type CastVoteUseCase struct {
	repo      voting.TransactableRepository
	service   voting.Service
	txManager voting.TransactionManager
}

func NewCastVoteUseCase(repo voting.TransactableRepository, service voting.Service, txManager voting.TransactionManager) *CastVoteUseCase {
	if txManager == nil {
		txManager = noOpTxManager{}
	}
	return &CastVoteUseCase{
		repo:      repo,
		service:   service,
		txManager: txManager,
	}
}

// noOpTxManager executes the callback directly without a transaction, for
// callers over repositories without transaction support (in-memory fakes).
type noOpTxManager struct{}

func (noOpTxManager) WithTransaction(fn func(tx *sql.Tx) error) error {
	return fn(nil)
}

// NewCastVoteUseCaseWithoutTx builds the use case without a transaction
// manager. Only appropriate for repositories without transaction support;
// the SQLite-backed wiring must use NewCastVoteUseCase with a real
// TransactionManager.
func NewCastVoteUseCaseWithoutTx(repo voting.TransactableRepository, service voting.Service) *CastVoteUseCase {
	return NewCastVoteUseCase(repo, service, noOpTxManager{})
}

// txRepo returns the repository scoped to the given transaction when one is
// active, otherwise the use case's default repository. Every mutation inside
// the WithTransaction callback MUST go through txRepo so the voter claim,
// the vote row and the tally increment share one SQLite transaction and
// roll back together.
func (uc *CastVoteUseCase) txRepo(tx *sql.Tx) voting.Repository {
	if tx == nil {
		return uc.repo
	}
	return uc.repo.WithTx(tx)
}

func (uc *CastVoteUseCase) Execute(req CastVoteRequest) (*VoteResponse, error) {
	session, err := uc.repo.GetSession(req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil {
		return nil, voting.ErrSessionNotFound
	}

	now := time.Now().Unix()
	if now < session.StartTime {
		return nil, voting.ErrSessionNotStarted
	}
	if now > session.EndTime {
		return nil, voting.ErrSessionEnded
	}

	voter, err := uc.repo.GetVoter(req.VoterPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get voter: %w", err)
	}
	if voter == nil {
		return nil, voting.ErrVoterNotRegistered
	}

	candidate, err := uc.repo.GetCandidate(req.CandidateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get candidate: %w", err)
	}
	if candidate == nil {
		return nil, voting.ErrCandidateNotFound
	}

	// Ballot integrity: the candidate must belong to THIS session's roster.
	// Sessions declare their candidate set (session.Candidates), so a vote for
	// a registered-but-not-in-this-session candidate (possibly one in a
	// different election) must be rejected rather than inflating that
	// candidate's tally.
	if !containsString(session.Candidates, req.CandidateID) {
		return nil, voting.ErrCandidateNotInSession
	}

	privBytes, err := base64.StdEncoding.DecodeString(req.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid private key format: %w", err)
	}

	timestamp := time.Now().Unix()
	message := fmt.Sprintf("%s|%s|%d", req.VoterPublicKey, req.CandidateID, timestamp)
	signature, err := uc.service.SignVote(message, privBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to sign vote: %w", err)
	}

	// The three ballot mutations — claim the voter, save the vote row,
	// increment the tally — run in ONE transaction. The pre-transaction flow
	// claimed the voter first and, on a later failure, ran a best-effort
	// UnmarkVoted/DeleteVote compensation whose own errors were silently
	// swallowed; a failed compensation permanently locked the voter out or
	// left an orphan vote row (violating the votes<->voters invariant). With
	// a real transaction no partial ballot state can ever commit.
	//
	// TryMarkVoted remains the concurrency primitive: its conditional UPDATE
	// guarantees exactly one concurrent caller for the same voter wins; the
	// losers get ErrAlreadyVoted and their whole transaction aborts with no
	// side effects.
	voteHash := base64.StdEncoding.EncodeToString([]byte(message))
	var voteID string
	var blockHeight int64
	err = uc.txManager.WithTransaction(func(tx *sql.Tx) error {
		txRepo := uc.txRepo(tx)

		if err := txRepo.TryMarkVoted(req.VoterPublicKey, voteHash); err != nil {
			return err
		}

		vote := voting.NewVote(req.VoterPublicKey, req.CandidateID, signature, message)
		if err := txRepo.SaveVote(vote); err != nil {
			return fmt.Errorf("failed to save vote: %w", err)
		}
		voteID = vote.ID
		blockHeight = vote.BlockHeight

		if err := txRepo.IncrementCandidateVoteCount(req.CandidateID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, sqlite.ErrAlreadyVoted) {
			return nil, voting.ErrAlreadyVoted
		}
		if errors.Is(err, sqlite.ErrNotFound) {
			// A row vanished between the existence checks above and the
			// transactional mutations: either the voter (TryMarkVoted) or
			// the candidate (IncrementCandidateVoteCount) was deleted
			// concurrently. Re-check the voter to map to the right public
			// error.
			if v, vErr := uc.repo.GetVoter(req.VoterPublicKey); vErr == nil && v == nil {
				return nil, voting.ErrVoterNotRegistered
			}
			return nil, voting.ErrCandidateNotFound
		}
		return nil, fmt.Errorf("failed to cast vote: %w", err)
	}

	return &VoteResponse{
		ID:          voteID,
		BlockHeight: blockHeight,
	}, nil
}

type RegisterVoterUseCase struct {
	repo voting.Repository
}

func NewRegisterVoterUseCase(repo voting.Repository) *RegisterVoterUseCase {
	return &RegisterVoterUseCase{repo: repo}
}

func (uc *RegisterVoterUseCase) Execute(req RegisterVoterRequest) (*VoterResponse, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, voting.ErrVoterNameRequired
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	voter := voting.NewVoter(req.Name)
	voter.PublicKey = base64.StdEncoding.EncodeToString(pub)

	if err := uc.repo.SaveVoter(voter); err != nil {
		return nil, fmt.Errorf("failed to save voter: %w", err)
	}

	return &VoterResponse{
		ID:         voter.PublicKey,
		Name:       voter.Name,
		PublicKey:  voter.PublicKey,
		PrivateKey: base64.StdEncoding.EncodeToString(priv),
	}, nil
}

type RegisterCandidateUseCase struct {
	repo voting.Repository
}

func NewRegisterCandidateUseCase(repo voting.Repository) *RegisterCandidateUseCase {
	return &RegisterCandidateUseCase{repo: repo}
}

func (uc *RegisterCandidateUseCase) Execute(req RegisterCandidateRequest) (*CandidateResponse, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, voting.ErrCandidateNameRequired
	}

	candidate := voting.NewCandidate(req.Name, req.Party, req.Program)

	if err := uc.repo.SaveCandidate(candidate); err != nil {
		return nil, fmt.Errorf("failed to save candidate: %w", err)
	}

	return &CandidateResponse{
		ID:        candidate.ID,
		Name:      candidate.Name,
		Party:     candidate.Party,
		Program:   candidate.Program,
		VoteCount: candidate.VoteCount,
	}, nil
}

type GetCandidatesUseCase struct {
	repo voting.Repository
}

func NewGetCandidatesUseCase(repo voting.Repository) *GetCandidatesUseCase {
	return &GetCandidatesUseCase{repo: repo}
}

func (uc *GetCandidatesUseCase) Execute() ([]*CandidateResponse, error) {
	candidates, err := uc.repo.ListCandidates()
	if err != nil {
		return nil, err
	}

	responses := make([]*CandidateResponse, len(candidates))
	for i, c := range candidates {
		responses[i] = &CandidateResponse{
			ID:        c.ID,
			Name:      c.Name,
			Party:     c.Party,
			Program:   c.Program,
			VoteCount: c.VoteCount,
		}
	}
	return responses, nil
}

type CreateSessionUseCase struct {
	repo voting.Repository
}

func NewCreateSessionUseCase(repo voting.Repository) *CreateSessionUseCase {
	return &CreateSessionUseCase{repo: repo}
}

func (uc *CreateSessionUseCase) Execute(req CreateSessionRequest) (*SessionResponse, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, voting.ErrSessionTitleRequired
	}
	if len(req.CandidateIDs) == 0 {
		return nil, voting.ErrCandidatesRequired
	}
	if req.EndTime <= req.StartTime {
		return nil, voting.ErrInvalidSessionTime
	}
	// The session stores candidate IDs as references; reject dangling
	// references up front so a session never points at candidates that do
	// not exist.
	for _, id := range req.CandidateIDs {
		c, err := uc.repo.GetCandidate(id)
		if err != nil {
			return nil, fmt.Errorf("failed to get candidate: %w", err)
		}
		if c == nil {
			return nil, voting.ErrCandidateNotFound
		}
	}

	session := voting.NewSession(req.Title, req.Description, req.CandidateIDs, req.StartTime, req.EndTime)

	if err := uc.repo.SaveSession(session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return &SessionResponse{
		ID:          session.ID,
		Title:       session.Title,
		Description: session.Description,
		Status:      session.Status,
		Candidates:  session.Candidates,
	}, nil
}

// ListSessionsUseCase returns every session in the repository, exposing the
// sessions surface that CreateSession writes into but which previously had no
// read path (the API could only fetch a single session by id).
type ListSessionsUseCase struct {
	repo voting.Repository
}

func NewListSessionsUseCase(repo voting.Repository) *ListSessionsUseCase {
	return &ListSessionsUseCase{repo: repo}
}

func (uc *ListSessionsUseCase) Execute() ([]*SessionResponse, error) {
	sessions, err := uc.repo.ListSessions()
	if err != nil {
		return nil, err
	}
	responses := make([]*SessionResponse, len(sessions))
	for i, s := range sessions {
		responses[i] = &SessionResponse{
			ID:          s.ID,
			Title:       s.Title,
			Description: s.Description,
			Status:      s.Status,
			Candidates:  s.Candidates,
		}
	}
	return responses, nil
}

// containsString reports whether list contains target (small lists; linear scan).
func containsString(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}
