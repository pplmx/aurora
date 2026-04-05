# Aurora DDD 架构重构实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 Aurora 从单体模块重构为完整 DDD 架构，分离 domain/infra/app/ui 层

**Architecture:** 
- domain/ 层：实体、服务接口、仓储接口（无外部依赖）
- infra/ 层：SQLite 实现、加密实现、HTTP 客户端
- app/ 层：用例（Use Case）编排业务逻辑
- ui/ 层：TUI 界面
- cmd/ 层：CLI 入口

**Tech Stack:** Go 1.26+, Cobra, Bubble Tea v2, SQLite

---

## 文件结构映射

### 新目录（需要创建）
```
internal/domain/lottery/
internal/domain/voting/
internal/domain/nft/
internal/domain/oracle/
internal/domain/blockchain/
internal/infra/sqlite/
internal/infra/crypto/
internal/infra/http/
internal/app/lottery/
internal/app/voting/
internal/app/nft/
internal/app/oracle/
internal/ui/lottery/
internal/ui/voting/
internal/ui/nft/
internal/ui/oracle/
```

### 现有文件（需要迁移/修改）
```
internal/lottery/*      → internal/domain/lottery/ + internal/ui/lottery/
internal/voting/*       → internal/domain/voting/ + internal/ui/voting/
internal/nft/*          → internal/domain/nft/ + internal/ui/nft/
internal/oracle/*       → internal/domain/oracle/ + internal/ui/oracle/
internal/blockchain/*   → internal/domain/blockchain/ + internal/infra/sqlite/
cmd/aurora/cmd/*        → 更新 import 路径
```

---

## Task 1: 创建目录结构和 blockchain domain

**Files:**
- Create: `internal/domain/blockchain/block.go`
- Create: `internal/domain/blockchain/repo.go`
- Create: `internal/infra/sqlite/blockchain.go`

- [ ] **Step 1: 创建 domain/blockchain 目录和 entity**

```go
// internal/domain/blockchain/block.go
package blockchain

type Block struct {
    Height    int64
    Hash      []byte
    PrevHash  []byte
    Data      []byte
    Nonce     int64
    Timestamp int64
}

type BlockChain struct {
    Blocks []*Block
}

func NewBlock(data string, prevHash []byte) *Block { ... }
func Genesis() *Block { ... }
func (c *BlockChain) AddBlock(data string) int64 { ... }
```

- [ ] **Step 2: 创建 blockchain repository 接口**

```go
// internal/domain/blockchain/repo.go
package blockchain

type Repository interface {
    SaveBlock(height int64, block *Block) error
    GetBlock(height int64) (*Block, error)
    GetAllBlocks() ([]*Block, error)
    GetLotteryRecords() ([]string, error)
    AddLotteryRecord(data string) (int64, error)
    Close() error
}
```

- [ ] **Step 3: 创建 infra/sqlite/blockchain 实现**

```go
// internal/infra/sqlite/blockchain.go
package sqlite

type BlockchainRepository struct {
    db *sql.DB
}

func NewBlockchainRepository(db *sql.DB) *BlockchainRepository { ... }

func (r *BlockchainRepository) SaveBlock(height int64, block *domain.Block) error { ... }
func (r *BlockchainRepository) GetBlock(height int64) (*domain.Block, error) { ... }
func (r *BlockchainRepository) GetAllBlocks() ([]*domain.Block, error) { ... }
func (r *BlockchainRepository) GetLotteryRecords() ([]string, error) { ... }
func (r *BlockchainRepository) AddLotteryRecord(data string) (int64, error) { ... }
func (r *BlockchainRepository) Close() error { ... }
```

- [ ] **Step 4: 验证编译**

Run: `go build ./internal/domain/blockchain/... ./internal/infra/sqlite/...`
Expected: 编译通过

- [ ] **Step 5: 提交**

```bash
git add internal/domain/blockchain/ internal/infra/sqlite/blockchain.go
git commit -m "refactor: create blockchain domain and infra layers"
```

---

