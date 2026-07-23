package voting

import (
	"errors"
	"testing"
	"time"

	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/stretchr/testify/require"
)

type mockVotingRepo struct {
	candidates []*voting.Candidate
	voters     []*voting.Voter
	sessions   []*voting.Session

	errSaveCandidate   error
	errGetCandidate    error
	errListCandidates  error
	errUpdateCandidate error
	errSaveVoter       error
	errGetVoter        error
	errSaveVote        error
	errTryMarkVoted    error
	errSaveSession     error
	errGetSession      error
}

func (m *mockVotingRepo) SaveCandidate(c *voting.Candidate) error {
	if m.errSaveCandidate != nil {
		return m.errSaveCandidate
	}
	m.candidates = append(m.candidates, c)
	return nil
}

func (m *mockVotingRepo) GetCandidate(id string) (*voting.Candidate, error) {
	if m.errGetCandidate != nil {
		return nil, m.errGetCandidate
	}
	for _, c := range m.candidates {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, nil
}

func (m *mockVotingRepo) ListCandidates() ([]*voting.Candidate, error) {
	if m.errListCandidates != nil {
		return nil, m.errListCandidates
	}
	return m.candidates, nil
}

func (m *mockVotingRepo) UpdateCandidate(c *voting.Candidate) error {
	if m.errUpdateCandidate != nil {
		return m.errUpdateCandidate
	}
	return nil
}

func (m *mockVotingRepo) DeleteCandidate(id string) error {
	return nil
}

func (m *mockVotingRepo) SaveVoter(v *voting.Voter) error {
	if m.errSaveVoter != nil {
		return m.errSaveVoter
	}
	m.voters = append(m.voters, v)
	return nil
}

func (m *mockVotingRepo) GetVoter(id string) (*voting.Voter, error) {
	if m.errGetVoter != nil {
		return nil, m.errGetVoter
	}
	for _, v := range m.voters {
		if v.PublicKey == id {
			return v, nil
		}
	}
	return nil, nil
}

func (m *mockVotingRepo) SaveVote(v *voting.Vote) error {
	if m.errSaveVote != nil {
		return m.errSaveVote
	}
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

func (m *mockVotingRepo) DeleteVote(id string) error {
	return nil
}

// TryMarkVoted mirrors the real SQLite repo semantics: the first
// caller succeeds, subsequent callers get an "already voted" error.
// The fake's lock-free duplicate-detection is fine for unit tests
// because tests don't run it from multiple goroutines.
func (m *mockVotingRepo) TryMarkVoted(publicKey, voteHash string) error {
	if m.errTryMarkVoted != nil {
		return m.errTryMarkVoted
	}
	for _, v := range m.voters {
		if v.PublicKey == publicKey {
			if v.HasVoted {
				return sqlite.ErrAlreadyVoted
			}
			v.HasVoted = true
			v.VoteHash = voteHash
			return nil
		}
	}
	return sqlite.ErrNotFound
}

func (m *mockVotingRepo) SaveSession(s *voting.Session) error {
	if m.errSaveSession != nil {
		return m.errSaveSession
	}
	m.sessions = append(m.sessions, s)
	return nil
}

func (m *mockVotingRepo) GetSession(id string) (*voting.Session, error) {
	if m.errGetSession != nil {
		return nil, m.errGetSession
	}
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

func TestCastVoteUseCase_GetSessionRepoError(t *testing.T) {
	now := time.Now().Unix()
	repo := &mockVotingRepo{
		errGetSession: errors.New("db connection lost"),
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

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for repo failure on GetSession")
	}
}

func TestCastVoteUseCase_GetVoterRepoError(t *testing.T) {
	now := time.Now().Unix()
	repo := &mockVotingRepo{
		errGetVoter: errors.New("db connection lost"),
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

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for repo failure on GetVoter")
	}
}

func TestCastVoteUseCase_GetCandidateRepoError(t *testing.T) {
	now := time.Now().Unix()
	repo := &mockVotingRepo{
		errGetCandidate: errors.New("db connection lost"),
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

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for repo failure on GetCandidate")
	}
}

func TestCastVoteUseCase_SignVoteError(t *testing.T) {
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
	service := &mockVotingService{err: errors.New("signing failed")}
	uc := NewCastVoteUseCase(repo, service)

	req := CastVoteRequest{
		VoterPublicKey: "dm90ZXIx",
		CandidateID:    "candidate1",
		PrivateKey:     "dGVzdC1wcml2YXRlLWtleQ==",
		SessionID:      "session1",
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for signing failure")
	}
}

func TestCastVoteUseCase_TryMarkVotedGenericError(t *testing.T) {
	now := time.Now().Unix()
	repo := &mockVotingRepo{
		errTryMarkVoted: errors.New("transaction failed"),
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

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for generic TryMarkVoted failure")
	}
}

func TestCastVoteUseCase_SaveVoteError(t *testing.T) {
	now := time.Now().Unix()
	repo := &mockVotingRepo{
		errSaveVote: errors.New("db write failed"),
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

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for SaveVote failure")
	}
}

func TestCastVoteUseCase_UpdateCandidateError(t *testing.T) {
	now := time.Now().Unix()
	repo := &mockVotingRepo{
		errUpdateCandidate: errors.New("db write failed"),
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

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for UpdateCandidate failure")
	}
}

func TestCastVoteUseCase_TryMarkVotedNotFound(t *testing.T) {
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
		VoterPublicKey: "dm90ZXJx",
		CandidateID:    "candidate1",
		PrivateKey:     "dGVzdC1wcml2YXRlLWtleQ==",
		SessionID:      "session1",
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for voter not found in TryMarkVoted")
	}
	if err.Error() != "voter not registered" {
		t.Errorf("Expected 'voter not registered', got '%v'", err)
	}
}

func TestRegisterCandidateUseCase_SaveError(t *testing.T) {
	repo := &mockVotingRepo{errSaveCandidate: errors.New("db write failed")}
	uc := NewRegisterCandidateUseCase(repo)

	req := RegisterCandidateRequest{
		Name:    "Alice",
		Party:   "Party A",
		Program: "Platform A",
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for SaveCandidate failure")
	}
}

func TestGetCandidatesUseCase_ListError(t *testing.T) {
	repo := &mockVotingRepo{errListCandidates: errors.New("db read failed")}
	uc := NewGetCandidatesUseCase(repo)

	_, err := uc.Execute()
	if err == nil {
		t.Fatal("Expected error for ListCandidates failure")
	}
}

func TestCreateSessionUseCase_SaveError(t *testing.T) {
	repo := &mockVotingRepo{errSaveSession: errors.New("db write failed")}
	uc := NewCreateSessionUseCase(repo)

	req := CreateSessionRequest{
		Title:        "Election 2024",
		Description:  "Annual election",
		CandidateIDs: []string{"c1", "c2"},
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for SaveSession failure")
	}
}

func TestRegisterVoterUseCase_SaveError(t *testing.T) {
	repo := &mockVotingRepo{errSaveVoter: errors.New("db write failed")}
	uc := NewRegisterVoterUseCase(repo)

	req := RegisterVoterRequest{
		Name: "Bob",
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for SaveVoter failure")
	}
}
