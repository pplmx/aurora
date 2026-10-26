# 架构与依赖规则

这份文档描述 Aurora 当前的模块边界、已经执行的架构调整，以及后续拆分顺序。
它是代码评审时的依赖契约，不是愿景清单：每一条“现状”都能在 `go list` 的
import 边上验证。

## 入口与分层

三个入口共用同一套应用层：

```text
cmd/aurora (CLI) ─┐
cmd/api (REST)  ─┼─> presentation (api / desktop nativeui) ─> app ─> domain
desktop          ─┘                                              ^
                                                                 │
                                            infra (sqlite/http/...) 实现端口
```

规则：

1. `domain` 只依赖标准库和同一层（含 `internal/blockchain` 的纯类型/端口）；
   不读配置文件、不打开数据库、不写文件。
2. `app` 只依赖 `domain` 的接口和纯工具；不得为了一个错误哨兵 import
   `infra`。
3. `infra` 实现 `domain` 声明的仓储/事务/网络端口，可以依赖 `domain`。
4. `api` 和 `desktop` 是表示层；它们可以依赖 `app`，但不得互相依赖，也不得
   成为数据库的第二个所有者。
5. 组合根（打开 SQLite、构造仓储、启动后台循环、关闭句柄）只能有一个明确的
   位置；表示层使用组合根给出的端口，而不是自己再拼一套。
6. 依赖方向不得成环；`internal/repocheck` 和 `go list` 是最后的机械检查。

## 当前依赖事实

`go list` 显示的内部边（只列项目包）：

| 包 | 依赖 | 说明 |
|---|---|---|
| `internal/domain/{lottery,nft,token,voting,oracle,events}` | 仅标准库与同层 | 纯领域模型、服务与仓储接口 |
| `internal/blockchain` | `internal/logger`, `internal/utils` | 共享账本：区块、PoW、SQLite 持久化、进程级单例；已从 `domain/` 移出 |
| `internal/app/*` | `domain`（`container` 例外） | 用例编排；错误哨兵与 HTTP fetcher 都已端口化 |
| `internal/infra/sqlite` | 全部领域包 | 仓储实现 |
| `internal/api` | `app`, `app/container`, `domain`, `infra/sqlite`, `infra/events` | HTTP 表示层；数据库/仓储/用例由 `app/container` 组装 |
| `desktop/internal/deskapp` | `internal/api`, `infra/sqlite`, `app`, `domain` | 桌面适配器；通过 `api.Server.Services()` 取到 `app/container` |
| `desktop/internal/nativeui` | `internal/i18n` | 纯视图状态与 MyGo 控件，不直接打开数据库 |

## 已识别的边界问题

### 1. `internal/domain/blockchain` 把存储引导藏在领域层

原目录同时包含区块/PoW 和 `sql.Open`、`viper`、SQLite 驱动、文件权限收紧、
进程级单例。它让 `domain` 不再是纯领域，也让“不打开数据库就无法使用账本类型”
成为事实。

**已做**：目录迁移到 `internal/blockchain`，文档和 import 路径同步；它被明确
标记为共享账本子系统，而不是领域实体包。

**仍未做**：把 `init.go` 的单例/引导、每个方法的 SQL 拆到仓储接口和
`infra/sqlite`。这是下一步，不应在本次目录重构里顺手改掉写入基底。

### 2. 应用层为了错误哨兵依赖基础设施

`oracleapp/usecase.go` 和 `votingapp/usecase.go` 用
`errors.Is(err, sqlite.ErrNotFound)` / `sqlite.ErrAlreadyVoted` 把仓储结果映射成
领域错误。应用层因此知道 SQLite 的存在。

**已做**：通用 not-found 哨兵移到 `internal/domain/repoerr`；`sqlite.ErrNotFound`
和 `sqlite.ErrAlreadyVoted` 保留为 alias，分别指向 `repoerr.ErrNotFound` 与
`voting.ErrAlreadyVoted`。`oracleapp`/`votingapp` 不再 import `infra/sqlite`。
`oracleapp` 的默认 `http.NewFetcher` 也移出：fetcher 由 container/CLI/API 注入，
应用层只认识 `FetcherInterface`。

### 3. `api.Server` 曾同时是表示层、组合根和服务容器

`internal/api/server.go` 原来打开数据库、构造全部仓储、事件总线、token 服务、
六个 handler，并暴露 `Services()` 给桌面端，字段数量接近“上帝对象”。

**已做**：新增 `internal/app/container`，由它打开/关闭 SQLite、构造仓储、事件
总线、replay protection、token service 和账本单例；`api.Server` 只接收容器、
构造 handler 和 oracle 调度器，`Close`/`Services` 委托给容器。容器是唯一的
存储句柄所有者，桌面仍通过同一份实例驱动窗口。

