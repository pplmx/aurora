package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	tokenapp "github.com/pplmx/aurora/internal/app/token"
	"github.com/pplmx/aurora/internal/domain/token"
)

// defaultHistoryLimit is the default limit for token transfer history API responses.
const defaultHistoryLimit = 20

// maxHistoryLimit caps the user-supplied ?limit= for token history so a
// key-holding caller cannot force an arbitrarily large event scan.
const maxHistoryLimit = 100

// maxHistoryOffset caps ?offset= for token history, bounding how far back a
// caller can page in one request.
const maxHistoryOffset = 1000

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
	r.Post("/approve", h.Approve)
	r.Post("/burn", h.Burn)
	r.Post("/transfer_from", h.TransferFrom)
	r.Get("/balance", h.Balance)
	r.Get("/allowance", h.Allowance)
	r.Get("/history", h.History)
	r.Get("/info", h.Info)
}

// Approve sets an allowance of amount from owner to spender (POST /approve).
// This closes the CLI/API parity gap: the CLI exposed `token approve` but the
// REST API had no matching endpoint even though the service implements it.
func (h *TokenHandler) Approve(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TokenID    string `json:"token_id"`
		Owner      string `json:"owner"`
		Spender    string `json:"spender"`
		Amount     string `json:"amount"`
		PrivateKey string `json:"private_key"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	uc := tokenapp.NewApproveUseCase(h.service)
	result, err := uc.Execute(&tokenapp.ApproveRequest{
		TokenID:    req.TokenID,
		Owner:      req.Owner,
		Spender:    req.Spender,
		Amount:     req.Amount,
		PrivateKey: req.PrivateKey,
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// Allowance returns the amount spender may transfer on behalf of owner.
func (h *TokenHandler) Allowance(w http.ResponseWriter, r *http.Request) {
	tokenID := r.URL.Query().Get("token_id")
	owner := r.URL.Query().Get("owner")
	spender := r.URL.Query().Get("spender")

	if tokenID == "" || owner == "" || spender == "" {
		writeBadRequest(w, "token_id, owner and spender required")
		return
	}

	uc := tokenapp.NewGetAllowanceUseCase(h.service)
	result, err := uc.Execute(&tokenapp.AllowanceRequest{
		TokenID: tokenID,
		Owner:   owner,
		Spender: spender,
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *TokenHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Symbol      string `json:"symbol"`
		TotalSupply string `json:"total_supply"`
		Owner       string `json:"owner"`
	}
	if !decodeJSON(w, r, &req) {
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
		writeUseCaseError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *TokenHandler) Mint(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TokenID    string `json:"token_id"`
		To         string `json:"to"`
		Amount     string `json:"amount"`
		PrivateKey string `json:"private_key"`
	}
	if !decodeJSON(w, r, &req) {
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
		writeUseCaseError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *TokenHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TokenID    string `json:"token_id"`
		From       string `json:"from"`
		To         string `json:"to"`
		Amount     string `json:"amount"`
		PrivateKey string `json:"private_key"`
	}
	if !decodeJSON(w, r, &req) {
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
		writeUseCaseError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *TokenHandler) Burn(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TokenID    string `json:"token_id"`
		From       string `json:"from"`
		Amount     string `json:"amount"`
		PrivateKey string `json:"private_key"`
	}
	if !decodeJSON(w, r, &req) {
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
		writeUseCaseError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *TokenHandler) Balance(w http.ResponseWriter, r *http.Request) {
	tokenID := r.URL.Query().Get("token_id")
	owner := r.URL.Query().Get("owner")

	if tokenID == "" || owner == "" {
		writeBadRequest(w, "token_id and owner required")
		return
	}

	uc := tokenapp.NewGetBalanceUseCase(h.service)
	result, err := uc.Execute(&tokenapp.BalanceRequest{
		TokenID: tokenID,
		Owner:   owner,
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *TokenHandler) History(w http.ResponseWriter, r *http.Request) {
	tokenID := r.URL.Query().Get("token_id")
	owner := r.URL.Query().Get("owner")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := defaultHistoryLimit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			if limit > maxHistoryLimit {
				limit = maxHistoryLimit
			}
		}
	}

	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
			if offset > maxHistoryOffset {
				offset = maxHistoryOffset
			}
		}
	}

	uc := tokenapp.NewGetHistoryUseCase(h.service)
	result, err := uc.Execute(&tokenapp.HistoryRequest{
		TokenID: tokenID,
		Owner:   owner,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	// Encode the bare transfer array, not the HistoryResponse wrapper: every
	// other list endpoint (lottery history, NFT list/history, oracle query)
	// returns a top-level JSON array, and the web UI (web/js/app.js) reads
	// history as `Array.isArray(data) ? data : (data.data || [])`, so the old
	// {"transfers":[...]} envelope made the token history page render empty
	// forever. The use case still speaks HistoryResponse internally (TASK-093,
	// ISS-086).
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result.Transfers)
}

// Info returns a token's metadata (name/symbol/total supply/decimals/owner) by
// token_id (v1.36). This closes the CLI-only gap: `token info` existed in the
// CLI but the REST API and web UI had no read path for token metadata.
func (h *TokenHandler) Info(w http.ResponseWriter, r *http.Request) {
	tokenID := r.URL.Query().Get("token_id")
	if tokenID == "" {
		writeBadRequest(w, "token_id required")
		return
	}

	uc := tokenapp.NewGetTokenInfoUseCase(h.service)
	result, err := uc.Execute(tokenID)
	if err != nil {
		if errors.Is(err, token.ErrTokenNotFound) {
			writeError(w, "not found", "NOT_FOUND", http.StatusNotFound)
			return
		}
		writeUseCaseError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// TransferFrom spends an allowance on the owner's behalf: the spender signs
// with their own private key and the owner's allowance is drawn down toward
// the recipient. The domain/app layer implemented this, but it was exposed
// nowhere (v1.38), so an approved allowance could not actually be spent
// through any CLI/REST/web surface.
func (h *TokenHandler) TransferFrom(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TokenID    string `json:"token_id"`
		Owner      string `json:"owner"`
		To         string `json:"to"`
		Amount     string `json:"amount"`
		Spender    string `json:"spender"`
		SpenderKey string `json:"spender_key"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	uc := tokenapp.NewTransferFromUseCase(h.service)
	result, err := uc.Execute(&tokenapp.TransferFromRequest{
		TokenID:    req.TokenID,
		Owner:      req.Owner,
		To:         req.To,
		Amount:     req.Amount,
		Spender:    req.Spender,
		SpenderKey: req.SpenderKey,
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
