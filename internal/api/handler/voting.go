package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/pplmx/aurora/internal/app/votingapp"
	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/pplmx/aurora/internal/ledger"
)

const (
	defaultVotingListLimit = 20
	maxVotingListLimit     = 100
	maxVotingListOffset    = 10000
)

type VotingHandler struct {
	repo      voting.TransactableRepository
	txManager voting.TransactionManager
	service   voting.Service
	chain     ledger.BlockWriter
}

// NewVotingHandler wires the voting use cases over a transaction-capable
// repository. txManager may be nil (handler tests); CastVote then falls back
// to non-transactional writes via the use case's nil guard.
func NewVotingHandler(repo voting.TransactableRepository, txManager voting.TransactionManager) *VotingHandler {
	return &VotingHandler{
		repo:      repo,
		txManager: txManager,
		service:   voting.NewEd25519Service(),
	}
}

// SetChain wires the blockchain writer so CastVote records accepted ballots
// on-chain (documented "blockchain-based vote recording"). Optional and
// nil-safe; the API server supplies it, handler tests leave it nil.
func (h *VotingHandler) SetChain(chain ledger.BlockWriter) {
	h.chain = chain
}

func (h *VotingHandler) Routes(r chi.Router) {
	r.Post("/register/voter", h.RegisterVoter)
	r.Post("/register/candidate", h.RegisterCandidate)
	r.Post("/session", h.CreateSession)
	r.Post("/session/{id}/start", h.StartSession)
	r.Post("/session/{id}/end", h.EndSession)
	r.Post("/vote", h.Vote)
	r.Get("/candidates", h.ListCandidates)
	r.Get("/sessions", h.ListSessions)
	r.Get("/voters", h.ListVoters)
	r.Get("/session/{id}", h.GetSession)
	r.Get("/results/{id}", h.GetResults)
}

func (h *VotingHandler) RegisterVoter(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	uc := votingapp.NewRegisterVoterUseCase(h.repo)
	result, err := uc.Execute(votingapp.RegisterVoterRequest{Name: req.Name})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *VotingHandler) RegisterCandidate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		Party   string `json:"party"`
		Program string `json:"program"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	uc := votingapp.NewRegisterCandidateUseCase(h.repo)
	result, err := uc.Execute(votingapp.RegisterCandidateRequest{
		Name:    req.Name,
		Party:   req.Party,
		Program: req.Program,
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *VotingHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title        string   `json:"title"`
		Description  string   `json:"description"`
		CandidateIDs []string `json:"candidate_ids"`
		StartTime    int64    `json:"start_time"`
		EndTime      int64    `json:"end_time"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	uc := votingapp.NewCreateSessionUseCase(h.repo)
	result, err := uc.Execute(votingapp.CreateSessionRequest{
		Title:        req.Title,
		Description:  req.Description,
		CandidateIDs: req.CandidateIDs,
		StartTime:    req.StartTime,
		EndTime:      req.EndTime,
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *VotingHandler) Vote(w http.ResponseWriter, r *http.Request) {
	var req struct {
		VoterPublicKey string `json:"voter_public_key"`
		CandidateID    string `json:"candidate_id"`
		PrivateKey     string `json:"private_key"`
		SessionID      string `json:"session_id"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	uc := votingapp.NewCastVoteUseCase(h.repo, h.service, h.txManager)
	uc.SetChain(h.chain)
	result, err := uc.Execute(votingapp.CastVoteRequest{
		VoterPublicKey: req.VoterPublicKey,
		CandidateID:    req.CandidateID,
		PrivateKey:     req.PrivateKey,
		SessionID:      req.SessionID,
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *VotingHandler) ListCandidates(w http.ResponseWriter, r *http.Request) {
	limit, offset := votingPage(r)
	uc := votingapp.NewGetCandidatesUseCase(h.repo)
	result, err := uc.Execute(&votingapp.ListCandidatesRequest{Limit: limit, Offset: offset})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *VotingHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	limit, offset := votingPage(r)
	uc := votingapp.NewListSessionsUseCase(h.repo)
	result, err := uc.Execute(&votingapp.ListSessionsRequest{Limit: limit, Offset: offset})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *VotingHandler) ListVoters(w http.ResponseWriter, r *http.Request) {
	limit, offset := votingPage(r)
	uc := votingapp.NewListVotersUseCase(h.repo)
	result, err := uc.Execute(&votingapp.ListVotersRequest{Limit: limit, Offset: offset})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// votingPage parses the candidate/session/voter list page with the same
// forgiving, capped rules as the other collection endpoints: malformed or
// non-positive limits use the default, and offsets clamp instead of
// overflowing storage.
func votingPage(r *http.Request) (limit, offset int) {
	limit = defaultVotingListLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = min(parsed, maxVotingListLimit)
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = min(parsed, maxVotingListOffset)
		}
	}
	return limit, offset
}

func (h *VotingHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Route through the use case (sibling pattern: ListSessions/GetResults) so
	// a missing session is a 404 SESSION_NOT_FOUND while a genuine DB failure
	// surfaces as an unclassified 500 — previously EVERY GetSession error was
	// flattened into 404 "not found" (TASK-171, ISS-166).
	uc := votingapp.NewGetSessionUseCase(h.repo)
	session, err := uc.Execute(id)
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, votingapp.SessionResponseOf(session))
}

// GetResults returns the per-candidate tally for a session (v1.23 Voting
// Results API), closing the CLI-only `voting results` parity gap.
func (h *VotingHandler) GetResults(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uc := votingapp.NewGetResultsUseCase(h.repo)
	result, err := uc.Execute(id)
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// updateSessionStatus applies a status transition to an existing session,
// mirroring the CLI's `session start` / `session end`. This closes the
// CLI↔API parity gap: the REST API previously had no way to drive the session
// lifecycle because CastVote admits ballots only while a session is active.
func (h *VotingHandler) updateSessionStatus(w http.ResponseWriter, r *http.Request, status string) {
	id := chi.URLParam(r, "id")

	// Same not-found-vs-server-fault split as GetSession (TASK-171, ISS-166).
	uc := votingapp.NewGetSessionUseCase(h.repo)
	session, err := uc.Execute(id)
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	session.Status = status
	if err := h.repo.UpdateSession(session); err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, votingapp.SessionResponseOf(session))
}

func (h *VotingHandler) StartSession(w http.ResponseWriter, r *http.Request) {
	h.updateSessionStatus(w, r, voting.SessionStatusActive)
}

func (h *VotingHandler) EndSession(w http.ResponseWriter, r *http.Request) {
	h.updateSessionStatus(w, r, voting.SessionStatusEnded)
}
