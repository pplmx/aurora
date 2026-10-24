# Aurora 纯 Go 与模块消融报告

**分析日期：** 2026-10-23  
**基线：** `3c6a700`（`main`）  
**范围：** 生产入口、SQLite 存储、事件系统，以及五个业务模块的可达性。

## 结论

1. **生产代码已经不需要 cgo。** SQLite 使用 `modernc.org/sqlite`，CLI、API、
   Docker、桌面壳和发布打包的默认构建都使用 `CGO_ENABLED=0`。
2. **cgo 只剩 race detector 这一项测试例外。** Go 在 Linux 上的 `-race`
   运行时需要 cgo；这不是生产依赖，也不应被当成默认沙箱门禁。若要求连
   race 检测也不能使用 cgo，只能删除该门禁，而不是继续“纯 Go 重构”。
3. **五个业务模块都不是死代码。** lottery、voting、NFT、token、oracle 都能
   从 CLI/API/Web/桌面端或其中多个入口到达；blockchain 是这些模块的审计
   存证基底。没有证据支持整体删除。
4. **本轮只删除生产不可达的表面。** 异步/组合事件总线、重复的内存 Oracle
   仓储，以及若干没有调用者的领域/仓储查询接口被消融；同步事件总线、审计
   outbox、迁移、备份和全部业务入口保留。
5. **四个 TUI 已在 2026-10-24 的全局消融中删除。** 它们覆盖的操作在 Web 与
   CLI 中都有入口，NFT/token 两个还是退出即丢的内存沙盒；保留三套表示层只增加
   维护面。界面能力由桌面端与 `aurora-api` 的 Web UI 承担。

## cgo 构建矩阵

| 目标          | 入口                                 | 构建模式             | 说明                                       |
| ------------- | ------------------------------------ | -------------------- | ------------------------------------------ |
| 核心测试      | `just test`                          | `CGO_ENABLED=0`      | 默认沙箱门禁                               |
| 格式化与 vet  | `just check`                         | `CGO_ENABLED=0`      | `gofmt`/`goimports`/`go vet`               |
| CLI 当前平台  | `just build-current`                 | `CGO_ENABLED=0`      | 纯 Go 二进制                               |
| CLI 发布矩阵  | `just build`、`build.yml`            | `CGO_ENABLED=0`      | linux/darwin/windows 交叉构建              |
| API + Web     | `just api`                           | `CGO_ENABLED=0`      | 独立 `aurora-api`                          |
| Docker        | `Dockerfile`                         | `CGO_ENABLED=0`      | distroless/static 运行时                   |
| 桌面构建/检查 | `just desktop`、`just desktop-check` | `CGO_ENABLED=0`      | 嵌套 `desktop/` 模块                       |
| 桌面打包      | `just desktop-package`               | `CGO_ENABLED=0`      | `go tool mygo build`                       |
| 发布元数据    | `release.yml`、`desktop-build.yml`   | 要求 `CGO_ENABLED=0` | 检查每个发布可执行文件的 Go build metadata |
| 核心 race     | CI 的 `go test -race`                | `CGO_ENABLED=1`      | **仅测试工具需要 cgo**                     |
| 桌面 race     | `just desktop-race`                  | `CGO_ENABLED=1`      | 显式 opt-in，不进入默认沙箱门禁            |

纯 Go 契约由根模块的 `toolchain_contract_test.go` 和
`desktop/internal/deskapp/ci_test.go` 锁定：若 SQLite 驱动退回
`github.com/mattn/go-sqlite3`、Docker/justfile 退回 cgo，或桌面打包改用 cgo，
测试会失败。

## 模块可达性

| 模块       | 生产入口                                                                                       | 结论                                     |
| ---------- | ---------------------------------------------------------------------------------------------- | ---------------------------------------- |
| lottery    | `aurora lottery ...`、`/api/v1/lottery/*`、`web/lottery.html`、桌面路由 | 保留；VRF 证明与审计记录是核心功能       |
| voting     | `aurora voting ...`、`/api/v1/voting/*`、`web/voting.html`、桌面路由                           | 保留                                     |
| NFT        | `aurora nft ...`、`/api/v1/nft/*`、`web/nft.html`、桌面路由                                   | 保留                                     |
| token      | `aurora token ...`、`/api/v1/token/*`、`web/token.html`、桌面路由                               | 保留                                     |
| oracle     | `aurora oracle ...`、`/api/v1/oracle/*`、`web/oracle.html`、桌面路由、调度器                   | 保留；定时抓取、健康度与链上存证都有入口 |
| blockchain | 各业务模块的 `AddLotteryRecord`/`AddBlock`、`/api/v1/blockchain/verify`、`web/blockchain.html` | 保留；不是独立产品页面，而是审计基底     |
| events     | CLI/API/桌面端的同步发布、审计 handler、`pending_events` outbox                                | 保留；生产只使用 `SyncEventBus`          |
| migrations | `aurora migrate`、启动配置与仓储自举 schema                                                    | 保留；迁移与自举共同服务新库/旧库        |
| backup     | `aurora backup ...`、桌面 File 菜单                                                            | 保留；用户数据恢复路径                   |

