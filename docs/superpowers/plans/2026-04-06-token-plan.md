# Token (FT) 系统实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现完整的同质化代币(FT)系统，支持铸造、转账、授权、销毁功能，基于 Event Sourcing 架构

**Architecture:** 采用 DDD 四层架构 + Event Sourcing，事件不可变存储，余额从事件聚合，Ed25519 签名验证

**Tech Stack:** Go, SQLite, Ed25519, Bubble Tea (TUI), Cobra (CLI)

---

## 文件结构

```text
internal/domain/token/
├── entity.go        # Token, Approval 定义
├── event.go         # TransferEvent, MintEvent, BurnEvent, ApproveEvent
├── service.go       # TokenService 接口
├── errors.go        # 领域错误
└── validator.go     # 业务规则验证

internal/infra/sqlite/
├── token.go         # Token 仓储实现
├── event_store.go   # 事件存储实现
└── allowance.go    # 授权仓储实现

internal/app/token/
├── dto.go           # 请求/响应对象
├── create.go        # CreateTokenUseCase
├── mint.go          # MintTokenUseCase
├── transfer.go      # TransferUseCase
├── approve.go       # ApproveUseCase
└── burn.go          # BurnTokenUseCase

internal/ui/token/
└── tui.go          # Bubble Tea TUI

cmd/aurora/cmd/
└── token.go        # CLI 命令
```

---

## Task 1: Domain Layer - Entity & Events

**Files:**

- Create: `internal/domain/token/entity.go`
- Create: `internal/domain/token/event.go`
- Create: `internal/domain/token/errors.go`
- Create: `internal/domain/token/validator.go`
- Test: `internal/domain/token/entity_test.go`

- [ ] **Step 1: 创建 entity.go**

```go
package token

import (
	"math/big"
	"time"
)

type TokenID string

type PublicKey []byte

type Signature []byte

type Amount struct {
	*big.Int
}

func NewAmount(value int64) *Amount {
	return &Amount{big.NewInt(value)}
}

func NewAmountFromString(s string) (*Amount, error) {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return nil, ErrInvalidAmount
	}
	return &Amount{v}, nil
}

func (a *Amount) String() string {
	if a == nil || a.Int == nil {
		return "0"
	}
	return a.Int.String()
}

func (a *Amount) IsPositive() bool {
	return a != nil && a.Int != nil && a.Int.Sign() > 0
}

type Token struct {
	id          TokenID
	name        string
	symbol      string
	totalSupply *Amount
	decimals    int8
	owner       PublicKey
	isMintable  bool
	isBurnable  bool
	createdAt   time.Time
}

func NewToken(id TokenID, name, symbol string, totalSupply *Amount, owner PublicKey) *Token {
	return &Token{
		id:          id,
		name:        name,
		symbol:      symbol,
		totalSupply: totalSupply,
		decimals:    8,
		owner:       owner,
		isMintable:  true,
		isBurnable:  true,
		createdAt:   time.Now(),
	}
}

func (t *Token) ID() TokenID        { return t.id }
func (t *Token) Name() string      { return t.name }
func (t *Token) Symbol() string    { return t.symbol }
func (t *Token) TotalSupply() *Amount { return t.totalSupply }
func (t *Token) Decimals() int8    { return t.decimals }
func (t *Token) Owner() PublicKey   { return t.owner }
func (t *Token) IsMintable() bool  { return t.isMintable }
func (t *Token) IsBurnable() bool  { return t.isBurnable }
func (t *Token) CreatedAt() time.Time { return t.createdAt }

type Approval struct {
	tokenID   TokenID
	owner     PublicKey
	spender   PublicKey
	amount    *Amount
	expiresAt time.Time
}

func NewApproval(tokenID TokenID, owner, spender PublicKey, amount *Amount) *Approval {
	return &Approval{
		tokenID: tokenID,
		owner:   owner,
		spender: spender,
		amount:  amount,
	}
}

func (a *Approval) TokenID() TokenID     { return a.tokenID }
func (a *Approval) Owner() PublicKey    { return a.owner }
func (a *Approval) Spender() PublicKey  { return a.spender }
func (a *Approval) Amount() *Amount    { return a.amount }
func (a *Approval) ExpiresAt() time.Time { return a.expiresAt }
```

