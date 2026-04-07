package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	tokenapp "github.com/pplmx/aurora/internal/app/token"
	"github.com/pplmx/aurora/internal/domain/token"
)

type TokenHandler struct {
	service token.Service
}

func NewTokenHandler(service token.Service) *TokenHandler {
	return &TokenHandler{service: service}
}

func (h *TokenHandler) Routes(r chi.Router) {
	r.Post("/create", h.Create)
	r.Post("/mint", h.Mint)
	r.Post("/transfer", h.Transfer)
	r.Post("/burn", h.Burn)
	r.Get("/balance", h.Balance)
	r.Get("/history", h.History)
}

func (h *TokenHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Symbol      string `json:"symbol"`
		TotalSupply string `json:"total_supply"`
		Owner       string `json:"owner"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request","code":"INVALID_REQUEST"}`, http.StatusBadRequest)
		return
	}

	uc := tokenapp.NewCreateTokenUseCase(h.service)
	result, err := uc.Execute(&tokenapp.CreateTokenRequest{
		Name:        req.Name,
		Symbol:      req.Symbol,
		TotalSupply: req.TotalSupply,
		Owner:       req.Owner,
	})
	if err != nil {
		http.Error(w, `{"error":"failed to create token","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *TokenHandler) Mint(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TokenID    string `json:"token_id"`
		To         string `json:"to"`
		Amount     string `json:"amount"`
		PrivateKey string `json:"private_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request","code":"INVALID_REQUEST"}`, http.StatusBadRequest)
		return
	}

	uc := tokenapp.NewMintUseCase(h.service)
	result, err := uc.Execute(&tokenapp.MintRequest{
		TokenID:    req.TokenID,
		To:         req.To,
		Amount:     req.Amount,
		PrivateKey: req.PrivateKey,
	})
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *TokenHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TokenID    string `json:"token_id"`
		From       string `json:"from"`
		To         string `json:"to"`
		Amount     string `json:"amount"`
		PrivateKey string `json:"private_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request","code":"INVALID_REQUEST"}`, http.StatusBadRequest)
		return
	}

	uc := tokenapp.NewTransferUseCase(h.service)
	result, err := uc.Execute(&tokenapp.TransferRequest{
		TokenID:    req.TokenID,
		From:       req.From,
		To:         req.To,
		Amount:     req.Amount,
		PrivateKey: req.PrivateKey,
	})
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *TokenHandler) Burn(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TokenID    string `json:"token_id"`
		From       string `json:"from"`
		Amount     string `json:"amount"`
		PrivateKey string `json:"private_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request","code":"INVALID_REQUEST"}`, http.StatusBadRequest)
		return
	}

	uc := tokenapp.NewBurnUseCase(h.service)
	result, err := uc.Execute(&tokenapp.BurnRequest{
		TokenID:    req.TokenID,
		From:       req.From,
		Amount:     req.Amount,
		PrivateKey: req.PrivateKey,
	})
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *TokenHandler) Balance(w http.ResponseWriter, r *http.Request) {
	tokenID := r.URL.Query().Get("token_id")
	owner := r.URL.Query().Get("owner")

	if tokenID == "" || owner == "" {
		http.Error(w, `{"error":"token_id and owner required","code":"INVALID_REQUEST"}`, http.StatusBadRequest)
		return
	}

	uc := tokenapp.NewGetBalanceUseCase(h.service)
	result, err := uc.Execute(&tokenapp.BalanceRequest{
		TokenID: tokenID,
		Owner:   owner,
	})
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *TokenHandler) History(w http.ResponseWriter, r *http.Request) {
	tokenID := r.URL.Query().Get("token_id")
	owner := r.URL.Query().Get("owner")

	uc := tokenapp.NewGetHistoryUseCase(h.service)
	result, err := uc.Execute(&tokenapp.HistoryRequest{
		TokenID: tokenID,
		Owner:   owner,
		Limit:   20,
	})
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
