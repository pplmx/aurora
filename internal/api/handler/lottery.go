package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	lotteryapp "github.com/pplmx/aurora/internal/app/lottery"
	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/infra/sqlite"
)

type LotteryHandler struct{}

func NewLotteryHandler() *LotteryHandler {
	return &LotteryHandler{}
}

type CreateLotteryRequest struct {
	Participants string `json:"participants"`
	Seed         string `json:"seed"`
	WinnerCount  int    `json:"winner_count"`
}

func (h *LotteryHandler) Routes(r chi.Router) {
	r.Post("/create", h.Create)
	r.Get("/history", h.History)
	r.Get("/{id}", h.Get)
}

func (h *LotteryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateLotteryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request","code":"INVALID_REQUEST"}`, http.StatusBadRequest)
		return
	}

	lotteryRepo, err := sqlite.NewLotteryRepository(blockchain.DBPath())
	if err != nil {
		http.Error(w, `{"error":"failed to create repository","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}
	defer func() { _ = lotteryRepo.Close() }()

	blockChain := blockchain.InitBlockChain()
	uc := lotteryapp.NewCreateLotteryUseCase(lotteryRepo, blockChain)

	appReq := lotteryapp.CreateLotteryRequest{
		Participants: req.Participants,
		Seed:         req.Seed,
		WinnerCount:  req.WinnerCount,
	}

	result, err := uc.Execute(appReq)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *LotteryHandler) History(w http.ResponseWriter, r *http.Request) {
	lotteryRepo, err := sqlite.NewLotteryRepository(blockchain.DBPath())
	if err != nil {
		http.Error(w, `{"error":"failed to create repository","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}
	defer func() { _ = lotteryRepo.Close() }()

	results, err := lotteryRepo.GetAll()
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (h *LotteryHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	lotteryRepo, err := sqlite.NewLotteryRepository(blockchain.DBPath())
	if err != nil {
		http.Error(w, `{"error":"failed to create repository","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}
	defer func() { _ = lotteryRepo.Close() }()

	result, err := lotteryRepo.GetByID(id)
	if err != nil {
		http.Error(w, `{"error":"not found","code":"NOT_FOUND"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