## Task 2: 重构 lottery 模块

**Files:**
- Create: `internal/domain/lottery/entity.go`
- Create: `internal/domain/lottery/service.go`
- Create: `internal/domain/lottery/repo.go`
- Create: `internal/infra/sqlite/lottery.go`
- Create: `internal/app/lottery/dto.go`
- Create: `internal/app/lottery/usecase.go`
- Create: `internal/ui/lottery/tui.go`
- Modify: `cmd/aurora/cmd/lottery.go`

- [ ] **Step 1: 创建 lottery entity**

```go
// internal/domain/lottery/entity.go
package lottery

type LotteryRecord struct {
    ID              string
    BlockHeight     int64
    Seed            string
    Participants    []string
    Winners         []string
    WinnerAddresses []string
    Timestamp       int64
    Verified        bool
}

func (r *LotteryRecord) Validate() error { ... }
func (r *LotteryRecord) GetWinners() []string { ... }
```

- [ ] **Step 2: 创建 lottery service 接口**

```go
// internal/domain/lottery/service.go
package lottery

type Service interface {
    DrawWinners(participants []string, seed string, count int) ([]string, error)
    VerifyDraw(record *LotteryRecord) (bool, error)
}

type lotteryService struct {
    vrf VRF
}

func NewService(vrf VRF) Service { ... }
func (s *lotteryService) DrawWinners(...) ([]string, error) { ... }
func (s *lotteryService) VerifyDraw(record *LotteryRecord) (bool, error) { ... }
```

- [ ] **Step 3: 创建 lottery repository 接口**

```go
// internal/domain/lottery/repo.go
package lottery

type Repository interface {
    Save(record *LotteryRecord) error
    GetByID(id string) (*LotteryRecord, error)
    GetAll() ([]*LotteryRecord, error)
    GetByBlockHeight(height int64) ([]*LotteryRecord, error)
}
```

- [ ] **Step 4: 从现有代码复制 VRF 实现到 domain**

从 `internal/lottery/vrf.go` 移动 VRF 核心逻辑到 `internal/domain/lottery/vrf.go`

- [ ] **Step 5: 创建 SQLite 实现**

```go
// internal/infra/sqlite/lottery.go
package sqlite

type LotteryRepository struct {
    db *sql.DB
}

func NewLotteryRepository(db *sql.DB) *LotteryRepository { ... }
func (r *LotteryRepository) Save(record *domain.LotteryRecord) error { ... }
func (r *LotteryRepository) GetByID(id string) (*domain.LotteryRecord, error) { ... }
func (r *LotteryRepository) GetAll() ([]*domain.LotteryRecord, error) { ... }
```

- [ ] **Step 6: 创建 app layer (DTO + UseCase)**

```go
// internal/app/lottery/dto.go
package lottery

type CreateLotteryRequest struct {
    Participants string // comma-separated
    Seed         string
    WinnerCount  int
}

type LotteryResponse struct {
    ID              string
    BlockHeight     int64
    Winners         []string
    WinnerAddresses []string
}
```

```go
// internal/app/lottery/usecase.go
package lottery

type CreateLotteryUseCase struct {
    lotteryRepo domain.LotteryRepository
    blockRepo   domain.BlockchainRepository
    service     domain.LotteryService
}

func NewCreateLotteryUseCase(...) *CreateLotteryUseCase { ... }
func (uc *CreateLotteryUseCase) Execute(req CreateLotteryRequest) (*LotteryResponse, error) { ... }
```

- [ ] **Step 7: 移动 TUI 到 ui/lottery**

从 `internal/lottery/tui.go` 移动到 `internal/ui/lottery/tui.go`，更新 import

- [ ] **Step 8: 更新 cmd/lottery.go**

更新 import 使用新的 app 层

- [ ] **Step 9: 验证编译**

Run: `go build ./...`
Expected: 编译通过，所有测试通过

- [ ] **Step 10: 提交**

