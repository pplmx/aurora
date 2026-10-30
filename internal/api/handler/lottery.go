package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pplmx/aurora/internal/app/lotteryapp"
	"github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/pplmx/aurora/internal/ledger"
)

// Lottery history pagination bounds. The REST handler passes the page into
// storage rather than materialising the whole table and slicing it afterwards.
const (
	defaultLotteryHistoryLimit = 50
	maxLotteryHistoryLimit     = 100
	maxLotteryHistoryOffset    = 10000
)

type LotteryHandler struct {
	repo               lottery.Repository
	defaultWinnerCount int
	chain              ledger.RecordWriter
}

// NewLotteryHandler wires the lottery REST surface. defaultWinnerCount is the
// winner count applied when a create request omits winner_count (the CLI's
// configured `lottery.defaultCount` defaults to 3); the server injects
// config.LotteryDefaultCount() so REST and CLI agree on a non-3 configured
// default instead of the API silently drawing 3 (TASK-247).
func NewLotteryHandler(repo lottery.Repository, defaultWinnerCount int, chain ledger.RecordWriter) *LotteryHandler {
	if defaultWinnerCount <= 0 {
		defaultWinnerCount = lottery.DefaultWinnerCount
	}
	return &LotteryHandler{repo: repo, defaultWinnerCount: defaultWinnerCount, chain: chain}
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

	uc := lotteryapp.NewCreate(h.repo, h.chain)

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

	writeJSON(w, http.StatusOK, result)
}

func (h *LotteryHandler) History(w http.ResponseWriter, r *http.Request) {
	limit, offset := lotteryHistoryPage(r)

	results, err := h.repo.List(limit, offset)
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	// A no-rows result is a nil slice, which JSON-encodes as `null`; every other
	// list endpoint (token history, NFT list, oracle query, voting sessions)
	// returns `[]`. Encode the empty array for envelope consistency (TASK-114).
	if results == nil {
		results = []*lottery.LotteryRecord{}
	}

	writeJSON(w, http.StatusOK, results)
}

func lotteryHistoryPage(r *http.Request) (limit, offset int) {
	return pageParams(r, defaultLotteryHistoryLimit, maxLotteryHistoryLimit, maxLotteryHistoryOffset)
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

	writeJSON(w, http.StatusOK, result)
}

// Verify re-verifies a persisted draw and returns the same report the CLI and
// the desktop show (the record, every check and the verdict), so all three
// surfaces answer with one shape instead of the REST shortcut {id, valid,
// reason}. The input resolves like the CLI's: exact id, block height, or an
// id/seed substring.
func (h *LotteryHandler) Verify(w http.ResponseWriter, r *http.Request) {
	input := chi.URLParam(r, "id")

	report, err := lotteryapp.VerifyInput(h.repo, input)
	if err != nil {
		if errors.Is(err, lottery.ErrNotFound) {
			writeError(w, "not found", "NOT_FOUND", http.StatusNotFound)
			return
		}
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, report)
}
