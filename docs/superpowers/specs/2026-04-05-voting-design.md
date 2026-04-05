# VRF 透明投票系统设计文档

## 概述

基于 Ed25519 签名和区块链存储的透明投票系统，支持候选人注册、投票人注册、匿名投票、实时计票和结果上链存证。

## 核心特性

- Ed25519 签名验证（投票人身份）
- 候选人管理（注册、竞选纲领）
- 区块链存证（投票记录上链）
- 实时计票与公示
- 结果可验证
- CLI + TUI 交互界面
- 内存 + SQLite 混合存储

## 技术选型

| 组件 | 技术           | 版本                        |
| ---- | -------------- | --------------------------- |
| 语言 | Go             | 1.26+                       |
| 签名 | crypto/ed25519 | 标准库                      |
| 存储 | 内存 + SQLite  | github.com/mattn/go-sqlite3 |
| TUI  | Bubble Tea     | latest                      |
| CLI  | Cobra          | latest                      |

## 架构

```text
┌─────────────────────────────────────────────────────────────┐
│                       CLI/TUI 层                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ 候选人管理  │  │ 投票人注册  │  │  投票/计票          │  │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘  │
└─────────┼────────────────┼────────────────────┼─────────────┘
          │                │                    │
┌─────────┼────────────────┼────────────────────┼─────────────┐
│         ▼                ▼                    ▼              │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                    业务逻辑层                            ││
│  │  ┌──────────┐  ┌──────────┐  ┌────────────────────┐    ││
│  │  │候选人管理│  │投票人管理│  │  投票/计票         │    ││
│  │  └──────────┘  └──────────┘  └────────────────────┘    ││
│  └─────────────────────────────────────────────────────────┘│
│                          │                                   │
│  ┌───────────────────────▼────────────────────────────────┐ │
│  │                  存储层                                  │ │
│  │        内存缓存  ←→  SQLite  ←→  区块链                │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## 数据结构

### 候选人 (Candidate)

```go
type Candidate struct {
    ID          string    `json:"id"`           // 唯一ID (UUID)
    Name        string    `json:"name"`         // 候选人名称
    Party       string    `json:"party"`        // 党派/团体
    Program     string    `json:"program"`      // 竞选纲领
    Description string    `json:"description"`  // 详细介绍
    ImageURL    string    `json:"image_url"`    // 头像URL
    VoteCount   int       `json:"vote_count"`   // 得票数（运行时）
    CreatedAt   int64     `json:"created_at"`   // 创建时间
}
```

### 投票人 (Voter)

```go
type Voter struct {
    PublicKey   string    `json:"public_key"`   // Ed25519 公钥 (Base64)
    Name        string    `json:"name"`         // 投票人名称（可选）
    HasVoted    bool      `json:"has_voted"`    // 是否已投票
    VoteHash    string    `json:"vote_hash"`    // 投票内容哈希
    RegisteredAt int64    `json:"registered_at"` // 注册时间
}
```

### 投票记录 (VoteRecord)

```go
type VoteRecord struct {
    ID           string    `json:"id"`            // 投票ID (UUID)
    VoterPK      string    `json:"voter_pk"`      // 投票人公钥 (Base64)
    CandidateID  string    `json:"candidate_id"`  // 候选人ID
    Signature    string    `json:"signature"`     // Ed25519 签名 (Base64)
    Message      string    `json:"message"`       // 签名原文
    Timestamp    int64     `json:"timestamp"`     // 投票时间
    BlockHeight  int64     `json:"block_height"`  // 区块高度
}
```

### 投票会话 (VotingSession)

```go
type VotingSession struct {
    ID          string       `json:"id"`            // 会话ID
    Title       string       `json:"title"`         // 投票标题
    Description string       `json:"description"`   // 投票描述
    StartTime   int64        `json:"start_time"`    // 开始时间
    EndTime     int64        `json:"end_time"`      // 结束时间
    Status      string       `json:"status"`        // draft/active/ended
    Candidates  []string     `json:"candidates"`    // 候选人ID列表
    CreatedAt   int64        `json:"created_at"`
}
```

## 核心流程

### 1. 候选人注册

```go
func RegisterCandidate(name, party, program string) (*Candidate, error) {
    id := generateUUID()
    candidate := &Candidate{
        ID:          id,
        Name:        name,
        Party:       party,
        Program:     program,
        VoteCount:   0,
        CreatedAt:   time.Now().Unix(),
    }
    saveToStorage(candidate)
    return candidate, nil
}
```

### 2. 投票人注册

```go
func RegisterVoter() (publicKey, privateKey []byte, err error) {
    pub, priv, err := ed25519.GenerateKey(rand.Reader)
    if err != nil {
        return nil, nil, err
    }

    voter := &Voter{
        PublicKey:   base64.StdEncoding.EncodeToString(pub),
        HasVoted:    false,
        RegisteredAt: time.Now().Unix(),
    }
    saveToStorage(voter)
    return pub, priv, nil
}
```

### 3. 投票

```go
func CastVote(voterPK []byte, candidateID string, privateKey []byte) (*VoteRecord, error) {
    // 验证投票人
    voter := getVoter(base64.StdEncoding.EncodeToString(voterPK))
    if voter.HasVoted {
        return nil, ErrAlreadyVoted
    }

    // 验证候选人
    candidate := getCandidate(candidateID)
    if candidate == nil {
        return nil, ErrCandidateNotFound
    }

    // 创建投票消息
    message := fmt.Sprintf("%s|%s|%d", voterPK, candidateID, time.Now().Unix())

    // 签名
    signature := ed25519.Sign(privateKey, []byte(message))

    // 创建记录
    record := &VoteRecord{
        ID:          generateUUID(),
        VoterPK:     base64.StdEncoding.EncodeToString(voterPK),
        CandidateID: candidateID,
        Signature:   base64.StdEncoding.EncodeToString(signature),
        Message:     message,
        Timestamp:   time.Now().Unix(),
        BlockHeight: 0, // 上链后更新
    }

    // 上链
    height := chain.AddBlock(record.ToJSON())
    record.BlockHeight = height

    // 更新投票人状态
    voter.HasVoted = true
    voter.VoteHash = sha256.Sum256([]byte(message))
    updateVoter(voter)

    // 更新候选人票数
    candidate.VoteCount++
    updateCandidate(candidate)

    return record, nil
}
```

### 4. 验证投票

```go
func VerifyVote(record *VoteRecord) bool {
    // 解码公钥和签名
    pubBytes, _ := base64.StdEncoding.DecodeString(record.VoterPK)
    sigBytes, _ := base64.StdEncoding.DecodeString(record.Signature)

    // 验证签名
    valid := ed25519.Verify(pubBytes, []byte(record.Message), sigBytes)
    if !valid {
        return false
    }

    // 验证链上存在
    block := chain.GetBlock(record.BlockHeight)
    return block != nil
}
```

### 5. 计票

```go
func CountVotes(candidateID string) int {
    records := getVoteRecordsByCandidate(candidateID)
    return len(records)
}

