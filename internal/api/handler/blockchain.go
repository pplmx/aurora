package handler

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pplmx/aurora/internal/ledger"
)

// blockchainChain is the part of the shared ledger view the audit endpoints
// read: the integrity report, the total height and the newest-first page of
// payloads. The container's Chain satisfies it.
type blockchainChain interface {
	VerifyIntegrity() *ledger.IntegrityReport
	Len() int
	BlockEntriesPage(limit, offset int) []ledger.BlockEntry
	BlockEntriesPageMatching(limit, offset int, match func(ledger.BlockEntry) bool) ([]ledger.BlockEntry, int)
}

// BlockchainHandler exposes the ledger's integrity and its recorded payloads
// over the REST API. Verify does not dereference the receiver, so it is safe to
// register even on a nil handler (route registration binds the method value).
type BlockchainHandler struct {
	chain blockchainChain
}

func NewBlockchainHandler(chain blockchainChain) *BlockchainHandler {
	return &BlockchainHandler{chain: chain}
}

func (h *BlockchainHandler) Routes(r chi.Router) {
	r.Get("/verify", h.Verify)
	r.Get("/blocks", h.Blocks)
}

// BlockPage is one newest-first page of recorded audit entries. The envelope
// carries the total because a bare array leaves a client unable to tell whether
// another page exists. Payload is the exact text the module appended.
type BlockPage struct {
	Blocks []BlockEntry `json:"blocks"`
	Total  int          `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

// BlockEntry is one recorded payload with the height and timestamp it was
// written at.
type BlockEntry struct {
	Height    int64  `json:"height"`
	Timestamp int64  `json:"timestamp"`
	Payload   string `json:"payload"`
}

const (
	defaultBlockPageLimit = 20
	maxBlockPageLimit     = 100
	maxBlockPageOffset    = 10000
)

// Blocks returns a page of non-genesis entries, newest first. The chain's page
// counts entries (not heights), so a ledger with gaps still pages correctly;
// the total excludes the genesis block, matching the desktop's pager. A q
// filters on the exact recorded text (case-insensitive substring), and the
// total then counts matches, so a client still knows how many pages remain.
func (h *BlockchainHandler) Blocks(w http.ResponseWriter, r *http.Request) {
	limit, offset := pageParams(r, defaultBlockPageLimit, maxBlockPageLimit, maxBlockPageOffset)
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	var entries []ledger.BlockEntry
	total := 0
	if query == "" {
		entries = h.chain.BlockEntriesPage(limit, offset)
		if total = h.chain.Len() - 1; total < 0 {
			total = 0
		}
	} else {
		needle := strings.ToLower(query)
		entries, total = h.chain.BlockEntriesPageMatching(limit, offset, func(entry ledger.BlockEntry) bool {
			return strings.Contains(strings.ToLower(entry.Data), needle)
		})
	}
	page := BlockPage{Blocks: make([]BlockEntry, 0, len(entries)), Total: total, Limit: limit, Offset: offset}
	for _, entry := range entries {
		page.Blocks = append(page.Blocks, BlockEntry{
			Height:    entry.Height,
			Timestamp: entry.Timestamp,
			Payload:   entry.Data,
		})
	}
	writeJSON(w, http.StatusOK, page)
}

// Verify runs a full integrity verification of the in-memory chain and returns
// the result (valid, length, first broken index + reason) as JSON.
func (h *BlockchainHandler) Verify(w http.ResponseWriter, r *http.Request) {
	report := h.chain.VerifyIntegrity()
	writeJSON(w, http.StatusOK, report)
}
