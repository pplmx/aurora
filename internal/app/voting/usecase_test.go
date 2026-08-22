package voting

import (
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
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
	// votes records every SaveVote so rollback tests can assert the votes
	// table state (the old mock discarded them).
	votes []*voting.Vote

	errSaveCandidate               error
	errGetCandidate                error
	errListCandidates              error
	errUpdateCandidate             error
	errIncrementCandidateVoteCount error
	errSaveVoter                   error
	errGetVoter                    error
	errSaveVote                    error
	errTryMarkVoted                error
	errSaveSession                 error
	errGetSession                  error

	// snapshot supports the mock TransactionManager: beginTx captures voter
	// flags + the votes slice so rollbackTx can restore them.
	snapshot *mockVotingTxSnapshot
}

// mockVotingTxSnapshot captures the mutation-bearing state of the mock repo.
type mockVotingTxSnapshot struct {
	voterState map[string]mockVoterFlags
	numVotes   int
}

type mockVoterFlags struct {
	hasVoted bool
	voteHash string
}

func (m *mockVotingRepo) beginTx() {
	snap := &mockVotingTxSnapshot{
		voterState: make(map[string]mockVoterFlags, len(m.voters)),
		numVotes:   len(m.votes),
	}
	for _, v := range m.voters {
		snap.voterState[v.PublicKey] = mockVoterFlags{hasVoted: v.HasVoted, voteHash: v.VoteHash}
	}
	m.snapshot = snap
}

func (m *mockVotingRepo) rollbackTx() {
	if m.snapshot == nil {
		return
	}
	for _, v := range m.voters {
		if flags, ok := m.snapshot.voterState[v.PublicKey]; ok {
			v.HasVoted = flags.hasVoted
			v.VoteHash = flags.voteHash
		}
	}
	m.votes = m.votes[:m.snapshot.numVotes]
	m.snapshot = nil
}

func (m *mockVotingRepo) commitTx() {
	m.snapshot = nil
}

// WithTx satisfies voting.TransactableRepository. The mock has no real
// transaction handle; it returns itself and relies on the beginTx/rollbackTx/
// commitTx snapshot hooks driven by mockTxManager.
func (m *mockVotingRepo) WithTx(_ *sql.Tx) voting.Repository {
	return m
}

// mockTxManager simulates TransactionManager semantics on top of the mock
// repo's snapshot hooks (same pattern as the token/nft service tests).
// Single-goroutine only.
type mockTxManager struct {
	repo       *mockVotingRepo
	shouldFail bool
	failStep   int
	step       int
}

func (m *mockTxManager) WithTransaction(fn func(tx *sql.Tx) error) error {
	if m.repo != nil {
		m.repo.beginTx()
	}

	if m.shouldFail {
		m.step++
		if m.step == m.failStep {
			if m.repo != nil {
				m.repo.rollbackTx()
			}
			return errors.New("transaction failed")
		}
	}

	err := fn(nil)

	if err != nil {
		if m.repo != nil {
			m.repo.rollbackTx()
		}
		return err
	}

	if m.repo != nil {
		m.repo.commitTx()
	}

	return nil
}

