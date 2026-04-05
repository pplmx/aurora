# Aurora DDD 架构重构设计

## 概述

将 Aurora 项目从单体模块重构为完整 DDD（领域驱动设计）架构。

## 当前状态

### 问题
- 每个模块（lottery/voting/nft/oracle）内部混合了实体、存储、业务逻辑
- 难以单独测试业务逻辑
- 代码重复（验证逻辑、存储模式）
- 添加新存储实现需要修改整个模块

### 目标
- 清晰的层分离（Domain / Application / Infrastructure / UI）
- 可测试的业务逻辑
- 可替换的存储实现
- 符合 Go + DDD 业界最佳实践

## 架构设计

### 目录结构

```
internal/
├── domain/           # 领域层（核心，无外部依赖）
│   ├── lottery/      # 抽奖限界上下文
│   │   ├── entity.go    # LotteryRecord, Winner
│   │   ├── service.go   # 抽奖逻辑（VRF）
│   │   └── repo.go      # 仓储接口
│   │
│   ├── voting/       # 投票限界上下文
│   │   ├── entity.go    # Vote, Voter, Candidate, Session
│   │   ├── service.go   # 投票、计票逻辑
│   │   └── repo.go      # 仓储接口
│   │
│   ├── nft/          # NFT 限界上下文
│   │   ├── entity.go    # NFT, Operation
│   │   ├── service.go   # NFT 转移逻辑
│   │   └── repo.go      # 仓储接口
│   │
│   ├── oracle/       # 预言机限界上下文
│   │   ├── entity.go    # OracleData, DataSource
│   │   ├── service.go   # 数据获取逻辑
│   │   └── repo.go      # 仓储接口
│   │
│   └── blockchain/   # 共享内核
│       ├── block.go     # Block 实体
│       └── repo.go      # 区块仓储接口
│
├── infra/            # 基础设施层（实现接口）
│   ├── sqlite/
│   │   ├── lottery.go
│   │   ├── voting.go
│   │   ├── nft.go
│   │   ├── oracle.go
│   │   └── blockchain.go
│   │
│   ├── crypto/
│   │   └── ed25519.go   # Ed25519 签名实现
│   │
│   └── http/
│       └── fetcher.go   # HTTP 数据获取
│
├── app/              # 应用层（用例）
│   ├── lottery/
│   │   ├── create.go    # 创建抽奖用例
│   │   ├── draw.go      # 抽奖用例
│   │   └── dto.go       # 数据传输对象
│   │
│   ├── voting/
│   │   ├── register.go
│   │   ├── vote.go
│   │   └── dto.go
│   │
│   ├── nft/
│   │   ├── mint.go
│   │   ├── transfer.go
│   │   ├── burn.go
│   │   └── dto.go
│   │
│   └── oracle/
│       ├── fetch.go
│       └── dto.go
│
└── ui/               # 表示层
    ├── lottery/
    │   └── tui.go
    ├── voting/
    │   └── tui.go
    ├── nft/
    │   └── tui.go
    └── oracle/
        └── tui.go
```

### 层依赖关系

```
cmd/aurora/cmd (CLI 入口)
        │
        ▼
    app/ (用例) ◄──┐
        │          │
        ▼          │
domain/service ◄──┤ 依赖抽象接口
domain/entity ◄──┤ 无外部依赖
domain/repo ◄────┤ 接口定义
        │        │
        ▼        │
infra/ (实现) ◄──┘
        │
        ▼
ui/tui (展示) ◄── app
```

### 各层职责

| 层 | 职责 | 规则 |
|---|---|---|
| **domain/entity** | 业务实体、状态 | 无外部依赖，只包含业务规则 |
| **domain/service** | 核心业务算法 | 包含 VRF、签名验证、计票等 |
| **domain/repo** | 仓储接口（抽象） | 只定义接口，不实现 |
| **app** | 用例、流程编排 | 协调领域对象，完成用例 |
| **infra** | 存储、HTTP、加密实现 | 实现 domain 定义的接口 |
| **ui** | TUI 界面 | 调用 app 层 |
| **cmd** | CLI 命令 | 入口，组装依赖 |

## 实施计划

### 阶段 1: 创建目录结构

1. 创建 `domain/{lottery,voting,nft,oracle,blockchain}/` 目录
2. 创建 `infra/{sqlite,crypto,http}/` 目录
3. 创建 `app/{lottery,voting,nft,oracle}/` 目录

### 阶段 2: 迁移 lottery 模块（示范）

1. 从现有 `lottery/lottery.go` 提取 entity 到 `domain/lottery/entity.go`
2. 创建 `domain/lottery/service.go` 包含抽奖逻辑
3. 创建 `domain/lottery/repo.go` 定义仓储接口
4. 创建 `infra/sqlite/lottery.go` 实现仓储
5. 创建 `app/lottery/` 用例
6. 移动 `lottery/tui.go` 到 `ui/lottery/`
7. 更新 `cmd/lottery.go` 使用新的 app 层

### 阶段 3: 迁移 voting 模块

同上模式

### 阶段 4: 迁移 nft 模块

同上模式

### 阶段 5: 迁移 oracle 模块

同上模式

### 阶段 6: 清理

1. 删除旧的混合文件
2. 验证编译和测试
3. 更新 import 路径

## 命名规范

- 文件名使用简短命名：`entity.go`, `service.go`, `repo.go`, `dto.go`
- 同一限界上下文内，import 路径清晰
- 接口命名：`Repository`, `Service`（实现加前缀如 `SQLiteLotteryRepo`）

## 向后兼容

- 保持 CLI 命令不变
- 保持数据格式不变
- 渐进式重构，不破坏现有功能

## 验收标准

- [ ] 所有模块重构完成
- [ ] `go build ./...` 编译通过
- [ ] `go test ./...` 测试通过
- [ ] `just lint` 无错误
- [ ] 功能与重构前一致