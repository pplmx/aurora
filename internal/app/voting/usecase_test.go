package voting

import (
	"testing"
	"time"

	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/stretchr/testify/require"
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
		if v.PublicKey == id {
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
	require.NoError(t, err)
	require.NotNil(t, resp)

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
	require.NoError(t, err)
	require.NotNil(t, resp)

	if resp.Name != "Bob" {
		t.Errorf("Expected name 'Bob', got '%s'", resp.Name)
	}
}

type mockVotingService struct {
	signature string
	err       error
}

func (m *mockVotingService) SignVote(message string, privateKey []byte) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.signature, nil
}

func (m *mockVotingService) VerifyVote(voterPK, message, signature string) bool {
	return true
}

func (m *mockVotingService) CountVotes(candidates []voting.Candidate) map[string]int {
	results := make(map[string]int)
	for _, c := range candidates {
		results[c.ID] = c.VoteCount
	}
	return results
}

func TestCastVoteUseCase_Execute(t *testing.T) {
	now := time.Now().Unix()
	repo := &mockVotingRepo{
		voters: []*voting.Voter{
			{Name: "voter1", PublicKey: "dm90ZXIx", HasVoted: false},
		},
		candidates: []*voting.Candidate{
			{ID: "candidate1", Name: "Alice", VoteCount: 0},
		},
		sessions: []*voting.Session{
			{ID: "session1", StartTime: now - 3600, EndTime: now + 3600},
		},
	}
	service := &mockVotingService{signature: "dGVzdC1zaWduYXR1cmU="}
	uc := NewCastVoteUseCase(repo, service)

	req := CastVoteRequest{
		VoterPublicKey: "dm90ZXIx",
		CandidateID:    "candidate1",
		PrivateKey:     "dGVzdC1wcml2YXRlLWtleQ==",
		SessionID:      "session1",
	}

	resp, err := uc.Execute(req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response should not be nil")
	}
}

func TestCastVoteUseCase_VoterNotFound(t *testing.T) {
	repo := &mockVotingRepo{}
	service := &mockVotingService{}
	uc := NewCastVoteUseCase(repo, service)

	req := CastVoteRequest{
		VoterPublicKey: "nonexistent",
		CandidateID:    "candidate1",
		PrivateKey:     "dGVzdA==",
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for nonexistent voter")
	}
}

func TestCastVoteUseCase_AlreadyVoted(t *testing.T) {
	repo := &mockVotingRepo{
		voters: []*voting.Voter{
			{Name: "voter1", PublicKey: "dm90ZXIx", HasVoted: true},
		},
	}
	service := &mockVotingService{}
	uc := NewCastVoteUseCase(repo, service)

	req := CastVoteRequest{
		VoterPublicKey: "dm90ZXIx",
		CandidateID:    "candidate1",
		PrivateKey:     "dGVzdA==",
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for already voted")
	}
}

func TestCastVoteUseCase_CandidateNotFound(t *testing.T) {
	repo := &mockVotingRepo{
		voters: []*voting.Voter{
			{Name: "voter1", PublicKey: "dm90ZXIx", HasVoted: false},
		},
	}
	service := &mockVotingService{}
	uc := NewCastVoteUseCase(repo, service)

	req := CastVoteRequest{
		VoterPublicKey: "dm90ZXIx",
		CandidateID:    "nonexistent",
		PrivateKey:     "dGVzdA==",
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for nonexistent candidate")
	}
}

func TestCastVoteUseCase_InvalidPrivateKey(t *testing.T) {
	repo := &mockVotingRepo{
		voters: []*voting.Voter{
			{Name: "voter1", PublicKey: "dm90ZXIx", HasVoted: false},
		},
		candidates: []*voting.Candidate{
			{ID: "candidate1", Name: "Alice"},
		},
	}
	service := &mockVotingService{}
	uc := NewCastVoteUseCase(repo, service)

	req := CastVoteRequest{
		VoterPublicKey: "dm90ZXIx",
		CandidateID:    "candidate1",
		PrivateKey:     "!!!invalid!!!",
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for invalid private key")
	}
}

