package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	votingapp "github.com/pplmx/aurora/internal/app/voting"
	domainvoting "github.com/pplmx/aurora/internal/domain/voting"
)

type VotingHandler struct {
	repo    domainvoting.Repository
	service domainvoting.Service
}

func NewVotingHandler(repo domainvoting.Repository) *VotingHandler {
	return &VotingHandler{
		repo:    repo,
		service: domainvoting.NewEd25519Service(),
	}
}

func (h *VotingHandler) Routes(r chi.Router) {
	r.Post("/register/voter", h.RegisterVoter)
	r.Post("/register/candidate", h.RegisterCandidate)
	r.Post("/session", h.CreateSession)
	r.Post("/vote", h.Vote)
	r.Get("/candidates", h.ListCandidates)
	r.Get("/session/{id}", h.GetSession)
}

func (h *VotingHandler) RegisterVoter(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request","code":"INVALID_REQUEST"}`, http.StatusBadRequest)
		return
	}

	uc := votingapp.NewRegisterVoterUseCase(h.repo)
	result, err := uc.Execute(votingapp.RegisterVoterRequest{Name: req.Name})
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *VotingHandler) RegisterCandidate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		Party   string `json:"party"`
		Program string `json:"program"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request","code":"INVALID_REQUEST"}`, http.StatusBadRequest)
		return
	}

	uc := votingapp.NewRegisterCandidateUseCase(h.repo)
	result, err := uc.Execute(votingapp.RegisterCandidateRequest{
		Name:    req.Name,
		Party:   req.Party,
		Program: req.Program,
	})
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
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
		http.Error(w, `{"error":"invalid request","code":"INVALID_REQUEST"}`, http.StatusBadRequest)
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
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *VotingHandler) Vote(w http.ResponseWriter, r *http.Request) {
	var req struct {
		VoterPublicKey string `json:"voter_public_key"`
		CandidateID    string `json:"candidate_id"`
		PrivateKey     string `json:"private_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request","code":"INVALID_REQUEST"}`, http.StatusBadRequest)
		return
	}

	uc := votingapp.NewCastVoteUseCase(h.repo, h.service)
	result, err := uc.Execute(votingapp.CastVoteRequest{
		VoterPublicKey: req.VoterPublicKey,
		CandidateID:    req.CandidateID,
		PrivateKey:     req.PrivateKey,
	})
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *VotingHandler) ListCandidates(w http.ResponseWriter, r *http.Request) {
	uc := votingapp.NewGetCandidatesUseCase(h.repo)
	result, err := uc.Execute()
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *VotingHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	session, err := h.repo.GetSession(id)
	if err != nil || session == nil {
		http.Error(w, `{"error":"not found","code":"NOT_FOUND"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}