- [ ] **Step 2: 创建 event.go**

```go
package token

import (
	"time"
)

type TransferEvent struct {
	id          string
	tokenID     TokenID
	from        PublicKey
	to          PublicKey
	amount      *Amount
	nonce       uint64
	signature   Signature
	blockHeight int64
	timestamp   time.Time
}

func NewTransferEvent(tokenID TokenID, from, to PublicKey, amount *Amount, nonce uint64, signature Signature) *TransferEvent {
	return &TransferEvent{
		id:        generateEventID(),
		tokenID:   tokenID,
		from:      from,
		to:        to,
		amount:    amount,
		nonce:     nonce,
		signature: signature,
		timestamp: time.Now(),
	}
}

func (e *TransferEvent) ID() string        { return e.id }
func (e *TransferEvent) TokenID() TokenID  { return e.tokenID }
func (e *TransferEvent) From() PublicKey   { return e.from }
func (e *TransferEvent) To() PublicKey     { return e.to }
func (e *TransferEvent) Amount() *Amount  { return e.amount }
func (e *TransferEvent) Nonce() uint64     { return e.nonce }
func (e *TransferEvent) Signature() Signature { return e.signature }
func (e *TransferEvent) BlockHeight() int64 { return e.blockHeight }
func (e *TransferEvent) Timestamp() time.Time { return e.timestamp }

type MintEvent struct {
	id          string
	tokenID     TokenID
	to          PublicKey
	amount      *Amount
	blockHeight int64
	timestamp   time.Time
}

func NewMintEvent(tokenID TokenID, to PublicKey, amount *Amount) *MintEvent {
	return &MintEvent{
		id:        generateEventID(),
		tokenID:   tokenID,
		to:        to,
		amount:    amount,
		timestamp: time.Now(),
	}
}

func (e *MintEvent) ID() string       { return e.id }
func (e *MintEvent) TokenID() TokenID { return e.tokenID }
func (e *MintEvent) To() PublicKey   { return e.to }
func (e *MintEvent) Amount() *Amount { return e.amount }
func (e *MintEvent) Timestamp() time.Time { return e.timestamp }

type BurnEvent struct {
	id          string
	tokenID     TokenID
	from        PublicKey
	amount      *Amount
	blockHeight int64
	timestamp   time.Time
}

func NewBurnEvent(tokenID TokenID, from PublicKey, amount *Amount) *BurnEvent {
	return &BurnEvent{
		id:        generateEventID(),
		tokenID:   tokenID,
		from:      from,
		amount:    amount,
		timestamp: time.Now(),
	}
}

func (e *BurnEvent) ID() string       { return e.id }
func (e *BurnEvent) TokenID() TokenID { return e.tokenID }
func (e *BurnEvent) From() PublicKey  { return e.from }
func (e *BurnEvent) Amount() *Amount { return e.amount }
func (e *BurnEvent) Timestamp() time.Time { return e.timestamp }

type ApproveEvent struct {
	id          string
	tokenID     TokenID
	owner       PublicKey
	spender     PublicKey
	amount      *Amount
	expiresAt   time.Time
	timestamp   time.Time
}

func NewApproveEvent(tokenID TokenID, owner, spender PublicKey, amount *Amount) *ApproveEvent {
	return &ApproveEvent{
		id:        generateEventID(),
		tokenID:   tokenID,
		owner:     owner,
		spender:   spender,
		amount:    amount,
		timestamp: time.Now(),
	}
}

func generateEventID() string {
	return ""
}
```

- [ ] **Step 3: 创建 errors.go**

```go
package token

import "errors"

var (
	ErrTokenNotFound         = errors.New("token not found")
	ErrInsufficientBalance   = errors.New("insufficient balance")
	ErrInsufficientAllowance = errors.New("insufficient allowance")
	ErrInvalidSignature      = errors.New("invalid signature")
	ErrNonceTooLow           = errors.New("nonce too low")
	ErrAmountMustBePositive  = errors.New("amount must be positive")
	ErrNotTokenOwner         = errors.New("not token owner")
	ErrTokenNotMintable      = errors.New("token not mintable")
	ErrTokenNotBurnable      = errors.New("token not burnable")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrTransferToZero        = errors.New("cannot transfer to zero address")
	ErrInvalidAmount         = errors.New("invalid amount")
	ErrDuplicateTransfer     = errors.New("duplicate transfer")
)
```

