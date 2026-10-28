package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockVotingRepo struct {
	voters         map[string]*voting.Voter
	candidates     map[string]*voting.Candidate
	sessions       map[string]*voting.Session
	saveVoterErr   error
	saveCandErr    error
	saveSessionErr error
	getSessionErr  error

	lastCandidateLimit  int
	lastCandidateOffset int
	lastSessionLimit    int
	lastSessionOffset   int
	lastVoterLimit      int
	lastVoterOffset     int
}

func newMockVotingRepo() *mockVotingRepo {
	return &mockVotingRepo{
		voters:     make(map[string]*voting.Voter),
		candidates: make(map[string]*voting.Candidate),
		sessions:   make(map[string]*voting.Session),
	}
}

func (m *mockVotingRepo) SaveVote(*voting.Vote) error                        { return nil }
func (m *mockVotingRepo) GetVotesByCandidate(string) ([]*voting.Vote, error) { return nil, nil }
func (m *mockVotingRepo) GetVotesByVoter(string) ([]*voting.Vote, error)     { return nil, nil }
func (m *mockVotingRepo) CountVotesBySession(_, candidateID string) (int, error) {
	if c, ok := m.candidates[candidateID]; ok {
		return c.VoteCount, nil
	}
	return 0, nil
}

// WithTx satisfies voting.TransactableRepository; the mock has no
// transaction state, so it returns itself.
func (m *mockVotingRepo) WithTx(_ *sql.Tx) voting.Repository { return m }

func (m *mockVotingRepo) SaveVoter(voter *voting.Voter) error {
	if m.saveVoterErr != nil {
		return m.saveVoterErr
	}
	m.voters[voter.PublicKey] = voter
	return nil
}
func (m *mockVotingRepo) GetVoter(pk string) (*voting.Voter, error) {
	return m.voters[pk], nil
}
func (m *mockVotingRepo) UpdateVoter(*voting.Voter) error { return nil }
func (m *mockVotingRepo) TryMarkVoted(_, _ string) error  { return nil }
func (m *mockVotingRepo) ListVoters(limit, offset int) ([]*voting.Voter, error) {
	m.lastVoterLimit = limit
	m.lastVoterOffset = offset
	voters := make([]*voting.Voter, 0, len(m.voters))
	for _, voter := range m.voters {
		voters = append(voters, voter)
	}
	sort.Slice(voters, func(i, j int) bool { return voters[i].PublicKey < voters[j].PublicKey })
	return pageVotingTestRows(voters, limit, offset), nil
}

