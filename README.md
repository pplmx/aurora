# Aurora

<div align="center">

**基于区块链的数字系统套件**

VRF 透明抽奖 · Ed25519 签名投票 · 数据预言机 · NFT · FT 代币

[![Awesome MyGo](https://awesomemygo.dev/badge.svg)](https://awesomemygo.dev/apps/aurora)
[![Go Report Card](https://goreportcard.com/badge/github.com/pplmx/aurora?style=flat-square)](https://goreportcard.com/report/github.com/pplmx/aurora)
[![CI](https://github.com/pplmx/aurora/actions/workflows/ci.yml/badge.svg)](https://github.com/pplmx/aurora/actions)
[![Release](https://img.shields.io/github/v/release/pplmx/aurora)](https://github.com/pplmx/aurora/releases)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

</div>

核心用 Go 实现，按 DDD 分层；同一份后端同时提供**桌面应用**、CLI 和 REST API。
五个业务模块共用一套领域模型、一个 SQLite 数据库和一条区块链审计账本。

## 目录

- [三个入口](#三个入口)
- [安装](#安装)
- [快速开始](#快速开始)
- [桌面应用](#桌面应用)
- [CLI](#cli)
- [REST API](#rest-api)
- [功能模块](#功能模块)
- [配置](#配置)
- [项目结构](#项目结构)
- [技术栈](#技术栈)
- [开发](#开发)
- [测试覆盖率](#测试覆盖率)
- [文档](#文档)
- [贡献](#贡献)
- [许可证](#许可证)

## 三个入口

| 入口         | 二进制           | 适合谁                                       |
| ------------ | ---------------- | -------------------------------------------- |
| **桌面应用** | `aurora-desktop` | 主要使用方式：双击即用，无需终端或浏览器     |
| **CLI**      | `aurora`         | 脚本、批量任务、服务器与无图形环境           |
| **REST API** | `aurora-api`     | 集成方与需要监听的部署；只提供 JSON 接口     |

桌面端用 MyGo 原生控件绘制界面，API 提供同一套应用层能力；两者共用同一份
SQLite 数据库，因此窗口里创建的记录 CLI 立刻能读到。设计细节见
[docs/desktop.md](docs/desktop.md)。

## 安装

### 下载预编译包

从 [Releases](https://github.com/pplmx/aurora/releases) 下载：

| 平台        | 桌面端                                              | CLI                          |
| ----------- | --------------------------------------------------- | ---------------------------- |
| **Linux**   | `.deb` / `.tar.gz` + `install.sh`（amd64、arm64）   | `aurora-linux-<arch>`        |
| **macOS**   | `.dmg` + `Aurora.app`（arm64、amd64）               | `aurora-darwin-<arch>`       |
| **Windows** | `Aurora.exe` / `*Setup*.exe`（amd64）               | `aurora-windows-<arch>.exe`  |

CLI 覆盖 linux/darwin/windows 的 amd64 与 arm64；桌面端发布 linux amd64+arm64、
macOS arm64+amd64、Windows amd64。

发布时会附 `checksums.txt`，并校验版本号与构建时间——不会发布 `0.0.1` /
`Build Time: unknown` 的产物。

**构建来源证明**：发布前为最终资产生成 GitHub artifact attestation，下载后可验证：

```bash
gh attestation verify <artifact> --repo pplmx/aurora
```

它绑定仓库、commit、工作流和文件摘要，能发现校验和文件本身被整体替换；但它
不改变 Gatekeeper / SmartScreen 的发布者判断，不能替代操作系统级签名。

**签名状态**：Windows 包正在申请 SignPath Foundation 的免费开源签名；macOS 包
需要维护者持有的 Apple Developer Program 身份。发布门槛、角色与验收方式见
[代码签名策略](docs/code-signing-policy.md)，本机数据与网络边界见
[隐私说明](docs/privacy.md)。

### 从源码构建

需要 Go 1.27+；桌面端另需 WebKitGTK（Linux）或 WebView2（Windows）。

```bash
CGO_ENABLED=0 go build -o aurora ./cmd/aurora      # CLI
CGO_ENABLED=0 go build -o aurora-api ./cmd/api     # REST API
just desktop                                       # 桌面壳
```

`just` 是唯一的任务入口（没有 Makefile），直接运行 `just` 会列出全部配方。

## 快速开始

```bash
# 桌面端：双击打开，或
./aurora-desktop

# CLI：一次 VRF 抽奖
./aurora lottery create -p "A,B,C,D" -s "seed" -c 3
./aurora lottery verify <id或高度>          # 任何人可复验

# 校验本地区块链完整性
./aurora blockchain verify
```

## 桌面应用

安装后直接打开 Aurora：首次运行在用户数据目录创建 `aurora.db`，日志写在同目录
`logs/`；数据、备份和导出都能从 File 菜单完成。

```bash
aurora-desktop                                  # 打开窗口
aurora-desktop --version                        # 版本、构建时间、Go 与 MyGo
aurora-desktop --data-dir /path/to/data         # 换数据库位置
aurora-desktop 'aurora://lottery?id=<draw>'     # 深链：直接验该期开奖
aurora-desktop aurora-lotteries.json            # 导入一份导出文件
```

### 主要特性

- **File 菜单**：Back Up Database…（WAL 完整快照 + 校验和）、Export/Import
  Lotteries、Open Recent。导入/导出与 CLI 共用同一 app 层实现，格式、覆盖保护
  与失败报告完全一致；导出目标若就是数据库本身（含符号链接、硬链接与
  `-wal`/`-shm`/`-journal`）会被拒绝。
- **Settings**：侧栏 Application 区（或 `Ctrl/Cmd+,`）打开，集中语言、会话密钥、
  数据位置/占用/备份、实时更新、外观和版本信息；数据库路径可复制、占用随卡片
  绘制实时统计，实时更新选择会保存到数据库同目录。
- **多语言**：菜单、页面、对话框、`--help` 跟随系统语言，也可在
  Settings → Language 运行时切换（跟随系统 / English / 简体中文 / 繁體中文）。
- **会话密钥**：Settings 里可选开启内存密钥记忆（默认关闭），成功的签名操作
  可在同公钥的后续表单用 "Use saved key"；不写磁盘，退出或点 Forget 即清空。
- **系统集成**：`aurora://` 深链在 macOS 由 bundle 注册，Linux/Windows 每次
  启动注册；第二次启动不会起第二个数据库写者，而是把链接或导出文件交给已运行
  的窗口；导出文件也可直接拖到窗口导入。
- **原生界面**：窗口用 MyGo 自己的控件绘制（`desktop/internal/nativeui`），
  不是网页。没有 HTML、没有 JavaScript、没有 CSP，界面和业务代码一样受类型检查，
  也能在没有窗口的情况下测试。
- **键盘**：Ctrl/Cmd+K 打开命令面板（跳页、刷新、备份、导出/导入、复制诊断），
  Ctrl/Cmd+1…8 切页，Ctrl/Cmd+=/−/0 缩放内容。
- **诊断**：About 卡片与命令面板共用同一份可复制诊断（版本、构建时间、
  Go/MyGo 工具链、库路径与占用、账本事实、outbox 待投递与最久等待）。
- **失败可见**：打包后的应用没有终端，启动失败（数据目录不可用、缺少图形运行时）
  会弹出原生对话框说明原因。

完整架构、窗口生命周期、打包与签名流程见 [docs/desktop.md](docs/desktop.md)。

## CLI

桌面端就是同一套功能的图形入口；下面的命令用于脚本、自动化和无图形环境。

```bash
# 抽奖
./aurora lottery create -p "A,B,C,D" -s "seed" -c 3   # VRF 抽奖
./aurora lottery verify <id或高度>                    # 验证开奖与 VRF 证明
./aurora blockchain verify                             # 校验本地区块链完整性
./aurora blockchain blocks -s transfer                 # 审计记录（可过滤）
./aurora doctor                                        # 配置/库/迁移/账本/outbox 自检

# 投票
./aurora voting candidate add -n "Name" -p "Party" -m "Program"
./aurora voting vote -v <voter-pk> -c <candidate-id> -s <session-id> -k <priv>

# 预言机
./aurora oracle source add -n "btc" -u "https://api.example.com/price" -p "data.price"
./aurora oracle source add -n "private" -u <url> --headers '{"Authorization":"Bearer ..."}'

# NFT 与代币
./aurora nft mint -n "My NFT" -c "<creator-pub>" -t <token-uri>
./aurora token create -n "MyToken" -s "SYMBOL" --supply 1000000
```

全部子命令、参数与运维命令（`backup`、`migrate`、`version`）见
[docs/cli.md](docs/cli.md)，或 `aurora <module> --help`。

> **破坏性操作确认门**
>
> `token burn`、`nft burn`、`oracle source delete`、`migrate down`、
> `migrate repair`、`backup restore`、`lottery reset` 都需要 `-y/--confirm`。
> 缺少确认标志会以非零退出码拒绝执行，脚本可据此判断。完整清单与错误/退出码
> 约定见 [docs/cli.md](docs/cli.md#确认门)。

## REST API

```bash
AURORA_API_KEY="replace-with-at-least-32-random-characters" ./aurora-api
./aurora-api --host 127.0.0.1 --port 9090        # 或 AURORA_SERVER_HOST/PORT
```

监听地址优先级：**flag > 环境变量 > 配置文件 > 默认**。服务同时运行预言机调度器
与审计 outbox 补偿循环，SIGINT/SIGTERM 优雅停机；`/metrics`、`/metrics/oracle`
暴露请求与抓取健康统计。

网关只提供 `/api/v1`、健康检查与指标接口，**没有静态页面**：界面只有桌面端的
原生窗口。鉴权用 `X-API-Key`，请求体上限 4 MiB，支持按客户端限流与 CORS 白名单。
详见 [docs/api.md](docs/api.md)。

## 功能模块

| 模块           | 能力                                                                                  |
| -------------- | ------------------------------------------------------------------------------------- |
| **VRF 抽奖**   | Edwards25519 VRF 抽签，结果上链存证，任何人可复验                                     |
| **透明投票**   | Ed25519 签名投票、候选人/选民注册、会期生命周期与实时计票；每个会期每位选民只能投一票 |
| **数据预言机** | 通用 HTTP 数据源（method/path/interval）、内置模板、定时抓取与链上存证                |
| **NFT**        | 铸造、转让、销毁、查询与操作历史，Ed25519 签名授权                                    |
| **FT 代币**    | ERC-20 风格 Mint/Transfer/Burn/Approve，余额与授权管理、分页历史                      |

每个模块的操作都从 CLI、REST API 与桌面端三处可达，共用同一套应用层用例。

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
cmd/api/        # REST API 服务入口
desktop/        # MyGo 桌面壳（嵌套模块；cmd/aurora-desktop + internal/deskapp）
internal/
├── ledger/     # 账本端口：BlockWriter、事务追加助手（无存储实现）
├── domain/     # 领域层：lottery / voting / nft / token / oracle / events
├── app/        # 应用层：用例编排（<module>app 包）与 container 组合根
├── infra/      # 基础设施：SQLite、账本实现、HTTP 客户端、备份、迁移、事件
├── api/        # 路由、handler、中间件
├── config/     # 配置加载、默认值与校验
├── i18n/       # 国际化（英文、简体、繁体；繁体由 tools/genzhhant 生成）
├── logger/     # zerolog 装配（级别、文件、权限）
├── metrics/    # 请求指标注册表与 Prometheus 文本
├── netguard/   # 出站地址策略（SSRF 守卫，纯函数）：阻断段、主机解析、拨号地址
├── records/    # 账本载荷解码：模块 / 动作 / 对象 / 事实（三端共用）
├── background/ # 后台工作：停止函数等在途工作真正退出后才返回
├── repocheck/  # 仓库级契约测试（格式、工具链、纯 Go 边界、文档一致性）
└── utils/      # 工具（含原子写文件）
e2e/            # E2E 测试
tools/          # 仓库工具：genzhhant（繁体表）、coverfloor（覆盖率底线）
```

**DDD 分层**：`domain` 持有实体、领域服务与仓储接口，不依赖任何其他层；`app`
编排用例；`infra` 实现外部依赖；`api`（REST）与 `desktop`（原生 Go 窗口）是两种
表示层，共用 `app`。分层契约由 `internal/repocheck` 用测试锁定。

## 技术栈

| 类别     | 选型                                                                       |
| -------- | -------------------------------------------------------------------------- |
| 语言     | Go 1.27+                                                                   |
| CLI      | Cobra + pflag                                                              |
| 配置     | Viper（TOML）                                                              |
| 日志     | Zerolog                                                                    |
| 存储     | SQLite（`modernc.org/sqlite`，纯 Go，无 cgo）                              |
| 密码学   | `filippo.io/edwards25519`（Ed25519 / VRF）                                 |
| 桌面     | [MyGo](https://github.com/egoist/mygo) 原生控件                            |
| 可观测性 | Prometheus 文本格式指标                                                    |

## 开发

```bash
just check          # gofmt + goimports + go vet（会改写未格式化的文件）
just check-verify   # 只检查不写回，供 just verify 使用
just test           # go test ./...
just lint           # golangci-lint（不含 errcheck；含 bodyclose、errorlint 等）
just race           # 可选：根模块 race detector（需要 cgo）
go test ./... -cover
```

### 全部配方

```bash
just dep              # 下载依赖
just verify           # 本地全部门槛：check/test/lint/覆盖率底线 + 桌面门槛
just test-shuffle     # 随机顺序跑测试，暴露顺序依赖
just test-coverage    # 覆盖率报告
just build            # check + test + 全平台 CLI
just build-current    # 只构建当前平台
just run              # 构建当前平台并跑一次示例抽奖
just api              # ./aurora-api，注入 git 版本与构建时间
just coverage-floor         # Domain/App 覆盖率底线（80%）
just desktop          # ./aurora-desktop（本地开发构建）
just desktop-run      # 构建并打开窗口
just desktop-test     # 桌面模块测试（根目录 just test 不进嵌套模块）
just desktop-test-shuffle # 桌面测试随机顺序，暴露顺序依赖
just desktop-check    # gofmt + build + vet + lint + 纯 Go 测试（无需 C 工具链）
just desktop-race     # 可选 race 检测（显式启用 cgo）
just desktop-race-window # race 检测仅跑窗口包，跳过本地截图草稿
just desktop-coverage-floor # 桌面 deskapp/nativeui 覆盖率底线（70%）
just desktop-shot     # 真实窗口逐页截图（MyGo automation，无显示器时起 Xvfb）
just desktop-verbose [包] [测试名]  # 单包 verbose 跑一次，输出留档便于定位失败测试
just desktop-stress [次数] [包]     # 单包重复 N 次，偶发失败也会带出测试名
just desktop-package  # mygo build：.deb / tar.gz / desktop entry
just desktop-deps     # 检查 MyGo 是否有新版本
just desktop-update-mygo # 升级 MyGo 依赖
just gen-zh-hant <OpenCC 词典目录> # 从简体表重新生成繁体表
just image            # 只构建 Docker 镜像
just clean            # 清理构建产物
```

### Docker

镜像只包含 CLI，没有 HTTP 监听器；`aurora-api` 只提供 REST API（没有静态页面），
因此 compose/k8s 清单没有端口映射——用 `docker compose run --rm aurora <命令>`
执行一次性任务。

```bash
just image
docker compose run --rm aurora lottery create -p "A,B,C" -s "seed" -c 3
docker compose run --rm aurora lottery history
```

贡献约定（门槛、分层约束、提交粒度）见 [AGENTS.md](AGENTS.md)。

## 测试覆盖率

`CGO_ENABLED=0 go test ./... -cover -count=1` 实测（2026-11-07）：

| 模块    | Domain | App   |
| ------- | ------ | ----- |
| Lottery | 94.2%  | 89.3% |
| Voting  | 93.1%  | 84.1% |
| NFT     | 80.6%  | 81.6% |
| Token   | 91.1%  | 91.2% |
| Oracle  | 84.8%  | 91.1% |
| SQLite  | -      | 80.8% |
| 组合根  | -      | 87.1% |

所有模块都在 80% 门槛之上，`just coverage-floor`（Domain/App）与
`just desktop-coverage-floor`（desktop 70%）在本地和 CI 逐包校验这两条底线。

数字在连续多次运行中保持一致：此前抽奖模块在 89.8%–90.8% 之间漂移，原因是拒绝
路径的测试用随机密钥对生成输入，走到哪个校验分支取决于随机结果；改用固定字节后
不再漂移。抽奖的解析顺序（精确 ID → 区块高度 → 子串）由
`verify_resolution_test.go` 直接锁定；`nft_cascade_test.go` 用一个旧 schema
数据库验证审计表重建不丢行。

SQLite 剩余缺口集中在仓储构造函数的错误分支（目录不可建、建表失败、权限收紧
失败）；正常路径与事务委托都有直接测试。

## 文档

| 文档                                                       | 内容                                          |
| ---------------------------------------------------------- | --------------------------------------------- |
| [docs/architecture.md](docs/architecture.md)               | 分层依赖契约、边界问题与重构顺序              |
| [docs/desktop.md](docs/desktop.md)                         | 桌面端架构、菜单、深链、打包与签名            |
| [docs/cli.md](docs/cli.md)                                 | 完整 CLI 与运维命令参考                       |
| [docs/api.md](docs/api.md)                                 | REST API、鉴权、限流、可观测性                |
| [docs/ablation.md](docs/ablation.md)                       | 纯 Go 构建边界与模块消融结论                  |
| [docs/code-signing-policy.md](docs/code-signing-policy.md) | 发布签名门槛、角色与验收方式                  |
| [docs/privacy.md](docs/privacy.md)                         | 本机数据与网络边界                            |
| [CHANGELOG.md](CHANGELOG.md)                               | 逐版本变更                                    |
| [SECURITY.md](SECURITY.md)                                 | 漏洞报告流程                                  |
| [docs/superpowers/lottery/](docs/superpowers/lottery/)     | 抽奖模块的早期设计记录（保持原样，非现行指令）|

## 贡献

欢迎提交 Issue 和 PR！提交前请跑：

```bash
just check && just test          # 基础门槛
just desktop-check               # 改动桌面端时
just desktop-race                # 需要 race 检测时（cgo）
```

## 许可证

[MIT](LICENSE)