- [ ] **Step 4: 创建 validator.go**

```go
package token

import "fmt"

func ValidateTokenName(name string) error {
	if name == "" {
		return fmt.Errorf("token name is required")
	}
	if len(name) > 100 {
		return fmt.Errorf("token name too long")
	}
	return nil
}

func ValidateTokenSymbol(symbol string) error {
	if symbol == "" {
		return fmt.Errorf("token symbol is required")
	}
	if len(symbol) > 10 {
		return fmt.Errorf("token symbol too long")
	}
	return nil
}

func ValidateAmount(amount *Amount) error {
	if amount == nil || !amount.IsPositive() {
		return ErrAmountMustBePositive
	}
	return nil
}

func ValidatePublicKey(pk PublicKey) error {
	if len(pk) == 0 {
		return fmt.Errorf("public key is required")
	}
	if len(pk) != 32 {
		return fmt.Errorf("invalid public key length")
	}
	return nil
}

func ValidateNonce(nonce uint64, currentNonce uint64) error {
	if nonce <= currentNonce {
		return ErrNonceTooLow
	}
	return nil
}
```

- [ ] **Step 5: 创建 entity_test.go**

```go
package token

import (
	"math/big"
	"testing"
)

func TestNewToken(t *testing.T) {
	owner := PublicKey([]byte("owner-public-key-12345678901234"))
	supply := NewAmount(1000000)
	token := NewToken("aurora", "Aurora Token", "AUR", supply, owner)

	if token.Name() != "Aurora Token" {
		t.Errorf("Expected name 'Aurora Token', got '%s'", token.Name())
	}
	if token.Symbol() != "AUR" {
		t.Errorf("Expected symbol 'AUR', got '%s'", token.Symbol())
	}
	if token.Decimals() != 8 {
		t.Errorf("Expected decimals 8, got %d", token.Decimals())
	}
	if !token.IsMintable() {
		t.Error("Expected mintable")
	}
}

func TestAmount_NewAmountFromString(t *testing.T) {
	amount, err := NewAmountFromString("123456789")
	if err != nil {
		t.Fatalf("Failed to parse amount: %v", err)
	}
	if amount.Int64() != 123456789 {
		t.Errorf("Expected 123456789, got %d", amount.Int64())
	}

	_, err = NewAmountFromString("invalid")
	if err == nil {
		t.Fatal("Expected error for invalid amount")
	}
}

func TestAmount_IsPositive(t *testing.T) {
	pos := NewAmount(100)
	if !pos.IsPositive() {
		t.Error("Expected positive amount")
	}

	zero := NewAmount(0)
	if zero.IsPositive() {
		t.Error("Expected zero to not be positive")
	}
}

func TestApproval(t *testing.T) {
	owner := PublicKey([]byte("owner-key"))
	spender := PublicKey([]byte("spender-key"))
	amount := NewAmount(500)

	approval := NewApproval("token1", owner, spender, amount)

	if approval.Owner() != owner {
		t.Error("Owner mismatch")
	}
	if approval.Spender() != spender {
		t.Error("Spender mismatch")
	}
	if approval.Amount().Int64() != 500 {
		t.Error("Amount mismatch")
	}
}

func TestValidateAmount(t *testing.T) {
	err := ValidateAmount(NewAmount(100))
	if err != nil {
		t.Errorf("Expected valid amount, got error: %v", err)
	}

	err = ValidateAmount(NewAmount(0))
	if err != ErrAmountMustBePositive {
		t.Errorf("Expected ErrAmountMustBePositive, got: %v", err)
	}

	err = ValidateAmount(nil)
	if err != ErrAmountMustBePositive {
		t.Errorf("Expected ErrAmountMustBePositive for nil, got: %v", err)
	}
}
```

- [ ] **Step 6: 运行测试**

```bash
go test ./internal/domain/token/... -v
```

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/domain/token/
git commit -m "feat(token): add domain layer - entity, events, errors, validator"
```

---

## Task 2: Domain Layer - Service Interface

**Files:**

- Create: `internal/domain/token/service.go`
- Test: `internal/domain/token/service_test.go`

- [ ] **Step 1: 创建 service.go**

```go
package token

