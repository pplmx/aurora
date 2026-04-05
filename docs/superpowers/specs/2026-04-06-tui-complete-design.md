# TUI 功能补全设计方案

## 目标

为 NFT、Oracle、Token 三个 TUI 模块添加完整交互功能，参考 lottery TUI 的实现模式。

## 背景

当前状态：

- **lottery/tui.go**: 完整实现（346行），包含创建、历史、帮助视图，使用 textinput 和 viewport
- **nft/tui.go**: stub 状态（117行），只有菜单和 info 视图
- **oracle/tui.go**: 部分实现（176行），有菜单和 3 个视图，但无表单输入
- **token/tui.go**: 部分实现（208行），有菜单和 5 个视图，View() 使用 tea.View 但功能不完整

## 功能需求

### NFT TUI 功能

| 功能          | 描述                                      |
| ------------- | ----------------------------------------- |
| Mint NFT      | 输入 NFT 名称、描述、公钥，铸造 NFT       |
| Transfer NFT  | 输入 NFT ID、转出方私钥、接收方地址，转账 |
| Query NFT     | 输入 NFT ID，查询详情                     |
| List by Owner | 输入公钥，列出该地址拥有的所有 NFT        |

### Oracle TUI 功能

| 功能              | 描述                        |
| ----------------- | --------------------------- |
| Source Management | 列出所有数据源，启用/禁用   |
| Fetch Data        | 输入数据源 ID，触发数据获取 |
| Query Data        | 输入数据源 ID，查看最新数据 |

### Token TUI 功能

| 功能         | 描述                               |
| ------------ | ---------------------------------- |
| Create Token | 输入名称、符号、总供应量，创建代币 |
| Mint         | 输入接收地址、数量、私钥，铸造代币 |
| Transfer     | 输入接收方地址、数量、私钥，转账   |
| Balance      | 输入地址，查询余额                 |
| History      | 列出交易历史                       |

## 架构设计

### Model 结构

```go
type model struct {
    view              string
    menuIndex         int

    // Form inputs (textinput.Model)
    input1            textinput.Model
    input2            textinput.Model
    input3            textinput.Model

    // List display (viewport.Model)
    viewport          viewport.Model

    // Business data
    result            interface{}
    err               string
    successMsg        string

    // Domain references
    chain             *blockchain.BlockChain
    // or
    repo              domain.Repository
}
```

### View 状态机

```text
menu
├── create/form1 → form2 → form3 → result
├── mint/form1 → form2 → form3 → result
├── transfer/form1 → form2 → form3 → result
├── query/form → result
└── list/form → result
```

### 通用模式

1. **输入表单**: 使用 `textinput.Model`，支持 placeholder、验证
2. **列表展示**: 使用 `viewport.Model`，支持滚动
3. **导航**: 上下键选择菜单，回车确认，ESC 返回
4. **错误处理**: 显示错误信息，允许重新输入

## 组件使用

```go
import (
    "charm.land/bubbles/v2/textinput"
    "charm.land/bubbles/v2/viewport"
    tea "charm.land/bubbletea/v2"

    "github.com/pplmx/aurora/internal/ui/components"
)
```

## 验收标准

- [ ] NFT TUI 可完整执行 mint/transfer/query/list 操作
- [ ] Oracle TUI 可完整执行 source 管理/fetch/query 操作
- [ ] Token TUI 可完整执行 create/mint/transfer/balance/history 操作
- [ ] 所有模块使用 components 包统一样式
- [ ] 编译通过，无 lint 错误
- [ ] 代码风格与 lottery/tui.go 一致