func (m *mockVotingRepo) SaveCandidate(candidate *voting.Candidate) error {
	if m.saveCandErr != nil {
		return m.saveCandErr
	}
	m.candidates[candidate.ID] = candidate
	return nil
}
func (m *mockVotingRepo) GetCandidate(id string) (*voting.Candidate, error) {
	return m.candidates[id], nil
}
func (m *mockVotingRepo) IncrementCandidateVoteCount(string) error { return nil }
func (m *mockVotingRepo) ListCandidates(limit, offset int) ([]*voting.Candidate, error) {
	m.lastCandidateLimit = limit
	m.lastCandidateOffset = offset
	candidates := make([]*voting.Candidate, 0, len(m.candidates))
	for _, c := range m.candidates {
		candidates = append(candidates, c)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
	return pageVotingTestRows(candidates, limit, offset), nil
}

func (m *mockVotingRepo) SaveSession(session *voting.Session) error {
	if m.saveSessionErr != nil {
		return m.saveSessionErr
	}
	m.sessions[session.ID] = session
	return nil
}
func (m *mockVotingRepo) GetSession(id string) (*voting.Session, error) {
	if m.getSessionErr != nil {
		return nil, m.getSessionErr
	}
	return m.sessions[id], nil
}
func (m *mockVotingRepo) UpdateSession(*voting.Session) error { return nil }
func (m *mockVotingRepo) ListSessions(limit, offset int) ([]*voting.Session, error) {
	m.lastSessionLimit = limit
	m.lastSessionOffset = offset
	sessions := make([]*voting.Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	sort.Slice(sessions, func(i, j int) bool { return sessions[i].ID < sessions[j].ID })
	return pageVotingTestRows(sessions, limit, offset), nil
}

func (m *mockVotingRepo) Counts() (voting.Counts, error) {
	counts := voting.Counts{
		Candidates: len(m.candidates),
		Sessions:   len(m.sessions),
	}
	for _, candidate := range m.candidates {
		counts.Votes += candidate.VoteCount
	}
	return counts, nil
}

func pageVotingTestRows[T any](rows []T, limit, offset int) []T {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(rows) {
		return nil
	}
	rows = rows[offset:]
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return rows
}

func TestVotingHandler_ListSessions(t *testing.T) {
	repo := newMockVotingRepo()
	repo.sessions["s1"] = &voting.Session{ID: "s1", Title: "Board Vote", Status: "active"}
	repo.sessions["s2"] = &voting.Session{ID: "s2", Title: "Referendum", Status: "closed"}

	h := NewVotingHandler(repo, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/voting/sessions", nil)
	rr := httptest.NewRecorder()

	h.ListSessions(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Board Vote")
	assert.Contains(t, rr.Body.String(), "Referendum")
}

func TestVotingHandler_ListSessions_Empty(t *testing.T) {
	h := NewVotingHandler(newMockVotingRepo(), nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/voting/sessions", nil)
	rr := httptest.NewRecorder()

	h.ListSessions(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, "[]", rr.Body.String())
}

func TestVotingHandler_ListSessionsPagesFromQuery(t *testing.T) {
	repo := newMockVotingRepo()
	repo.sessions["s1"] = &voting.Session{ID: "s1", Title: "Board Vote"}
	repo.sessions["s2"] = &voting.Session{ID: "s2", Title: "Referendum"}

	h := NewVotingHandler(repo, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/voting/sessions?limit=1&offset=1", nil)
	rr := httptest.NewRecorder()

	h.ListSessions(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, 1, repo.lastSessionLimit)
	assert.Equal(t, 1, repo.lastSessionOffset)
	assert.Contains(t, rr.Body.String(), "Referendum")
	assert.NotContains(t, rr.Body.String(), "Board Vote")
}

func TestVotingHandler_ListCandidatesClampsPage(t *testing.T) {
	repo := newMockVotingRepo()
	h := NewVotingHandler(repo, nil)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/voting/candidates?limit=999&offset=999999", nil)
	rr := httptest.NewRecorder()

	h.ListCandidates(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, maxVotingListLimit, repo.lastCandidateLimit)
	assert.Equal(t, maxVotingListOffset, repo.lastCandidateOffset)
}

func TestVotingHandler_ListDefaultsToFirstPage(t *testing.T) {
	repo := newMockVotingRepo()
	h := NewVotingHandler(repo, nil)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/voting/candidates?limit=nope&offset=-1", nil)
	rr := httptest.NewRecorder()

	h.ListCandidates(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, defaultVotingListLimit, repo.lastCandidateLimit)
	assert.Equal(t, 0, repo.lastCandidateOffset)
}

func TestVotingHandler_ListVotersPagesWithoutPrivateKeys(t *testing.T) {
	repo := newMockVotingRepo()
	repo.voters["pk-a"] = &voting.Voter{PublicKey: "pk-a", Name: "Alice"}
	repo.voters["pk-b"] = &voting.Voter{PublicKey: "pk-b", Name: "Bob", HasVoted: true}
	h := NewVotingHandler(repo, nil)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/voting/voters?limit=1&offset=1", nil)
	rr := httptest.NewRecorder()

	h.ListVoters(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, 1, repo.lastVoterLimit)
	require.Equal(t, 1, repo.lastVoterOffset)
	var voters []struct {
		Name      string `json:"name"`
		PublicKey string `json:"public_key"`
		HasVoted  bool   `json:"has_voted"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &voters))
	require.Len(t, voters, 1)
	require.Equal(t, "Bob", voters[0].Name)
	require.Equal(t, "pk-b", voters[0].PublicKey)
	require.True(t, voters[0].HasVoted)
	require.NotContains(t, rr.Body.String(), "private_key",
		"a list read must not expose the one-time registration private key")
}

func TestVotingHandler_RegisterVoter_InvalidJSON(t *testing.T) {
	handler := NewVotingHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voting/register/voter", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.RegisterVoter(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestVotingHandler_RegisterCandidate_InvalidJSON(t *testing.T) {
	handler := NewVotingHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voting/register/candidate", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.RegisterCandidate(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestVotingHandler_CreateSession_InvalidJSON(t *testing.T) {
	handler := NewVotingHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voting/session", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.CreateSession(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestVotingHandler_Vote_InvalidJSON(t *testing.T) {
	handler := NewVotingHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voting/vote", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.Vote(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestVotingHandler_RegisterVoter_Success(t *testing.T) {
	handler := &VotingHandler{repo: newMockVotingRepo(), service: voting.NewEd25519Service()}

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
	handler := &VotingHandler{repo: repo, service: voting.NewEd25519Service()}

	body, _ := json.Marshal(map[string]string{"name": "Alice"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/voting/register/voter", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.RegisterVoter(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestVotingHandler_RegisterCandidate_Success(t *testing.T) {
	handler := &VotingHandler{repo: newMockVotingRepo(), service: voting.NewEd25519Service()}

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
	handler := &VotingHandler{repo: repo, service: voting.NewEd25519Service()}

	body, _ := json.Marshal(map[string]string{"name": "Bob", "party": "ABC", "program": "Do stuff"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/voting/register/candidate", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.RegisterCandidate(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestVotingHandler_CreateSession_Success(t *testing.T) {
	repo := newMockVotingRepo()
	repo.candidates["cand1"] = &voting.Candidate{ID: "cand1", Name: "Cand1"}
	handler := &VotingHandler{repo: repo, service: voting.NewEd25519Service()}

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
	repo.candidates["cand1"] = &voting.Candidate{ID: "cand1", Name: "Cand1"}
	handler := &VotingHandler{repo: repo, service: voting.NewEd25519Service()}

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
	repo := newMockVotingRepo()
	handler := &VotingHandler{repo: repo, service: voting.NewEd25519Service()}

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
	handler := &VotingHandler{repo: newMockVotingRepo(), service: voting.NewEd25519Service()}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/voting/session/nonexistent", nil)
	rr := httptest.NewRecorder()

	handler.GetSession(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "SESSION_NOT_FOUND")
}

// TestVotingHandler_GetSession_ServerErrorIsNot404 guards the error
// classification fix (TASK-171, ISS-166): a genuine repository failure (lock
// contention, disk fault) must surface as an unclassified 500, not be masked
// as "not found" — otherwise an outage looks like a missing resource and
// misdirects clients to re-check IDs.
func TestVotingHandler_GetSession_ServerErrorIsNot404(t *testing.T) {
	repo := newMockVotingRepo()
	repo.getSessionErr = errors.New("database is locked")
	handler := &VotingHandler{repo: repo, service: voting.NewEd25519Service()}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/voting/session/s1", nil)
	rr := httptest.NewRecorder()

	handler.GetSession(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	require.NotContains(t, rr.Body.String(), "NOT_FOUND",
		"a real DB error must not masquerade as a missing session (ISS-166)")
	require.NotContains(t, rr.Body.String(), "database is locked",
		"the raw internal error must not leak to the client")
}

func TestVotingHandler_Routes(t *testing.T) {
	handler := NewVotingHandler(nil, nil)
	assert.NotNil(t, handler)
}

func TestVotingHandler_NewVotingHandler(t *testing.T) {
	handler := NewVotingHandler(nil, nil)
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.service)
}

func TestVotingHandler_Vote_MissingSessionID(t *testing.T) {
	repo := newMockVotingRepo()
	handler := &VotingHandler{repo: repo, service: voting.NewEd25519Service()}

	body, _ := json.Marshal(map[string]string{
		"voter_public_key": "pk1",
		"candidate_id":     "cand1",
		"private_key":      "pk1",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/voting/vote", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.Vote(rr, req)

	// No session_id provided → session not found → 404 with SESSION_NOT_FOUND code
	assert.Equal(t, http.StatusNotFound, rr.Code)

	var resp ErrorResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "SESSION_NOT_FOUND", resp.Code)
}

func TestVotingHandler_StartEndSession(t *testing.T) {
	repo := newMockVotingRepo()
	repo.sessions["s1"] = &voting.Session{ID: "s1", Title: "Election", Status: "draft"}
	handler := NewVotingHandler(repo, nil)

	setParam := func(h http.HandlerFunc, method, id string) *httptest.ResponseRecorder {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", id)
		req := httptest.NewRequest(method, "/api/v1/voting/session/"+id, nil).
			WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rctx))
		rr := httptest.NewRecorder()
		h(rr, req)
		return rr
	}

	// Start: draft -> active
	rr := setParam(handler.StartSession, http.MethodPost, "s1")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "active", repo.sessions["s1"].Status)

	// End: active -> ended
	rr = setParam(handler.EndSession, http.MethodPost, "s1")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "ended", repo.sessions["s1"].Status)

	// Missing session -> 404
	rr = setParam(handler.StartSession, http.MethodPost, "nope")
	assert.Equal(t, http.StatusNotFound, rr.Code)
	rr = setParam(handler.EndSession, http.MethodPost, "nope")
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func requestWithParam(method, path, id string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return httptest.NewRequest(method, path, nil).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rctx))
}

func TestVotingHandler_GetResults(t *testing.T) {
	repo := newMockVotingRepo()
	repo.sessions["s1"] = &voting.Session{ID: "s1", Candidates: []string{"c1", "c2"}}
	repo.candidates["c1"] = &voting.Candidate{ID: "c1", Name: "Alice", Party: "A", VoteCount: 5}
	repo.candidates["c2"] = &voting.Candidate{ID: "c2", Name: "Bob", Party: "B", VoteCount: 3}

	h := NewVotingHandler(repo, nil)
	rr := httptest.NewRecorder()
	h.GetResults(rr, requestWithParam(http.MethodGet, "/api/v1/voting/results/s1", "s1"))
	assert.Equal(t, http.StatusOK, rr.Code)

	var res struct {
		SessionID  string `json:"session_id"`
		TotalVotes int    `json:"total_votes"`
		Candidates []struct {
			ID        string `json:"id"`
			VoteCount int    `json:"vote_count"`
		} `json:"candidates"`
	}
	assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &res))
	assert.Equal(t, "s1", res.SessionID)
	assert.Equal(t, 8, res.TotalVotes)
	assert.Len(t, res.Candidates, 2)
}

func TestVotingHandler_GetResults_NotFound(t *testing.T) {
	repo := newMockVotingRepo()
	h := NewVotingHandler(repo, nil)

	rr := httptest.NewRecorder()
	h.GetResults(rr, requestWithParam(http.MethodGet, "/api/v1/voting/results/missing", "missing"))
	assert.Equal(t, http.StatusNotFound, rr.Code)
}