import "blockchain"

type Service interface {
	CreateToken(req *CreateTokenRequest) (*Token, error)
	
	Mint(req *MintRequest) (*MintEvent, error)
	Transfer(req *TransferRequest) (*TransferEvent, error)
	TransferFrom(req *TransferFromRequest) (*TransferEvent, error)
	
	Approve(req *ApproveRequest) (*ApproveEvent, error)
	IncreaseAllowance(req *AllowanceRequest) (*ApproveEvent, error)
	DecreaseAllowance(req *AllowanceRequest) (*ApproveEvent, error)
	
	Burn(req *BurnRequest) (*BurnEvent, error)
	
	GetTokenInfo(tokenID TokenID) (*Token, error)
	GetBalance(tokenID TokenID, owner PublicKey) (*Amount, error)
	GetAllowance(tokenID TokenID, owner, spender PublicKey) (*Amount, error)
	GetTransferHistory(tokenID TokenID, owner PublicKey, limit int) ([]*TransferEvent, error)
}

type TokenService struct {
	repo        Repository
	eventStore  EventStore
	chain       blockchain.BlockWriter
}

func NewService(repo Repository, eventStore EventStore, chain blockchain.BlockWriter) *TokenService {
	return &TokenService{
		repo:       repo,
		eventStore: eventStore,
		chain:      chain,
	}
}

type Repository interface {
	SaveToken(token *Token) error
	GetToken(id TokenID) (*Token, error)
	
	SaveApproval(approval *Approval) error
	GetApproval(tokenID TokenID, owner, spender PublicKey) (*Approval, error)
	GetApprovalsByOwner(tokenID TokenID, owner PublicKey) ([]*Approval, error)
	
	GetAccountBalance(tokenID TokenID, owner PublicKey) (*Amount, error)
}

type EventStore interface {
	SaveTransferEvent(event *TransferEvent) error
	SaveMintEvent(event *MintEvent) error
	SaveBurnEvent(event *BurnEvent) error
	SaveApproveEvent(event *ApproveEvent) error
	
	GetTransferEventsByToken(tokenID TokenID) ([]*TransferEvent, error)
	GetTransferEventsByOwner(tokenID TokenID, owner PublicKey) ([]*TransferEvent, error)
	GetMintEventsByToken(tokenID TokenID) ([]*MintEvent, error)
	GetBurnEventsByToken(tokenID TokenID) ([]*BurnEvent, error)
	
	GetLastNonce(tokenID TokenID, owner PublicKey) (uint64, error)
}

func (s *TokenService) CreateToken(req *CreateTokenRequest) (*Token, error) {
	if err := ValidateTokenName(req.Name); err != nil {
		return nil, err
	}
	if err := ValidateTokenSymbol(req.Symbol); err != nil {
		return nil, err
	}
	if err := ValidateAmount(req.TotalSupply); err != nil {
		return nil, err
	}
	if err := ValidatePublicKey(req.Owner); err != nil {
		return nil, err
	}
	
	token := NewToken(TokenID(req.Symbol), req.Name, req.Symbol, req.TotalSupply, req.Owner)
	
	if err := s.repo.SaveToken(token); err != nil {
		return nil, err
	}
	
	return token, nil
}

func (s *TokenService) GetTokenInfo(tokenID TokenID) (*Token, error) {
	return s.repo.GetToken(tokenID)
}

func (s *TokenService) GetBalance(tokenID TokenID, owner PublicKey) (*Amount, error) {
	return s.repo.GetAccountBalance(tokenID, owner)
}
```

- [ ] **Step 2: 补全 service.go 方法实现**

(详细实现见实际文件)

- [ ] **Step 3: Commit**

```bash
git add internal/domain/token/service.go
git commit -m "feat(token): add domain service interface"
```

---

## Task 3: Infrastructure Layer - SQLite Repository

**Files:**

- Create: `internal/infra/sqlite/token.go`
- Create: `internal/infra/sqlite/event_store.go`
- Create: `internal/infra/sqlite/allowance.go`
- Test: `internal/infra/sqlite/token_test.go`

- [ ] **Step 1: 创建 token.go**

```go
package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pplmx/aurora/internal/domain/token"
	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
)

type TokenRepository struct {
	db     *sql.DB
	dbPath string
}