func newMockTxManagerWithRepo(repo *mockVotingRepo) *mockTxManager {
	return &mockTxManager{repo: repo}
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

func (m *mockVotingRepo) IncrementCandidateVoteCount(id string) error {
	if m.errIncrementCandidateVoteCount != nil {
		return m.errIncrementCandidateVoteCount
	}
	for _, c := range m.candidates {
		if c.ID == id {
			c.VoteCount++
			return nil
		}
	}
	return sqlite.ErrNotFound
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
	m.votes = append(m.votes, v)
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

func (m *mockVotingRepo) UnmarkVoted(publicKey string) error {
	for _, v := range m.voters {
		if v.PublicKey == publicKey {
			v.HasVoted = false
			v.VoteHash = ""
			return nil
		}
	}
	return nil
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
			{ID: "session1", StartTime: now - 3600, EndTime: now + 3600, Candidates: []string{"candidate1"}},
		},
	}
	service := &mockVotingService{signature: "dGVzdC1zaWduYXR1cmU="}
	uc := NewCastVoteUseCaseWithoutTx(repo, service)

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
	uc := NewCastVoteUseCaseWithoutTx(repo, service)

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
	uc := NewCastVoteUseCaseWithoutTx(repo, service)

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
	uc := NewCastVoteUseCaseWithoutTx(repo, service)

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
	uc := NewCastVoteUseCaseWithoutTx(repo, service)

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

func TestListSessionsUseCase(t *testing.T) {
	repo := &mockVotingRepo{
		sessions: []*voting.Session{
			{ID: "s1", Title: "Board Vote", Status: "active", Candidates: []string{"c1"}},
			{ID: "s2", Title: "Referendum", Status: "closed", Candidates: []string{"c2"}},
		},
	}
	uc := NewListSessionsUseCase(repo)

	resp, err := uc.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(resp) != 2 {
		t.Fatalf("Expected 2 sessions, got %d", len(resp))
	}
	if resp[0].Title != "Board Vote" || resp[0].Status != "active" {
		t.Errorf("unexpected first session: %+v", resp[0])
	}
	if len(resp[0].Candidates) != 1 || resp[0].Candidates[0] != "c1" {
		t.Errorf("unexpected candidates mapping: %+v", resp[0].Candidates)
	}
}

func TestListSessionsUseCase_Empty(t *testing.T) {
	repo := &mockVotingRepo{}
	uc := NewListSessionsUseCase(repo)

	resp, err := uc.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(resp) != 0 {
		t.Errorf("Expected 0 sessions, got %d", len(resp))
	}
}

func TestCreateSessionUseCase(t *testing.T) {
	now := time.Now().Unix()
	repo := &mockVotingRepo{
		candidates: []*voting.Candidate{
			{ID: "c1", Name: "Alice"},
			{ID: "c2", Name: "Bob"},
		},
	}
	uc := NewCreateSessionUseCase(repo)

	req := CreateSessionRequest{
		Title:        "Election 2024",
		Description:  "Annual election",
		CandidateIDs: []string{"c1", "c2"},
		StartTime:    now,
		EndTime:      now + 3600,
	}

	resp, err := uc.Execute(req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	if resp.Title != "Election 2024" {
		t.Errorf("Expected title 'Election 2024', got '%s'", resp.Title)
	}
}

func TestCreateSessionUseCase_Validation(t *testing.T) {
	now := time.Now().Unix()
	tests := []struct {
		name string
		req  CreateSessionRequest
		want error
	}{
		{
			name: "empty title",
			req:  CreateSessionRequest{Title: "  ", CandidateIDs: []string{"c1"}, StartTime: now, EndTime: now + 3600},
			want: voting.ErrSessionTitleRequired,
		},
		{
			name: "no candidates",
			req:  CreateSessionRequest{Title: "Election", CandidateIDs: nil, StartTime: now, EndTime: now + 3600},
			want: voting.ErrCandidatesRequired,
		},
		{
			name: "end before start",
			req:  CreateSessionRequest{Title: "Election", CandidateIDs: []string{"c1"}, StartTime: now, EndTime: now - 60},
			want: voting.ErrInvalidSessionTime,
		},
		{
			name: "unknown candidate",
			req: CreateSessionRequest{
				Title:        "Election",
				CandidateIDs: []string{"ghost"},
				StartTime:    now,
				EndTime:      now + 3600,
			},
			want: voting.ErrCandidateNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockVotingRepo{candidates: []*voting.Candidate{{ID: "c1", Name: "Alice"}}}
			uc := NewCreateSessionUseCase(repo)
			_, err := uc.Execute(tt.req)
			require.Error(t, err)
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestRegisterCandidateUseCase_EmptyName(t *testing.T) {
	repo := &mockVotingRepo{}
	uc := NewRegisterCandidateUseCase(repo)

	_, err := uc.Execute(RegisterCandidateRequest{Name: "   "})
	require.ErrorIs(t, err, voting.ErrCandidateNameRequired)
}

func TestRegisterVoterUseCase_EmptyName(t *testing.T) {
	repo := &mockVotingRepo{}
	uc := NewRegisterVoterUseCase(repo)

	_, err := uc.Execute(RegisterVoterRequest{Name: ""})
	require.ErrorIs(t, err, voting.ErrVoterNameRequired)
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
	uc := NewCastVoteUseCaseWithoutTx(repo, service)

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
	uc := NewCastVoteUseCaseWithoutTx(repo, service)

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

func TestCastVoteUseCase_SessionEndedWithinTimeWindow(t *testing.T) {
	// A session that is explicitly marked "ended" must reject votes even while
	// still inside its start/end window — otherwise the CLI `session end`
	// lifecycle is a no-op and closed sessions keep tallying votes.
	now := time.Now().Unix()
	repo := &mockVotingRepo{
		voters: []*voting.Voter{
			{Name: "voter1", PublicKey: "dm90ZXIx", HasVoted: false},
		},
		candidates: []*voting.Candidate{
			{ID: "candidate1", Name: "Alice", VoteCount: 0},
		},
		sessions: []*voting.Session{
			{ID: "session1", StartTime: now - 60, EndTime: now + 3600, Status: "ended"},
		},
	}
	service := &mockVotingService{signature: "dGVzdC1zaWduYXR1cmU="}
	uc := NewCastVoteUseCaseWithoutTx(repo, service)

	req := CastVoteRequest{
		VoterPublicKey: "dm90ZXIx",
		CandidateID:    "candidate1",
		PrivateKey:     "dGVzdC1wcml2YXRlLWtleQ==",
		SessionID:      "session1",
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for vote on a session explicitly marked ended")
	}
	if err.Error() != "voting session has ended" {
		t.Errorf("Expected 'voting session has ended', got '%v'", err)
	}
}

func TestCastVoteUseCase_SessionNotFound(t *testing.T) {
	repo := &mockVotingRepo{}
	service := &mockVotingService{}
	uc := NewCastVoteUseCaseWithoutTx(repo, service)

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
			{ID: "session1", StartTime: now - 3600, EndTime: now + 3600, Candidates: []string{"candidate1"}},
		},
	}
	service := &mockVotingService{signature: "dGVzdC1zaWduYXR1cmU="}
	uc := NewCastVoteUseCaseWithoutTx(repo, service)

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
			{ID: "session1", StartTime: now - 3600, EndTime: now + 3600, Candidates: []string{"candidate1"}},
		},
	}
	service := &mockVotingService{signature: "dGVzdC1zaWduYXR1cmU="}
	uc := NewCastVoteUseCaseWithoutTx(repo, service)

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
			{ID: "session1", StartTime: now - 3600, EndTime: now + 3600, Candidates: []string{"candidate1"}},
		},
	}
	service := &mockVotingService{signature: "dGVzdC1zaWduYXR1cmU="}
	uc := NewCastVoteUseCaseWithoutTx(repo, service)

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
			{ID: "session1", StartTime: now - 3600, EndTime: now + 3600, Candidates: []string{"candidate1"}},
		},
	}
	service := &mockVotingService{err: errors.New("signing failed")}
	uc := NewCastVoteUseCaseWithoutTx(repo, service)

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
			{ID: "session1", StartTime: now - 3600, EndTime: now + 3600, Candidates: []string{"candidate1"}},
		},
	}
	service := &mockVotingService{signature: "dGVzdC1zaWduYXR1cmU="}
	uc := NewCastVoteUseCaseWithoutTx(repo, service)

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

