package voting

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/pplmx/aurora/internal/domain/voting"
)

type CastVoteUseCase struct {
	repo    voting.Repository
	service voting.Service
}

func NewCastVoteUseCase(repo voting.Repository, service voting.Service) *CastVoteUseCase {
	return &CastVoteUseCase{
		repo:    repo,
		service: service,
	}
}

func (uc *CastVoteUseCase) Execute(req CastVoteRequest) (*VoteResponse, error) {
	session, err := uc.repo.GetSession(req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil {
		return nil, fmt.Errorf("session not found")
	}

	now := time.Now().Unix()
	if now < session.StartTime {
		return nil, fmt.Errorf("voting session has not started yet")
	}
	if now > session.EndTime {
		return nil, fmt.Errorf("voting session has ended")
	}

	voter, err := uc.repo.GetVoter(req.VoterPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get voter: %w", err)
	}
	if voter == nil {
		return nil, fmt.Errorf("voter not registered")
	}
	if voter.HasVoted {
		return nil, fmt.Errorf("already voted")
	}

	candidate, err := uc.repo.GetCandidate(req.CandidateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get candidate: %w", err)
	}
	if candidate == nil {
		return nil, fmt.Errorf("candidate not found")
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

	vote := voting.NewVote(req.VoterPublicKey, req.CandidateID, signature, message)
	if err := uc.repo.SaveVote(vote); err != nil {
		return nil, fmt.Errorf("failed to save vote: %w", err)
	}

	candidate.VoteCount++
	if err := uc.repo.UpdateCandidate(candidate); err != nil {
		return nil, fmt.Errorf("failed to update candidate: %w", err)
	}

	voter.HasVoted = true
	voter.VoteHash = base64.StdEncoding.EncodeToString([]byte(message))
	if err := uc.repo.UpdateVoter(voter); err != nil {
		return nil, fmt.Errorf("failed to update voter: %w", err)
	}

	return &VoteResponse{
		ID:          vote.ID,
		BlockHeight: vote.BlockHeight,
	}, nil
}

type RegisterVoterUseCase struct {
	repo voting.Repository
}

func NewRegisterVoterUseCase(repo voting.Repository) *RegisterVoterUseCase {
	return &RegisterVoterUseCase{repo: repo}
}

func (uc *RegisterVoterUseCase) Execute(req RegisterVoterRequest) (*VoterResponse, error) {
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