func NewTokenRepository(dbPath string) (*TokenRepository, error) {
	database, err := sql.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=ON", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	
	repo := &TokenRepository{
		db:     database,
		dbPath: dbPath,
	}
	
	if err := repo.createTables(); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}
	
	return repo, nil
}

func (r *TokenRepository) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS tokens (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			symbol TEXT NOT NULL,
			total_supply TEXT NOT NULL,
			decimals INTEGER DEFAULT 8,
			owner TEXT NOT NULL,
			is_mintable BOOLEAN DEFAULT 1,
			is_burnable BOOLEAN DEFAULT 1,
			created_at INTEGER
		)`,
	}
	
	for _, q := range queries {
		if _, err := r.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func (r *TokenRepository) SaveToken(t *token.Token) error {
	ownerB64 := base64.StdEncoding.EncodeToString(t.Owner())
	totalSupplyJSON, _ := json.Marshal(t.TotalSupply())
	
	_, err := r.db.Exec(`
		INSERT INTO tokens (id, name, symbol, total_supply, decimals, owner, is_mintable, is_burnable, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID(), t.Name(), t.Symbol(), string(totalSupplyJSON), t.Decimals(), ownerB64, t.IsMintable(), t.IsBurnable(), t.CreatedAt().Unix())
	return err
}

func (r *TokenRepository) GetToken(id token.TokenID) (*token.Token, error) {
	row := r.db.QueryRow("SELECT id, name, symbol, total_supply, decimals, owner, is_mintable, is_burnable, created_at FROM tokens WHERE id = ?", id)
	
	var t token.Token
	var name, symbol, totalSupplyJSON, ownerB64 string
	var decimals int8
	var isMintable, isBurnable bool
	var createdAt int64
	
	err := row.Scan(&t.id, &name, &symbol, &totalSupplyJSON, &decimals, &ownerB64, &isMintable, &isBurnable, &createdAt)
	if err == sql.ErrNoRows {
		return nil, token.ErrTokenNotFound
	}
	if err != nil {
		return nil, err
	}
	
	owner, _ := base64.StdEncoding.DecodeString(ownerB64)
	amount, _ := token.NewAmountFromString(totalSupplyJSON)
	
	return &token.Token{
		id:          t.id,
		name:        name,
		symbol:      symbol,
		totalSupply: amount,
		decimals:    decimals,
		owner:       owner,
		isMintable:  isMintable,
		isBurnable:  isBurnable,
		createdAt:   time.Unix(createdAt, 0),
	}, nil
}
```

- [ ] **Step 2: 创建 event_store.go**

```go
package sqlite

type EventStore struct {
	db *sql.DB
}

func NewEventStore(db *sql.DB) *EventStore {
	return &EventStore{db: db}
}

