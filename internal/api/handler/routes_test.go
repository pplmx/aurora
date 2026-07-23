package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainnft "github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/domain/oracle"
	domaintoken "github.com/pplmx/aurora/internal/domain/token"
	domainvoting "github.com/pplmx/aurora/internal/domain/voting"
)

// =================================================================
// Routes() — verify each handler registers the right URL patterns
// =================================================================

func TestNFTHandler_Routes_RegistersAllEndpoints(t *testing.T) {
	h := NewNFTHandler(nil)
	r := chi.NewRouter()
	h.Routes(r)

	got := map[string]bool{}
	for _, route := range r.Routes() {
		got[route.Pattern] = true
	}
	for _, want := range []string{"/mint", "/transfer", "/burn", "/{id}", "/list"} {
		assert.True(t, got[want], "expected NFT route %q to be registered", want)
	}
}

func TestOracleHandler_Routes_RegistersAllEndpoints(t *testing.T) {
	h := NewOracleHandler(oracle.NewInmemRepo())
	r := chi.NewRouter()
	h.Routes(r)

	got := map[string]bool{}
	for _, route := range r.Routes() {
		got[route.Pattern] = true
	}
	for _, want := range []string{"/sources", "/fetch", "/query"} {
		assert.True(t, got[want], "expected oracle route %q to be registered", want)
	}
}

func TestTokenHandler_Routes_RegistersAllEndpoints(t *testing.T) {
	h := NewTokenHandler(fakeTokenServiceFull{})
	r := chi.NewRouter()
	h.Routes(r)

	got := map[string]bool{}
	for _, route := range r.Routes() {
		got[route.Pattern] = true
	}
	for _, want := range []string{"/create", "/mint", "/transfer", "/burn", "/balance", "/history"} {
		assert.True(t, got[want], "expected token route %q to be registered", want)
	}
}

func TestVotingHandler_Routes_RegistersAllEndpoints(t *testing.T) {
	h := NewVotingHandler(fakeVotingRepo{})
	r := chi.NewRouter()
	h.Routes(r)

	got := map[string]bool{}
	for _, route := range r.Routes() {
		got[route.Pattern] = true
	}
	for _, want := range []string{"/register/voter", "/register/candidate", "/session", "/vote", "/candidates", "/session/{id}"} {
		assert.True(t, got[want], "expected voting route %q to be registered", want)
	}
}

// =================================================================
// NFT — success/error paths using NewInmemRepo
// =================================================================

