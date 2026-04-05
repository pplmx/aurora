# Voting 投票系统实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现基于 Ed25519 签名和区块链存储的透明投票系统，支持候选人注册、投票人注册、匿名投票、实时计票和结果上链存证

**Architecture:** 使用 Go 标准库 Ed25519 签名，Bubble Tea 构建 TUI，SQLite + 内存混合存储，复用现有 blockchain 模块上链

**Tech Stack:** Go 1.26+, crypto/ed25519, github.com/mattn/go-sqlite3, Bubble Tea, Cobra

---

## 文件结构

```text
internal/
├── voting/
│   ├── candidate.go       # 候选人管理
│   ├── voter.go           # 投票人管理（已存在，需扩展）
│   ├── vote.go            # 投票逻辑（已存在，需扩展）
│   ├── session.go         # 投票会话
│   ├── storage.go         # SQLite 存储
│   ├── storage_test.go    # 存储测试
│   ├── tui.go             # TUI 界面
│   └── voting_test.go     # 单元测试
├── blockchain/
│   └── block.go           # 已存在，复用上链
└── ...

cmd/aurora/cmd/
└── voting.go              # CLI 命令
```

---

## Task 1: 依赖添加

**Files:**

- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: 添加 SQLite 依赖**

Run: `go get github.com/mattn/go-sqlite3`

- [ ] **Step 2: 添加 UUID 依赖**

Run: `go get github.com/google/uuid`

- [ ] **Step 3: 运行 go mod tidy**

Run: `go mod tidy`

- [ ] **Step 4: Commit**

```bash
go mod tidy
git add go.mod go.sum
git commit -m "deps: add sqlite3 and uuid for voting"
```

---

## Task 2: SQLite 存储层

**Files:**

- Create: `internal/voting/storage.go`

- [ ] **Step 1: 创建存储接口和数据结构**

```go
package voting

import "github.com/google/uuid"

type Candidate struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Party       string `json:"party"`
    Program     string `json:"program"`
    Description string `json:"description"`
    ImageURL    string `json:"image_url"`
    VoteCount   int    `json:"vote_count"`
    CreatedAt   int64  `json:"created_at"`
}

type Voter struct {
    PublicKey    string `json:"public_key"`
    Name         string `json:"name"`
    HasVoted     bool   `json:"has_voted"`
    VoteHash     string `json:"vote_hash"`
    RegisteredAt int64  `json:"registered_at"`
}

type VoteRecord struct {
    ID          string `json:"id"`
    VoterPK     string `json:"voter_pk"`
    CandidateID string `json:"candidate_id"`
    Signature   string `json:"signature"`
    Message     string `json:"message"`
    Timestamp   int64  `json:"timestamp"`
    BlockHeight int64  `json:"block_height"`
}

type VotingSession struct {
    ID          string   `json:"id"`
    Title       string   `json:"title"`
    Description string   `json:"description"`
    StartTime   int64    `json:"start_time"`
    EndTime     int64    `json:"end_time"`
    Status      string   `json:"status"`
    Candidates  []string `json:"candidates"`
    CreatedAt   int64    `json:"created_at"`
}

type Storage interface {
    // Candidate
    SaveCandidate(c *Candidate) error
    GetCandidate(id string) (*Candidate, error)
    ListCandidates() ([]*Candidate, error)
    UpdateCandidate(c *Candidate) error
    DeleteCandidate(id string) error

    // Voter
    SaveVoter(v *Voter) error
    GetVoter(pk string) (*Voter, error)
    UpdateVoter(v *Voter) error
    ListVoters() ([]*Voter, error)

    // Vote
    SaveVote(v *VoteRecord) error
    GetVote(id string) (*VoteRecord, error)
    GetVotesByCandidate(candidateID string) ([]*VoteRecord, error)
    GetVotesByVoter(voterPK string) ([]*VoteRecord, error)
    ListVotes() ([]*VoteRecord, error)

    // Session
    SaveSession(s *VotingSession) error
    GetSession(id string) (*VotingSession, error)
    ListSessions() ([]*VotingSession, error)
    UpdateSession(s *VotingSession) error

    // Transaction
    Begin() error
    Commit() error
    Rollback() error
    Close() error
}
```

- [ ] **Step 2: 创建 SQLite 实现**

```go
package voting

import (
    "database/sql"
    "time"

    _ "github.com/mattn/go-sqlite3"
)

type SQLiteStorage struct {
    db *sql.DB
}

func NewSQLiteStorage(path string) (*SQLiteStorage, error) {
    db, err := sql.Open("sqlite3", path)
    if err != nil {
        return nil, err
    }

    s := &SQLiteStorage{db: db}
    if err := s.initTables(); err != nil {
        return nil, err
    }
    return s, nil
}

func (s *SQLiteStorage) initTables() error {
    queries := []string{
        `CREATE TABLE IF NOT EXISTS candidates (
            id TEXT PRIMARY KEY,
            name TEXT NOT NULL,
            party TEXT,
            program TEXT,
            description TEXT,
            image_url TEXT,
            vote_count INTEGER DEFAULT 0,
            created_at INTEGER
        )`,
        `CREATE TABLE IF NOT EXISTS voters (
            public_key TEXT PRIMARY KEY,
            name TEXT,
            has_voted INTEGER DEFAULT 0,
            vote_hash TEXT,
            registered_at INTEGER
        )`,
        `CREATE TABLE IF NOT EXISTS votes (
            id TEXT PRIMARY KEY,
            voter_pk TEXT NOT NULL,
            candidate_id TEXT NOT NULL,
            signature TEXT NOT NULL,
            message TEXT NOT NULL,
            timestamp INTEGER,
            block_height INTEGER,
            FOREIGN KEY (candidate_id) REFERENCES candidates(id)
        )`,
        `CREATE TABLE IF NOT EXISTS voting_sessions (
            id TEXT PRIMARY KEY,
            title TEXT NOT NULL,
            description TEXT,
            start_time INTEGER,
            end_time INTEGER,
            status TEXT DEFAULT 'draft',
            created_at INTEGER
        )`,
        `CREATE TABLE IF NOT EXISTS session_candidates (
            session_id TEXT,
            candidate_id TEXT,
            PRIMARY KEY (session_id, candidate_id)
        )`,
    }

    for _, q := range queries {
        if _, err := s.db.Exec(q); err != nil {
            return err
        }
    }
    return nil
}

// Implement all Storage interface methods...
func (s *SQLiteStorage) SaveCandidate(c *Candidate) error {
    _, err := s.db.Exec(
        `INSERT OR REPLACE INTO candidates (id, name, party, program, description, image_url, vote_count, created_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
        c.ID, c.Name, c.Party, c.Program, c.Description, c.ImageURL, c.VoteCount, c.CreatedAt,
    )
    return err
}