func (e *EventStore) SaveTransferEvent(event *token.TransferEvent) error {
	fromB64 := base64.StdEncoding.EncodeToString(event.From())
	toB64 := base64.StdEncoding.EncodeToString(event.To())
	sigB64 := base64.StdEncoding.EncodeToString(event.Signature())
	
	_, err := e.db.Exec(`
		INSERT INTO transfer_events (id, token_id, from_owner, to_owner, amount, nonce, signature, block_height, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, event.ID(), event.TokenID(), fromB64, toB64, event.Amount().String(), event.Nonce(), sigB64, event.BlockHeight(), event.Timestamp().Unix())
	return err
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/infra/sqlite/
git commit -m "feat(token): add infra/sqlite repository implementation"
```

---

## Task 4: Application Layer - Use Cases

**Files:**

- Create: `internal/app/token/dto.go`
- Create: `internal/app/token/create.go`
- Create: `internal/app/token/mint.go`
- Create: `internal/app/token/transfer.go`
- Create: `internal/app/token/approve.go`
- Create: `internal/app/token/burn.go`
- Test: `internal/app/token/usecase_test.go`

- [ ] **Step 1: 创建 dto.go**

```go
package token

type CreateTokenRequest struct {
	Name        string
	Symbol      string
	TotalSupply string
	Owner       string // base64
}

type CreateTokenResponse struct {
	ID          string
	Name        string
	Symbol      string
	TotalSupply string
	Decimals    int8
	Owner       string
}

type MintRequest struct {
	TokenID    string
	To         string // base64
	Amount     string
	PrivateKey string // base64
}

type MintResponse struct {
	ID        string
	TokenID   string
	To        string
	Amount    string
	Timestamp int64
}

type TransferRequest struct {
	TokenID    string
	From       string // base64
	To         string // base64
	Amount     string
	PrivateKey string // base64
}

type TransferResponse struct {
	ID        string
	TokenID   string
	From      string
	To        string
	Amount    string
	Timestamp int64
}

type ApproveRequest struct {
	TokenID    string
	Spender    string // base64
	Amount     string
	PrivateKey string // base64
}

type ApproveResponse struct {
	ID        string
	TokenID   string
	Owner     string
	Spender   string
	Amount    string
	Timestamp int64
}

type BurnRequest struct {
	TokenID    string
	Amount     string
	PrivateKey string // base64
}

type BurnResponse struct {
	ID        string
	TokenID   string
	From      string
	Amount    string
	Timestamp int64
}
```

- [ ] **Step 2: 实现 use cases**

(详细实现见实际文件)

- [ ] **Step 3: Commit**

```bash
git add internal/app/token/
git commit -m "feat(token): add application layer use cases"
```

---

## Task 5: CLI Command

**Files:**

- Create: `cmd/aurora/cmd/token.go`

- [ ] **Step 1: 创建 token.go**

```go
package cmd

import (
	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Token management",
}

func init() {
	rootCmd.AddCommand(tokenCmd)
	
	tokenCmd.AddCommand(tokenCreateCmd)
	tokenCmd.AddCommand(tokenMintCmd)
	tokenCmd.AddCommand(tokenTransferCmd)
	tokenCmd.AddCommand(tokenApproveCmd)
	tokenCmd.AddCommand(tokenBurnCmd)
	tokenCmd.AddCommand(tokenBalanceCmd)
	tokenCmd.AddCommand(tokenInfoCmd)
	tokenCmd.AddCommand(tokenTuiCmd)
}
```

- [ ] **Step 2: Commit**

```bash
git add cmd/aurora/cmd/token.go
git commit -m "feat(token): add CLI commands"
```

---

## Task 6: TUI Interface

**Files:**

- Create: `internal/ui/token/tui.go`

- [ ] **Step 1: 创建 tui.go**

```go
package token

import (
	"github.com/charmbracelet/bubbletea"
)

type TokenTUI struct {
	// state
}

func NewTokenTUI() *TokenTUI {
	return &TokenTUI{}
}

func (m *TokenTUI) Init() tea.Model {
	return m
}

func (m *TokenTUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *TokenTUI) View() string {
	return "Token TUI"
}

func RunTokenTUI() error {
	p := tea.NewProgram(NewTokenTUI())
	_, err := p.Run()
	return err
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/ui/token/tui.go
git commit -m "feat(token): add TUI interface"
```

---

## Task 7: Integration & Testing

**Files:**

- Test: `e2e/token_e2e_test.go`

- [ ] **Step 1: 创建 e2e 测试**

```go
package test

import (
	"testing"
	
	"github.com/pplmx/aurora/internal/domain/token"
)

func TestToken_CreateAndTransfer(t *testing.T) {
	// 1. Create token
	// 2. Mint tokens
	// 3. Transfer tokens
	// 4. Verify balance
}
```

- [ ] **Step 2: 运行所有测试**

```bash
go test ./internal/domain/token/... -v
go test ./internal/app/token/... -v
go test ./internal/infra/sqlite/... -v
go test ./e2e/... -v
```

- [ ] **Step 3: 运行 linter**

```bash
golangci-lint run
```

- [ ] **Step 4: 最终 commit**

```bash
git add -A
git commit -m "feat: complete token (FT) system implementation

- Event Sourcing architecture
- DDD four-layer architecture
- Ed25519 signature verification
- CLI and TUI support
- Full test coverage"
```

---

## 总结

此计划包含 7 个主要任务，涵盖:

- Domain Layer (entity, event, service, errors, validator)
- Infrastructure Layer (SQLite repository, event store)
- Application Layer (use cases with DTOs)
- CLI commands
- TUI interface
- E2E tests

**Plan complete and saved to `docs/superpowers/plans/2026-04-06-token-plan.md`. Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
