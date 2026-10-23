# Aurora

[![Go Report Card](https://goreportcard.com/badge/github.com/pplmx/aurora?style=flat-square)](https://goreportcard.com/report/github.com/pplmx/aurora)
[![Tests](https://github.com/pplmx/aurora/actions/workflows/ci.yml/badge.svg)](https://github.com/pplmx/aurora/actions)
[![Release](https://img.shields.io/github/v/release/pplmx/aurora)](https://github.com/pplmx/aurora/releases)

基于区块链的数字系统套件：VRF 透明抽奖、Ed25519 签名投票、数据预言机、NFT 与
FT 代币。核心用 Go 实现，按 DDD 分层；同一份后端同时提供**桌面应用**、CLI/TUI
和 REST API + Web 界面。

## 三个入口

| 入口           | 二进制           | 适合谁                                       |
| -------------- | ---------------- | -------------------------------------------- |
| 桌面应用       | `aurora-desktop` | **主要使用方式**：双击即用，无需终端或浏览器 |
| CLI / TUI      | `aurora`         | 脚本、批量任务、服务器与无图形环境           |
| REST API + Web | `aurora-api`     | 集成方、浏览器访问、需要监听的部署           |

桌面端与 API 用同一套路由和同一份 SQLite 数据库；桌面端通过自定义 scheme
`aurora://localhost/` 在进程内加载页面，不监听端口，API key 不出本机。设计细节见
[docs/desktop.md](docs/desktop.md)。

## 安装

从 [Releases](https://github.com/pplmx/aurora/releases) 下载：

- **桌面端**：Linux `.deb` / `.tar.gz`+`install.sh`（amd64、arm64）、macOS
  `.dmg`（arm64、amd64）、Windows `Aurora.exe` / `*Setup*.exe`。
- **CLI**：`aurora-<os>-<arch>`，纯 Go 构建，发布时附 `checksums.txt`。
- 发布流程会校验版本号与构建时间，不发布 `0.0.1` / `Build Time: unknown`
  的产物。
- **构建来源证明**：发布前会为 `checksums.txt` 中的最终资产生成 GitHub
  artifact attestation。下载后可用
  `gh attestation verify <artifact> --repo pplmx/aurora` 检查文件是否由本仓库
  的发布工作流构建；它补充校验和，但不能替代 macOS/Windows 的操作系统级签名。
- **签名与隐私**：Windows 包正在申请 SignPath Foundation 的免费开源签名；
  macOS 包仍需要维护者持有的 Apple Developer Program 身份。发布门槛、角色与
  验收方式见[代码签名策略](docs/code-signing-policy.md)，本机数据与网络边界见
  [隐私说明](docs/privacy.md)。

从源码构建：

```bash
CGO_ENABLED=0 go build -o aurora ./cmd/aurora      # CLI（Go 1.27+）
CGO_ENABLED=0 go build -o aurora-api ./cmd/api     # REST API + Web
just desktop                             # 桌面壳（Go 1.27+、WebKitGTK/WebView2）
```

`just` 是唯一的任务入口（没有 Makefile）：

```bash
just test            # 核心模块全部测试
just build           # check + test + 全平台 CLI
just api             # ./aurora-api，注入 git 版本与构建时间
just desktop         # ./aurora-desktop（本地开发构建）
just desktop-run     # 构建并打开窗口
just desktop-test    # 桌面模块测试（根目录 just test 不进嵌套模块）
just desktop-check   # gofmt + build + vet + 纯 Go 测试（无需 C 工具链）
just desktop-race    # 可选 race 检测（显式启用 cgo）
just desktop-package # mygo build：.deb / tar.gz / desktop entry
just lint            # golangci-lint
just dev             # Docker 开发镜像
```

## 桌面端（推荐）

安装后直接打开 Aurora：首次运行在用户数据目录创建 `aurora.db`，日志写在同目录
`logs/`；数据、备份和导出都能从 File 菜单完成。

```bash
aurora-desktop                                  # 打开窗口
aurora-desktop --version                        # 版本与构建时间
aurora-desktop --data-dir /path/to/data         # 换数据库位置
aurora-desktop --web-root /path/to/web          # 开发 UI：改用磁盘页面
aurora-desktop 'aurora://lottery?id=<draw>'     # 深链：直接验该期开奖
aurora-desktop aurora-lotteries.json            # 导入一份导出文件
```

- **File 菜单**：Back Up Database…（WAL 完整快照 + 校验和）、Export/Import
  Lotteries、Open Recent、Show Data Folder。导入/导出与 CLI 共用同一 app 层
  实现，格式、覆盖保护与失败报告完全一致；导出目标若就是数据库本身（含符号
  链接、硬链接与 `-wal`/`-shm`）会被拒绝。
- **多语言**：菜单、对话框、`--help` 跟随系统语言，也可在 Help → Language
  运行时切换（跟随系统 / English / 简体中文 / 繁體中文）。
- **系统集成**：`aurora://` 深链在 macOS 由 bundle 注册，Linux/Windows 每次
  启动注册；第二次启动不会起第二个数据库写者，而是把链接或导出文件交给已运行
  的窗口；导出文件也可直接拖到窗口导入。
- **失败可见**：打包后的应用没有终端，启动失败（`--web-root` 无效、数据目录
  不可用、缺少 WebView 运行时）会弹出原生对话框说明原因。

完整架构、窗口生命周期、打包与签名流程见 [docs/desktop.md](docs/desktop.md)。

## CLI 快速开始

桌面端就是上面同一套功能的图形入口；下面的命令用于脚本、自动化和无图形
环境。

```bash
./aurora lottery create -p "A,B,C,D" -s "seed" -c 3   # VRF 抽奖
./aurora lottery verify <id或高度>                    # 验证开奖与 VRF 证明
./aurora voting candidate add -n "Name" -p "Party" -m "Program"
./aurora voting vote -v <voter-pk> -c <candidate-id> -s <session-id> -k <priv>
./aurora oracle source add -n "btc" -u "https://api.example.com/price" -p "data.price"
./aurora nft mint -n "My NFT" -c "<creator-pub>" -t <token-uri>
./aurora token create -n "MyToken" -s "SYMBOL" --supply 1000000
./aurora lottery tui                                  # 任一模块都有 TUI
```

全部子命令、参数与运维命令（`backup`、`migrate`、`version`）见
[docs/cli.md](docs/cli.md)，或 `aurora <module> --help`。

> **破坏性操作确认门：** `token burn`、`nft burn`、`oracle source delete`、
> `migrate down`、`backup restore`、`lottery reset` 都需要 `-y/--confirm`
> （`lottery reset` 也接受旧写法 `--yes`）。缺少确认标志会以非零退出码拒绝
> 执行，脚本可据此判断。

## REST API + Web

```bash
AURORA_API_KEY="your-strong-key" ./aurora-api   # 默认 0.0.0.0:8080
./aurora-api --host 127.0.0.1 --port 9090        # 或 AURORA_SERVER_HOST/PORT
```

监听地址优先级：**flag > 环境变量 > 配置文件 > 默认**。服务同时运行预言机调度器
与审计 outbox 补偿循环，SIGINT/SIGTERM 优雅停机；`/metrics`、`/metrics/oracle`
暴露请求与抓取健康统计。鉴权用 `X-API-Key`，Web 页面同源自动注入；请求体上限
4 MiB，支持按客户端限流与 CORS 白名单，Web 响应带 nonce 化 CSP。
详见 [docs/api.md](docs/api.md)。

## 功能模块

- **VRF 透明抽奖**：Edwards25519 VRF 抽签，结果上链存证，任何人可复验；支持
  CLI、TUI、Web 与桌面端。
- **透明投票**：Ed25519 签名投票、候选人/选民注册、会期生命周期与实时计票。
- **数据预言机**：通用 HTTP 数据源（method/path/interval）、内置模板、定时
  抓取、链上存证与抓取健康统计。
- **NFT**：铸造、转让、销毁、查询与操作历史，Ed25519 签名授权。
- **FT 代币**：ERC-20 风格 Mint/Transfer/Burn/Approve，余额与授权管理、分页
  历史。

## 配置

TOML，查找顺序：CLI `--config` → `$HOME/aurora.toml` → `./config/aurora.toml`。
完整示例与每一节的语义见 [config/aurora.toml](config/aurora.toml)。

```toml
[server]
host = "0.0.0.0"
port = 8080

[db]
path = "./data/aurora.db"     # SQLite 是唯一后端

[log]
level = "info"
path = "./logs/"

[lottery]
defaultCount = 3              # 未传 -c 时的获奖人数；种子始终由调用方提供
```

## 项目结构

```text
cmd/aurora/     # CLI 入口
cmd/api/        # REST API + Web 服务入口
desktop/        # MyGo 桌面壳（嵌套模块；cmd/aurora-desktop + internal/deskapp）
web/            # Web 前端（HTML + Alpine.js，嵌入二进制）
internal/
├── domain/     # 领域层：blockchain / lottery / voting / nft / token / oracle
├── app/        # 应用层：用例编排
├── infra/      # 基础设施：SQLite、HTTP 客户端、备份、迁移、事件
├── ui/         # 表示层：Bubble Tea TUI
├── api/        # 路由、handler、中间件
├── i18n/       # 国际化
└── utils/      # 工具（含原子写文件）
e2e/            # E2E 测试
```

DDD 分层：`domain` 持有实体、领域服务与仓储接口；`app` 编排用例；`infra` 实现
外部依赖；`ui`/`api`/`desktop` 是三种表示层，共用 `app`。

## 技术栈

Go 1.27+、Cobra、Viper、Zerolog、Bubble Tea v2、SQLite
（modernc.org/sqlite，纯 Go）、filippo.io/edwards25519、MyGo、Alpine.js。

## 开发

```bash
just check      # gofmt + goimports + go vet
just test       # go test ./...
just lint       # golangci-lint（errcheck 关闭）
just desktop-check
just desktop-race # 可选：race detector 需要 cgo
go test ./... -cover

just dev        # 构建并启动 CLI 工具箱容器
just start      # docker compose up -d
just stop       # docker compose down
```

贡献约定见 [AGENTS.md](AGENTS.md)：直接在 `main` 开发、禁止
`git commit --amend`、提交粒度以“一个自洽的逻辑单元”为单位、过程记录
（`.planning/ril/graph.json`）批量提交。

Docker 镜像只包含 CLI，没有 HTTP 监听器；Web UI 由独立的 `aurora-api` 提供，
因此 compose/k8s 清单没有端口映射——用
`docker compose run --rm aurora <命令>` 执行一次性任务。

## 测试覆盖率

`go test ./... -cover -count=1` 实测（2026-10-23）：

| 模块    | Domain | App   |
| ------- | ------ | ----- |
| Lottery | 90.6%  | 50.8% |
| Voting  | 94.1%  | 82.2% |
| NFT     | 88.2%  | 88.5% |
| Token   | 90.1%  | 91.8% |
| Oracle  | 88.7%  | 87.4% |
| SQLite  | -      | 80.4% |

`internal/app/lottery` 偏低是因为 `verify.go` 的 `VerifyInput`/`VerifyRecord`
目前只被 CLI 与桌面端的集成路径覆盖；这是已知测试缺口，不虚构数字。

## 文档

- [docs/desktop.md](docs/desktop.md)：桌面端架构、菜单、深链、打包与签名
- [docs/cli.md](docs/cli.md)：完整 CLI / TUI 与运维命令参考
- [docs/api.md](docs/api.md)：REST API、鉴权、限流、可观测性
- [docs/ablation.md](docs/ablation.md)：纯 Go 构建边界与模块消融结论
- [docs/lottery/](docs/lottery/)：抽奖模块的 API、测试与使用说明
- [CHANGELOG.md](CHANGELOG.md)：逐版本变更
- [SECURITY.md](SECURITY.md)：漏洞报告流程

## 贡献

欢迎提交 Issue 和 PR！提交前请跑 `just check && just test`（改动桌面端时再加
`just desktop-check`；需要 race 检测时另跑 `just desktop-race`）。

## 许可证

MIT
