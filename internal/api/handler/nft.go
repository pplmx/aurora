package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/pplmx/aurora/internal/app/nftapp"
	"github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/ledger"
)

// defaultNFTListLimit is the default page size for GET /nft/list?owner=.
const defaultNFTListLimit = 20

// maxNFTListLimit caps the user-supplied ?limit= so a key-holding caller
// cannot force an unbounded NFT scan/response (TASK-101, ISS-093).
const maxNFTListLimit = 100

// maxNFTListOffset caps ?offset=, bounding how far a caller can page in
// one request. Mirrors the token-history caps.
const maxNFTListOffset = 1000

const (
	defaultNFTHistoryLimit = 50
	maxNFTHistoryLimit     = 100
	maxNFTHistoryOffset    = 10000
)

type NFTHandler struct {
	repo    nft.Repository
	service nft.Service
	chain   ledger.BlockWriter
}

// NewNFTHandler wires the NFT service over a transaction-capable repository.
// txManager may be nil (handler tests); the service then falls back to
// non-transactional writes via NewService's nil guard.
func NewNFTHandler(repo nft.TransactableRepository, txManager nft.TransactionManager, chain ledger.BlockWriter) *NFTHandler {
	return &NFTHandler{
		repo:    repo,
		service: nft.NewService(repo, txManager),
		chain:   chain,
	}
}

func (h *NFTHandler) Routes(r chi.Router) {
	r.Post("/mint", h.Mint)
	r.Post("/transfer", h.Transfer)
	r.Post("/burn", h.Burn)
	r.Get("/{id}", h.Get)
	r.Get("/{id}/history", h.History)
	r.Get("/list", h.List)
}

func (h *NFTHandler) Mint(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		ImageURL    string `json:"image_url"`
		TokenURI    string `json:"token_uri"`
		Creator     string `json:"creator"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	uc := nftapp.NewMint(h.service, h.chain)
	result, err := uc.Execute(&nftapp.MintNFTRequest{
		Name:        req.Name,
		Description: req.Description,
		ImageURL:    req.ImageURL,
		TokenURI:    req.TokenURI,
		Creator:     req.Creator,
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *NFTHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NFTID      string `json:"nft_id"`
		From       string `json:"from"`
		To         string `json:"to"`
		PrivateKey string `json:"private_key"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	uc := nftapp.NewTransferNFTUseCase(h.service, h.chain)
	result, err := uc.Execute(&nftapp.TransferNFTRequest{
		NFTID:      req.NFTID,
		From:       req.From,
		To:         req.To,
		PrivateKey: req.PrivateKey,
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *NFTHandler) Burn(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NFTID      string `json:"nft_id"`
		Owner      string `json:"owner"`
		PrivateKey string `json:"private_key"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	uc := nftapp.NewBurnNFTUseCase(h.service, h.chain)
	err := uc.Execute(&nftapp.BurnNFTRequest{
		NFTID:      req.NFTID,
		Owner:      req.Owner,
		PrivateKey: req.PrivateKey,
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "burned"})
}

func (h *NFTHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	uc := nftapp.NewGet(h.service)
	result, err := uc.Execute(id)
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *NFTHandler) List(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner")

	// The list must be bounded: pre-fix it returned every NFT the owner
	// had with no limit or caps, letting a key-holding caller force an
	// unbounded response (TASK-101, ISS-093). Parse and clamp limit/offset
	// exactly like the token-history handler.
	limit := defaultNFTListLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 {
			limit = l
			if limit > maxNFTListLimit {
				limit = maxNFTListLimit
			}
		}
	}
	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		if o, err := strconv.Atoi(v); err == nil && o >= 0 {
			offset = o
			if offset > maxNFTListOffset {
				offset = maxNFTListOffset
			}
		}
	}

	// Without an owner the call lists the whole collection: an owner key is
	// exactly what a caller who did not mint the asset does not have, and
	// requiring it made every NFT unreachable unless its id was already known.
	if owner == "" {
		all, err := nftapp.NewListAll(h.service).Execute(&nftapp.ListNFTsRequest{
			Limit: limit, Offset: offset,
		})
		if err != nil {
			writeUseCaseError(w, err)
			return
		}
		if all == nil {
			all = []*nftapp.NFTResponse{}
		}
		writeJSON(w, http.StatusOK, all)
		return
	}

	uc := nftapp.NewListByOwner(h.service)
	result, err := uc.Execute(&nftapp.ListNFTsByOwnerRequest{
		Owner:  owner,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// History returns the mint/transfer/burn operation trail for one NFT
// (v1.35). The CLI exposed this as `nft history` but the REST API and the
// NFT web page had no way to see an NFT's ownership/audit history in the
// browser — closing that parity gap.
func (h *NFTHandler) History(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	limit, offset := nftHistoryPage(r)

	uc := nftapp.NewGetNFTOperationsUseCase(h.service)
	result, err := uc.Execute(&nftapp.GetNFTOperationsRequest{
		NFTID: id, Limit: limit, Offset: offset,
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// nftHistoryPage uses the same forgiving, capped rules as the other collection
// reads: malformed or non-positive limits take the default, and offsets clamp
// instead of allowing an arbitrarily deep storage scan.
func nftHistoryPage(r *http.Request) (limit, offset int) {
	limit = defaultNFTHistoryLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = min(parsed, maxNFTHistoryLimit)
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = min(parsed, maxNFTHistoryOffset)
		}
	}
	return limit, offset
}
