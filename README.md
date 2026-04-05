# aurora

[![Go Report Card](https://goreportcard.com/badge/github.com/golang-standards/project-layout?style=flat-square)](https://goreportcard.com/report/github.com/pplmx/aurora)
[![Go Doc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/pplmx/aurora)
[![Release](https://img.shields.io/github/release/golang-standards/project-layout.svg?style=flat-square)](https://github.com/golang-standards/project-layout/releases/latest)

Aurora - 基于区块链的数字投票系统，更安全、透明、不可篡改。

## 功能

### VRF 透明抽奖系统
- 🎲 基于 VRF（可验证随机函数）的抽奖
- ✅ 结果上链存证，可验证
- 🖥️ TUI 和 CLI 界面

## 快速开始

### 构建

```bash
make build
# 或
go build -o aurora ./cmd/aurora
```

### CLI

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

### TUI（交互式）

```bash
./aurora lottery tui
```

## 开发

```bash
make test         # 运行所有测试
make check        # 代码检查 (gofmt, goimports, go vet)
make lint         # linter (需要 golangci-lint)
make build        # 构建所有平台
make dev          # Docker 开发环境
```

## 测试

```bash
# 单元测试
go test ./internal/lottery/ -v

# E2E 功能测试
go test ./test/ -v

# 所有测试
go test ./...
```

## 项目结构

```
cmd/aurora/       # CLI 入口
internal/
├── blockchain/   # 区块链核心
├── lottery/      # 抽奖系统
├── voting/       # 投票系统
├── logger/       # 日志
└── utils/        # 工具
test/
└── lottery_e2e_test.go  # E2E 功能测试
```

## 技术栈

- Go 1.26+
- Cobra (CLI)
- Viper (配置)
- Zerolog (日志)
- Edwards25519 (VRF)
- Bubble Tea (TUI)
- Lipgloss (样式)

## 文档

- [抽奖系统文档](docs/lottery/README.md)
- [测试文档](docs/lottery/testing.md)
