package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	votingapp "github.com/pplmx/aurora/internal/app/voting"
	"github.com/pplmx/aurora/internal/domain/blockchain"
	domainvoting "github.com/pplmx/aurora/internal/domain/voting"
)

type VotingHandler struct {
	repo      domainvoting.TransactableRepository
	txManager domainvoting.TransactionManager
	service   domainvoting.Service
	chain     blockchain.BlockWriter
}

// NewVotingHandler wires the voting use cases over a transaction-capable
// repository. txManager may be nil (handler tests); CastVote then falls back
// to non-transactional writes via the use case's nil guard.
func NewVotingHandler(repo domainvoting.TransactableRepository, txManager domainvoting.TransactionManager) *VotingHandler {
	return &VotingHandler{
		repo:      repo,
		txManager: txManager,
		service:   domainvoting.NewEd25519Service(),
	}
}

// SetChain wires the blockchain writer so CastVote records accepted ballots
// on-chain (documented "blockchain-based vote recording"). Optional and
// nil-safe; the API server supplies it, handler tests leave it nil.
func (h *VotingHandler) SetChain(chain blockchain.BlockWriter) {
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
	r.Get("/session/{id}", h.GetSession)
	r.Get("/results/{id}", h.GetResults)
}

func (h *VotingHandler) RegisterVoter(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid request")
		return
	}

	uc := votingapp.NewRegisterVoterUseCase(h.repo)
	result, err := uc.Execute(votingapp.RegisterVoterRequest{Name: req.Name})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *VotingHandler) RegisterCandidate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		Party   string `json:"party"`
		Program string `json:"program"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid request")
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

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *VotingHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title        string   `json:"title"`
		Description  string   `json:"description"`
		CandidateIDs []string `json:"candidate_ids"`
		StartTime    int64    `json:"start_time"`
		EndTime      int64    `json:"end_time"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid request")
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

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *VotingHandler) Vote(w http.ResponseWriter, r *http.Request) {
	var req struct {
		VoterPublicKey string `json:"voter_public_key"`
		CandidateID    string `json:"candidate_id"`
		PrivateKey     string `json:"private_key"`
		SessionID      string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid request")
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

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *VotingHandler) ListCandidates(w http.ResponseWriter, r *http.Request) {
	uc := votingapp.NewGetCandidatesUseCase(h.repo)
	result, err := uc.Execute()
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *VotingHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	uc := votingapp.NewListSessionsUseCase(h.repo)
	result, err := uc.Execute()
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *VotingHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	session, err := h.repo.GetSession(id)
	if err != nil || session == nil {
		writeError(w, "not found", "NOT_FOUND", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(session)
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

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// updateSessionStatus applies a status transition to an existing session,
// mirroring the CLI's `session start` / `session end`. This closes the
// CLI↔API parity gap: the REST API previously had no way to drive the session
// lifecycle, even though the voting flow now rejects votes on "ended"
// sessions (see CastVote).
func (h *VotingHandler) updateSessionStatus(w http.ResponseWriter, r *http.Request, status string) {
	id := chi.URLParam(r, "id")

	session, err := h.repo.GetSession(id)
	if err != nil || session == nil {
		writeError(w, "not found", "NOT_FOUND", http.StatusNotFound)
		return
	}

	session.Status = status
	if err := h.repo.UpdateSession(session); err != nil {
		writeError(w, "internal server error", "INTERNAL_ERROR", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(session)
}

func (h *VotingHandler) StartSession(w http.ResponseWriter, r *http.Request) {
	h.updateSessionStatus(w, r, "active")
}

func (h *VotingHandler) EndSession(w http.ResponseWriter, r *http.Request) {
	h.updateSessionStatus(w, r, "ended")
}
