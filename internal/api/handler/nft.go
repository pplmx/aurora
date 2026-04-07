package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	nftapp "github.com/pplmx/aurora/internal/app/nft"
	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	domainnft "github.com/pplmx/aurora/internal/domain/nft"
)

type NFTHandler struct {
	repo    domainnft.Repository
	service domainnft.Service
	chain   blockchain.BlockWriter
}

func NewNFTHandler(repo domainnft.Repository) *NFTHandler {
	return &NFTHandler{
		repo:    repo,
		service: domainnft.NewService(repo),
		chain:   blockchain.InitBlockChain(),
	}
}

func (h *NFTHandler) Routes(r chi.Router) {
	r.Post("/mint", h.Mint)
	r.Post("/transfer", h.Transfer)
	r.Post("/burn", h.Burn)
	r.Get("/{id}", h.Get)
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid request")
		return
	}

	uc := nftapp.NewMintNFTUseCase(h.service, h.chain)
	result, err := uc.Execute(&nftapp.MintNFTRequest{
		Name:        req.Name,
		Description: req.Description,
		ImageURL:    req.ImageURL,
		TokenURI:    req.TokenURI,
		Creator:     req.Creator,
	})
	if err != nil {
		writeInternalError(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *NFTHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NFTID      string `json:"nft_id"`
		From       string `json:"from"`
		To         string `json:"to"`
		PrivateKey string `json:"private_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid request")
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
		writeInternalError(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *NFTHandler) Burn(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NFTID      string `json:"nft_id"`
		Owner      string `json:"owner"`
		PrivateKey string `json:"private_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid request")
		return
	}

	uc := nftapp.NewBurnNFTUseCase(h.service, h.chain)
	err := uc.Execute(&nftapp.BurnNFTRequest{
		NFTID:      req.NFTID,
		Owner:      req.Owner,
		PrivateKey: req.PrivateKey,
	})
	if err != nil {
		writeInternalError(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "burned"})
}

func (h *NFTHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	uc := nftapp.NewGetNFTUseCase(h.service)
	result, err := uc.Execute(id)
	if err != nil {
		http.Error(w, `{"error":"not found","code":"NOT_FOUND"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *NFTHandler) List(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner")

	uc := nftapp.NewListNFTsByOwnerUseCase(h.service)
	result, err := uc.Execute(owner)
	if err != nil {
		writeInternalError(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
