package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	nftapp "github.com/pplmx/aurora/internal/app/nft"
	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	domainnft "github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/infra/sqlite"
)

type NFTHandler struct{}

func NewNFTHandler() *NFTHandler {
	return &NFTHandler{}
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
		http.Error(w, `{"error":"invalid request","code":"INVALID_REQUEST"}`, http.StatusBadRequest)
		return
	}

	nftRepo, err := sqlite.NewNFTRepository(blockchain.DBPath())
	if err != nil {
		http.Error(w, `{"error":"failed to create repository","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	nftService := domainnft.NewService(nftRepo)
	chain := blockchain.InitBlockChain()

	uc := nftapp.NewMintNFTUseCase(nftService, chain)
	result, err := uc.Execute(&nftapp.MintNFTRequest{
		Name:        req.Name,
		Description: req.Description,
		ImageURL:    req.ImageURL,
		TokenURI:    req.TokenURI,
		Creator:     req.Creator,
	})
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
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
		http.Error(w, `{"error":"invalid request","code":"INVALID_REQUEST"}`, http.StatusBadRequest)
		return
	}

	nftRepo, err := sqlite.NewNFTRepository(blockchain.DBPath())
	if err != nil {
		http.Error(w, `{"error":"failed to create repository","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	nftService := domainnft.NewService(nftRepo)
	chain := blockchain.InitBlockChain()

	uc := nftapp.NewTransferNFTUseCase(nftService, chain)
	result, err := uc.Execute(&nftapp.TransferNFTRequest{
		NFTID:      req.NFTID,
		From:       req.From,
		To:         req.To,
		PrivateKey: req.PrivateKey,
	})
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
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
		http.Error(w, `{"error":"invalid request","code":"INVALID_REQUEST"}`, http.StatusBadRequest)
		return
	}

	nftRepo, err := sqlite.NewNFTRepository(blockchain.DBPath())
	if err != nil {
		http.Error(w, `{"error":"failed to create repository","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	nftService := domainnft.NewService(nftRepo)
	chain := blockchain.InitBlockChain()

	uc := nftapp.NewBurnNFTUseCase(nftService, chain)
	err = uc.Execute(&nftapp.BurnNFTRequest{
		NFTID:      req.NFTID,
		Owner:      req.Owner,
		PrivateKey: req.PrivateKey,
	})
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "burned"})
}

func (h *NFTHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	nftRepo, err := sqlite.NewNFTRepository(blockchain.DBPath())
	if err != nil {
		http.Error(w, `{"error":"failed to create repository","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	nftService := domainnft.NewService(nftRepo)
	uc := nftapp.NewGetNFTUseCase(nftService)
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

	nftRepo, err := sqlite.NewNFTRepository(blockchain.DBPath())
	if err != nil {
		http.Error(w, `{"error":"failed to create repository","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	nftService := domainnft.NewService(nftRepo)
	uc := nftapp.NewListNFTsByOwnerUseCase(nftService)
	result, err := uc.Execute(owner)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
