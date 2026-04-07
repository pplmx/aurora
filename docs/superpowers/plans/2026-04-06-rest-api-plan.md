# REST API Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 Aurora 添加 REST API 服务器，使现有功能可通过 HTTP API 访问

**Architecture:** 使用 Chi 框架创建轻量级 HTTP 服务，复用现有 domain 和 app 层，API 作为新的表示层

**Tech Stack:** go-chi/chi, zerolog, Viper

---

## File Structure

```text
cmd/
├── aurora/           # 现有 CLI
│   └── main.go
└── api/              # 新增 API 服务
    └── main.go

internal/
├── config/           # 新增: 配置结构
│   └── config.go
└── api/              # 新增 API 层
    ├── router.go
    ├── handler/
    │   ├── lottery.go
    │   ├── voting.go
    │   ├── nft.go
    │   ├── token.go
    │   └── oracle.go
    └── middleware/
        ├── logger.go
        └── recovery.go

config/
└── aurora.toml       # 修改: 添加 server 配置
```

---

## Task 1: 添加 Server 配置

**Files:**

- Create: `internal/config/config.go`
- Modify: `cmd/aurora/cmd/root.go` (添加 server 配置加载)

- [ ] **Step 1: 创建配置结构**

```go
// internal/config/config.go
package config

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	Log    LogConfig    `mapstructure:"log"`
	DB     DBConfig     `mapstructure:"db"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
	Path  string `mapstructure:"path"`
}

type DBConfig struct {
	Type string `mapstructure:"type"`
	Path string `mapstructure:"path"`
}

func Load(path string) (*Config, error) {
	viper.SetConfigName("aurora")
	viper.SetConfigType("toml")
	viper.AddConfigPath("$HOME")
	viper.AddConfigPath("./config")
	
	// 默认值
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.path", "./logs")
	
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
```

- [ ] **Step 2: 更新 config/aurora.toml 添加 server 配置**

```toml
[server]
host = "0.0.0.0"
port = 8080

[log]
level = "info"
path = "./logs"

[db]
type = "sqlite"
path = "./data/aurora.db"
```

- [ ] **Step 3: 提交**

```bash
git add internal/config/config.go config/aurora.toml
git commit -m "config: add server configuration"
```

---

## Task 2: 创建 API 中间件

**Files:**

- Create: `internal/api/middleware/logger.go`
- Create: `internal/api/middleware/recovery.go`

- [ ] **Step 1: 创建 Logger 中间件**

```go
// internal/api/middleware/logger.go
package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/pplmx/aurora/internal/logger"
)

func Logger(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()
		next.ServeHTTP(ww, r)
		
		logger.Log().Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", ww.Status()).
			Dur("latency", time.Since(start)).
			Msg("request")
	}
	return http.HandlerFunc(fn)
}
```

- [ ] **Step 2: 创建 Recovery 中间件**

```go
// internal/api/middleware/recovery.go
package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/pplmx/aurora/internal/logger"
)

func Recovery(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.Log().Error().Interface("error", err).Msg("panic recovered")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "Internal server error",
					"code":  "INTERNAL_ERROR",
				})
			}
		}()
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
```

- [ ] **Step 3: 提交**

```bash
git add internal/api/middleware/logger.go internal/api/middleware/recovery.go
git commit -m "api: add middleware (logger, recovery)"
```

---

## Task 3: 创建 Lottery Handler

**Files:**

- Create: `internal/api/handler/lottery.go`

- [ ] **Step 1: 创建 Lottery Handler**

```go
// internal/api/handler/lottery.go
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pplmx/aurora/internal/app/lottery"
	"github.com/pplmx/aurora/internal/domain/lottery"
)

type LotteryHandler struct {
	uc *lottery.UseCase
}

func NewLotteryHandler(uc *lottery.UseCase) *LotteryHandler {
	return &LotteryHandler{uc: uc}
}

