# TUI 统一样式库设计方案

## 目标

统一 Aurora 项目中所有 TUI 模块的样式定义，扩展 `internal/ui/components/theme.go` 为完整的样式库，并重构各 TUI 模块使用它。

## 背景

当前状态：
- `components/theme.go`: 238行，已有基础样式函数
- `lottery/tui.go`: 375行，完整实现但未使用 theme.go
- `token/tui.go`: 241行，部分使用 theme.go 但有语法错误
- `nft/tui.go`: 138行，stub 状态
- `oracle/tui.go`: 200行，stub 状态

问题：
- 各模块重复定义样式变量（~100行重复代码）
- 样式不统一，维护困难
- token/tui.go 有语法错误（import 位置错误）

## 设计

### 1. 扩展 theme.go 样式库

#### 1.1 模块专属颜色（已存在，部分需增强）

```go
// Color values
var (
    Primary    = lipgloss.Color("86")    // 主色
    Secondary  = lipgloss.Color("75")    // 次色
    Accent     = lipgloss.Color("205")   // 强调色
    LotteryAcc = lipgloss.Color("212")   // 抽奖模块
    NFTAccent  = lipgloss.Color("219")   // NFT模块
    OracleAcc  = lipgloss.Color("75")    // Oracle模块
    TokenAcc   = lipgloss.Color("82")    // Token模块
    VotingAcc  = lipgloss.Color("205")   // 投票模块
)
```

#### 1.2 新增样式函数

| 函数 | 用途 |
|------|------|
| `ModuleTitleStyle(module string)` | 模块标题，根据模块名返回对应颜色 |
| `InputStyle()` | 文本输入框样式 |
| `ViewportStyle()` | 列表容器样式 |
| `MenuActiveStyle()` | 选中菜单项 |
| `MenuInactiveStyle()` | 未选中菜单项 |
| `HelpTextStyle()` | 帮助文本 |
| `StatusBarStyle()` | 底部状态栏 |

#### 1.3 布局辅助函数

| 函数 | 用途 |
|------|------|
| `Center(width int, content string)` | 居中内容 |
| `PadLeft(text string, width int)` | 左侧填充 |
| `TableRow(columns []string, widths []int)` | 表格行 |

### 2. 重构各 TUI 模块

#### 2.1 lottery/tui.go

- 移除重复的样式定义（~30行）
- 使用 `components.*` 替代本地样式
- 保持功能完整

#### 2.2 token/tui.go

- 修复 import 语法错误
- 使用 `components.*` 统一样式
- 保持功能完整

#### 2.3 nft/tui.go

- 使用 `components.*` 统一样式
- 补全 mint/transfer/query 视图

#### 2.4 oracle/tui.go

- 使用 `components.*` 统一样式
- 补全功能

### 3. 代码示例

#### 使用 theme.go 前后对比

**Before** (lottery/tui.go):
```go
var (
    headerStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("86")).
        Bold(true).
        Padding(0, 1)
    menuItemStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("252"))
    menuSelectedStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("86")).
        Bold(true)
)
```

**After**:
```go
import "github.com/pplmx/aurora/internal/ui/components"

// 使用现有样式
s := components.HeaderStyle().Render("🌟 Lottery 🌟")
s += components.ModuleTitleStyle("lottery").Render("Create")
```

## 实现计划

### 阶段 1: 扩展 theme.go

1. 添加模块标题样式函数
2. 添加输入框/视口样式
3. 添加菜单样式函数
4. 添加布局辅助函数

### 阶段 2: 重构 lottery/tui.go

1. 移除重复样式定义
2. 导入 components 包
3. 替换所有样式调用

### 阶段 3: 修复 token/tui.go

1. 修复 import 语法错误
2. 统一使用 components

### 阶段 4: 重构 nft/tui.go & oracle/tui.go

1. 应用统一样式
2. 补全功能视图

## 验收标准

- [ ] 所有 TUI 模块使用统一的样式定义
- [ ] theme.go 包含所有通用样式
- [ ] 代码中无重复样式定义
- [ ] 编译通过，无 lint 错误
- [ ] TUI 功能正常