func GetResults(sessionID string) map[string]int {
    session := getSession(sessionID)
    results := make(map[string]int)
    for _, cid := range session.Candidates {
        results[cid] = CountVotes(cid)
    }
    return results
}
```

## 存储设计

### SQLite 表结构

```sql
-- 候选人表
CREATE TABLE candidates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    party TEXT,
    program TEXT,
    description TEXT,
    image_url TEXT,
    vote_count INTEGER DEFAULT 0,
    created_at INTEGER
);

-- 投票人表
CREATE TABLE voters (
    public_key TEXT PRIMARY KEY,
    name TEXT,
    has_voted INTEGER DEFAULT 0,
    vote_hash TEXT,
    registered_at INTEGER
);

-- 投票记录表
CREATE TABLE votes (
    id TEXT PRIMARY KEY,
    voter_pk TEXT NOT NULL,
    candidate_id TEXT NOT NULL,
    signature TEXT NOT NULL,
    message TEXT NOT NULL,
    timestamp INTEGER,
    block_height INTEGER,
    FOREIGN KEY (candidate_id) REFERENCES candidates(id)
);

-- 投票会话表
CREATE TABLE voting_sessions (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    start_time INTEGER,
    end_time INTEGER,
    status TEXT DEFAULT 'draft',
    created_at INTEGER
);

-- 会话-候选人关联表
CREATE TABLE session_candidates (
    session_id TEXT,
    candidate_id TEXT,
    PRIMARY KEY (session_id, candidate_id)
);
```

### 存储接口

```go
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

    // Vote
    SaveVote(v *VoteRecord) error
    GetVote(id string) (*VoteRecord, error)
    GetVotesByCandidate(candidateID string) ([]*VoteRecord, error)
    GetVotesByVoter(voterPK string) ([]*VoteRecord, error)

    // Session
    SaveSession(s *VotingSession) error
    GetSession(id string) (*VotingSession, error)
    ListSessions() ([]*VotingSession, error)
}
```

## CLI 命令设计

```bash
# 候选人管理
./aurora voting candidate add --name "张三" --party "党A" --program "纲领内容"
./aurora voting candidate list
./aurora voting candidate show <id>

# 投票人会话
./aurora voting voter register --name "投票人A"
./aurora voting voter list

# 投票
./aurora voting vote --voter <public-key> --candidate <candidate-id> --private-key <key>

