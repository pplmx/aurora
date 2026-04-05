# TUI 功能补全实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 NFT、Oracle、Token 三个 TUI 模块添加完整交互功能，参考 lottery TUI 的实现模式。

**Architecture:** 在现有 stub 基础上添加 textinput 表单、viewport 列表、domain 业务调用，实现完整的创建/查询/操作流程。

**Tech Stack:** Go, bubbletea v2, textinput, viewport, lipgloss

---

## 文件结构

```text
internal/ui/
├── nft/
│   └── tui.go         # 从 117 行扩展到 ~300 行
├── oracle/
│   └── tui.go         # 从 176 行扩展到 ~350 行
└── token/
    └── tui.go         # 从 208 行扩展到 ~350 行
```

---

## 任务分解

### Task 1: NFT TUI - 添加 Mint 功能

**Files:**

- Modify: `internal/ui/nft/tui.go`

需要添加：

- `textinput.Model` 用于 NFT 名称、描述、公钥输入
- mint 视图：3 个表单字段
- `handleMint()` 处理函数
- 调用 domain NFT 服务

- [ ] **Step 1: 添加 imports**

```go
import (
    "charm.land/bubbles/v2/textinput"
    "charm.land/bubbles/v2/viewport"
    // ... existing imports
)
```

- [ ] **Step 2: 扩展 model 结构体**

```go
type model struct {
    view          string
    menuIndex     int

    // Form inputs for mint
    nameInput     textinput.Model
    descInput     textinput.Model
    pubkeyInput   textinput.Model

    // Result display
    result        *nft.NFT
    err           string
    successMsg    string

    // Domain
    chain         *blockchain.BlockChain
}
```

- [ ] **Step 3: 初始化 textinput**

在 `NewNFTApp()` 中添加：

```go
nameInput := textinput.New()
nameInput.Placeholder = "NFT Name"
nameInput.Focus()

descInput := textinput.New()
descInput.Placeholder = "Description"

pubkeyInput := textinput.New()
pubkeyInput.Placeholder = "Creator Public Key"
```

- [ ] **Step 4: 添加 mint 视图**

```go
func (m *model) mintView() string {
    s := components.ModuleTitleStyle("nft").Render("🖼️ Mint NFT") + "\n\n"
    s += components.InfoStyle().Render("Name:") + "\n"
    s += m.nameInput.View() + "\n\n"
    s += components.InfoStyle().Render("Description:") + "\n"
    s += m.descInput.View() + "\n\n"
    s += components.InfoStyle().Render("Public Key:") + "\n"
    s += m.pubkeyInput.View() + "\n\n"

    if m.err != "" {
        s += components.ErrorStyle().Render("⚠ "+m.err) + "\n\n"
    }
    if m.successMsg != "" {
        s += components.SuccessStyle().Render("✓ "+m.successMsg) + "\n\n"
    }

    s += components.BorderStyle().Render("[Enter] Mint | [ESC] Back")
    return s
}
```

- [ ] **Step 5: 添加 Update 处理**

在 `Update()` 的 `case "enter":` 中添加：

```go
case "mint":
    m.nameInput.Focus()
    // ... handle form navigation
```

- [ ] **Step 6: 添加 handleMint()**

```go
func (m *model) handleMint() tea.Msg {
    name := m.nameInput.Value()
    desc := m.descInput.Value()
    pubkey := m.pubkeyInput.Value()

    if name == "" {
        m.err = "Name is required"
        return nil
    }
    if pubkey == "" {
        m.err = "Public key is required"
        return nil
    }

    // Call domain service
    nft := &nft.NFT{
        Name:        name,
        Description: desc,
        Creator:     pubkey,
    }
    jsonData, _ := json.Marshal(nft)
    height, _ := m.chain.AddNFT(jsonData)

    m.result = nft
    m.view = "result"
    m.successMsg = "NFT minted successfully"
    return nil
}
```

- [ ] **Step 7: 添加 result 视图**

```go
func (m *model) resultView() string {
    if m.result == nil {
        return "No result"
    }
    s := components.SuccessStyle().Render("✓ Minted!") + "\n\n"
    s += components.KeyValue("Name", m.result.Name) + "\n"
    s += components.KeyValue("ID", m.result.ID) + "\n"
    s += components.KeyValue("Creator", m.result.Creator) + "\n"
    s += "\n" + components.BorderStyle().Render("[ESC] Back")
    return s
}
```

- [ ] **Step 8: 验证编译**

```bash
go build ./internal/ui/nft/
```

- [ ] **Step 9: Commit**

```bash
git add internal/ui/nft/tui.go
git commit -m "feat(nft-tui): add mint functionality"
```

---

