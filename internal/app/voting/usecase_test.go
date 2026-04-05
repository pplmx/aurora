package voting

import (
	"testing"

	"github.com/pplmx/aurora/internal/domain/voting"
)

type mockVotingRepo struct {
	candidates []*voting.Candidate
	voters     []*voting.Voter
	sessions   []*voting.Session
}

func (m *mockVotingRepo) SaveCandidate(c *voting.Candidate) error {
	m.candidates = append(m.candidates, c)
	return nil
}

func (m *mockVotingRepo) GetCandidate(id string) (*voting.Candidate, error) {
	for _, c := range m.candidates {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, nil
}

func (m *mockVotingRepo) ListCandidates() ([]*voting.Candidate, error) {
	return m.candidates, nil
}

func (m *mockVotingRepo) UpdateCandidate(c *voting.Candidate) error {
	return nil
}

func (m *mockVotingRepo) DeleteCandidate(id string) error {
	return nil
}

func (m *mockVotingRepo) SaveVoter(v *voting.Voter) error {
	m.voters = append(m.voters, v)
	return nil
}

func (m *mockVotingRepo) GetVoter(id string) (*voting.Voter, error) {
	for _, v := range m.voters {
		if v.Name == id {
			return v, nil
		}
	}
	return nil, nil
}

func (m *mockVotingRepo) SaveVote(v *voting.Vote) error {
	return nil
}

func (m *mockVotingRepo) GetVote(id string) (*voting.Vote, error) {
	return nil, nil
}

func (m *mockVotingRepo) GetVotesByCandidate(candidateID string) ([]*voting.Vote, error) {
	return nil, nil
}

func (m *mockVotingRepo) GetVotesByVoter(voterPK string) ([]*voting.Vote, error) {
	return nil, nil
}

func (m *mockVotingRepo) SaveSession(s *voting.Session) error {
	m.sessions = append(m.sessions, s)
	return nil
}

func (m *mockVotingRepo) GetSession(id string) (*voting.Session, error) {
	for _, s := range m.sessions {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, nil
}

func (m *mockVotingRepo) GetAllVoters() ([]*voting.Voter, error) {
	return m.voters, nil
}

func (m *mockVotingRepo) ListVoters() ([]*voting.Voter, error) {
	return m.voters, nil
}

func (m *mockVotingRepo) UpdateVoter(v *voting.Voter) error {
	return nil
}

func (m *mockVotingRepo) ListSessions() ([]*voting.Session, error) {
	return m.sessions, nil
}

func (m *mockVotingRepo) UpdateSession(s *voting.Session) error {
	return nil
}

func TestRegisterCandidateUseCase(t *testing.T) {
	repo := &mockVotingRepo{}
	uc := NewRegisterCandidateUseCase(repo)

	req := RegisterCandidateRequest{
		Name:    "Alice",
		Party:   "Party A",
		Program: "Platform A",
	}

	resp, err := uc.Execute(req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response should not be nil")
	}

	if resp.Name != "Alice" {
		t.Errorf("Expected name 'Alice', got '%s'", resp.Name)
	}
}

func TestRegisterVoterUseCase(t *testing.T) {
	repo := &mockVotingRepo{}
	uc := NewRegisterVoterUseCase(repo)

	req := RegisterVoterRequest{
		Name: "Bob",
	}

	resp, err := uc.Execute(req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response should not be nil")
	}

	if resp.Name != "Bob" {
		t.Errorf("Expected name 'Bob', got '%s'", resp.Name)
	}
}