# 投票会话
./aurora voting session create --title "2024选举" --candidates "id1,id2,id3"
./aurora voting session start <session-id>
./aurora voting session end <session-id>
./aurora voting session results <session-id>

# 验证
./aurora voting verify --vote-id <id>
```

## TUI 界面设计

### 主菜单

```text
┌────────────────────────────────────────────┐
│          🌟 VRF 透明投票系统 🌟             │
│                                            │
│  [1] 创建投票会话                           │
│  [2] 候选人管理                             │
│  [3] 投票                                   │
│  [4] 查看结果                               │
│  [5] 验证投票                               │
│  [6] 退出                                   │
│                                            │
│  输入选项: _                                │
└────────────────────────────────────────────┘
```

### 创建投票会话

```text
┌────────────────────────────────────────────┐
│           创建投票会话                      │
│                                            │
│  投票标题: ___________________              │
│  描述:     ___________________              │
│                                            │
│  选择候选人:                                │
│  ☐ [1] 张三 (党A)                          │
│  ☑ [2] 李四 (党B)                          │
│  ☑ [3] 王五 (无党派)                       │
│                                            │
│  开始时间: ___________________              │
│  结束时间: ___________________              │
│                                            │
│  [创建]  [取消]                             │
└────────────────────────────────────────────┘
```

### 投票界面

```text
┌────────────────────────────────────────────┐
│           投票                             │
│                                            │
│  投票会话: 2024主席选举                     │
│  状态: 进行中                               │
│                                            │
│  请选择候选人:                             │
│                                            │
│  [1] 张三                                  │
│      党A | 纲领：发展经济，改善民生         │
│                                            │
│  [2] 李四                                  │
│      党B | 纲领：环境保护，科技创新         │
│                                            │
│  [3] 王五                                  │
│      无党派 | 纲领：公平公正，透明治理      │
│                                            │
│  [选择投票]                                │
│                                            │
│  您的公钥: 0x7a3f...                       │
└────────────────────────────────────────────┘
```

### 结果展示

```text
┌────────────────────────────────────────────┐
│           投票结果                         │
│                                            │
│  投票会话: 2024主席选举                     │
│  总投票数: 1,234                           │
│                                            │
│  ┌────────────────────────────────────┐    │
│  │ ████████████░░░░░░░░░░ 45%        │    │
│  │ 张三 (党A) - 555 票                │    │
│  ├────────────────────────────────────┤    │
│  │ █████████░░░░░░░░░░░░░ 40%        │    │
│  │ 李四 (党B) - 493 票                │    │
│  ├────────────────────────────────────┤    │
│  │ ███░░░░░░░░░░░░░░░░░░ 15%         │    │
│  │ 王五 (无党派) - 186 票             │    │
│  └────────────────────────────────────┘    │
│                                            │
│  🏆 获胜者: 张三                           │
│                                            │
│  [验证结果]  [返回]                         │
└────────────────────────────────────────────┘
```

## 安全性考虑

1. **一人一票**：通过 Ed25519 公钥唯一标识，hasVoted 标记防止重复
2. **签名验证**：每票必验，防止篡改
3. **区块链存证**：投票记录上链，不可篡改
4. **隐私保护**：只存储公钥和签名，不存储投票人身份信息
5. **时间戳**：防止重放攻击

## 测试计划

### 单元测试

- `TestRegisterCandidate` - 候选人注册
- `TestRegisterVoter` - 投票人注册
- `TestCastVote` - 投票功能
- `TestVerifyVote` - 投票验证
- `TestDuplicateVote` - 防止重复投票
- `TestCountVotes` - 计票功能
- `TestStorage` - SQLite 存储

### 集成测试

- 完整投票流程（注册 → 投票 → 验证 → 计票）
- 多投票人投票
- 投票会话状态流转
- 结果统计正确性

## 文件结构

```text
internal/
├── voting/
│   ├── candidate.go       # 候选人管理
│   ├── voter.go           # 投票人管理
│   ├── vote.go            # 投票逻辑
│   ├── session.go         # 投票会话
│   ├── storage.go         # SQLite 存储
│   ├── tui.go             # TUI 界面
│   └── voting_test.go     # 测试

cmd/aurora/cmd/
└── voting.go              # CLI 命令
```

## 待定事项 (TBD)

- [ ] 多选投票支持
- [ ] 权重投票支持
- [ ] 匿名投票（零知识证明）
- [ ] 委托投票
- [ ] 投票结果 Merkle 证明

## 风险与注意事项

1. **私钥安全**：投票人需妥善保管私钥
2. **上链成本**：每次投票上链，需考虑 gas/存储成本
3. **链上数据膨胀**：大量投票记录需考虑存储清理
4. **时间同步**：投票有时间限制，需保证节点时间准确