// TestCastVoteUseCase_SaveVoteError verifies that a SaveVote failure
// mid-transaction rolls back the voter claim: no vote row survives, the
// voter is NOT left marked as voted, and the tally is untouched. The
// pre-transaction flow relied on a best-effort UnmarkVoted compensation;
// with a real transaction no compensation exists.
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
			{ID: "session1", StartTime: now - 3600, EndTime: now + 3600, Candidates: []string{"candidate1"}},
		},
	}
	service := &mockVotingService{signature: "dGVzdC1zaWduYXR1cmU="}
	uc := NewCastVoteUseCase(repo, service, newMockTxManagerWithRepo(repo))

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

	// Rollback assertions: no partial ballot state may survive.
	if len(repo.votes) != 0 {
		t.Errorf("expected 0 saved votes after rolled-back ballot, got %d", len(repo.votes))
	}
	voter, _ := repo.GetVoter("dm90ZXIx")
	if voter == nil {
		t.Fatal("voter row must still exist")
	}
	if voter.HasVoted {
		t.Error("voter claim must be rolled back when SaveVote fails (voter must not be locked out)")
	}
	cand, _ := repo.GetCandidate("candidate1")
	if cand.VoteCount != 0 {
		t.Errorf("candidate tally must be untouched, got %d", cand.VoteCount)
	}
}