type CreateLotteryRequest struct {
	Participants []string `json:"participants"`
	Seed         string   `json:"seed"`
	Count        int      `json:"count"`
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

	dto := lottery.CreateLotteryDTO{
		Participants: req.Participants,
		Seed:         req.Seed,
		Count:        req.Count,
	}

	result, err := h.uc.Create(r.Context(), dto)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *LotteryHandler) History(w http.ResponseWriter, r *http.Request) {
	results, err := h.uc.History(r.Context())
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (h *LotteryHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	result, err := h.uc.Get(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not found","code":"NOT_FOUND"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
```

- [ ] **Step 2: 提交**

```bash
git add internal/api/handler/lottery.go
git commit -m "api: add lottery handler"
```

---

## Task 4: 创建 Voting Handler

**Files:**

- Create: `internal/api/handler/voting.go`

- [ ] **Step 1: 创建 Voting Handler**

```go
// internal/api/handler/voting.go
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pplmx/aurora/internal/app/voting"
)

type VotingHandler struct {
	uc *voting.UseCase
}

func NewVotingHandler(uc *voting.UseCase) *VotingHandler {
	return &VotingHandler{uc: uc}
}

func (h *VotingHandler) Routes(r chi.Router) {
	r.Post("/create", h.Create)
	r.Post("/vote", h.Vote)
	r.Get("/{id}", h.Get)
}

func (h *VotingHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title       string `json:"title"`
		OwnerPubkey string `json:"owner_pubkey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request","code":"INVALID_REQUEST"}`, http.StatusBadRequest)
		return
	}

	dto := voting.CreateVoteDTO{
		Title:       req.Title,
		OwnerPubkey: req.OwnerPubkey,
	}

	result, err := h.uc.Create(r.Context(), dto)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *VotingHandler) Vote(w http.ResponseWriter, r *http.Request) {
	var req struct {
		VoteID    string `json:"vote_id"`
		VoterPubkey string `json:"voter_pubkey"`
		Choice    string `json:"choice"`
		Signature string `json:"signature"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request","code":"INVALID_REQUEST"}`, http.StatusBadRequest)
		return
	}

	dto := voting.CastVoteDTO{
		VoteID:     req.VoteID,
		VoterPubkey: req.VoterPubkey,
		Choice:     req.Choice,
		Signature:  req.Signature,
	}

	result, err := h.uc.CastVote(r.Context(), dto)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *VotingHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	result, err := h.uc.Get(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not found","code":"NOT_FOUND"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
```

- [ ] **Step 2: 提交**

```bash
git add internal/api/handler/voting.go
git commit -m "api: add voting handler"
```

---

## Task 5: 创建 NFT Handler

**Files:**

- Create: `internal/api/handler/nft.go`

- [ ] **Step 1: 创建 NFT Handler**

```go
// internal/api/handler/nft.go
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pplmx/aurora/internal/app/nft"
)

type NFTHandler struct {
	uc *nft.UseCase
}

func NewNFTHandler(uc *nft.UseCase) *NFTHandler {
	return &NFTHandler{uc: uc}
}

func (h *NFTHandler) Routes(r chi.Router) {
	r.Post("/mint", h.Mint)
	r.Post("/transfer", h.Transfer)
	r.Get("/{id}", h.Get)
	r.Get("/list", h.List)
}

func (h *NFTHandler) Mint(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		CreatorPubkey string `json:"creator_pubkey"`
		Signature   string `json:"signature"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request","code":"INVALID_REQUEST"}`, http.StatusBadRequest)
		return
	}

	dto := nft.MintNFTDTO{
		Name:          req.Name,
		Description:   req.Description,
		CreatorPubkey: req.CreatorPubkey,
		Signature:     req.Signature,
	}

	result, err := h.uc.Mint(r.Context(), dto)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *NFTHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NFTID    string `json:"nft_id"`
		To       string `json:"to"`
		From     string `json:"from"`
		Signature string `json:"signature"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request","code":"INVALID_REQUEST"}`, http.StatusBadRequest)
		return
	}

	dto := nft.TransferNFTDTO{
		ID:         req.NFTID,
		To:         req.To,
		From:       req.From,
		Signature:  req.Signature,
	}

	result, err := h.uc.Transfer(r.Context(), dto)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *NFTHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	result, err := h.uc.Get(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not found","code":"NOT_FOUND"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *NFTHandler) List(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner")
	if owner == "" {
		http.Error(w, `{"error":"owner required","code":"INVALID_REQUEST"}`, http.StatusBadRequest)
		return
	}

	result, err := h.uc.ListByOwner(r.Context(), owner)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
```

- [ ] **Step 2: 提交**

```bash
git add internal/api/handler/nft.go
git commit -m "api: add nft handler"
```

---

## Task 6: 创建 Token Handler

**Files:**

- Create: `internal/api/handler/token.go`

- [ ] **Step 1: 创建 Token Handler**

```go
// internal/api/handler/token.go
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/pplmx/aurora/internal/app/token"
)

type TokenHandler struct {
	uc *token.UseCase
}

func NewTokenHandler(uc *token.UseCase) *TokenHandler {
	return &TokenHandler{uc: uc}
}