**下一步**：把 `api.Server` 中剩余的 handler/调度器组装也收进一个 HTTP
模块的函数，让 `Server` 成为纯路由宿主；`Services()` 可进一步改成窄接口，
避免桌面看到具体 SQLite 类型。

### 4. 桌面适配器重复组合并泄漏具体类型

`desktop/internal/deskapp/native_services.go` 直接持有
`*sqlite.LotteryRepository` 等具体类型，并在 `newNativeServices` 里再建
`sqlite.NewTxManager`。这既是重复组合，也是表示层对 infra 的硬依赖。

**已做**：`native_services.go` 只保留结构体、构造和接口断言，按领域拆成
`native_lottery.go`、`native_ledger.go`、`native_voting.go`、`native_keys.go`、
`native_token.go`、`native_oracle.go`、`native_nft.go`。单文件从 779 行降到
80 行，领域映射的修改范围可定位。

构造参数和字段已改为 `lottery.Repository`、`voting.TransactableRepository`、
`nft.TransactableRepository`、`oracle.Repository` 等领域端口；NFT 事务管理器
由 container 注入。`ports_test.go` 禁止 `native_*.go` import infra/api。

**已做**：`Shell` 接收 container 的 `lottery.Repository`，File 菜单的导入/
导出与拖放导入都走这份共享句柄；只有测试或 runtime 建立前的 shell 才回退到
`openLotteryRepo`。生产路径不再为一次文件操作打开第二个 SQLite writer。

**下一步**：删除 fallback 与 `sqlite.NewLotteryRepository` 的菜单依赖，让
notebook 风格的 standalone 测试改用显式注入的仓储。

### 5. 桌面 UI 文案曾放在核心 i18n

`internal/i18n/messages_ui_zh.go` 的 `desktopUIMessages` 是桌面专用文案，
却与 CLI 文案一起编译进核心模块。

**已做**：表移到 `desktop/internal/nativeui/messages_zh.go`，`tr`/`Translate`
在中文 locale 下优先查桌面表，未知 key 回退核心目录再回退英文。核心 i18n
只保留 CLI/服务层文案，并用 `cloneMessages` 隔离 `LoadLocaleFile` 的写入。

### 6. 命名不一致

原 API 的 `BlockChain`、`InitBlockChain`、`GetBlockChain`、`AddLotteryRecord`
把实现细节（“这是 lottery 的链”）写进了共享账本 API；同一概念还有
`blockChain` 局部变量。

**已做**：统一为 `Chain`、`New`、`NewWithDB`、`Init`、`Get`、`AddRecord`、
`WithRecordTransaction`；局部变量统一为 `chain`，测试名同步。编译器、全量
测试和桌面模块确认了所有调用点。

## 目标结构

```text
internal/
├── blockchain/        # 共享账本：纯区块/PoW + 明确的持久化端口（最终）
├── domain/            # 只放实体、领域服务、仓储端口、领域错误
├── app/               # 用例；只依赖 domain
├── app/container/     # 组合根：打开存储、构造仓储/用例、生命周期
├── infra/             # SQLite/HTTP/事件/备份/迁移实现
├── api/               # HTTP 路由、handler、中间件
├── i18n/              # CLI/服务文案；桌面文案由桌面注册（计划）
├── repocheck/         # 仓库级机械契约
└── utils/             # 无业务语义的工具
```

## 执行顺序

- [x] 建立依赖事实表，确定“domain 不碰 infra”的硬规则。
- [x] `internal/domain/blockchain` → `internal/blockchain`，同步 import/文档。
- [x] 领域错误包，移除 `app` → `infra/sqlite` 的错误哨兵依赖。
- [x] 把 oracle fetcher 从 `oracleapp` 默认构造改为注入。
- [x] 拆 `nativeServices` 为 `native_lottery.go` / `native_voting.go` /
      `native_token.go` / `native_nft.go` / `native_oracle.go` /
      `native_ledger.go` / `native_keys.go`，并改用领域仓储端口。
- [x] 让 Shell 的 File 导入/导出复用 container 的 lottery 仓储。
- [x] 统一账本对外命名。
- [x] 抽出 `app/container` 组合根；`api.Server` 已委托存储生命周期。
- [x] 把 `desktopUIMessages` 移出核心 i18n。
- [ ] 评估把账本 SQL 移到 `infra/sqlite`；这一步需要单独的迁移/回归计划。

## 验证

每一阶段至少运行：

```bash
just check && just test
just desktop-check
just desktop-race        # 触及桌面/并发路径时
python3 .agents/skills/graph-engineering/scripts/ril.py check
```

`just check` 会重排 i18n 映射和 import，提交前必须复查 `git status`，只 stage
本阶段改动的文件。`internal/repocheck/layers_test.go` 用 AST 解析生产 import，
把 domain/app/infra 的禁止方向变成测试；`internal/app/container` 之外的应用
包 import 任何 `infra` 都会在 `just test` 失败。