## 全局消融：按「真实调用者」重判（2026-10-24）

上一轮消融（2026-10-23）按「生产入口是否可达」判断模块去留。这一轮换了判据：
**每个导出符号是否有真实生产调用者**。可达模块里的死 API 和「有调用者但结果
没人读」的代码，都按同一个标准处理。

### 判定方法

1. 对每个包的导出函数与方法，统计 `internal/`、`cmd/`、`desktop/`、`e2e/`
   中排除 `_test.go` 后的引用数；`refs<=1`（只有定义本身）进入候选。
2. 候选逐个确认调用点，区分三类：
   - **死 API**：生产零调用，测试只在自测它 → 删除。
   - **重复实现**：同一算法有两条路径，生产只走一条 → 合并到保留的那条。
   - **测试基础设施**：生产不用，但真实回归测试用它断言行为 → 保留。
3. 删除后跑 `just check` 与 `just test`，并对保留/删除有争议的项做故障注入。

### 删除的死 API 与重复实现

| 位置 | 符号 | 证据 | 处理 |
| --- | --- | --- | --- |
| `infra/http` | `NewFetcherWithConfig` | 生产只用 `NewFetcher()`；与 `NewFetcherWithTimeout` 三份几乎相同的构造器 | 删除；调用方改用 `NewFetcher(WithRateLimit(...))` |
| `infra/http` | `FetchDataWithValidation` | 无 ctx 的薄封装，生产只走 `FetchDataContext` | 删除；测试改调 Context 版本 |
| `domain/blockchain` | `MineBlockWithContext` | 与 `ProofOfWork.Run` 完整重复，仅多一个 ctx 检查 | 把 ctx 折进 `Run(ctx)`，删除重复函数 |
| `domain/blockchain` | `ValidateProof` | 与 `ProofOfWork.Validate` 同一比较 | 删除；测试改断 `pow.Validate()` |
| `domain/blockchain` | `Serialize` / `Deserialize` | 块从 SQLite 按列重建，生产不读 gob | 删除这一对及其往返测试 |
| `domain/token` | `ValidateNonce` + `ErrNonceTooLow` + `NONCE_TOO_LOW` 映射 | nonce 由 `ClaimNextNonce` 原子分配，该错误从未被返回 | 删除整条链；409 映射随之移除 |
| `domain/token` | `NewMintEventFromData` / `NewBurnEventFromData` | 生产用 `NewMintEvent`/`NewBurnEvent` 创建；无 mint/burn 事件回读器 | 删除（transfer 的 `FromData` 保留，历史回读在用） |
| `domain/lottery` | `VRFOutputToBytes` | `SelectWinners` 自行处理输出，无调用者 | 删除 |
| `domain/lottery` | `GetWinners` | 纯 getter，`Winners` 字段本就导出 | 删除及其自测 |
| `domain/token` | `ApproveEvent.SetExpiresAt` | 无调用者 | 删除 |
| `infra/events` | `StatsHandler`（含 `GetCount`/`GetAllCounts`） | `server.go` 订阅了 `Handle`，但计数结果无人读取 | 删除 handler 与订阅；订阅/退订语义改由审计 handler 与计数闭包断言 |

### 有意保留（看似死代码，实际有真实用途）

- **`Operation.IsBurn` / `IsMint` / `IsTransfer`**：`IsTransfer` 在
  `nft/service.go` 的生产路径使用；三者是同族谓词，`IsBurn` 还被
  `burn_sqlite_regression_test.go` 当作断言工具。保留全部三个以维持对称。
- **`RateLimiter.Remaining` / `Reset`**：生产不调用，但限流器测试用它们断言
  预算与窗口语义；`Remaining` 也是 `WithRateLimit` 的可观测面。保留。
- **`LotteryRecord.FromJSON`**：与 `ToJSON` 构成导入文档的解码契约。
- **`Fetcher.FetchData`**：`app/oracle/usecase.go` 的默认路径在用。
- **`NewMockRepository` / `NewMockEventStore`**：它们定义在 `_test.go` 内，
  是测试替身而非生产 API。

### 结论

六个业务模块全部保留：lottery、voting、NFT、token、oracle 各自有 CLI、REST、
Web 与桌面路由中的多个真实调用点，blockchain 是它们的审计基底。本轮删掉的是
模块**内部**的重复实现与死 API，不改变任何用户可见行为；`just test` 的 34 个
包在删除前后都是绿的。

## 本轮删除的不可达表面

- `internal/infra/events/async_bus.go` 与测试：生产从未构造 `AsyncEventBus`。
- `internal/infra/events/composite_bus.go` 与测试：生产从未构造
  `CompositeEventBus`；集成测试改用真实生产使用的 `SyncEventBus`。