### Task 2: NFT TUI - 添加 Transfer/Query/List 功能

**Files:**

- Modify: `internal/ui/nft/tui.go`

- [ ] **Step 1: 添加 Transfer 功能**

- 添加 transfer 相关的 textinput（nftID, fromPrivkey, toAddress）
- 添加 transfer 视图
- 添加 handleTransfer() 函数

- [ ] **Step 2: 添加 Query 功能**

- 添加 query 相关的 textinput（nftID）
- 添加 query 视图
- 添加 handleQuery() 函数

- [ ] **Step 3: 添加 List 功能**

- 添加 list 相关的 textinput（ownerPubkey）
- 添加 viewport 用于显示列表
- 添加 handleList() 函数

- [ ] **Step 4: 更新 menu 索引**

菜单从 3 项改为 4 项（mint, transfer, query, list, exit）

- [ ] **Step 5: 验证编译**

```bash
go build ./internal/ui/nft/
```

- [ ] **Step 6: Commit**

```bash
git add internal/ui/nft/tui.go
git commit -m "feat(nft-tui): add transfer, query, list functionality"
```

---

### Task 3: Oracle TUI - 完善 Source Management

**Files:**

- Modify: `internal/ui/oracle/tui.go`

- [ ] **Step 1: 添加 textinput 用于添加数据源**

```go
nameInput := textinput.New()
nameInput.Placeholder = "Source Name"

urlInput := textinput.New()
urlInput.Placeholder = "URL"

typeInput := textinput.New()
typeInput.Placeholder = "Type (http, ws, api)"
```

- [ ] **Step 2: 添加 addSource 视图**

- [ ] **Step 3: 添加 handleAddSource()**

调用 `oracleapp.CreateSourceUseCase`

- [ ] **Step 4: 添加 toggle source 功能**

- [ ] **Step 5: 验证编译**

```bash
go build ./internal/ui/oracle/
```

- [ ] **Step 6: Commit**

```bash
git add internal/ui/oracle/tui.go
git commit -m "feat(oracle-tui): add source management functionality"
```

---

### Task 4: Oracle TUI - 添加 Fetch/Query 功能

**Files:**

- Modify: `internal/ui/oracle/tui.go`

- [ ] **Step 1: 添加 Fetch 功能**

- textinput: sourceID
- 视图和 handler
- 调用 `oracleapp.FetchDataUseCase`

- [ ] **Step 2: 添加 Query 功能**

- textinput: sourceID, timeRange
- viewport 显示结果

- [ ] **Step 3: 验证编译**

- [ ] **Step 4: Commit**

```bash
git add internal/ui/oracle/tui.go
git commit -m "feat(oracle-tui): add fetch and query functionality"
```

---

### Task 5: Token TUI - 完善表单交互

**Files:**

- Modify: `internal/ui/token/tui.go`

当前 token/tui.go 已有基础视图，但需要：

- 添加实际 textinput 输入
- 添加业务逻辑调用
- 修复 View() 返回类型

- [ ] **Step 1: 添加 textinput 到 model**

```go
type model struct {
    view          string
    menuIndex     int

    // Form inputs
    nameInput     textinput.Model
    symbolInput   textinput.Model
    supplyInput   textinput.Model
    toInput       textinput.Model
    amountInput   textinput.Model
    privkeyInput  textinput.Model
    addressInput  textinput.Model

    // Display
    viewport      viewport.Model
    result        interface{}
    err           string
    successMsg    string

    // Domain
    chain         *blockchain.BlockChain
}
```

- [ ] **Step 2: 初始化 inputs**

- [ ] **Step 3: 更新各视图添加 input.View()**

- [ ] **Step 4: 添加 handler 函数**

- [ ] **Step 5: 修复 View() tea.View 返回**

- [ ] **Step 6: 验证编译**

```bash
go build ./internal/ui/token/
```

- [ ] **Step 7: Commit**

```bash
git add internal/ui/token/tui.go
git commit -m "feat(token-tui): add form inputs and handlers"
```

---

### Task 6: 最终验证

- [ ] **Step 1: 运行完整构建**

```bash
go build ./...
```

- [ ] **Step 2: 运行 lint**

```bash
golangci-lint run ./internal/ui/...
```

- [ ] **Step 3: 运行测试**

```bash
go test ./...
```

- [ ] **Step 4: Commit**

```bash
git add .
git commit -m "chore: complete TUI functionality"
```

---

## 验收标准

- [ ] NFT TUI: mint/transfer/query/list 功能完整
- [ ] Oracle TUI: source management/fetch/query 功能完整
- [ ] Token TUI: create/mint/transfer/balance/history 功能完整
- [ ] 编译通过
- [ ] Lint 0 issues
- [ ] 测试通过
