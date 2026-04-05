# VRF 透明抽奖系统

基于 VRF（可验证随机函数）的透明抽奖系统，使用 Go 实现。

## 特性

- 🎲 **VRF 随机数** - 基于 Edwards25519 曲线生成可验证的随机数
- ✅ **上链存证** - 抽奖结果存储在区块链上，不可篡改
- 🔍 **可验证** - 任何人都可以验证抽奖结果的公平性
- 🖥️ **TUI 界面** - 使用 Bubble Tea 的交互式终端界面
- 📖 **CLI 命令** - 便捷的命令行工具

## 快速开始

### 构建

```bash
go build -o aurora ./cmd/aurora
```

### CLI 模式

```bash
# 创建抽奖
./aurora lottery create -p "张三,李四,王五,赵六" -s "种子字符串" -c 3

# 参数说明
# -p, --participants: 参与者名单（逗号分隔）
# -s, --seed: 随机种子
# -c, --count: 获奖人数（默认3）

# 查看历史
./aurora lottery history
```

### TUI 模式

```bash
./aurora lottery tui
```

## 技术实现

### VRF 流程

1. 用户输入参与者名单和随机种子
2. 系统生成 Edwards25519 密钥对
3. 使用私钥对种子进行 VRF 运算，生成随机输出和证明
4. 使用随机输出选择中奖者
5. 将结果写入区块链

### 地址转换

参与者名字通过 SHA256 哈希转换为虚拟地址：

```text
张三 → SHA256 → 0x7a3f...
```

## 项目结构

```text
cmd/aurora/cmd/
└── lottery.go           # CLI 命令入口

internal/lottery/
├── address.go           # 名字转地址
├── address_test.go      # 测试
├── vrf.go               # VRF 实现 (Edwards25519)
├── vrf_test.go          # 测试
├── lottery.go           # 抽奖核心逻辑
├── lottery_test.go      # 单元测试
└── tui.go               # TUI 界面 (Bubble Tea)

test/
└── lottery_e2e_test.go  # E2E 功能测试
```

## 测试

### 单元测试

```bash
go test ./internal/lottery/ -v
```

### E2E 功能测试

```bash
go test ./test/ -v
```

### 覆盖的测试场景

| 测试文件              | 说明                                     |
| --------------------- | ---------------------------------------- |
| `lottery_test.go`     | 单元测试：地址转换、VRF、抽奖逻辑        |
| `lottery_e2e_test.go` | E2E 测试：完整流程、多次抽奖、数据完整性 |

## 依赖

- Go 1.26+
- filippo.io/edwards25519 - Edwards25519 曲线实现
- github.com/charmbracelet/bubbletea - TUI 框架
- github.com/charmbracelet/lipgloss - 样式
- github.com/spf13/cobra - CLI 框架

## 相关文档

- [测试文档](testing.md)