- `EventRepository.GetByAggregateAndType`：没有生产调用者；token 历史使用
  `GetByAggregateAndTypePayloadEither`。
- `domain/events` 中未使用的错误值：只保留实际使用的
  `ErrInvalidPayload`。
- `sqlite.InMemoryOracleRepository` 及其重复测试：保留
  `domain/oracle.InmemRepo` 作为唯一的内存测试实现。
- token 的 `GetMintEventsByToken`、`GetBurnEventsByToken`、
  `GetApprovalsByOwner`、`Token.AddToSupply`：无生产调用者。
- NFT 的 `GetNFTsByCreator`：CLI/API/Web 均只按 owner 查询。
- `Migrator.DB`：没有调用者；迁移器只暴露 `Version`/`Status`/`Up`/`Down`。
- 上述接口的测试替身、mock 和重复断言同步删除，避免测试继续维护死 API。

## 有意保留的重复

这些不是可以盲目“去重”的死代码：

- **仓储自举 schema + SQL migrations**：新安装可能先打开仓储再执行迁移；
  自举保证每个模块单独可用，迁移负责已有数据库的版本化演进。删除任一侧都会
  破坏已有启动路径。
- **内存仓储与 SQLite 仓储**：内存实现用于 E2E 和领域测试，SQLite
  实现用于生产持久化；接口相同，生命周期和故障模型不同。
- **`GetByType`/`GetByModule`**：当前主要由事件存储的断言/回归测试使用；
  它们保留事件存储的可观察查询面，不应因为生产 handler 不调用就删除。
- **迁移文件中的历史表**：迁移是历史记录，不能因为当前代码不读某张表就
  改写已发布迁移。

## TUI 消融（2026-10-24）

四个 Bubble Tea TUI 已整体删除。判据不是"能否到达"，而是"是否是同一能力的
必要表示层"：

| 判据 | 结论 |
| --- | --- |
| 覆盖范围 | lottery/voting/NFT/token/oracle 的每个操作在 Web 页面与 CLI 中都有入口 |
| 独有能力 | 无。lottery/oracle TUI 只是读写同一个 SQLite；NFT/token TUI 是内存沙盒，退出即丢，与 CLI 的持久化语义**相反** |
| 维护成本 | `internal/ui/` 约 4k 行生产代码 + 对应测试，另有 234 行 TUI 专用 i18n 键 |
| 用户影响 | 无图形环境下的交互式入口消失；这些场景本就用 CLI，且 TUI 的沙盒语义会误导用户 |

删除内容：`internal/ui/{components,lottery,nft,oracle,token}`、四个
`<module> tui` 子命令、`charm.land/{bubbles,bubbletea,lipgloss}` 依赖、
TUI 专用 i18n 键，以及只锁定 TUI 存在性的测试
（`TestTUISubcommandsUseRunE`、`TestTokenTUI_Wired`）。界面能力由桌面端与
`aurora-api` 的 Web UI 承担，两者与 CLI 共用同一份 app 层与数据库。

## 暂缓的破坏性清理

`migrations/000002_token_transfers.up.sql` 创建的 `token_transfers` 表目前
没有生产读写者，只有迁移测试断言它存在。删除历史表属于不可逆数据库变更，
可能影响已有数据库或外部工具；本轮不删除。若确认没有数据保留需求，应通过
新的 down/up 迁移和显式批准处理，而不是直接修改旧迁移文件。

## 验证

本轮已执行并通过：

- `CGO_ENABLED=0 go test -count=1 ./...`：通过。
- `CGO_ENABLED=0 go vet ./...`：通过。
- `just build`：格式化、vet、核心测试及 darwin/linux/windows 五个目标交叉
  构建通过。
- `just desktop-check`：桌面模块 gofmt/build/vet/纯 Go 测试通过。
- `just desktop-race` 与根模块 `CGO_ENABLED=1 go test -race -count=1 ./...`：
  通过；仅该门禁需要 cgo。
- 根模块与 `desktop/` 的 `go mod tidy -diff`：无差异。
- `CGO_ENABLED=0 go test ./... -cover -count=1`：通过，用于核对 README
  覆盖率基线。
- `git diff --check`：通过。

`just lint` 未通过，但失败发生在加载配置之前：本机
`golangci-lint 2.11.4` 由 Go 1.26.1 构建，而模块目标为 Go 1.27.2，报错为
`can't load config: the Go language version (go1.26) used to build
golangci-lint is lower than the targeted Go version (1.27.2)`。CI 固定使用
`golangci-lint v2.15.0`；本报告不把本机 lint 记为通过。

故障注入结果：

- 反转 `SyncEventBus` 的全局/类型 handler 顺序后，
  `TestSyncEventBus_Publish_GlobalHandlersFirst` 失败；恢复后通过。
- 让 Oracle `GetData` 对不存在的记录返回错误后，对应“空结果是 nil, nil”回归
  测试失败；恢复后通过。