func (h *TokenHandler) Routes(r Router) {
	r.Post("/create", h.Create)
	r.Post("/mint", h.Mint)
	r.Post("/transfer", h.Transfer)
	r.Post("/burn", h.Burn)
	r.Get("/balance", h.Balance)
	r.Get("/history", h.History)
}

func (h *TokenHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string `json:"name"`
		Symbol string `json:"symbol"`
		Supply int64  `json:"supply"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request","code":"INVALID_REQUEST"}`, http.StatusBadRequest)
		return
	}

	dto := token.CreateTokenDTO{
		Name:   req.Name,
		Symbol: req.Symbol,
		Supply: req.Supply,
	}

	result, err := h.uc.Create(r.Context(), dto)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *TokenHandler) Mint(w http.ResponseWriter, r *http.Request) {
	var req struct {
		To     string `json:"to"`
		Amount int64  `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request","code":"INVALID_REQUEST"}`, http.StatusBadRequest)
		return
	}

	dto := token.MintTokenDTO{
		To:     req.To,
		Amount: req.Amount,
	}

	result, err := h.uc.Mint(r.Context(), dto)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *TokenHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		From   string `json:"from"`
		To     string `json:"to"`
		Amount int64  `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request","code":"INVALID_REQUEST"}`, http.StatusBadRequest)
		return
	}

	dto := token.TransferTokenDTO{
		From:   req.From,
		To:     req.To,
		Amount: req.Amount,
	}

	result, err := h.uc.Transfer(r.Context(), dto)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *TokenHandler) Burn(w http.ResponseWriter, r *http.Request) {
	var req struct {
		From   string `json:"from"`
		Amount int64  `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request","code":"INVALID_REQUEST"}`, http.StatusBadRequest)
		return
	}

	dto := token.BurnTokenDTO{
		From:   req.From,
		Amount: req.Amount,
	}

	result, err := h.uc.Burn(r.Context(), dto)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *TokenHandler) Balance(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner")
	if owner == "" {
		http.Error(w, `{"error":"owner required","code":"INVALID_REQUEST"}`, http.StatusBadRequest)
		return
	}

	result, err := h.uc.Balance(r.Context(), owner)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *TokenHandler) History(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.History(r.Context())
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
```

- [ ] **Step 2: 提交**

```bash
git add internal/api/handler/token.go
git commit -m "api: add token handler"
```

---

## Task 7: 创建 Oracle Handler

**Files:**

- Create: `internal/api/handler/oracle.go`

- [ ] **Step 1: 创建 Oracle Handler**

```go
// internal/api/handler/oracle.go
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/pplmx/aurora/internal/app/oracle"
)

type OracleHandler struct {
	uc *oracle.UseCase
}

func NewOracleHandler(uc *oracle.UseCase) *OracleHandler {
	return &OracleHandler{uc: uc}
}

func (h *OracleHandler) Routes(r Router) {
	r.Get("/sources", h.Sources)
	r.Post("/fetch", h.Fetch)
	r.Get("/query", h.Query)
}

func (h *OracleHandler) Sources(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.ListSources(r.Context())
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *OracleHandler) Fetch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request","code":"INVALID_REQUEST"}`, http.StatusBadRequest)
		return
	}

	dto := oracle.FetchDataDTO{
		Source: req.Source,
	}

	result, err := h.uc.Fetch(r.Context(), dto)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *OracleHandler) Query(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	limit := r.URL.Query().Get("limit")

	dto := oracle.QueryDataDTO{
		Source: source,
		Limit:  10,
	}
	if limit != "" {
		dto.Limit = 10 // parse limit if needed
	}

	result, err := h.uc.Query(r.Context(), dto)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
```

- [ ] **Step 2: 提交**

```bash
git add internal/api/handler/oracle.go
git commit -m "api: add oracle handler"
```

---

## Task 8: 创建 Router 和主程序

**Files:**

- Create: `internal/api/router.go`
- Create: `cmd/api/main.go`

- [ ] **Step 1: 创建 Router**

```go
// internal/api/router.go
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/pplmx/aurora/internal/api/handler"
	"github.com/pplmx/aurora/internal/api/middleware/api_middleware"
)

type Router interface {
	Method(method, path string, handler http.HandlerFunc)
	Group(prefix string, fn func(r Router))
}

type chiRouter struct {
	r *chi.Mux
}

func (c *chiRouter) Method(method, path string, handler http.HandlerFunc) {
	c.r.Method(method, path, handler)
}

func (c *chiRouter) Group(prefix string, fn func(r Router)) {
	fn(c)
}

