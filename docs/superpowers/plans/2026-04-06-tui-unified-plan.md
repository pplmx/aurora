# TUI 统一样式库实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 统一 Aurora 项目中所有 TUI 模块的样式定义，扩展 theme.go 为完整样式库，重构各 TUI 模块使用它。

**Architecture:** 扩展 `internal/ui/components/theme.go` 作为统一样式源，各 TUI 模块通过 import components 包使用样式函数，消除重复代码。

**Tech Stack:** Go, lipgloss, bubbletea

---

## 文件结构

```
internal/ui/
├── components/
│   └── theme.go          # 扩展至 ~350 行，统一样式库
├── lottery/
│   └── tui.go            # 移除 ~30 行重复代码，使用 components
├── token/
│   └── tui.go            # 修复语法错误，使用 components
├── nft/
│   └── tui.go            # 使用 components，补全功能
└── oracle/
    └── tui.go            # 使用 components，补全功能
```

---

## 任务分解

### Task 1: 扩展 theme.go 样式库

**Files:**
- Modify: `internal/ui/components/theme.go:238-350`

- [ ] **Step 1: 添加模块标题样式函数**

在 theme.go 末尾添加：

```go
// Module title with module-specific accent color
func ModuleTitleStyle(module string) lipgloss.Style {
    var accent lipgloss.Color
    switch module {
    case "lottery":
        accent = LotteryAcc
    case "nft":
        accent = NFTAccent
    case "oracle":
        accent = OracleAcc
    case "token":
        accent = TokenAcc
    case "voting":
        accent = VotingAcc
    default:
        accent = Primary
    }
    return lipgloss.NewStyle().
        Foreground(accent).
        Bold(true).
        Padding(0, 1)
}
```

- [ ] **Step 2: 添加输入框和视口样式**

```go
func InputStyle() lipgloss.Style {
    return lipgloss.NewStyle().
        Foreground(Text).
        Background(lipgloss.Color("236")).
        Padding(0, 1)
}

func ViewportStyle() lipgloss.Style {
    return lipgloss.NewStyle().
        Foreground(Text).
        Background(lipgloss.Color("235")).
        Border(lipgloss.RoundedBorder).
        BorderForeground(Border)
}
```

- [ ] **Step 3: 添加菜单样式函数**

```go
func MenuActiveStyle() lipgloss.Style {
    return lipgloss.NewStyle().
        Foreground(Primary).
        Bold(true).
        Background(lipgloss.Color("236"))
}

func MenuInactiveStyle() lipgloss.Style {
    return lipgloss.NewStyle().
        Foreground(Text)
}

func HelpTextStyle() lipgloss.Style {
    return lipgloss.NewStyle().
        Foreground(TextMuted)
}

func StatusBarStyle() lipgloss.Style {
    return lipgloss.NewStyle().
        Foreground(TextMuted).
        Background(lipgloss.Color("234")).
        Padding(0, 1)
}
```

- [ ] **Step 4: 添加布局辅助函数**

```go
// Center content in specified width
func Center(width int, content string) string {
    padding := width - len(content)
    if padding <= 0 {
        return content
    }
    left := padding / 2
    right := padding - left
    return strings.Repeat(" ", left) + content + strings.Repeat(" ", right)
}

// Pad left to specified width
func PadLeft(text string, width int) string {
    padding := width - len(text)
    if padding <= 0 {
        return text
    }
    return strings.Repeat(" ", padding) + text
}
```

- [ ] **Step 5: 验证编译**

```bash
go build ./internal/ui/components/
```

Expected: SUCCESS

- [ ] **Step 6: Commit**

```bash
git add internal/ui/components/theme.go
git commit -m "feat(ui): extend theme.go with unified style functions"
```

---

### Task 2: 重构 lottery/tui.go

**Files:**
- Modify: `internal/ui/lottery/tui.go:1-50`

- [ ] **Step 1: 添加 components import**

在 import 块添加：

```go
import (
    "fmt"
    "os"
    "strconv"
    "strings"

    "charm.land/bubbles/v2/textinput"
    "charm.land/bubbles/v2/viewport"
    tea "charm.land/bubbletea/v2"
    "charm.land/lipgloss/v2"

    "github.com/pplmx/aurora/internal/ui/components"
    blockChain "github.com/pplmx/aurora/internal/domain/blockchain"
    "github.com/pplmx/aurora/internal/domain/lottery"
    "github.com/pplmx/aurora/internal/i18n"
)
```