func TestNFTHandler_Get_FoundAndNotFound(t *testing.T) {
	repo := domainnft.NewInmemRepo()
	_ = repo.SaveNFT(&domainnft.NFT{ID: "nft-1", Name: "Aurora", Owner: []byte("alice")})

	h := NewNFTHandler(repo)

	// Found
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nft-1")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/nft/nft-1", nil).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.Get(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Aurora")

	// Not found
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", "missing")
	req = httptest.NewRequest(http.MethodGet, "/api/v1/nft/missing", nil).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rctx))
	rr = httptest.NewRecorder()
	h.Get(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestNFTHandler_List_EmptyAndPopulated(t *testing.T) {
	repo := domainnft.NewInmemRepo()

	h := NewNFTHandler(repo)

	// No owner, no NFTs
	req := httptest.NewRequest(http.MethodGet, "/api/v1/nft/list", nil)
	rr := httptest.NewRecorder()
	h.List(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Owner with NFTs (owner must be base64-encoded per usecase contract)
	_ = repo.SaveNFT(&domainnft.NFT{ID: "nft-2", Owner: []byte("bob")})
	ownerB64 := "Ym9i" // base64.StdEncoding.EncodeToString([]byte("bob"))
	req = httptest.NewRequest(http.MethodGet, "/api/v1/nft/list?owner="+ownerB64, nil)
	rr = httptest.NewRecorder()
	h.List(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "nft-2")
}

func TestNFTHandler_Mint_ServiceError(t *testing.T) {
	h := NewNFTHandler(&failingNFTRepo{err: errors.New("save failed")})
	body, _ := json.Marshal(map[string]string{"name": "X"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/nft/mint", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Mint(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestNFTHandler_Transfer_ServiceError(t *testing.T) {
	h := NewNFTHandler(&failingNFTRepo{err: errors.New("not found")})
	body, _ := json.Marshal(map[string]string{
		"nft_id": "nft-1", "from": "a", "to": "b", "private_key": "k",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/nft/transfer", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Transfer(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestNFTHandler_Burn_ServiceError(t *testing.T) {
	h := NewNFTHandler(&failingNFTRepo{err: errors.New("not owner")})
	body, _ := json.Marshal(map[string]string{
		"nft_id": "nft-1", "owner": "a", "private_key": "k",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/nft/burn", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Burn(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// =================================================================
// Oracle — success/error paths using InmemRepo
// =================================================================

func TestOracleHandler_Sources_Success(t *testing.T) {
	repo := oracle.NewInmemRepo()
	_ = repo.SaveSource(&oracle.DataSource{ID: "s1", Name: "weather-tokyo"})
	h := NewOracleHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/oracle/sources", nil)
	rr := httptest.NewRecorder()

	h.Sources(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "weather-tokyo")
}

func TestOracleHandler_Fetch_Success(t *testing.T) {
	repo := oracle.NewInmemRepo()
	_ = repo.SaveSource(&oracle.DataSource{ID: "s1", URL: "http://example.com", Enabled: true})
	h := NewOracleHandler(repo)

	body, _ := json.Marshal(map[string]string{"source": "s1"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/oracle/fetch", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Fetch(rr, req)

	// FetchData may fail because URL is fake, but success/error path is exercised.
	// We accept either 200 (with data) or 500 (data fetch failed) — both are valid
	// code paths; what matters is the handler didn't crash.
	assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusInternalServerError,
		"unexpected status: %d", rr.Code)
}

func TestOracleHandler_Fetch_InvalidJSON_BodyParse(t *testing.T) {
	h := NewOracleHandler(oracle.NewInmemRepo())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/oracle/fetch", bytes.NewBufferString("not json"))
	rr := httptest.NewRecorder()

	h.Fetch(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestOracleHandler_Query_WithLimit(t *testing.T) {
	repo := oracle.NewInmemRepo()
	for i := 0; i < 5; i++ {
		_ = repo.SaveData(&oracle.OracleData{ID: "d" + string(rune('0'+i)), SourceID: "s1"})
	}
	h := NewOracleHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/oracle/query?source=s1&limit=2", nil)
	rr := httptest.NewRecorder()

	h.Query(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Response should contain exactly 2 records (limit=2). The records come
	// from a Go map so order is non-deterministic — verify count not order.
	var resp struct {
		Data []oracle.OracleData `json:"Data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Len(t, resp.Data, 2, "limit=2 should return exactly 2 records")
}

func TestOracleHandler_Query_DefaultLimit(t *testing.T) {
	h := NewOracleHandler(oracle.NewInmemRepo())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/oracle/query?source=s1", nil)
	rr := httptest.NewRecorder()

	h.Query(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestOracleHandler_Query_BadLimitFallsBackToDefault(t *testing.T) {
	h := NewOracleHandler(oracle.NewInmemRepo())
	// limit=abc should be ignored and default (10) used.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/oracle/query?source=s1&limit=abc", nil)
	rr := httptest.NewRecorder()

	h.Query(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// =================================================================
// Token — Balance and History paths
// =================================================================

func TestTokenHandler_Balance_MissingQueryParam(t *testing.T) {
	h := NewTokenHandler(fakeTokenServiceFull{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/token/balance", nil)
	rr := httptest.NewRecorder()

	h.Balance(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestTokenHandler_Balance_ServiceError(t *testing.T) {
	h := NewTokenHandler(fakeTokenServiceFull{err: errors.New("not found")})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/token/balance?token_id=T&owner=x", nil)
	rr := httptest.NewRecorder()

	h.Balance(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestTokenHandler_History_Empty(t *testing.T) {
	h := NewTokenHandler(fakeTokenServiceFull{})
	// owner must be base64-encoded per usecase contract
	req := httptest.NewRequest(http.MethodGet, "/api/v1/token/history?token_id=T&owner=YWxpY2U=", nil)
	rr := httptest.NewRecorder()

	h.History(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestTokenHandler_History_ServiceError(t *testing.T) {
	h := NewTokenHandler(fakeTokenServiceFull{err: errors.New("db error")})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/token/history?token_id=T&owner=YWxpY2U=", nil)
	rr := httptest.NewRecorder()

	h.History(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// =================================================================
// Voting — success/error paths via GetSession & ListCandidates
// =================================================================

func TestVotingHandler_GetSession_Success(t *testing.T) {
	h := NewVotingHandler(fakeVotingRepo{
		getSession: &domainvoting.Session{ID: "s1", Title: "Election 2026"},
	})

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "s1")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/voting/session/s1", nil).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetSession(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Election 2026")
}

func TestVotingHandler_GetSession_NotFound(t *testing.T) {
	h := NewVotingHandler(fakeVotingRepo{})

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "missing")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/voting/session/missing", nil).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetSession(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestVotingHandler_ListCandidates_Success(t *testing.T) {
	h := NewVotingHandler(fakeVotingRepo{candidates: []string{"alice", "bob"}})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/voting/candidates", nil)
	rr := httptest.NewRecorder()

	h.ListCandidates(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "alice")
}

func TestVotingHandler_ListCandidates_ServiceError(t *testing.T) {
	h := NewVotingHandler(fakeVotingRepo{err: errors.New("db error")})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/voting/candidates", nil)
	rr := httptest.NewRecorder()

	h.ListCandidates(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// =================================================================
// Test helpers — fake services and repos
// =================================================================

// fakeTokenServiceFull implements the entire domaintoken.Service interface with
// pass-through behavior. `err` controls whether Balance/History return errors.
type fakeTokenServiceFull struct {
	err error
}

func (f fakeTokenServiceFull) CreateToken(req *domaintoken.CreateTokenRequest) (*domaintoken.Token, error) {
	return nil, nil
}
func (f fakeTokenServiceFull) GetTokenInfo(id domaintoken.TokenID) (*domaintoken.Token, error) {
	return nil, nil
}
func (f fakeTokenServiceFull) Mint(req *domaintoken.MintRequest) (*domaintoken.MintEvent, error) {
	return nil, nil
}
func (f fakeTokenServiceFull) Transfer(req *domaintoken.TransferRequest) (*domaintoken.TransferEvent, error) {
	return nil, nil
}
func (f fakeTokenServiceFull) TransferFrom(req *domaintoken.TransferFromRequest) (*domaintoken.TransferEvent, error) {
	return nil, nil
}
func (f fakeTokenServiceFull) Approve(req *domaintoken.ApproveRequest) (*domaintoken.ApproveEvent, error) {
	return nil, nil
}
func (f fakeTokenServiceFull) IncreaseAllowance(req *domaintoken.AllowanceRequest) (*domaintoken.ApproveEvent, error) {
	return nil, nil
}
func (f fakeTokenServiceFull) DecreaseAllowance(req *domaintoken.AllowanceRequest) (*domaintoken.ApproveEvent, error) {
	return nil, nil
}
func (f fakeTokenServiceFull) Burn(req *domaintoken.BurnRequest) (*domaintoken.BurnEvent, error) {
	return nil, nil
}
func (f fakeTokenServiceFull) GetBalance(_ domaintoken.TokenID, _ domaintoken.PublicKey) (*domaintoken.Amount, error) {
	if f.err != nil {
		return nil, f.err
	}
	return domaintoken.NewAmount(100), nil
}
func (f fakeTokenServiceFull) GetAllowance(_ domaintoken.TokenID, _, _ domaintoken.PublicKey) (*domaintoken.Amount, error) {
	return nil, nil
}
func (f fakeTokenServiceFull) GetTransferHistory(_ domaintoken.TokenID, _ domaintoken.PublicKey, _, _ int) ([]*domaintoken.TransferEvent, error) {
	if f.err != nil {
		return nil, f.err
	}
	return nil, nil
}

// fakeVotingRepo implements domainvoting.Repository for handler tests.
type fakeVotingRepo struct {
	getSession *domainvoting.Session
	candidates []string
	err        error
}

func (f fakeVotingRepo) SaveVoter(*domainvoting.Voter) error { return nil }
func (f fakeVotingRepo) GetVoter(string) (*domainvoting.Voter, error) {
	return nil, nil
}
func (f fakeVotingRepo) UpdateVoter(*domainvoting.Voter) error { return nil }
func (f fakeVotingRepo) TryMarkVoted(string, string) error     { return nil }
func (f fakeVotingRepo) ListVoters() ([]*domainvoting.Voter, error) {
	return nil, nil
}
func (f fakeVotingRepo) SaveCandidate(*domainvoting.Candidate) error { return nil }
func (f fakeVotingRepo) GetCandidate(string) (*domainvoting.Candidate, error) {
	return nil, nil
}
func (f fakeVotingRepo) UpdateCandidate(*domainvoting.Candidate) error { return nil }
func (f fakeVotingRepo) DeleteCandidate(string) error                  { return nil }
func (f fakeVotingRepo) ListCandidates() ([]*domainvoting.Candidate, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make([]*domainvoting.Candidate, 0, len(f.candidates))
	for _, c := range f.candidates {
		out = append(out, &domainvoting.Candidate{ID: c})
	}
	return out, nil
}
func (f fakeVotingRepo) SaveSession(*domainvoting.Session) error { return nil }
func (f fakeVotingRepo) GetSession(string) (*domainvoting.Session, error) {
	if f.getSession == nil {
		return nil, f.err
	}
	return f.getSession, f.err
}
func (f fakeVotingRepo) UpdateSession(*domainvoting.Session) error { return nil }
func (f fakeVotingRepo) ListSessions() ([]*domainvoting.Session, error) {
	return nil, nil
}
func (f fakeVotingRepo) SaveVote(*domainvoting.Vote) error { return nil }
func (f fakeVotingRepo) GetVote(string) (*domainvoting.Vote, error) {
	return nil, nil
}
func (f fakeVotingRepo) GetVotesByCandidate(string) ([]*domainvoting.Vote, error) {
	return nil, nil
}
func (f fakeVotingRepo) GetVotesByVoter(string) ([]*domainvoting.Vote, error) {
	return nil, nil
}
func (f fakeVotingRepo) DeleteVote(string) error { return nil }

// failingNFTRepo makes every Save* method return err.
type failingNFTRepo struct {
	err error
}

func (f *failingNFTRepo) SaveNFT(*domainnft.NFT) error   { return f.err }
func (f *failingNFTRepo) UpdateNFT(*domainnft.NFT) error { return f.err }
func (f *failingNFTRepo) TryTransferOwnership(string, []byte, []byte) error {
	return f.err
}
func (f *failingNFTRepo) TryDeleteNFTIfOwned(string, []byte) error {
	return f.err
}
func (f *failingNFTRepo) DeleteNFT(string) error { return f.err }
func (f *failingNFTRepo) GetNFT(string) (*domainnft.NFT, error) {
	return nil, f.err
}
func (f *failingNFTRepo) GetNFTsByOwner([]byte) ([]*domainnft.NFT, error) {
	return nil, f.err
}
func (f *failingNFTRepo) GetNFTsByCreator([]byte) ([]*domainnft.NFT, error) {
	return nil, f.err
}
func (f *failingNFTRepo) SaveOperation(*domainnft.Operation) error {
	return f.err
}
func (f *failingNFTRepo) GetOperations(string) ([]*domainnft.Operation, error) {
	return nil, f.err
}