// TestCastVoteUseCase_IncrementCandidateVoteCountError verifies that a tally
// increment failure mid-transaction rolls back the ENTIRE ballot: the voter
// claim and the vote row must not survive, so the votes<->voters invariant
// ("every vote has a has_voted=1 voter") can never be violated by a partial
// commit.
func TestCastVoteUseCase_IncrementCandidateVoteCountError(t *testing.T) {
	now := time.Now().Unix()
	repo := &mockVotingRepo{
		errIncrementCandidateVoteCount: errors.New("db write failed"),
		voters: []*voting.Voter{
			{Name: "voter1", PublicKey: "dm90ZXIx", HasVoted: false},
		},
		candidates: []*voting.Candidate{
			{ID: "candidate1", Name: "Alice", VoteCount: 3},
		},
		sessions: []*voting.Session{
			{ID: "session1", StartTime: now - 3600, EndTime: now + 3600, Candidates: []string{"candidate1"}},
		},
	}
	service := &mockVotingService{signature: "dGVzdC1zaWduYXR1cmU="}
	uc := NewCastVoteUseCase(repo, service, newMockTxManagerWithRepo(repo))

	req := CastVoteRequest{
		VoterPublicKey: "dm90ZXIx",
		CandidateID:    "candidate1",
		PrivateKey:     "dGVzdC1wcml2YXRlLWtleQ==",
		SessionID:      "session1",
	}

	_, err := uc.Execute(req)
	if err == nil {
		t.Fatal("Expected error for CandidateVoteCount increment failure")
	}

	// Rollback assertions: the claim and the vote row must be undone too.
	if len(repo.votes) != 0 {
		t.Errorf("expected 0 saved votes after rolled-back ballot, got %d (orphan vote row)", len(repo.votes))
	}
	voter, _ := repo.GetVoter("dm90ZXIx")
	if voter == nil {
		t.Fatal("voter row must still exist")
	}
	if voter.HasVoted {
		t.Error("voter claim must be rolled back when the tally increment fails")
	}
	cand, _ := repo.GetCandidate("candidate1")
	if cand.VoteCount != 3 {
		t.Errorf("candidate tally must be untouched, got %d", cand.VoteCount)
	}
}

func TestCastVoteUseCase_IncrementCandidateVoteCountAppliesTally(t *testing.T) {
	now := time.Now().Unix()
	repo := &mockVotingRepo{
		voters: []*voting.Voter{
			{Name: "voter1", PublicKey: "dm90ZXIx", HasVoted: false},
		},
		candidates: []*voting.Candidate{
			{ID: "candidate1", Name: "Alice", VoteCount: 7},
		},
		sessions: []*voting.Session{
			{ID: "session1", StartTime: now - 3600, EndTime: now + 3600, Candidates: []string{"candidate1"}},
		},
	}
	service := &mockVotingService{signature: "dGVzdC1zaWduYXR1cmU="}
	uc := NewCastVoteUseCaseWithoutTx(repo, service)

	_, err := uc.Execute(CastVoteRequest{
		VoterPublicKey: "dm90ZXIx",
		CandidateID:    "candidate1",
		PrivateKey:     "dGVzdC1wcml2YXRlLWtleQ==",
		SessionID:      "session1",
	})
	require.NoError(t, err)

	got, _ := repo.GetCandidate("candidate1")
	if got.VoteCount != 8 {
		t.Fatalf("expected candidate tally 8 after one vote, got %d", got.VoteCount)
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
			{ID: "session1", StartTime: now - 3600, EndTime: now + 3600, Candidates: []string{"candidate1"}},
		},
	}
	service := &mockVotingService{signature: "dGVzdC1zaWduYXR1cmU="}
	uc := NewCastVoteUseCaseWithoutTx(repo, service)

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

