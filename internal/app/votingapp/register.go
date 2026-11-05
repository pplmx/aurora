// Registration: the two standing registries a session draws on — voters with
// their signing keys, and candidates with their party and program.
package votingapp

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/pplmx/aurora/internal/domain/voting"
)

type RegisterVoterUseCase struct {
	repo voting.Repository
}

func NewRegisterVoterUseCase(repo voting.Repository) *RegisterVoterUseCase {
	return &RegisterVoterUseCase{repo: repo}
}

func (uc *RegisterVoterUseCase) Execute(req RegisterVoterRequest) (*VoterResponse, error) {
	if err := voting.ValidateVoterName(req.Name); err != nil {
		return nil, err
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

type ListVotersUseCase struct {
	repo voting.Repository
}

func NewListVotersUseCase(repo voting.Repository) *ListVotersUseCase {
	return &ListVotersUseCase{repo: repo}
}

func (uc *ListVotersUseCase) Execute(req *ListVotersRequest) ([]*VoterSummaryResponse, error) {
	if req == nil {
		req = &ListVotersRequest{}
	}
	voters, err := uc.repo.ListMatchingVoters(req.Search, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}
	responses := make([]*VoterSummaryResponse, len(voters))
	for i, voter := range voters {
		responses[i] = &VoterSummaryResponse{
			Name:      voter.Name,
			PublicKey: voter.PublicKey,
			HasVoted:  voter.HasVoted,
		}
	}
	return responses, nil
}

type RegisterCandidateUseCase struct {
	repo voting.Repository
}

func NewRegisterCandidateUseCase(repo voting.Repository) *RegisterCandidateUseCase {
	return &RegisterCandidateUseCase{repo: repo}
}

func (uc *RegisterCandidateUseCase) Execute(req RegisterCandidateRequest) (*CandidateResponse, error) {
	if err := voting.ValidateCandidateName(req.Name); err != nil {
		return nil, err
	}
	if err := voting.ValidateCandidateParty(req.Party); err != nil {
		return nil, err
	}
	if err := voting.ValidateCandidateProgram(req.Program); err != nil {
		return nil, err
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

func (uc *GetCandidatesUseCase) Execute(req *ListCandidatesRequest) ([]*CandidateResponse, error) {
	if req == nil {
		req = &ListCandidatesRequest{}
	}
	candidates, err := uc.repo.ListMatchingCandidates(req.Search, req.Limit, req.Offset)
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