func TestGetCandidatesUseCase(t *testing.T) {
	repo := &mockVotingRepo{
		candidates: []*voting.Candidate{
			{ID: "1", Name: "Alice", Party: "Party A", VoteCount: 10},
			{ID: "2", Name: "Bob", Party: "Party B", VoteCount: 5},
		},
	}
	uc := NewGetCandidatesUseCase(repo)

	resp, err := uc.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(resp) != 2 {
		t.Errorf("Expected 2 candidates, got %d", len(resp))
	}
}

func TestGetCandidatesUseCase_Empty(t *testing.T) {
	repo := &mockVotingRepo{}
	uc := NewGetCandidatesUseCase(repo)

	resp, err := uc.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(resp) != 0 {
		t.Errorf("Expected 0 candidates, got %d", len(resp))
	}
}

func TestCreateSessionUseCase(t *testing.T) {
	repo := &mockVotingRepo{}
	uc := NewCreateSessionUseCase(repo)

	req := CreateSessionRequest{
		Title:        "Election 2024",
		Description:  "Annual election",
		CandidateIDs: []string{"c1", "c2"},
	}

	resp, err := uc.Execute(req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	if resp.Title != "Election 2024" {
		t.Errorf("Expected title 'Election 2024', got '%s'", resp.Title)
	}
}

func TestCastVoteUseCase_SessionNotStarted(t *testing.T) {
	now := time.Now().Unix()
	repo := &mockVotingRepo{
		voters: []*voting.Voter{
			{Name: "voter1", PublicKey: "dm90ZXIx", HasVoted: false},
		},
		candidates: []*voting.Candidate{
			{ID: "candidate1", Name: "Alice", VoteCount: 0},
		},
		sessions: []*voting.Session{
			{ID: "session1", StartTime: now + 3600, EndTime: now + 7200},
		},
	}
	service := &mockVotingService{signature: "dGVzdC1zaWduYXR1cmU="}
	uc := NewCastVoteUseCase(repo, service)

	req := CastVoteRequest{
		VoterPublicKey: "dm90ZXIx",
		CandidateID:    "candidate1",
		PrivateKey:     "dGVzdC1wcml2YXRlLWtleQ==",
		SessionID:      "session1",
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for vote before session starts")
	}
	if err.Error() != "voting session has not started yet" {
		t.Errorf("Expected 'voting session has not started yet', got '%v'", err)
	}
}

func TestCastVoteUseCase_SessionEnded(t *testing.T) {
	now := time.Now().Unix()
	repo := &mockVotingRepo{
		voters: []*voting.Voter{
			{Name: "voter1", PublicKey: "dm90ZXIx", HasVoted: false},
		},
		candidates: []*voting.Candidate{
			{ID: "candidate1", Name: "Alice", VoteCount: 0},
		},
		sessions: []*voting.Session{
			{ID: "session1", StartTime: now - 7200, EndTime: now - 3600},
		},
	}
	service := &mockVotingService{signature: "dGVzdC1zaWduYXR1cmU="}
	uc := NewCastVoteUseCase(repo, service)

	req := CastVoteRequest{
		VoterPublicKey: "dm90ZXIx",
		CandidateID:    "candidate1",
		PrivateKey:     "dGVzdC1wcml2YXRlLWtleQ==",
		SessionID:      "session1",
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for vote after session ends")
	}
	if err.Error() != "voting session has ended" {
		t.Errorf("Expected 'voting session has ended', got '%v'", err)
	}
}

func TestCastVoteUseCase_SessionNotFound(t *testing.T) {
	repo := &mockVotingRepo{}
	service := &mockVotingService{}
	uc := NewCastVoteUseCase(repo, service)

	req := CastVoteRequest{
		VoterPublicKey: "dm90ZXIx",
		CandidateID:    "candidate1",
		PrivateKey:     "dGVzdA==",
		SessionID:      "nonexistent",
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for nonexistent session")
	}
	if err.Error() != "session not found" {
		t.Errorf("Expected 'session not found', got '%v'", err)
	}
}
