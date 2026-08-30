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

### 使用 justfile

```bash
just test          # 运行测试
just build        # 构建所有平台
just lint         # 代码检查
just dev          # Docker 开发
```

### CLI 示例

```bash
# 抽奖
./aurora lottery create -p "A,B,C,D" -s "seed" -c 3
./aurora lottery history
./aurora lottery tui

# 投票
./aurora voting candidate add -n "Name" -p "Party"
./aurora voting voter register -n "Name"
./aurora voting session create -t "Proposal" -c <cand-id> --start-time <t> --end-time <t>
./aurora voting vote -v <voter-pk> -c <candidate-id> -s <session-id> -k <priv-key>
./aurora voting results -s <session-id>

# 预言机
./aurora oracle source list
./aurora oracle source add --name "btc" --url "https://api.example.com/price"
./aurora oracle source enable --id <source-id>
./aurora oracle source disable --id <source-id>
./aurora oracle source delete --id <source-id> --confirm   # 需 --confirm
./aurora oracle template list
./aurora oracle template add -t <template-name>            # 从预设模板建源
./aurora oracle fetch -i <source-id>            # -i/--id（旧拼写 -s 仍可用）
./aurora oracle data -i <source-id> --limit 10
./aurora oracle latest -i <source-id>
./aurora oracle tui

# NFT
./aurora nft mint -n "My NFT" -d "Description" -c "<creator-pub>"
./aurora nft transfer --nft <id> --from <owner> --to <to> -k <priv>
./aurora nft get --nft <nft_id>
./aurora nft list --owner <pubkey>
./aurora nft history --nft <nft_id>
./aurora nft burn --nft <nft_id> --owner <pub> -k <priv> --confirm   # 需 --confirm
./aurora nft tui

# Token
./aurora token create -n "MyToken" -s "SYMBOL" --supply 1000000
./aurora token info -t <token-id>
./aurora token mint -t <token-id> --to <address> --amount 100 -k <priv>
./aurora token transfer -t <token-id> --from <addr> --to <address> --amount 50 -k <priv>
./aurora token balance -t <token-id> --owner <address>
./aurora token history -t <token-id> --owner <address>
./aurora token approve -t <token-id> --owner <pub> -s <spender-pub> -a <amount> -k <priv>
./aurora token allowance -t <token-id> -o <pub> -s <spender-pub>
./aurora token transfer-from -t <token-id> -o <pub> -s <spender-pub> --to <recipient> -a <amount> -k <spender-priv>
./aurora token burn -t <token-id> --from <pub> -a <amount> -k <priv> --confirm   # 需 --confirm
./aurora token tui
```

> **破坏性操作确认门：** `token burn`、`nft burn`、`oracle source delete`、
> `migrate down`、`backup restore`、`lottery reset` 全部接受 `-y/--confirm`；
> `lottery reset` 也接受旧写法的 `--yes`。没有确认标志这些命令会直接拒绝执行
> （非零退出码），防止误删数据。

### 运维命令

```bash
./aurora version                     # 显示版本与构建信息

# 数据库备份（备份目录含元数据，可校验）
./aurora backup create ./backups/backup-$(date +%Y%m%d)
./aurora backup verify ./backups/backup-$(date +%Y%m%d)
./aurora backup restore ./backups/backup-$(date +%Y%m%d) --confirm   # 覆盖当前库

# 数据库迁移
./aurora migrate status             # 当前 / 已应用 / 待应用
./aurora migrate up                 # 应用全部待迁移
./aurora migrate up 2               # 应用接下来两步（up N 应用 N 步待迁移）
./aurora migrate down --confirm     # 回滚一步（默认 1，需 --confirm）
```

## Web / API 界面

Aurora 提供内置的 REST API 与 Web 界面，由独立的 `cmd/api` 二进制启动
（`./aurora` 是纯 CLI/TUI，不带 HTTP 服务）。Web 页面由 API 服务托管在
`web/` 目录（默认相对当前工作目录；若非从仓库根目录启动，可在 `[server]`
配置 `webRoot = "/path/to/web"` 指定实际路径，TASK-181）。

```bash
go build -o aurora-api ./cmd/api
AURORA_API_KEY="your-strong-key" ./aurora-api
# 默认监听 0.0.0.0:8080（可通过 [server] 配置 host / port 调整）
# 然后浏览器访问 http://localhost:8080
```

- **鉴权**：API 通过 `X-API-Key` 请求头校验密钥。密钥来自
  `AURORA_API_KEY` 环境变量或配置项 `api.key`；开发模式下未设置时会生成
  一个随机密钥并在启动日志打印。Web 页面会自动把密钥注入
  `window.AURORA_API_KEY`，同源浏览器请求无需手动带 key。
- **端到端**：API 服务同时启动预言机定时调度器（按源的 interval 拉取）与
  审计事件 outbox 补偿循环，SIGINT/SIGTERM 优雅停机（15s 超时）。
- **可观测**：`/metrics`（Prometheus 文本）暴露请求计数器与按模块计数；
  `/metrics/oracle` 暴露各预言机源的抓取健康统计。
- **安全**：API 支持请求体大小上限（4 MiB）、速率限制（按客户端）、CORS
  白名单，生产环境拒绝弱 API key（`ErrInsecureAPIKey`）。

## 项目结构 (DDD 架构)

```text
cmd/aurora/              # CLI 入口
cmd/api/                 # REST API + Web 服务入口（go build ./cmd/api）
web/                     # Web 前端（HTML + Alpine.js，由 cmd/api 托管）
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
just build
go build -o aurora ./cmd/aurora

# Docker
just dev      # 开发模式
just start    # 启动容器
just stop     # 停止容器
```

## 测试覆盖率

| 模块    | Domain | App   |
| ------- | ------ | ----- |
| Lottery | 92.8%  | 85.7% |
| Voting  | 87.5%  | 87.8% |
| NFT     | 90.3%  | 89.2% |
| Token   | 91.1%  | 91.8% |
| Oracle  | 89.3%  | 88.9% |
| SQLite  | -      | 85.0% |

> 覆盖率为 `go test -cover` 实测（2026-08-29，`-count=1`）。

## 贡献

欢迎提交 Issue 和 PR！

## 许可证

MIT
