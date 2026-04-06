# REST API Server Design

## Overview

为 Aurora 项目添加 REST API 服务器，使现有功能（抽奖、投票、NFT、代币、预言机）可通过 HTTP API 访问。

## Goals

- 将 CLI 工具能力暴露为 HTTP API
- 保持 DDD 架构，API 作为新的表示层
- 最小可用的 API 服务，后续可扩展

## Technical Choice

**Framework**: Chi (go-chi/chi)
- 轻量级，纯 Go 实现
- 中间件生态丰富
- 与项目风格一致

## API Specification

### Base URL
```
http://localhost:8080/api/v1
```

### Endpoints

#### Health Check
| Method | Path | Description |
|--------|------|-------------|
| GET | /health | 服务健康检查 |

#### Lottery
| Method | Path | Description |
|--------|------|-------------|
| POST | /lottery/create | 创建抽奖 |
| GET | /lottery/history | 抽奖历史 |
| GET | /lottery/:id | 抽奖详情 |

Request body (POST /lottery/create):
```json
{
  "participants": ["A", "B", "C", "D"],
  "seed": "random-seed",
  "count": 3
}
```

Response:
```json
{
  "id": "uuid",
  "winners": ["A", "B", "C"],
  "seed": "...",
  "created_at": "2026-04-06T10:00:00Z"
}
```

#### Voting
| Method | Path | Description |
|--------|------|-------------|
| POST | /voting/create | 创建投票 |
| POST | /voting/vote | 投票 |
| GET | /voting/:id | 投票结果 |

Request body (POST /voting/create):
```json
{
  "title": "Proposal Title",
  "owner_pubkey": "ed25519-pubkey"
}
```

Request body (POST /voting/vote):
```json
{
  "vote_id": "uuid",
  "voter_pubkey": "ed25519-pubkey",
  "choice": "yes",
  "signature": "ed25519-signature"
}
```

#### NFT
| Method | Path | Description |
|--------|------|-------------|
| POST | /nft/mint | 铸造 NFT |
| POST | /nft/transfer | 转移 NFT |
| GET | /nft/:id | NFT 详情 |
| GET | /nft/list?owner=<pubkey> | NFT 列表 |

Request body (POST /nft/mint):
```json
{
  "name": "My NFT",
  "description": "Description",
  "creator_pubkey": "ed25519-pubkey",
  "signature": "ed25519-signature"
}
```

Request body (POST /nft/transfer):
```json
{
  "nft_id": "uuid",
  "to": "ed25519-pubkey",
  "from": "ed25519-pubkey",
  "signature": "ed25519-signature"
}
```

#### Token
| Method | Path | Description |
|--------|------|-------------|
| POST | /token/create | 创建代币 |
| POST | /token/mint | 铸造代币 |
| POST | /token/transfer | 转账 |
| POST | /token/burn | 销毁 |
| GET | /token/balance?owner=<pubkey> | 余额查询 |
| GET | /token/history | 交易历史 |

Request body (POST /token/create):
```json
{
  "name": "MyToken",
  "symbol": "SYMBOL",
  "supply": 1000000
}
```

Request body (POST /token/transfer):
```json
{
  "from": "ed25519-privkey",
  "to": "ed25519-pubkey",
  "amount": 100
}
```

#### Oracle
| Method | Path | Description |
|--------|------|-------------|
| GET | /oracle/sources | 数据源列表 |
| POST | /oracle/fetch | 获取数据 |
| GET | /oracle/query?source=<id>&limit=10 | 查询历史 |

### Error Response Format

所有错误响应统一格式：
```json
{
  "error": "错误描述",
  "code": "ERROR_CODE"
}
```

常见错误码：
- `INVALID_REQUEST` - 请求参数错误
- `NOT_FOUND` - 资源不存在
- `INTERNAL_ERROR` - 服务器内部错误

## Project Structure

```
cmd/
├── aurora/           # 现有 CLI
└── api/              # 新增 API 服务
    └── main.go

internal/
├── domain/           # 现有（不变）
├── app/              # 现有（不变）
├── infra/            # 现有（不变）
└── api/              # 新增 API 层
    ├── handler/
    │   ├── lottery.go
    │   ├── voting.go
    │   ├── nft.go
    │   ├── token.go
    │   └── oracle.go
    ├── middleware/
    │   ├── logger.go
    │   └── recovery.go
    └── router.go
```

## Configuration

新增配置项（config/aurora.toml）：
```toml
[server]
host = "0.0.0.0"
port = 8080
```

## Implementation Notes

1. API 层复用现有的 domain 和 app 层，不重复业务逻辑
2. 请求参数通过 DTO 转换后调用 app 层用例
3. 响应同样通过 DTO 转换后返回 JSON
4. 日志使用现有 zerolog
5. 初始化阶段不加认证，后续通过 middleware 扩展

## Acceptance Criteria

- [ ] API 服务可正常启动，监听 8080 端口
- [ ] /health 返回 200 OK
- [ ] 各模块 CRUD 接口正常工作
- [ ] 错误响应格式统一
- [ ] 单元测试覆盖新增代码
- [ ] E2E 测试验证 API 功能