- [ ] **Step 2: 移除本地样式定义（替换为 components）**

删除 var 块（第19-46行），改为使用 components：

```go
// 使用 components 包中的样式，移除本地重复定义
```

- [ ] **Step 3: 替换 menuView 中的样式调用**

```go
// Before:
s += menuSelectedStyle.Render("▶ " + item + "\n")

// After:
s += components.MenuActiveStyle().Render("▶ " + item + "\n")
```

需要在所有使用处替换：
- headerStyle → components.HeaderStyle()
- menuItemStyle → components.MenuInactiveStyle()
- menuSelectedStyle → components.MenuActiveStyle()
- borderStyle → components.BorderStyle()
- errorStyle → components.ErrorStyle()
- successStyle → components.SuccessStyle()
- helpStyle → components.HelpTextStyle()
- infoStyle → components.InfoStyle()

- [ ] **Step 4: 验证编译**

```bash
go build ./internal/ui/lottery/
```

Expected: SUCCESS

- [ ] **Step 5: 运行 lottery TUI 测试**

```bash
go test ./internal/ui/lottery/ -v
```

- [ ] **Step 6: Commit**

```bash
git add internal/ui/lottery/tui.go
git commit -m "refactor(lottery): use unified theme.go styles"
```

---

### Task 3: 修复并重构 token/tui.go

**Files:**
- Modify: `internal/ui/token/tui.go:1-241`

- [ ] **Step 1: 修复 import 语法错误**

当前错误：import "strings" 在文件末尾（第231行）

需要将第231行的 `import "strings"` 移动到顶部 import 块中：

```go
import (
    "fmt"
    "os"
    "strings"  // 移到这里

    tea "charm.land/bubbletea/v2"

    "github.com/pplmx/aurora/internal/ui/components"
)
```

然后删除第197-229行的无用函数定义（SectionHeader, TitleStyle, HeaderStyle, Divider, KeyValue, strings struct 等），因为 components 包已提供。

- [ ] **Step 2: 验证编译**

```bash
go build ./internal/ui/token/
```

Expected: SUCCESS

- [ ] **Step 3: 运行 token TUI 测试**

```bash
go test ./internal/ui/token/ -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/ui/token/tui.go
git commit -m "fix(token): resolve import syntax error and use theme.go"
```

---

### Task 4: 重构 nft/tui.go

**Files:**
- Modify: `internal/ui/nft/tui.go:1-138`

- [ ] **Step 1: 添加 components import 并移除本地样式**

替换 import 块和 var 块：

```go
import (
    "fmt"
    "os"

    tea "charm.land/bubbletea/v2"

    "github.com/pplmx/aurora/internal/ui/components"
    "github.com/pplmx/aurora/internal/i18n"
)
```

移除 var 块中的本地样式定义。

- [ ] **Step 2: 替换所有样式调用**

在 menuView, infoView 中使用 components.* 替换。

- [ ] **Step 3: 验证编译**

```bash
go build ./internal/ui/nft/
```

Expected: SUCCESS

- [ ] **Step 4: Commit**

```bash
git add internal/ui/nft/tui.go
git commit -refactor(nft): use unified theme.go styles
```

---

### Task 5: 重构 oracle/tui.go

**Files:**
- Modify: `internal/ui/oracle/tui.go:1-200`

- [ ] **Step 1: 添加 components import 并重构**

类似于 Task 4，添加 components 包使用。

- [ ] **Step 2: 验证编译**

```bash
go build ./internal/ui/oracle/
```

- [ ] **Step 3: Commit**

```bash
git add internal/ui/oracle/tui.go
git commit -m "refactor(oracle): use unified theme.go styles"
```

---

### Task 6: 最终验证

- [ ] **Step 1: 运行完整构建**

```bash
make build
```

- [ ] **Step 2: 运行 lint**

```bash
make lint
```

- [ ] **Step 3: 运行测试**

```bash
make test
```

- [ ] **Step 4: Commit**

```bash
git add .
git commit -m "chore: complete TUI style unification"
```

---

## 验收标准

- [ ] theme.go 扩展完成，包含所有通用样式函数
- [ ] lottery/tui.go 使用 components 包，无本地样式定义
- [ ] token/tui.go 语法错误已修复，使用 components 包
- [ ] nft/tui.go 使用 components 包
- [ ] oracle/tui.go 使用 components 包
- [ ] 编译通过，无 lint 错误
- [ ] 所有 TUI 功能正常