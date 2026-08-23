package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pplmx/aurora/internal/domain/blockchain"
)

// BlockchainHandler exposes the chain's integrity verification over the REST
// API (v1.25). The Verify handler does not dereference the receiver, so it is
// safe to register even on a nil handler (route registration binds the method
// value; the actual call uses the global chain).
type BlockchainHandler struct{}

func NewBlockchainHandler() *BlockchainHandler { return &BlockchainHandler{} }

func (h *BlockchainHandler) Routes(r chi.Router) {
	r.Get("/verify", h.Verify)
}

// Verify runs a full integrity verification of the in-memory chain and returns
// the result (valid, length, first broken index + reason) as JSON.
func (h *BlockchainHandler) Verify(w http.ResponseWriter, r *http.Request) {
	report := blockchain.GetBlockChain().VerifyIntegrity()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(report)
}
