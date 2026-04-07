# Aurora

[![Go Report Card](https://goreportcard.com/badge/github.com/pplmx/aurora?style=flat-square)](https://goreportcard.com/report/github.com/pplmx/aurora)
[![Tests](https://github.com/pplmx/aurora/actions/workflows/ci.yml/badge.svg)](https://github.com/pplmx/aurora/actions)
[![Release](https://img.shields.io/github/v/release/pplmx/aurora)](https://github.com/pplmx/aurora/releases)

基于区块链的数字系统套件，支持抽奖、投票、预言机、NFT 和代币。采用 **DDD (领域驱动设计)** 架构。

## 功能

### 🎲 VRF 透明抽奖

- 基于 VRF（可验证随机函数）的抽奖
- 结果上链存证，可验证
- CLI 和 TUI 界面

### 🗳️ 透明投票系统

- Ed25519 签名验证
- 区块链存证
- 实时计票

### 🔮 数据预言机

- 通用 HTTP API 数据获取
- 预设模板（BTC/ETH 价格）
- 数据上链存证

### 🖼️ NFT 系统

- Ed25519 签名转移
- 铸造、转让、查询
- 区块链存证

### 🪙 FT 代币系统

- ERC-20 风格 Fungible Token
- Mint、Transfer、Burn、Approve
- 完整余额和授权管理

## 快速开始

### 安装

```bash
# 下载 releases 或编译
go build -o aurora ./cmd/aurora
```

### 使用 makefile

```bash
make test          # 运行测试
make build        # 构建所有平台
make lint         # 代码检查
make dev          # Docker 开发
```

### CLI 示例

```bash
# 抽奖
./aurora lottery create -p "A,B,C,D" -s "seed" -c 3
./aurora lottery history
./aurora lottery tui

# 投票
./aurora voting create -t "Proposal" -o "owner_key"
./aurora voting vote -k "signing_key"
./aurora voting tui

# 预言机
./aurora oracle sources
./aurora oracle fetch --source <id>
./aurora oracle query --source <id> --limit 10
./aurora oracle tui

# NFT
./aurora nft mint -n "My NFT" -d "Description" -c "<creator-pub>"
./aurora nft transfer --nft <id> --to <to> -k <priv>
./aurora nft get --id <nft_id>
./aurora nft list --owner <pubkey>
./aurora nft tui

# Token
./aurora token create -n "MyToken" -s "SYMBOL" --supply 1000000
./aurora token mint --to <address> --amount 100 -k <priv>
./aurora token transfer --to <address> --amount 50 -k <priv>
./aurora token balance --owner <address>
./aurora token history
./aurora token tui
```

## 项目结构 (DDD 架构)

```text
cmd/aurora/              # CLI 入口
internal/
├── domain/               # 领域层 - 实体、服务、仓储接口
│   ├── blockchain/       # 区块链核心 (Block, BlockChain)
│   ├── lottery/         # 抽奖领域 (LotteryRecord, VRF Service)
│   ├── voting/          # 投票领域 (Vote, Voter, Candidate)
│   ├── nft/             # NFT 领域 (NFT, Operation)
│   ├── token/           # 代币领域 (Token, Amount, Approval)
│   └── oracle/          # 预言机领域 (OracleData, DataSource)
│
├── infra/               # 基础设施层 - 存储实现
│   ├── sqlite/          # SQLite 持久化
│   └── http/            # HTTP 客户端
│
├── app/                 # 应用层 - 用例
│   ├── lottery/         # CreateLotteryUseCase
│   ├── voting/          # CastVoteUseCase, RegisterVoterUseCase
│   ├── nft/            # MintNFTUseCase, TransferNFTUseCase
│   ├── token/          # CreateTokenUseCase, MintTokenUseCase
│   └── oracle/          # FetchDataUseCase
│
├── ui/                  # 表示层 - TUI 界面
│   ├── lottery/
│   ├── nft/
│   ├── token/
│   └── oracle/
│
├── i18n/                # 国际化
├── logger/               # 日志
└── utils/                # 工具
e2e/                     # E2E 测试
.github/workflows/        # CI/CD
```

### DDD 分层说明

| 层         | 职责                                   | 示例                                             |
| ---------- | -------------------------------------- | ------------------------------------------------ |
| **domain** | 核心业务逻辑、实体、领域服务、仓储接口 | `LotteryRecord`, `VRFService`, `Repository` 接口 |
| **app**    | 用例编排、DTO 转换                     | `CreateLotteryUseCase`                           |
| **infra**  | 外部依赖实现                           | `SQLiteRepository`, `HTTPFetcher`                |
| **ui**     | 用户界面                               | Bubble Tea TUI                                   |

## 技术栈

- Go 1.26+
- Cobra (CLI)
- Viper (配置)
- Zerolog (日志)
- Ed25519 (签名)
- Bubble Tea v2 (TUI)
- SQLite (持久化)

## 开发

```bash
# 测试
go test ./...           # 所有测试
go test ./... -cover   # 带覆盖率

# 代码检查
go vet ./...
golangci-lint run ./...

# 构建
make build
go build -o aurora ./cmd/aurora

# Docker
make dev      # 开发模式
make start    # 启动容器
make stop     # 停止容器
```

## 测试覆盖率

| 模块    | Domain | App   |
| ------- | ------ | ----- |
| Lottery | 73.8%  | 87.1% |
| Voting  | 87.5%  | 81.4% |
| NFT     | 72.7%  | 88.1% |
| Token   | 70.5%  | 91.9% |
| Oracle  | 76.1%  | 65.8% |
| SQLite  | -      | 45.5% |

## 贡献

欢迎提交 Issue 和 PR！

## 许可证

MIT
