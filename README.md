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

### CLI

```bash
# 创建抽奖
./aurora lottery create -p "张三,李四,王五" -s "种子" -c 3

# 查看历史
./aurora lottery history
```

### TUI

```bash
./aurora lottery tui
```

## 开发

```bash
make test         # 运行测试
make build        # 构建
make dev          # Docker 开发环境
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
```

## 技术栈

- Go 1.26+
- Cobra (CLI)
- Viper (配置)
- Zerolog (日志)
- Edwards25519 (VRF)
- tview (TUI)