func NewRouter(
	lotteryHandler *handler.LotteryHandler,
	votingHandler *handler.VotingHandler,
	nftHandler *handler.NFTHandler,
	tokenHandler *handler.TokenHandler,
	oracleHandler *handler.OracleHandler,
) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(api_middleware.Logger)
	r.Use(api_middleware.Recovery)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/v1/lottery", func(r chi.Router) {
		lotteryHandler.Routes(r)
	})

	r.Route("/api/v1/voting", func(r chi.Router) {
		votingHandler.Routes(r)
	})

	r.Route("/api/v1/nft", func(r chi.Router) {
		nftHandler.Routes(r)
	})

	r.Route("/api/v1/token", func(r chi.Router) {
		tokenHandler.Routes(r)
	})

	r.Route("/api/v1/oracle", func(r chi.Router) {
		oracleHandler.Routes(r)
	})

	return r
}
```

- [ ] **Step 2: 创建 API 主程序**

```go
// cmd/api/main.go
package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/pplmx/aurora/internal/api"
	"github.com/pplmx/aurora/internal/api/handler"
	"github.com/pplmx/aurora/internal/app/lottery"
	"github.com/pplmx/aurora/internal/app/nft"
	"github.com/pplmx/aurora/internal/app/oracle"
	"github.com/pplmx/aurora/internal/app/token"
	"github.com/pplmx/aurora/internal/app/voting"
	"github.com/pplmx/aurora/internal/config"
	"github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/pplmx/aurora/internal/logger"
)

func main() {
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger.Init(cfg.Log.Level, cfg.Log.Path)
	defer logger.Sync()

	db, err := sqlite.NewDB(cfg.DB.Path)
	if err != nil {
		logger.Log().Fatal().Err(err).Msg("Failed to connect database")
	}

	lotteryRepo := sqlite.NewLotteryRepository(db)
	votingRepo := sqlite.NewVotingRepository(db)
	nftRepo := sqlite.NewNFTRepository(db)
	tokenRepo := sqlite.NewTokenRepository(db)
	oracleRepo := sqlite.NewOracleRepository(db)

	lotteryUC := lottery.NewUseCase(lotteryRepo)
	votingUC := voting.NewUseCase(votingRepo)
	nftUC := nft.NewUseCase(nftRepo)
	tokenUC := token.NewUseCase(tokenRepo)
	oracleUC := oracle.NewUseCase(oracleRepo, nil) // TODO: add fetcher

	lotteryHandler := handler.NewLotteryHandler(lotteryUC)
	votingHandler := handler.NewVotingHandler(votingUC)
	nftHandler := handler.NewNFTHandler(nftUC)
	tokenHandler := handler.NewTokenHandler(tokenUC)
	oracleHandler := handler.NewOracleHandler(oracleUC)

	router := api.NewRouter(
		lotteryHandler,
		votingHandler,
		nftHandler,
		tokenHandler,
		oracleHandler,
	)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		logger.Log().Info().Str("addr", addr).Msg("Starting API server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log().Fatal().Err(err).Msg("Server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log().Info().Msg("Shutting down server...")
	if err := server.Close(); err != nil {
		logger.Log().Error().Err(err).Msg("Server closed with error")
	}
}
```

- [ ] **Step 3: 添加 go-chi/chi 依赖**

```bash
go get github.com/go-chi/chi/v5
```

- [ ] **Step 4: 提交**

```bash
git add internal/api/router.go cmd/api/main.go
git commit -m "api: add router and main program"
```

---

## Task 9: 运行测试并验证

- [ ] **Step 1: 构建项目**

```bash
go build ./cmd/api
```

- [ ] **Step 2: 测试健康检查**

```bash
./api &
curl http://localhost:8080/health
# Expected: {"status":"ok"}
```

- [ ] **Step 3: 测试 Lottery API**

```bash
curl -X POST http://localhost:8080/api/v1/lottery/create \
  -H "Content-Type: application/json" \
  -d '{"participants":["A","B","C","D"],"seed":"test","count":2}'
```

- [ ] **Step 4: 停止服务并提交**

```bash
pkill api
git commit -m "api: verify all endpoints work"
```

---

## Summary

| Task | Description          |
| ---- | -------------------- |
| 1    | 添加 Server 配置     |
| 2    | 创建 API 中间件      |
| 3    | 创建 Lottery Handler |
| 4    | 创建 Voting Handler  |
| 5    | 创建 NFT Handler     |
| 6    | 创建 Token Handler   |
| 7    | 创建 Oracle Handler  |
| 8    | 创建 Router 和主程序 |
| 9    | 运行测试并验证       |

---

**Plan complete and saved to `docs/superpowers/plans/2026-04-06-rest-api-plan.md`. Two execution options:**

1. **Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

2. **Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
