# 完整测试覆盖实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 Token、NFT、Oracle 模块添加完整测试覆盖，包括单元测试、集成测试和 E2E 测试。

**Architecture:** 使用内存仓库进行测试，避免外部依赖。参考现有 e2e/nft_e2e_test.go 的 in-memory repository 模式。

**Tech Stack:** Go, testing, assert, mock

---

## 文件结构

```text
e2e/
├── token_e2e_test.go      # 新建: Token E2E 测试
├── nft_e2e_test.go        # 扩展: 更多场景
└── oracle_e2e_test.go     # 扩展: 更多场景

internal/domain/token/
└── service_test.go        # 扩展: 更多测试

internal/domain/nft/
└── service_test.go        # 扩展: Transfer/Query/List

internal/domain/oracle/
└── service_test.go        # 新建: Oracle Service 测试

internal/app/token/
├── create_test.go         # 新建
├── mint_test.go           # 新建
├── transfer_test.go       # 新建
├── burn_test.go           # 新建
└── approve_test.go        # 新建
```

---

## 任务分解

### Task 1: Token E2E 测试

**Files:**

- Create: `e2e/token_e2e_test.go`

- [ ] **Step 1: 创建 Token E2E 测试文件**

```go
package test

import (
    "crypto/ed25519"
    "testing"

    "github.com/pplmx/aurora/internal/domain/blockchain"
    "github.com/pplmx/aurora/internal/domain/token"
)

func TestTokenE2E_FullFlow(t *testing.T) {
    blockchain.ResetForTest()

    // 1. Create token
    // 2. Mint tokens
    // 3. Transfer tokens
    // 4. Check balance
    // 5. Burn tokens
    // 6. Verify final state
}
```

- [ ] **Step 2: 运行测试**

```bash
go test ./e2e/ -run TestTokenE2E_FullFlow -v
```

- [ ] **Step 3: Commit**

```bash
git add e2e/token_e2e_test.go
git commit -m "test: add Token E2E full flow test"
```

---

### Task 2: Token Domain 单元测试扩展

**Files:**

- Modify: `internal/domain/token/service_test.go`

- [ ] **Step 1: 添加 Transfer 测试**

- [ ] **Step 2: 添加 Burn 测试**

- [ ] **Step 3: 添加 Approve 测试**

- [ ] **Step 4: 运行测试**

```bash
go test ./internal/domain/token/ -v
```

- [ ] **Step 5: Commit**

---

### Task 3: Token App 集成测试

**Files:**

- Create: `internal/app/token/create_test.go`
- Create: `internal/app/token/mint_test.go`
- Create: `internal/app/token/transfer_test.go`
- Create: `internal/app/token/burn_test.go`

- [ ] **Step 1: 创建 Token UseCase 测试助手**

```go
func setupTokenTest(t *testing.T) (*token.Service, *inmemRepo, *inmemEventStore, *blockchain.BlockChain) {
    repo := &inmemRepo{...}
    eventStore := &inmemEventStore{...}
    chain := blockchain.InitBlockChain()
    svc := token.NewService(repo, eventStore, chain)
    return svc, repo, eventStore, chain
}
```

- [ ] **Step 2: 创建 CreateTokenUseCase 测试**

- [ ] **Step 3: 创建 MintTokenUseCase 测试**

- [ ] **Step 4: 创建 TransferTokenUseCase 测试**

- [ ] **Step 5: 运行测试**

```bash
go test ./internal/app/token/ -v
```

- [ ] **Step 6: Commit**

---

### Task 4: NFT Domain 测试扩展

**Files:**

- Modify: `internal/domain/nft/service_test.go`

- [ ] **Step 1: 添加 Transfer 测试**

- [ ] **Step 2: 添加 Query 测试**

- [ ] **Step 3: 添加 ListByOwner 测试**

- [ ] **Step 4: 运行测试**

- [ ] **Step 5: Commit**

---

### Task 5: NFT E2E 扩展

**Files:**

- Modify: `e2e/nft_e2e_test.go`

- [ ] **Step 1: 添加 Transfer 场景测试**

- [ ] **Step 2: 添加 Query 场景测试**

- [ ] **Step 3: 运行测试**

- [ ] **Step 4: Commit**

---

### Task 6: Oracle Domain 单元测试

**Files:**

- Create: `internal/domain/oracle/service_test.go`

- [ ] **Step 1: 创建 Oracle Service 测试**

```go
func TestOracleService_AddSource(t *testing.T) {
    repo := &inmemOracleRepo{...}
    svc := oracle.NewService(repo)

    source := &oracle.DataSource{
        Name: "test",
        URL:  "https://test.com",
        Type: "http",
    }

    err := svc.AddSource(source)
    assert.NoError(t, err)
}
```

- [ ] **Step 2: 添加 ToggleSource 测试**

- [ ] **Step 3: 添加 DeleteSource 测试**

- [ ] **Step 4: 运行测试**

```bash
go test ./internal/domain/oracle/ -v
```

- [ ] **Step 5: Commit**

---

### Task 7: Oracle E2E 扩展

**Files:**

- Modify: `e2e/oracle_e2e_test.go`

- [ ] **Step 1: 添加完整流程测试**

- [ ] **Step 2: 运行测试**

- [ ] **Step 3: Commit**

---

### Task 8: 最终验证

- [ ] **Step 1: 运行所有测试**

```bash
go test ./... -cover
```

- [ ] **Step 2: 检查覆盖率**

Expected:

- Token: >80%
- NFT: >70%
- Oracle: >70%

- [ ] **Step 3: Commit**

```bash
git add .
git commit -m "test: complete test coverage for Token, NFT, Oracle"
```

---

## 验收标准

- [ ] Token E2E 测试通过
- [ ] Token Domain 测试覆盖 >80%
- [ ] Token App 测试通过
- [ ] NFT Domain 测试扩展
- [ ] NFT E2E 扩展测试通过
- [ ] Oracle Domain 测试覆盖 >70%
- [ ] Oracle E2E 扩展测试通过
- [ ] 所有测试无外部依赖