func (s *SQLiteStorage) GetCandidate(id string) (*Candidate, error) {
    row := s.db.QueryRow(`SELECT id, name, party, program, description, image_url, vote_count, created_at
        FROM candidates WHERE id = ?`, id)

    var c Candidate
    err := row.Scan(&c.ID, &c.Name, &c.Party, &c.Program, &c.Description, &c.ImageURL, &c.VoteCount, &c.CreatedAt)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    return &c, err
}

func (s *SQLiteStorage) ListCandidates() ([]*Candidate, error) {
    rows, err := s.db.Query(`SELECT id, name, party, program, description, image_url, vote_count, created_at FROM candidates`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var candidates []*Candidate
    for rows.Next() {
        var c Candidate
        if err := rows.Scan(&c.ID, &c.Name, &c.Party, &c.Program, &c.Description, &c.ImageURL, &c.VoteCount, &c.CreatedAt); err != nil {
            return nil, err
        }
        candidates = append(candidates, &c)
    }
    return candidates, nil
}

func (s *SQLiteStorage) UpdateCandidate(c *Candidate) error {
    return s.SaveCandidate(c)
}

func (s *SQLiteStorage) DeleteCandidate(id string) error {
    _, err := s.db.Exec(`DELETE FROM candidates WHERE id = ?`, id)
    return err
}

// Voter methods
func (s *SQLiteStorage) SaveVoter(v *Voter) error {
    _, err := s.db.Exec(
        `INSERT OR REPLACE INTO voters (public_key, name, has_voted, vote_hash, registered_at)
         VALUES (?, ?, ?, ?, ?)`,
        v.PublicKey, v.Name, v.HasVoted, v.VoteHash, v.RegisteredAt,
    )
    return err
}

func (s *SQLiteStorage) GetVoter(pk string) (*Voter, error) {
    row := s.db.QueryRow(`SELECT public_key, name, has_voted, vote_hash, registered_at FROM voters WHERE public_key = ?`, pk)

    var v Voter
    var hasVoted int
    err := row.Scan(&v.PublicKey, &v.Name, &hasVoted, &v.VoteHash, &v.RegisteredAt)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    v.HasVoted = hasVoted == 1
    return &v, err
}

func (s *SQLiteStorage) UpdateVoter(v *Voter) error {
    return s.SaveVoter(v)
}

func (s *SQLiteStorage) ListVoters() ([]*Voter, error) {
    rows, err := s.db.Query(`SELECT public_key, name, has_voted, vote_hash, registered_at FROM voters`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var voters []*Voter
    for rows.Next() {
        var v Voter
        var hasVoted int
        if err := rows.Scan(&v.PublicKey, &v.Name, &hasVoted, &v.VoteHash, &v.RegisteredAt); err != nil {
            return nil, err
        }
        v.HasVoted = hasVoted == 1
        voters = append(voters, &v)
    }
    return voters, nil
}

// Vote methods
func (s *SQLiteStorage) SaveVote(v *VoteRecord) error {
    _, err := s.db.Exec(
        `INSERT OR REPLACE INTO votes (id, voter_pk, candidate_id, signature, message, timestamp, block_height)
         VALUES (?, ?, ?, ?, ?, ?, ?)`,
        v.ID, v.VoterPK, v.CandidateID, v.Signature, v.Message, v.Timestamp, v.BlockHeight,
    )
    return err
}

func (s *SQLiteStorage) GetVote(id string) (*VoteRecord, error) {
    row := s.db.QueryRow(`SELECT id, voter_pk, candidate_id, signature, message, timestamp, block_height
        FROM votes WHERE id = ?`, id)

    var v VoteRecord
    err := row.Scan(&v.ID, &v.VoterPK, &v.CandidateID, &v.Signature, &v.Message, &v.Timestamp, &v.BlockHeight)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    return &v, err
}

func (s *SQLiteStorage) GetVotesByCandidate(candidateID string) ([]*VoteRecord, error) {
    rows, err := s.db.Query(`SELECT id, voter_pk, candidate_id, signature, message, timestamp, block_height
        FROM votes WHERE candidate_id = ?`, candidateID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var votes []*VoteRecord
    for rows.Next() {
        var v VoteRecord
        if err := rows.Scan(&v.ID, &v.VoterPK, &v.CandidateID, &v.Signature, &v.Message, &v.Timestamp, &v.BlockHeight); err != nil {
            return nil, err
        }
        votes = append(votes, &v)
    }
    return votes, nil
}

func (s *SQLiteStorage) GetVotesByVoter(voterPK string) ([]*VoteRecord, error) {
    rows, err := s.db.Query(`SELECT id, voter_pk, candidate_id, signature, message, timestamp, block_height
        FROM votes WHERE voter_pk = ?`, voterPK)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var votes []*VoteRecord
    for rows.Next() {
        var v VoteRecord
        if err := rows.Scan(&v.ID, &v.VoterPK, &v.CandidateID, &v.Signature, &v.Message, &v.Timestamp, &v.BlockHeight); err != nil {
            return nil, err
        }
        votes = append(votes, &v)
    }
    return votes, nil
}

func (s *SQLiteStorage) ListVotes() ([]*VoteRecord, error) {
    rows, err := s.db.Query(`SELECT id, voter_pk, candidate_id, signature, message, timestamp, block_height FROM votes`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var votes []*VoteRecord
    for rows.Next() {
        var v VoteRecord
        if err := rows.Scan(&v.ID, &v.VoterPK, &v.CandidateID, &v.Signature, &v.Message, &v.Timestamp, &v.BlockHeight); err != nil {
            return nil, err
        }
        votes = append(votes, &v)
    }
    return votes, nil
}

// Session methods
func (s *SQLiteStorage) SaveSession(session *VotingSession) error {
    tx, err := s.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    _, err = tx.Exec(
        `INSERT OR REPLACE INTO voting_sessions (id, title, description, start_time, end_time, status, created_at)
         VALUES (?, ?, ?, ?, ?, ?, ?)`,
        session.ID, session.Title, session.Description, session.StartTime, session.EndTime, session.Status, session.CreatedAt,
    )
    if err != nil {
        return err
    }

    // Clear and re-insert session candidates
    tx.Exec(`DELETE FROM session_candidates WHERE session_id = ?`, session.ID)
    for _, cid := range session.Candidates {
        tx.Exec(`INSERT INTO session_candidates (session_id, candidate_id) VALUES (?, ?)`, session.ID, cid)
    }

    return tx.Commit()
}

func (s *SQLiteStorage) GetSession(id string) (*VotingSession, error) {
    row := s.db.QueryRow(`SELECT id, title, description, start_time, end_time, status, created_at
        FROM voting_sessions WHERE id = ?`, id)

    var sess VotingSession
    err := row.Scan(&sess.ID, &sess.Title, &sess.Description, &sess.StartTime, &sess.EndTime, &sess.Status, &sess.CreatedAt)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }

    // Load candidates
    rows, err := s.db.Query(`SELECT candidate_id FROM session_candidates WHERE session_id = ?`, id)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    for rows.Next() {
        var cid string
        rows.Scan(&cid)
        sess.Candidates = append(sess.Candidates, cid)
    }

    return &sess, nil
}

func (s *SQLiteStorage) ListSessions() ([]*VotingSession, error) {
    rows, err := s.db.Query(`SELECT id, title, description, start_time, end_time, status, created_at FROM voting_sessions`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var sessions []*VotingSession
    for rows.Next() {
        var sess VotingSession
        if err := rows.Scan(&sess.ID, &sess.Title, &sess.Description, &sess.StartTime, &sess.EndTime, &sess.Status, &sess.CreatedAt); err != nil {
            return nil, err
        }
        sessions = append(sessions, &sess)
    }
    return sessions, nil
}

func (s *SQLiteStorage) UpdateSession(session *VotingSession) error {
    return s.SaveSession(session)
}

// Transaction stubs
func (s *SQLiteStorage) Begin() error { return nil }
func (s *SQLiteStorage) Commit() error { return nil }
func (s *SQLiteStorage) Rollback() error { return nil }
func (s *SQLiteStorage) Close() error { return s.db.Close() }
```

- [ ] **Step 3: 创建内存存储实现（可选，用于测试）**

```go
package voting

type InMemoryStorage struct {
    candidates map[string]*Candidate
    voters     map[string]*Voter
    votes      map[string]*VoteRecord
    sessions   map[string]*VotingSession
}

func NewInMemoryStorage() *InMemoryStorage {
    return &InMemoryStorage{
        candidates: make(map[string]*Candidate),
        voters:     make(map[string]*Voter),
        votes:      make(map[string]*VoteRecord),
        sessions:   make(map[string]*VotingSession),
    }
}

// Implement all methods same as SQLiteStorage but with in-memory maps
```

- [ ] **Step 4: 编写存储测试**

```go
package voting

import (
    "testing"
)

func TestSQLiteStorage(t *testing.T) {
    // Create temp file
    f, err := os.CreateTemp("", "voting-*.db")
    if err != nil {
        t.Fatal(err)
    }
    f.Close()
    defer os.Remove(f.Name())

    storage, err := NewSQLiteStorage(f.Name())
    if err != nil {
        t.Fatal(err)
    }
    defer storage.Close()

    // Test SaveCandidate
    candidate := &Candidate{
        ID:        "test-1",
        Name:      "张三",
        Party:     "党A",
        Program:   "发展经济",
        CreatedAt: 1234567890,
    }
    if err := storage.SaveCandidate(candidate); err != nil {
        t.Fatal(err)
    }

    // Test GetCandidate
    got, err := storage.GetCandidate("test-1")
    if err != nil {
        t.Fatal(err)
    }
    if got.Name != "张三" {
        t.Errorf("Name = %v, want 张三", got.Name)
    }

    // Test ListCandidates
    list, err := storage.ListCandidates()
    if err != nil {
        t.Fatal(err)
    }
    if len(list) != 1 {
        t.Errorf("len(list) = %v, want 1", len(list))
    }

    // Test DeleteCandidate
    if err := storage.DeleteCandidate("test-1"); err != nil {
        t.Fatal(err)
    }

    got, _ = storage.GetCandidate("test-1")
    if got != nil {
        t.Error("Candidate should be deleted")
    }
}
```

- [ ] **Step 5: 运行测试验证**

Run: `go test internal/voting/ -run TestSQLiteStorage -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/voting/storage.go internal/voting/storage_test.go
git commit -m "feat: add SQLite storage layer for voting"
```

---

## Task 3: 候选人管理

**Files:**

- Modify: `internal/voting/candidate.go`

- [ ] **Step 1: 更新 Candidate 结构体**

```go
package voting

import "github.com/google/uuid"

type Candidate struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Party       string `json:"party"`
    Program     string `json:"program"`
    Description string `json:"description"`
    ImageURL    string `json:"image_url"`
    VoteCount   int    `json:"vote_count"`
    CreatedAt   int64  `json:"created_at"`
}

func NewCandidate(name, party, program string) *Candidate {
    return &Candidate{
        ID:        uuid.New().String(),
        Name:      name,
        Party:     party,
        Program:   program,
        VoteCount: 0,
        CreatedAt: now(),
    }
}

func (c *Candidate) GetID() string { return c.ID }
func (c *Candidate) GetName() string { return c.Name }
func (c *Candidate) GetParty() string { return c.Party }
func (c *Candidate) GetProgram() string { return c.Program }
func (c *Candidate) GetVoteCount() int { return c.VoteCount }
func (c *Candidate) IncrementVote() { c.VoteCount++ }
```

- [ ] **Step 2: 添加候选人管理函数**

```go
var candidateStorage Storage

func SetCandidateStorage(s Storage) {
    candidateStorage = s
}

func RegisterCandidate(name, party, program string) (*Candidate, error) {
    candidate := NewCandidate(name, party, program)
    if err := candidateStorage.SaveCandidate(candidate); err != nil {
        return nil, err
    }
    return candidate, nil
}

func GetCandidate(id string) (*Candidate, error) {
    return candidateStorage.GetCandidate(id)
}

func ListCandidates() ([]*Candidate, error) {
    return candidateStorage.ListCandidates()
}

func UpdateCandidate(c *Candidate) error {
    return candidateStorage.UpdateCandidate(c)
}

func DeleteCandidate(id string) error {
    return candidateStorage.DeleteCandidate(id)
}
```

- [ ] **Step 3: 编写测试**

```go
func TestCandidateManagement(t *testing.T) {
    storage := NewInMemoryStorage()
    SetCandidateStorage(storage)

    // Register
    c, err := RegisterCandidate("张三", "党A", "纲领A")
    if err != nil {
        t.Fatal(err)
    }
    if c.Name != "张三" {
        t.Errorf("Name = %v, want 张三", c.Name)
    }

    // Get
    got, err := GetCandidate(c.ID)
    if err != nil {
        t.Fatal(err)
    }
    if got.Party != "党A" {
        t.Errorf("Party = %v, want 党A", got.Party)
    }

    // List
    list, err := ListCandidates()
    if err != nil {
        t.Fatal(err)
    }
    if len(list) != 1 {
        t.Errorf("len(list) = %v, want 1", len(list))
    }

    // Delete
    if err := DeleteCandidate(c.ID); err != nil {
        t.Fatal(err)
    }

    got, _ = GetCandidate(c.ID)
    if got != nil {
        t.Error("Candidate should be deleted")
    }
}
```

- [ ] **Step 4: 运行测试**

Run: `go test internal/voting/ -run TestCandidateManagement -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/voting/candidate.go
git commit -m "feat: implement candidate management"
```

---

## Task 4: 投票人会话

**Files:**

- Modify: `internal/voting/voter.go`

- [ ] **Step 1: 更新 Voter 结构体和注册函数**

```go
package voting

import (
    "crypto/ed25519"
    "crypto/rand"
    "encoding/base64"
    "time"
)

type Voter struct {
    PublicKey    string `json:"public_key"`
    Name         string `json:"name"`
    HasVoted     bool   `json:"has_voted"`
    VoteHash     string `json:"vote_hash"`
    RegisteredAt int64  `json:"registered_at"`
}

var voterStorage Storage

func SetVoterStorage(s Storage) {
    voterStorage = s
}

func RegisterVoter(name string) (publicKey []byte, privateKey []byte, err error) {
    pub, priv, err := ed25519.GenerateKey(rand.Reader)
    if err != nil {
        return nil, nil, err
    }

    voter := &Voter{
        PublicKey:    base64.StdEncoding.EncodeToString(pub),
        Name:         name,
        HasVoted:     false,
        RegisteredAt: time.Now().Unix(),
    }

    if err := voterStorage.SaveVoter(voter); err != nil {
        return nil, nil, err
    }

    return pub, priv, nil
}

func GetVoter(publicKey string) (*Voter, error) {
    return voterStorage.GetVoter(publicKey)
}

func ListVoters() ([]*Voter, error) {
    return voterStorage.ListVoters()
}

func CanVote(publicKey string) (bool, error) {
    voter, err := voterStorage.GetVoter(publicKey)
    if err != nil {
        return false, err
    }
    if voter == nil {
        return false, nil
    }
    return !voter.HasVoted, nil
}

func MarkVoted(publicKey, voteHash string) error {
    voter, err := voterStorage.GetVoter(publicKey)
    if err != nil {
        return err
    }
    voter.HasVoted = true
    voter.VoteHash = voteHash
    return voterStorage.UpdateVoter(voter)
}
```

- [ ] **Step 2: 编写测试**

```go
func TestVoterRegistration(t *testing.T) {
    storage := NewInMemoryStorage()
    SetVoterStorage(storage)

    pub, priv, err := RegisterVoter("投票人A")
    if err != nil {
        t.Fatal(err)
    }

    if len(pub) == 0 || len(priv) == 0 {
        t.Error("Keys should not be empty")
    }

    // Verify voter exists
    pkStr := base64.StdEncoding.EncodeToString(pub)
    voter, err := GetVoter(pkStr)
    if err != nil {
        t.Fatal(err)
    }
    if voter.Name != "投票人A" {
        t.Errorf("Name = %v, want 投票人A", voter.Name)
    }
    if voter.HasVoted {
        t.Error("Should not have voted yet")
    }

    // Test CanVote
    canVote, err := CanVote(pkStr)
    if err != nil {
        t.Fatal(err)
    }
    if !canVote {
        t.Error("Should be able to vote")
    }

    // Mark as voted
    if err := MarkVoted(pkStr, "some-hash"); err != nil {
        t.Fatal(err)
    }

    canVote, _ = CanVote(pkStr)
    if canVote {
        t.Error("Should not be able to vote again")
    }
}
```

- [ ] **Step 3: 运行测试**

Run: `go test internal/voting/ -run TestVoterRegistration -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/voting/voter.go
git commit -m "feat: implement voter registration with Ed25519"
```

---

## Task 5: 投票逻辑

**Files:**

- Modify: `internal/voting/vote.go`

- [ ] **Step 1: 更新投票函数**

```go
package voting

import (
    "crypto/ed25519"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/pplmx/aurora/internal/blockchain"
)

type VoteRecord struct {
    ID          string `json:"id"`
    VoterPK     string `json:"voter_pk"`
    CandidateID string `json:"candidate_id"`
    Signature   string `json:"signature"`
    Message     string `json:"message"`
    Timestamp   int64  `json:"timestamp"`
    BlockHeight int64  `json:"block_height"`
}

func CastVote(voterPK []byte, candidateID string, privateKey []byte, chain *blockchain.BlockChain) (*VoteRecord, error) {
    pkStr := base64.StdEncoding.EncodeToString(voterPK)

    // Verify voter exists and hasn't voted
    voter, err := voterStorage.GetVoter(pkStr)
    if err != nil {
        return nil, err
    }
    if voter == nil {
        return nil, fmt.Errorf("voter not registered")
    }
    if voter.HasVoted {
        return nil, fmt.Errorf("already voted")
    }

    // Verify candidate exists
    candidate, err := candidateStorage.GetCandidate(candidateID)
    if err != nil {
        return nil, err
    }
    if candidate == nil {
        return nil, fmt.Errorf("candidate not found")
    }

    // Create vote message
    timestamp := time.Now().Unix()
    message := fmt.Sprintf("%s|%s|%d", pkStr, candidateID, timestamp)

    // Sign
    signature := ed25519.Sign(privateKey, []byte(message))

    // Create record
    record := &VoteRecord{
        ID:          uuid.New().String(),
        VoterPK:     pkStr,
        CandidateID: candidateID,
        Signature:   base64.StdEncoding.EncodeToString(signature),
        Message:     message,
        Timestamp:   timestamp,
        BlockHeight: 0,
    }

    // Save to blockchain
    jsonData, _ := record.ToJSON()
    height := chain.AddBlock(jsonData)
    record.BlockHeight = height

    // Save to storage
    if err := voteStorage.SaveVote(record); err != nil {
        return nil, err
    }

    // Update voter as voted
    voteHash := sha256.Sum256([]byte(message))
    if err := MarkVoted(pkStr, fmt.Sprintf("%x", voteHash)); err != nil {
        return nil, err
    }

    // Update candidate vote count
    candidate.VoteCount++
    if err := candidateStorage.UpdateCandidate(candidate); err != nil {
        return nil, err
    }

    return record, nil
}

func VerifyVote(record *VoteRecord) bool {
    pubBytes, err := base64.StdEncoding.DecodeString(record.VoterPK)
    if err != nil {
        return false
    }

    sigBytes, err := base64.StdEncoding.DecodeString(record.Signature)
    if err != nil {
        return false
    }

    return ed25519.Verify(pubBytes, []byte(record.Message), sigBytes)
}

func GetVote(id string) (*VoteRecord, error) {
    return voteStorage.GetVote(id)
}

func GetVotesByCandidate(candidateID string) ([]*VoteRecord, error) {
    return voteStorage.GetVotesByCandidate(candidateID)
}

func CountVotes(candidateID string) (int, error) {
    votes, err := voteStorage.GetVotesByCandidate(candidateID)
    if err != nil {
        return 0, err
    }
    return len(votes), nil
}

func (r *VoteRecord) ToJSON() (string, error) {
    data, err := json.Marshal(r)
    if err != nil {
        return "", err
    }
    return string(data), nil
}
```

- [ ] **Step 2: 添加 voteStorage 全局变量**

```go
var voteStorage Storage

func SetVoteStorage(s Storage) {
    voteStorage = s
}
```

- [ ] **Step 3: 编写测试**

```go
func TestCastVote(t *testing.T) {
    storage := NewInMemoryStorage()
    SetCandidateStorage(storage)
    SetVoterStorage(storage)
    SetVoteStorage(storage)

    chain := blockchain.InitBlockChain()

    // Register candidate
    cand, _ := RegisterCandidate("张三", "党A", "纲领")

    // Register voter
    pub, priv, _ := RegisterVoter("投票人A")

    // Cast vote
    record, err := CastVote(pub, cand.ID, priv, chain)
    if err != nil {
        t.Fatal(err)
    }

    if record.ID == "" {
        t.Error("Vote ID should not be empty")
    }

    // Verify vote
    if !VerifyVote(record) {
        t.Error("Vote verification should pass")
    }

    // Verify can't vote again
    _, err = CastVote(pub, cand.ID, priv, chain)
    if err == nil {
        t.Error("Should not be able to vote twice")
    }

    // Verify vote count
    count, _ := CountVotes(cand.ID)
    if count != 1 {
        t.Errorf("Vote count = %v, want 1", count)
    }
}
```

- [ ] **Step 4: 运行测试**

Run: `go test internal/voting/ -run TestCastVote -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/voting/vote.go
git commit -m "feat: implement vote casting with Ed25519 signature"
```

---

## Task 6: 投票会话

**Files:**

- Create: `internal/voting/session.go`

- [ ] **Step 1: 创建会话管理**

```go
package voting

import (
    "time"

    "github.com/google/uuid"
)

func CreateSession(title, description string, candidateIDs []string, startTime, endTime int64) (*VotingSession, error) {
    session := &VotingSession{
        ID:          uuid.New().String(),
        Title:       title,
        Description: description,
        StartTime:   startTime,
        EndTime:     endTime,
        Status:      "draft",
        Candidates:  candidateIDs,
        CreatedAt:   time.Now().Unix(),
    }

    if err := sessionStorage.SaveSession(session); err != nil {
        return nil, err
    }

    return session, nil
}

func GetSession(id string) (*VotingSession, error) {
    return sessionStorage.GetSession(id)
}

func ListSessions() ([]*VotingSession, error) {
    return sessionStorage.ListSessions()
}

func StartSession(id string) error {
    session, err := sessionStorage.GetSession(id)
    if err != nil {
        return err
    }
    if session == nil {
        return nil // or error
    }

    session.Status = "active"
    session.StartTime = time.Now().Unix()

    return sessionStorage.UpdateSession(session)
}

func EndSession(id string) error {
    session, err := sessionStorage.GetSession(id)
    if err != nil {
        return err
    }
    if session == nil {
        return nil
    }

    session.Status = "ended"
    session.EndTime = time.Now().Unix()

    return sessionStorage.UpdateSession(session)
}

func GetSessionResults(sessionID string) (map[string]int, error) {
    session, err := sessionStorage.GetSession(sessionID)
    if err != nil {
        return nil, err
    }
    if session == nil {
        return nil, nil
    }

    results := make(map[string]int)
    for _, cid := range session.Candidates {
        count, err := CountVotes(cid)
        if err != nil {
            return nil, err
        }
        results[cid] = count
    }

    return results, nil
}

var sessionStorage Storage

func SetSessionStorage(s Storage) {
    sessionStorage = s
}
```

- [ ] **Step 2: 更新主初始化函数**

```go
func InitVoting(storage Storage) {
    SetCandidateStorage(storage)
    SetVoterStorage(storage)
    SetVoteStorage(storage)
    SetSessionStorage(storage)
}
```

- [ ] **Step 3: 编写测试**

```go
func TestVotingSession(t *testing.T) {
    storage := NewInMemoryStorage()
    SetCandidateStorage(storage)
    SetVoterStorage(storage)
    SetVoteStorage(storage)
    SetSessionStorage(storage)

    // Create candidates
    c1, _ := RegisterCandidate("张三", "党A", "纲领A")
    c2, _ := RegisterCandidate("李四", "党B", "纲领B")

    // Create session
    session, err := CreateSession("2024选举", "主席选举", []string{c1.ID, c2.ID}, 0, 0)
    if err != nil {
        t.Fatal(err)
    }

    if session.Status != "draft" {
        t.Errorf("Status = %v, want draft", session.Status)
    }

    // Start session
    if err := StartSession(session.ID); err != nil {
        t.Fatal(err)
    }

    s, _ := GetSession(session.ID)
    if s.Status != "active" {
        t.Errorf("Status = %v, want active", s.Status)
    }

    // End session
    if err := EndSession(session.ID); err != nil {
        t.Fatal(err)
    }

    s, _ = GetSession(session.ID)
    if s.Status != "ended" {
        t.Errorf("Status = %v, want ended", s.Status)
    }
}
```

- [ ] **Step 4: 运行测试**

Run: `go test internal/voting/ -run TestVotingSession -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/voting/session.go
git commit -m "feat: implement voting session management"
```

---

## Task 7: CLI 命令

**Files:**

- Create: `cmd/aurora/cmd/voting.go`

- [ ] **Step 1: 创建 CLI 命令**

```go
package cmd

import (
    "fmt"

    "github.com/pplmx/aurora/internal/blockchain"
    "github.com/pplmx/aurora/internal/voting"
    "github.com/spf13/cobra"
)

var votingCmd = &cobra.Command{
    Use:   "voting",
    Short: "Voting system",
    Long:  "Ed25519 signature based transparent voting system",
}

var candidateCmd = &cobra.Command{
    Use:   "candidate",
    Short: "Candidate management",
}

var candidateAddCmd = &cobra.Command{
    Use:   "add",
    Short: "Add a candidate",
    Run: func(cmd *cobra.Command, args []string) {
        name, _ := cmd.Flags().GetString("name")
        party, _ := cmd.Flags().GetString("party")
        program, _ := cmd.Flags().GetString("program")

        cand, err := voting.RegisterCandidate(name, party, program)
        if err != nil {
            fmt.Println("Error:", err)
            return
        }

        fmt.Printf("Candidate registered: %s (%s)\n", cand.Name, cand.ID)
    },
}

var candidateListCmd = &cobra.Command{
    Use:   "list",
    Short: "List candidates",
    Run: func(cmd *cobra.Command, args []string) {
        list, err := voting.ListCandidates()
        if err != nil {
            fmt.Println("Error:", err)
            return
        }

        fmt.Println("Candidates:")
        for _, c := range list {
            fmt.Printf("  - %s [%s] - %d votes\n", c.Name, c.Party, c.VoteCount)
        }
    },
}

var voterCmd = &cobra.Command{
    Use:   "voter",
    Short: "Voter management",
}

var voterRegisterCmd = &cobra.Command{
    Use:   "register",
    Short: "Register a new voter",
    Run: func(cmd *cobra.Command, args []string) {
        name, _ := cmd.Flags().GetString("name")

        pub, priv, err := voting.RegisterVoter(name)
        if err != nil {
            fmt.Println("Error:", err)
            return
        }

        fmt.Println("Voter registered successfully!")
        fmt.Printf("Public Key:  %s\n", base64.StdEncoding.EncodeToString(pub))
        fmt.Println("PRIVATE KEY: (save this!)")
        fmt.Println(base64.StdEncoding.EncodeToString(priv))
    },
}

var voteCmd = &cobra.Command{
    Use:   "vote",
    Short: "Cast a vote",
    Run: func(cmd *cobra.Command, args []string) {
        voterPK, _ := cmd.Flags().GetString("voter")
        candidateID, _ := cmd.Flags().GetString("candidate")
        privKey, _ := cmd.Flags().GetString("private-key")

        pubBytes, err := base64.StdEncoding.DecodeString(voterPK)
        if err != nil {
            fmt.Println("Invalid voter public key")
            return
        }

        privBytes, err := base64.StdEncoding.DecodeString(privKey)
        if err != nil {
            fmt.Println("Invalid private key")
            return
        }

        chain := blockchain.InitBlockChain()
        record, err := voting.CastVote(pubBytes, candidateID, privBytes, chain)
        if err != nil {
            fmt.Println("Error:", err)
            return
        }

        fmt.Println("Vote cast successfully!")
        fmt.Printf("Vote ID:     %s\n", record.ID)
        fmt.Printf("Block Height: %d\n", record.BlockHeight)
    },
}

var sessionCmd = &cobra.Command{
    Use:   "session",
    Short: "Voting session management",
}

var sessionCreateCmd = &cobra.Command{
    Use:   "create",
    Short: "Create a voting session",
    Run: func(cmd *cobra.Command, args []string) {
        title, _ := cmd.Flags().GetString("title")
        description, _ := cmd.Flags().GetString("description")
        candidates, _ := cmd.Flags().GetStringSlice("candidates")

        session, err := voting.CreateSession(title, description, candidates, 0, 0)
        if err != nil {
            fmt.Println("Error:", err)
            return
        }

        fmt.Printf("Session created: %s\n", session.ID)
    },
}

var sessionListCmd = &cobra.Command{
    Use:   "list",
    Short: "List voting sessions",
    Run: func(cmd *cobra.Command, args []string) {
        list, err := voting.ListSessions()
        if err != nil {
            fmt.Println("Error:", err)
            return
        }

        fmt.Println("Voting Sessions:")
        for _, s := range list {
            fmt.Printf("  - %s [%s] - %s\n", s.Title, s.Status, s.ID)
        }
    },
}

var resultsCmd = &cobra.Command{
    Use:   "results",
    Short: "Get voting results",
    Run: func(cmd *cobra.Command, args []string) {
        sessionID, _ := cmd.Flags().GetString("session")

        results, err := voting.GetSessionResults(sessionID)
        if err != nil {
            fmt.Println("Error:", err)
            return
        }

        fmt.Println("Results:")
        for cid, count := range results {
            cand, _ := voting.GetCandidate(cid)
            name := cid
            if cand != nil {
                name = cand.Name
            }
            fmt.Printf("  %s: %d votes\n", name, count)
        }
    },
}

func init() {
    rootCmd.AddCommand(votingCmd)

    // Candidate commands
    votingCmd.AddCommand(candidateCmd)
    candidateCmd.AddCommand(candidateAddCmd)
    candidateCmd.AddCommand(candidateListCmd)

    candidateAddCmd.Flags().StringP("name", "n", "", "Candidate name")
    candidateAddCmd.Flags().StringP("party", "p", "", "Candidate party")
    candidateAddCmd.Flags().StringP("program", "m", "", "Candidate program")
    candidateAddCmd.MarkFlagRequired("name")

    // Voter commands
    votingCmd.AddCommand(voterCmd)
    voterCmd.AddCommand(voterRegisterCmd)

    voterRegisterCmd.Flags().StringP("name", "n", "", "Voter name")
    voterRegisterCmd.MarkFlagRequired("name")

    // Vote command
    votingCmd.AddCommand(voteCmd)
    voteCmd.Flags().StringP("voter", "v", "", "Voter public key (Base64)")
    voteCmd.Flags().StringP("candidate", "c", "", "Candidate ID")
    voteCmd.Flags().StringP("private-key", "k", "", "Voter private key (Base64)")
    voteCmd.MarkFlagRequired("voter")
    voteCmd.MarkFlagRequired("candidate")
    voteCmd.MarkFlagRequired("private-key")

    // Session commands
    votingCmd.AddCommand(sessionCmd)
    sessionCmd.AddCommand(sessionCreateCmd)
    sessionCmd.AddCommand(sessionListCmd)

    sessionCreateCmd.Flags().StringP("title", "t", "", "Session title")
    sessionCreateCmd.Flags().StringP("description", "d", "", "Session description")
    sessionCreateCmd.Flags().StringSliceP("candidates", "c", []string{}, "Candidate IDs")
    sessionCreateCmd.MarkFlagRequired("title")
    sessionCreateCmd.MarkFlagRequired("candidates")

    // Results command
    votingCmd.AddCommand(resultsCmd)
    resultsCmd.Flags().StringP("session", "s", "", "Session ID")
    resultsCmd.MarkFlagRequired("session")
}
```

- [ ] **Step 2: 添加 base64 导入**

需要添加 `"encoding/base64"` 到 imports

- [ ] **Step 3: 初始化存储**

在 `cmd/aurora/main.go` 或 `root.go` 中添加：

```go
import "github.com/pplmx/aurora/internal/voting"

// In init() or main():
storage := voting.NewInMemoryStorage() // 或 SQLite
voting.InitVoting(storage)
```

- [ ] **Step 4: 验证编译**

Run: `go build ./...`
Expected: 无错误

- [ ] **Step 5: 测试 CLI**

Run: `./aurora voting --help`

- [ ] **Step 6: Commit**

```bash
git add cmd/aurora/cmd/voting.go
git commit -m "feat: add voting CLI commands"
```

---

## Task 8: TUI 界面

**Files:**

- Modify: `internal/voting/tui.go` (创建新文件)

- [ ] **Step 1: 创建 TUI 主界面**

```go
package voting

import (
    "fmt"

    "github.com/charmbracelet/bubbles/textinput"
    "github.com/charmbracelet/lipgloss"

    tea "github.com/charmbracelet/bubbletea"
)

var (
    headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true).Padding(0, 1)
    // ... 其他样式
)

type votingModel struct {
    view       string
    storage    Storage
    menuIndex  int
    // ... 其他字段
}

func NewVotingApp(storage Storage) *votingModel {
    return &votingModel{
        view:    "menu",
        storage: storage,
    }
}

func (m *votingModel) Init() tea.Cmd {
    return nil
}

func (m *votingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // 实现消息处理
    return m, nil
}

func (m *votingModel) View() string {
    switch m.view {
    case "menu":
        return m.menuView()
    case "create":
        return m.createView()
    case "vote":
        return m.voteView()
    case "results":
        return m.resultsView()
    }
    return ""
}

func (m *votingModel) menuView() string {
    s := headerStyle.Render("🗳️ VRF 透明投票系统 🗳️") + "\n\n"
    items := []string{"创建投票会话", "候选人管理", "投票", "查看结果", "退出"}
    for i, item := range items {
        if i == m.menuIndex {
            s += "▶ " + item + "\n"
        } else {
            s += "  " + item + "\n"
        }
    }
    return s
}

// ... 其他 view 方法

func RunVotingTUI(storage Storage) error {
    p := tea.NewProgram(NewVotingApp(storage), tea.WithAltScreen())
    if _, err := p.Run(); err != nil {
        return err
    }
    return nil
}
```

- [ ] **Step 2: 添加 tui 命令到 CLI**

在 `voting.go` 中添加：

```go
var tuiCmd = &cobra.Command{
    Use:   "tui",
    Short: "Launch TUI interface",
    Run: func(cmd *cobra.Command, args []string) {
        storage := voting.NewInMemoryStorage()
        voting.InitVoting(storage)
        if err := voting.RunVotingTUI(storage); err != nil {
            fmt.Println("Error:", err)
        }
    },
}

func init() {
    votingCmd.AddCommand(tuiCmd)
}
```

- [ ] **Step 3: 验证编译**

Run: `go build ./...`

- [ ] **Step 4: Commit**

```bash
git add internal/voting/tui.go cmd/aurora/cmd/voting.go
git commit -m "feat: add voting TUI interface"
```

---

## Task 9: 集成测试

**Files:**

- Create: `test/voting_e2e_test.go`

- [ ] **Step 1: 编写端到端测试**

```go
package test

import (
    "testing"

    "github.com/pplmx/aurora/internal/blockchain"
    "github.com/pplmx/aurora/internal/voting"
)

func TestVotingE2E(t *testing.T) {
    storage := voting.NewInMemoryStorage()
    voting.InitVoting(storage)
    chain := blockchain.InitBlockChain()

    // 1. Register candidates
    c1, err := voting.RegisterCandidate("张三", "党A", "纲领A")
    if err != nil {
        t.Fatal(err)
    }
    c2, err := voting.RegisterCandidate("李四", "党B", "纲领B")
    if err != nil {
        t.Fatal(err)
    }

    // 2. Register voters
    v1Pub, v1Priv, err := voting.RegisterVoter("投票人1")
    if err != nil {
        t.Fatal(err)
    }
    v2Pub, v2Priv, err := voting.RegisterVoter("投票人2")
    if err != nil {
        t.Fatal(err)
    }

    // 3. Create session
    session, err := voting.CreateSession("测试选举", "描述", []string{c1.ID, c2.ID}, 0, 0)
    if err != nil {
        t.Fatal(err)
    }

    // 4. Start session
    if err := voting.StartSession(session.ID); err != nil {
        t.Fatal(err)
    }

    // 5. Cast votes
    _, err = voting.CastVote(v1Pub, c1.ID, v1Priv, chain)
    if err != nil {
        t.Fatal(err)
    }

    _, err = voting.CastVote(v2Pub, c2.ID, v2Priv, chain)
    if err != nil {
        t.Fatal(err)
    }

    // 6. End session
    if err := voting.EndSession(session.ID); err != nil {
        t.Fatal(err)
    }

    // 7. Check results
    results, err := voting.GetSessionResults(session.ID)
    if err != nil {
        t.Fatal(err)
    }

    if results[c1.ID] != 1 {
        t.Errorf("c1 votes = %v, want 1", results[c1.ID])
    }
    if results[c2.ID] != 1 {
        t.Errorf("c2 votes = %v, want 1", results[c2.ID])
    }

    t.Log("E2E test passed!")
}
```

- [ ] **Step 2: 运行测试**

Run: `go test ./test/ -run TestVotingE2E -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add test/voting_e2e_test.go
git commit -m "test: add voting E2E test"
```

---

## Task 10: 最终验证

- [ ] **Step 1: 运行所有测试**

Run: `go test ./...`
Expected: 全部 PASS

- [ ] **Step 2: 运行 go vet**

Run: `go vet ./...`
Expected: 无警告

- [ ] **Step 3: 验证功能**

```bash
# 注册候选人
./aurora voting candidate add -n "张三" -p "党A" -m "纲领A"
./aurora voting candidate add -n "李四" -p "党B" -m "纲领B"

# 列出候选人
./aurora voting candidate list

# 注册投票人
./aurora voting voter register -n "投票人A"
./aurora voting voter register -n "投票人B"

# 投票
./aurora voting vote -v "<pub-key>" -c "<candidate-id>" -k "<priv-key>"

# 查看结果
./aurora voting results -s "<session-id>"
```

- [ ] **Step 4: Commit**

```bash
git status
git add -A
git commit -m "feat: complete voting system"
```

---

## 总结

完成所有任务后，你将拥有：

- ✅ SQLite 存储层（可替换为内存）
- ✅ 候选人管理（注册、列表、删除）
- ✅ 投票人注册（Ed25519 密钥对）
- ✅ 投票功能（签名验证、区块链存证）
- ✅ 投票会话管理（创建、开始、结束）
- ✅ 计票与结果查询
- ✅ CLI 命令行工具
- ✅ TUI 交互界面（基础）
- ✅ 单元测试与集成测试
