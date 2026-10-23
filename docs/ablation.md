# Aurora 纯 Go 与模块消融报告

**分析日期：** 2026-10-23  
**基线：** `3c6a700`（`main`）  
**范围：** 生产入口、SQLite 存储、事件系统、TUI，以及五个业务模块的可达性。

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
5. **四个 TUI 都保留，但产品边界分为持久化与沙盒两类。** lottery/oracle TUI
   与 CLI/API 共用 SQLite；NFT/token TUI 明确标记为内存沙盒，退出即丢弃状态。
   需要持久化的 NFT/token 操作应使用 CLI、REST/Web 或桌面端。

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
| lottery    | `aurora lottery ...`、`/api/v1/lottery/*`、`web/lottery.html`、桌面路由、lottery TUI           | 保留；VRF 证明与审计记录是核心功能       |
| voting     | `aurora voting ...`、`/api/v1/voting/*`、`web/voting.html`、桌面路由                           | 保留；没有独立的 voting TUI              |
| NFT        | `aurora nft ...`、`/api/v1/nft/*`、`web/nft.html`、桌面路由、NFT TUI                           | 保留；TUI 的持久化边界见下文             |
| token      | `aurora token ...`、`/api/v1/token/*`、`web/token.html`、桌面路由、token TUI                   | 保留；TUI 的持久化边界见下文             |
| oracle     | `aurora oracle ...`、`/api/v1/oracle/*`、`web/oracle.html`、桌面路由、oracle TUI、调度器       | 保留；定时抓取、健康度与链上存证都有入口 |
| blockchain | 各业务模块的 `AddLotteryRecord`/`AddBlock`、`/api/v1/blockchain/verify`、`web/blockchain.html` | 保留；不是独立产品页面，而是审计基底     |
| events     | CLI/API/桌面端的同步发布、审计 handler、`pending_events` outbox                                | 保留；生产只使用 `SyncEventBus`          |
| migrations | `aurora migrate`、启动配置与仓储自举 schema                                                    | 保留；迁移与自举共同服务新库/旧库        |
| backup     | `aurora backup ...`、桌面 File 菜单                                                            | 保留；用户数据恢复路径                   |

`internal/ui/voting/` 并不存在；旧分析文档中的 voting TUI 描述已过期。

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
- NFT 的 `GetNFTsByCreator`：CLI/API/Web/TUI 均只按 owner 查询。
- `Migrator.DB`：没有调用者；迁移器只暴露 `Version`/`Status`/`Up`/`Down`。
- 上述接口的测试替身、mock 和重复断言同步删除，避免测试继续维护死 API。

## 有意保留的重复

这些不是可以盲目“去重”的死代码：

- **仓储自举 schema + SQL migrations**：新安装可能先打开仓储再执行迁移；
  自举保证每个模块单独可用，迁移负责已有数据库的版本化演进。删除任一侧都会
  破坏已有启动路径。
- **内存仓储与 SQLite 仓储**：内存实现用于 TUI、E2E 和领域测试，SQLite
  实现用于生产持久化；接口相同，生命周期和故障模型不同。
- **`GetByType`/`GetByModule`**：当前主要由事件存储的断言/回归测试使用；
  它们保留事件存储的可观察查询面，不应因为生产 handler 不调用就删除。
- **迁移文件中的历史表**：迁移是历史记录，不能因为当前代码不读某张表就
  改写已发布迁移。

## TUI 产品边界

| TUI     | 仓储                          | 当前行为                  | 决策                                         |
| ------- | ----------------------------- | ------------------------- | -------------------------------------------- |
| lottery | `sqlite.NewLotteryRepository` | 与 CLI/API 共用持久化记录 | 保留；持久化交互入口                         |
| oracle  | `sqlite.NewOracleRepository`  | 数据源与抓取结果持久化    | 保留；持久化交互入口                         |
| NFT     | `nft.NewInmemRepo`            | 退出后状态丢失            | 保留为沙盒；界面必须持续提示不持久化         |
| token   | token TUI 的内存仓储/事件存储 | 退出后状态丢失            | 保留为沙盒；界面必须持续提示不持久化         |

这是有意收窄产品承诺，而不是删除可达代码。NFT/token TUI 适合探索签名、转移和
余额流程；持久化操作仍由 CLI、REST/Web 与桌面端承担。菜单级沙盒提示由
`internal/ui/{nft,token}` 的回归测试锁定，避免将来悄悄把临时会话伪装成持久化。

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
