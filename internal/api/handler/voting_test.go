package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domainvoting "github.com/pplmx/aurora/internal/domain/voting"
	"github.com/stretchr/testify/assert"
)

type mockVotingRepo struct {
	voters         map[string]*domainvoting.Voter
	candidates     map[string]*domainvoting.Candidate
	sessions       map[string]*domainvoting.Session
	saveVoterErr   error
	saveCandErr    error
	saveSessionErr error
}

func newMockVotingRepo() *mockVotingRepo {
	return &mockVotingRepo{
		voters:     make(map[string]*domainvoting.Voter),
		candidates: make(map[string]*domainvoting.Candidate),
		sessions:   make(map[string]*domainvoting.Session),
	}
}

func (m *mockVotingRepo) SaveVote(*domainvoting.Vote) error                        { return nil }
func (m *mockVotingRepo) GetVote(string) (*domainvoting.Vote, error)               { return nil, nil }
func (m *mockVotingRepo) GetVotesByCandidate(string) ([]*domainvoting.Vote, error) { return nil, nil }
func (m *mockVotingRepo) GetVotesByVoter(string) ([]*domainvoting.Vote, error)     { return nil, nil }
func (m *mockVotingRepo) DeleteVote(string) error                                  { return nil }

func (m *mockVotingRepo) SaveVoter(voter *domainvoting.Voter) error {
	if m.saveVoterErr != nil {
		return m.saveVoterErr
	}
	m.voters[voter.PublicKey] = voter
	return nil
}
func (m *mockVotingRepo) GetVoter(pk string) (*domainvoting.Voter, error) {
	return m.voters[pk], nil
}
func (m *mockVotingRepo) UpdateVoter(*domainvoting.Voter) error      { return nil }
func (m *mockVotingRepo) TryMarkVoted(_, _ string) error             { return nil }
func (m *mockVotingRepo) ListVoters() ([]*domainvoting.Voter, error) { return nil, nil }

func (m *mockVotingRepo) SaveCandidate(candidate *domainvoting.Candidate) error {
	if m.saveCandErr != nil {
		return m.saveCandErr
	}
	m.candidates[candidate.ID] = candidate
	return nil
}
func (m *mockVotingRepo) GetCandidate(id string) (*domainvoting.Candidate, error) {
	return m.candidates[id], nil
}
func (m *mockVotingRepo) UpdateCandidate(*domainvoting.Candidate) error { return nil }
func (m *mockVotingRepo) DeleteCandidate(string) error                  { return nil }
func (m *mockVotingRepo) ListCandidates() ([]*domainvoting.Candidate, error) {
	candidates := make([]*domainvoting.Candidate, 0, len(m.candidates))
	for _, c := range m.candidates {
		candidates = append(candidates, c)
	}
	return candidates, nil
}

func (m *mockVotingRepo) SaveSession(session *domainvoting.Session) error {
	if m.saveSessionErr != nil {
		return m.saveSessionErr
	}
	m.sessions[session.ID] = session
	return nil
}
func (m *mockVotingRepo) GetSession(id string) (*domainvoting.Session, error) {
	return m.sessions[id], nil
}
func (m *mockVotingRepo) UpdateSession(*domainvoting.Session) error      { return nil }
func (m *mockVotingRepo) ListSessions() ([]*domainvoting.Session, error) { return nil, nil }

func TestVotingHandler_RegisterVoter_InvalidJSON(t *testing.T) {
	handler := NewVotingHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voting/register/voter", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.RegisterVoter(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestVotingHandler_RegisterCandidate_InvalidJSON(t *testing.T) {
	handler := NewVotingHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voting/register/candidate", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.RegisterCandidate(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestVotingHandler_CreateSession_InvalidJSON(t *testing.T) {
	handler := NewVotingHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voting/session", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.CreateSession(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestVotingHandler_Vote_InvalidJSON(t *testing.T) {
	handler := NewVotingHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voting/vote", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.Vote(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestVotingHandler_RegisterVoter_Success(t *testing.T) {
	handler := &VotingHandler{repo: newMockVotingRepo(), service: domainvoting.NewEd25519Service()}

	body, _ := json.Marshal(map[string]string{"name": "Alice"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/voting/register/voter", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.RegisterVoter(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

func TestVotingHandler_RegisterVoter_RepoError(t *testing.T) {
	repo := newMockVotingRepo()
	repo.saveVoterErr = assert.AnError
	handler := &VotingHandler{repo: repo, service: domainvoting.NewEd25519Service()}

	body, _ := json.Marshal(map[string]string{"name": "Alice"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/voting/register/voter", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.RegisterVoter(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestVotingHandler_RegisterCandidate_Success(t *testing.T) {
	handler := &VotingHandler{repo: newMockVotingRepo(), service: domainvoting.NewEd25519Service()}

	body, _ := json.Marshal(map[string]string{"name": "Bob", "party": "ABC", "program": "Do stuff"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/voting/register/candidate", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.RegisterCandidate(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

func TestVotingHandler_RegisterCandidate_RepoError(t *testing.T) {
	repo := newMockVotingRepo()
	repo.saveCandErr = assert.AnError
	handler := &VotingHandler{repo: repo, service: domainvoting.NewEd25519Service()}

	body, _ := json.Marshal(map[string]string{"name": "Bob", "party": "ABC", "program": "Do stuff"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/voting/register/candidate", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.RegisterCandidate(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestVotingHandler_CreateSession_Success(t *testing.T) {
	repo := newMockVotingRepo()
	repo.candidates["cand1"] = &domainvoting.Candidate{ID: "cand1", Name: "Cand1"}
	handler := &VotingHandler{repo: repo, service: domainvoting.NewEd25519Service()}

	body, _ := json.Marshal(map[string]interface{}{
		"title":         "Test Session",
		"description":   "A test",
		"candidate_ids": []string{"cand1"},
		"start_time":    1000,
		"end_time":      2000,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/voting/session", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.CreateSession(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

func TestVotingHandler_CreateSession_RepoError(t *testing.T) {
	repo := newMockVotingRepo()
	repo.saveSessionErr = assert.AnError
	repo.candidates["cand1"] = &domainvoting.Candidate{ID: "cand1", Name: "Cand1"}
	handler := &VotingHandler{repo: repo, service: domainvoting.NewEd25519Service()}

	body, _ := json.Marshal(map[string]interface{}{
		"title":         "Test Session",
		"description":   "A test",
		"candidate_ids": []string{"cand1"},
		"start_time":    1000,
		"end_time":      2000,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/voting/session", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.CreateSession(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestVotingHandler_Vote_Success(t *testing.T) {
	handler := &VotingHandler{repo: newMockVotingRepo(), service: domainvoting.NewEd25519Service()}

	body, _ := json.Marshal(map[string]string{
		"voter_public_key": "pk1",
		"candidate_id":     "cand1",
		"private_key":      "pk1",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/voting/vote", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.Vote(rr, req)

	assert.NotEqual(t, http.StatusOK, rr.Code)
}

func TestVotingHandler_GetSession_NotFound_Repo(t *testing.T) {
	handler := &VotingHandler{repo: newMockVotingRepo(), service: domainvoting.NewEd25519Service()}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/voting/session/nonexistent", nil)
	rr := httptest.NewRecorder()

	handler.GetSession(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestVotingHandler_Routes(t *testing.T) {
	handler := NewVotingHandler(nil)
	assert.NotNil(t, handler)
}

func TestVotingHandler_NewVotingHandler(t *testing.T) {
	handler := NewVotingHandler(nil)
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.service)
}