```bash
git add internal/domain/lottery/ internal/infra/sqlite/lottery.go internal/app/lottery/ internal/ui/lottery/
git commit -m "refactor: migrate lottery to DDD structure"
```

---

## Task 3: 重构 voting 模块

**Files:**
- Create: `internal/domain/voting/entity.go`
- Create: `internal/domain/voting/service.go`
- Create: `internal/domain/voting/repo.go`
- Create: `internal/infra/sqlite/voting.go`
- Create: `internal/app/voting/dto.go`
- Create: `internal/app/voting/usecase.go`
- Create: `internal/ui/voting/tui.go`
- Modify: `cmd/aurora/cmd/voting.go`

- [ ] **Step 1: 创建 voting entity (Vote, Voter, Candidate, Session)**

```go
// internal/domain/voting/entity.go
package voting

type Vote struct {
    ID        string
    Voter     string
    Candidate string
    Signature string
    Timestamp int64
}

type Voter struct {
    ID        string
    Name      string
    PublicKey []byte
}

type Candidate struct {
    ID      string
    Name    string
    Party   string
    Program string
    Votes   int
}

type Session struct {
    ID          string
    Title       string
    Description string
    Candidates  []string
    Status      string // active/ended
    StartTime   int64
    EndTime     int64
}
```

- [ ] **Step 2: 创建 voting service**

```go
// internal/domain/voting/service.go
package voting

type Service interface {
    SignVote(vote *Vote, privateKey []byte) (string, error)
    VerifyVote(vote *Vote) (bool, error)
    CountVotes(candidates []Candidate) map[string]int
}
```

- [ ] **Step 3: 创建 voting repository 接口**

```go
// internal/domain/voting/repo.go
package voting

type Repository interface {
    SaveVote(vote *Vote) error
    GetVote(id string) (*Vote, error)
    SaveVoter(voter *Voter) error
    GetVoter(id string) (*Voter, error)
    SaveCandidate(candidate *Candidate) error
    GetCandidate(id string) (*Candidate, error)
    GetAllCandidates() ([]*Candidate, error)
    SaveSession(session *Session) error
    GetSession(id string) (*Session, error)
}
```

- [ ] **Step 4: 创建 SQLite 实现**

```go
// internal/infra/sqlite/voting.go
package sqlite

type VotingRepository struct {
    db *sql.DB
}
// 实现 domain/voting.Repository 接口
```

- [ ] **Step 5: 创建 app layer**

```go
// internal/app/voting/dto.go
package voting

type CastVoteRequest struct {
    VoterPublicKey string
    CandidateID    string
    PrivateKey     string
}

type RegisterVoterRequest struct {
    Name string
}

// internal/app/voting/usecase.go
type CastVoteUseCase struct { ... }
type RegisterVoterUseCase struct { ... }
```

- [ ] **Step 6: 移动 TUI**

移动 `internal/voting/tui.go` 到 `internal/ui/voting/tui.go`

- [ ] **Step 7: 更新 cmd/voting.go**

- [ ] **Step 8: 验证和提交**

Run: `go build ./...`
Commit: `git commit -m "refactor: migrate voting to DDD structure"`

---

## Task 4: 重构 nft 模块

**Files:**
- Create: `internal/domain/nft/entity.go`
- Create: `internal/domain/nft/service.go`
- Create: `internal/domain/nft/repo.go`
- Create: `internal/infra/sqlite/nft.go`
- Create: `internal/app/nft/dto.go`
- Create: `internal/app/nft/usecase.go`
- Create: `internal/ui/nft/tui.go`
- Modify: `cmd/aurora/cmd/nft.go`

- [ ] **Step 1: 创建 nft entity**

```go
// internal/domain/nft/entity.go
package nft

type NFT struct {
    ID          string
    Name        string
    Description string
    ImageURL    string
    TokenURI    string
    Owner       []byte
    Creator     []byte
    Timestamp   int64
}

type Operation struct {
    ID        string
    NFTID     string
    Type      string // mint/transfer/burn
    From      []byte
    To        []byte
    Signature []byte
    Timestamp int64
}
```

