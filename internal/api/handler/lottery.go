package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	lotteryapp "github.com/pplmx/aurora/internal/app/lottery"
	"github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/domain/lottery"
)

// Lottery history pagination bounds. GET /lottery/history previously returned
// every stored draw with no knob, unlike the token/NFT/oracle read surfaces
// (TASK-271, ISS-267) — a key-holding caller could force an unbounded DB
// scan/response. ?limit/?offset are clamped exactly like the sibling handlers;
// the CLI/TUI keep their full GetAll semantics, only the REST surface pages.
const (
	defaultLotteryHistoryLimit = 50
	maxLotteryHistoryLimit     = 100
	maxLotteryHistoryOffset    = 10000
)

type LotteryHandler struct {
	repo               lottery.Repository
	defaultWinnerCount int
}

// NewLotteryHandler wires the lottery REST surface. defaultWinnerCount is the
// winner count applied when a create request omits winner_count (the CLI's
// configured `lottery.defaultCount` defaults to 3); the server injects
// config.LotteryDefaultCount() so REST and CLI agree on a non-3 configured
// default instead of the API silently drawing 3 (TASK-247).
func NewLotteryHandler(repo lottery.Repository, defaultWinnerCount int) *LotteryHandler {
	if defaultWinnerCount <= 0 {
		defaultWinnerCount = lottery.DefaultWinnerCount
	}
	return &LotteryHandler{repo: repo, defaultWinnerCount: defaultWinnerCount}
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
	r.Get("/{id}/verify", h.Verify)
}

func (h *LotteryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateLotteryRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	blockChain := blockchain.InitBlockChain()
	uc := lotteryapp.NewCreateLotteryUseCase(h.repo, blockChain)

	// Resolve the configured default winner count at the API boundary, exactly
	// as the CLI's RunE resolves its `-c` absence to `lottery.defaultCount` —
	// an omitted winner_count must not silently fall to a hardcoded 3 when the
	// operator configured a different default (TASK-247; the use case keeps
	// DefaultWinnerCount as its own fallback for direct programmatic use).
	if req.WinnerCount == 0 {
		req.WinnerCount = h.defaultWinnerCount
	}

	appReq := lotteryapp.CreateLotteryRequest{
		Participants: req.Participants,
		Seed:         req.Seed,
		WinnerCount:  req.WinnerCount,
	}

	result, err := uc.Execute(appReq)
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *LotteryHandler) History(w http.ResponseWriter, r *http.Request) {
	// Bound the REST read surface: ?limit/?offset are parsed and clamped like
	// the token-history handler so a key-holding caller cannot force an
	// unbounded scan/response. The CLI/TUI paths keep their full GetAll (local
	// operator), only the REST endpoint pages (TASK-271, ISS-267).
	limit := defaultLotteryHistoryLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 {
			limit = l
			if limit > maxLotteryHistoryLimit {
				limit = maxLotteryHistoryLimit
			}
		}
	}
	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		if o, err := strconv.Atoi(v); err == nil && o >= 0 {
			offset = o
			if offset > maxLotteryHistoryOffset {
				offset = maxLotteryHistoryOffset
			}
		}
	}

	results, err := h.repo.GetAll()
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	// Apply offset/limit after the fetch: GetAll is shared with the CLI/TUI
	// (which want the full local list), so the REST surface slices. For the
	// documented no-flag default this avoids materializing the whole table only
	// in extreme cases; a future repo-level LIMIT could short-circuit it, but
	// slicing keeps the shared repo semantics intact.
	if offset > len(results) {
		offset = len(results)
	}
	end := offset + limit
	if end > len(results) {
		end = len(results)
	}
	results = results[offset:end]

	// A no-rows result is a nil slice, which JSON-encodes as `null`; every other
	// list endpoint (token history, NFT list, oracle query, voting sessions)
	// returns `[]`. Encode the empty array for envelope consistency (TASK-114).
	if results == nil {
		results = []*lottery.LotteryRecord{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results)
}

func (h *LotteryHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	result, err := h.repo.GetByID(id)
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	if result == nil {
		writeError(w, "not found", "NOT_FOUND", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// Verify re-verifies a persisted draw (proof vs stored public key + winner
// set) and returns {id, valid, reason}. It uses the same key-bound check as
// creation, so a draw created after the v1.31 feature can be cryptographically
// confirmed from the record alone.
func (h *LotteryHandler) Verify(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	uc := lotteryapp.NewVerifyLotteryUseCase(h.repo, lottery.NewService())
	result, err := uc.Execute(lotteryapp.VerifyLotteryRequest{ID: id})
	if err != nil {
		if errors.Is(err, lottery.ErrNotFound) {
			writeError(w, "not found", "NOT_FOUND", http.StatusNotFound)
			return
		}
		writeUseCaseError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