// TestCastVoteUseCase_CandidateNotInSession guards ballot integrity: a vote
// for a registered candidate that is NOT part of this session's roster must be
// rejected (otherwise a caller could inflate a candidate's tally across
// elections).
func TestCastVoteUseCase_CandidateNotInSession(t *testing.T) {
	now := time.Now().Unix()
	repo := &mockVotingRepo{
		voters: []*voting.Voter{
			{Name: "voter1", PublicKey: "dm90ZXIx", HasVoted: false},
		},
		// candidate2 exists globally but is NOT in session1's roster.
		candidates: []*voting.Candidate{
			{ID: "candidate1", Name: "Alice", VoteCount: 0},
			{ID: "candidate2", Name: "Bob", VoteCount: 0},
		},
		sessions: []*voting.Session{
			{ID: "session1", StartTime: now - 3600, EndTime: now + 3600, Candidates: []string{"candidate1"}},
		},
	}
	service := &mockVotingService{signature: "dGVzdC1zaWduYXR1cmU="}
	uc := NewCastVoteUseCaseWithoutTx(repo, service)

	_, err := uc.Execute(CastVoteRequest{
		VoterPublicKey: "dm90ZXIx",
		CandidateID:    "candidate2", // exists, but not in session1
		PrivateKey:     "dGVzdC1wcml2YXRlLWtleQ==",
		SessionID:      "session1",
	})
	require.ErrorIs(t, err, voting.ErrCandidateNotInSession)
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

// TestCastVoteUseCase_AlreadyVoted_NoStateChange verifies the ErrAlreadyVoted
// sentinel from TryMarkVoted surfaces as voting.ErrAlreadyVoted through the
// transactional path (the whole transaction aborts; no vote row, no tally
// change) — strengthening the older TestCastVoteUseCase_AlreadyVoted, which
// only asserts that some error is returned.
func TestCastVoteUseCase_AlreadyVoted_NoStateChange(t *testing.T) {
	now := time.Now().Unix()
	repo := &mockVotingRepo{
		voters: []*voting.Voter{
			{Name: "voter1", PublicKey: "dm90ZXIx", HasVoted: true, VoteHash: "previous"},
		},
		candidates: []*voting.Candidate{
			{ID: "candidate1", Name: "Alice", VoteCount: 1},
		},
		sessions: []*voting.Session{
			{ID: "session1", StartTime: now - 3600, EndTime: now + 3600, Candidates: []string{"candidate1"}},
		},
	}
	service := &mockVotingService{signature: "dGVzdC1zaWduYXR1cmU="}
	uc := NewCastVoteUseCase(repo, service, newMockTxManagerWithRepo(repo))

	_, err := uc.Execute(CastVoteRequest{
		VoterPublicKey: "dm90ZXIx",
		CandidateID:    "candidate1",
		PrivateKey:     "dGVzdC1wcml2YXRlLWtleQ==",
		SessionID:      "session1",
	})
	require.ErrorIs(t, err, voting.ErrAlreadyVoted)

	require.Empty(t, repo.votes, "no vote row may be saved for a double vote")
	cand, _ := repo.GetCandidate("candidate1")
	require.Equal(t, 1, cand.VoteCount, "tally must be untouched for a double vote")
	voter, _ := repo.GetVoter("dm90ZXIx")
	require.Equal(t, "previous", voter.VoteHash, "original vote hash must be untouched")
}

// TestCastVoteUseCase_TxManagerFailure verifies a transaction-level failure
// (the manager rejects the unit before the callback runs) leaves no state.
func TestCastVoteUseCase_TxManagerFailure(t *testing.T) {
	now := time.Now().Unix()
	repo := &mockVotingRepo{
		voters: []*voting.Voter{
			{Name: "voter1", PublicKey: "dm90ZXIx", HasVoted: false},
		},
		candidates: []*voting.Candidate{
			{ID: "candidate1", Name: "Alice", VoteCount: 0},
		},
		sessions: []*voting.Session{
			{ID: "session1", StartTime: now - 3600, EndTime: now + 3600, Candidates: []string{"candidate1"}},
		},
	}
	service := &mockVotingService{signature: "dGVzdC1zaWduYXR1cmU="}
	mgr := &mockTxManager{repo: repo, shouldFail: true, failStep: 1}
	uc := NewCastVoteUseCase(repo, service, mgr)

	_, err := uc.Execute(CastVoteRequest{
		VoterPublicKey: "dm90ZXIx",
		CandidateID:    "candidate1",
		PrivateKey:     "dGVzdC1wcml2YXRlLWtleQ==",
		SessionID:      "session1",
	})
	require.Error(t, err)

	voter, _ := repo.GetVoter("dm90ZXIx")
	require.False(t, voter.HasVoted, "voter must not be marked when the transaction itself fails")
	require.Empty(t, repo.votes)
	cand, _ := repo.GetCandidate("candidate1")
	require.Equal(t, 0, cand.VoteCount)
}

// TestCastVoteUseCase_RealEd25519_AcceptCorrectKey proves a vote signed with
// the registered voter's actual private key is accepted end to end.
func TestCastVoteUseCase_RealEd25519_AcceptCorrectKey(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	pubB64 := base64.StdEncoding.EncodeToString(pub)
	privB64 := base64.StdEncoding.EncodeToString(priv)

	now := time.Now().Unix()
	repo := &mockVotingRepo{
		voters: []*voting.Voter{
			{Name: "voter1", PublicKey: pubB64, HasVoted: false},
		},
		candidates: []*voting.Candidate{
			{ID: "candidate1", Name: "Alice", VoteCount: 0},
		},
		sessions: []*voting.Session{
			{ID: "session1", StartTime: now - 3600, EndTime: now + 3600, Status: "active", Candidates: []string{"candidate1"}},
		},
	}
	uc := NewCastVoteUseCaseWithoutTx(repo, voting.NewEd25519Service())

	resp, err := uc.Execute(CastVoteRequest{
		VoterPublicKey: pubB64,
		CandidateID:    "candidate1",
		PrivateKey:     privB64,
		SessionID:      "session1",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.ID)
}

// TestCastVoteUseCase_RealEd25519_RejectForgedKey proves that a caller who
// knows only the voter's PUBLIC key (which is public) cannot cast as that
// voter with a different private key — the ballot is rejected.
func TestCastVoteUseCase_RealEd25519_RejectForgedKey(t *testing.T) {
	voterPub, _, err := ed25519.GenerateKey(nil) // victim's public key
	require.NoError(t, err)
	pubB64 := base64.StdEncoding.EncodeToString(voterPub)

	_, attackerPriv, err := ed25519.GenerateKey(nil) // attacker's own key
	require.NoError(t, err)
	attackerPrivB64 := base64.StdEncoding.EncodeToString(attackerPriv)

	now := time.Now().Unix()
	repo := &mockVotingRepo{
		voters: []*voting.Voter{
			{Name: "voter1", PublicKey: pubB64, HasVoted: false},
		},
		candidates: []*voting.Candidate{
			{ID: "candidate1", Name: "Alice", VoteCount: 0},
		},
		sessions: []*voting.Session{
			{ID: "session1", StartTime: now - 3600, EndTime: now + 3600, Status: "active", Candidates: []string{"candidate1"}},
		},
	}
	uc := NewCastVoteUseCaseWithoutTx(repo, voting.NewEd25519Service())

	_, err = uc.Execute(CastVoteRequest{
		VoterPublicKey: pubB64,
		CandidateID:    "candidate1",
		PrivateKey:     attackerPrivB64,
		SessionID:      "session1",
	})
	require.ErrorIs(t, err, voting.ErrInvalidSignature)
}