- [ ] **Step 2: 创建 nft service**

```go
// internal/domain/nft/service.go
package nft

type Service interface {
    Mint(nft *NFT) error
    Transfer(nftID string, from, to []byte, privateKey []byte) error
    Burn(nftID string, owner []byte, privateKey []byte) error
    VerifyTransfer(nft *NFT, op *Operation) (bool, error)
}
```

- [ ] **Step 3: 创建 nft repository 接口**

```go
// internal/domain/nft/repo.go
package nft

type Repository interface {
    SaveNFT(nft *NFT) error
    GetNFT(id string) (*NFT, error)
    GetNFTsByOwner(owner []byte) ([]*NFT, error)
    DeleteNFT(id string) error
    SaveOperation(op *Operation) error
    GetOperations(nftID string) ([]*Operation, error)
}
```

- [ ] **Step 4-8: SQLite 实现, app layer, TUI, cmd, 验证**

(同 voting 模式)

- [ ] **Step 9: 提交**

```bash
git commit -m "refactor: migrate nft to DDD structure"
```

---

## Task 5: 重构 oracle 模块

**Files:**
- Create: `internal/domain/oracle/entity.go`
- Create: `internal/domain/oracle/service.go`
- Create: `internal/domain/oracle/repo.go`
- Create: `internal/infra/sqlite/oracle.go`
- Create: `internal/infra/http/fetcher.go`
- Create: `internal/app/oracle/dto.go`
- Create: `internal/app/oracle/usecase.go`
- Create: `internal/ui/oracle/tui.go`
- Modify: `cmd/aurora/cmd/oracle.go`

- [ ] **Step 1: 创建 oracle entity**

```go
// internal/domain/oracle/entity.go
package oracle

type OracleData struct {
    ID        string
    SourceID  string
    Value     string
    Timestamp int64
}

type DataSource struct {
    ID       string
    Name     string
    Type     string // http/api
    URL      string
    Enabled  bool
}
```

- [ ] **Step 2: 创建 oracle service**

```go
// internal/domain/oracle/service.go
package oracle

type Service interface {
    FetchData(source *DataSource) (*OracleData, error)
}
```

- [ ] **Step 3: 创建 oracle repository 接口**

```go
// internal/domain/oracle/repo.go
package oracle

type Repository interface {
    SaveData(data *OracleData) error
    GetData(id string) (*OracleData, error)
    GetLatestData(sourceID string) (*OracleData, error)
    SaveSource(source *DataSource) error
    GetSource(id string) (*DataSource, error)
    GetAllSources() ([]*DataSource, error)
}
```

- [ ] **Step 4: 创建 HTTP fetcher**

```go
// internal/infra/http/fetcher.go
package http

type Fetcher struct {
    client *http.Client
}

func (f *Fetcher) Get(url string) ([]byte, error) { ... }
```

- [ ] **Step 5-9: SQLite 实现, app layer, TUI, cmd, 验证**

(同模式)

- [ ] **Step 10: 提交**

```bash
git commit -m "refactor: migrate oracle to DDD structure"
```

---

## Task 6: 清理和最终验证

- [ ] **Step 1: 删除旧目录**

```bash
rm -rf internal/lottery internal/voting internal/nft internal/oracle
```

- [ ] **Step 2: 验证编译**

Run: `go build ./...`
Expected: 编译通过

- [ ] **Step 3: 运行测试**

Run: `go test ./...`
Expected: 所有测试通过

- [ ] **Step 4: 运行 lint**

Run: `just lint`
Expected: 0 issues

- [ ] **Step 5: 功能验证**

```bash
./aurora lottery create -p "A,B,C,D" -s "seed" -c 3
./aurora lottery history
./aurora version
```

- [ ] **Step 6: 最终提交**

```bash
git add -A
git commit -m "refactor: complete DDD architecture restructure"
```

---

## 预期结果

- 清晰的 DDD 分层结构
- 可单独测试的 domain service
- 可替换的存储实现
- 与重构前完全相同的功能
