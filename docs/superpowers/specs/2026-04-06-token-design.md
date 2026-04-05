# Token (FT) 系统设计文档

## 概述

基于 Event Sourcing 架构的同质化代币系统，支持铸造、转账、授权、销毁功能。

## 核心理念

```
Event Sourcing: 所有状态变化 → 不可变 Event → 聚合出 State
TransferEvent ──▶ TransferHistory ──▶ Balance(聚合)
```

## 数据模型

### Value Objects

| 类型 | 说明 |
|------|------|
| TokenID | 代币唯一标识 |
| PublicKey | Ed25519 公钥 |
| Signature | Ed25519 签名 |
| Amount | *big.Int 任意精度 |

### Entities

```go
type Token struct {
    id          TokenID
    name        string
    symbol      string
    totalSupply Amount
    decimals    int8        // = 8
    owner       PublicKey
    isMintable  bool
    isBurnable  bool
    createdAt   time.Time
}

type Approval struct {
    tokenID  TokenID
    owner    PublicKey
    spender  PublicKey
    amount   Amount
    expiresAt time.Time
}
```

### Events (不可变)

```go
type TransferEvent struct {
    id          string
    tokenID     TokenID
    from        PublicKey
    to          PublicKey
    amount      Amount
    nonce       uint64       // 防重放
    signature   Signature
    blockHeight int64
    timestamp   time.Time
}

type MintEvent struct {
    id          string
    tokenID     TokenID
    to          PublicKey
    amount      Amount
    blockHeight int64
    timestamp   time.Time
}

type BurnEvent struct {
    id          string
    tokenID     TokenID
    from        PublicKey
    amount      Amount
    blockHeight int64
    timestamp   time.Time
}

type ApproveEvent struct {
    id          string
    tokenID     TokenID
    owner       PublicKey
    spender     PublicKey
    amount      Amount
    expiresAt   time.Time
    timestamp   time.Time
}
```

## 存储表

```sql
CREATE TABLE tokens (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    symbol TEXT NOT NULL,
    total_supply TEXT NOT NULL,
    decimals INTEGER DEFAULT 8,
    owner TEXT NOT NULL,
    is_mintable BOOLEAN DEFAULT TRUE,
    is_burnable BOOLEAN DEFAULT TRUE,
    created_at INTEGER
);

CREATE TABLE transfer_events (
    id TEXT PRIMARY KEY,
    token_id TEXT NOT NULL,
    from_owner TEXT NOT NULL,
    to_owner TEXT NOT NULL,
    amount TEXT NOT NULL,
    nonce INTEGER NOT NULL,
    signature TEXT,
    block_height INTEGER,
    timestamp INTEGER
);

CREATE TABLE mint_events (
    id TEXT PRIMARY KEY,
    token_id TEXT NOT NULL,
    to_owner TEXT NOT NULL,
    amount TEXT NOT NULL,
    block_height INTEGER,
    timestamp INTEGER
);

CREATE TABLE burn_events (
    id TEXT PRIMARY KEY,
    token_id TEXT NOT NULL,
    from_owner TEXT NOT NULL,
    amount TEXT NOT NULL,
    block_height INTEGER,
    timestamp INTEGER
);

CREATE TABLE allowances (
    id TEXT PRIMARY KEY,
    token_id TEXT NOT NULL,
    owner TEXT NOT NULL,
    spender TEXT NOT NULL,
    amount TEXT NOT NULL,
    expires_at INTEGER,
    updated_at INTEGER,
    UNIQUE(token_id, owner, spender)
);
```

## 错误定义

```go
var (
    ErrTokenNotFound       = errors.New("token not found")
    ErrInsufficientBalance = errors.New("insufficient balance")
    ErrInsufficientAllowance = errors.New("insufficient allowance")
    ErrInvalidSignature   = errors.New("invalid signature")
    ErrNonceTooLow        = errors.New("nonce too low")
    ErrAmountMustBePositive = errors.New("amount must be positive")
    ErrNotTokenOwner      = errors.New("not token owner")
    ErrTokenNotMintable   = errors.New("token not mintable")
    ErrTokenNotBurnable   = errors.New("token not burnable")
    ErrUnauthorized       = errors.New("unauthorized")
    ErrTransferToZero     = errors.New("cannot transfer to zero address")
)
```

## CLI 命令

```bash
# 发行
aurora token create -n "Aurora" -s "AUR" -t 1000000 --mintable --burnable

# 铸造
aurora token mint -i aurora -t <pubkey> -a 1000 -k <privkey>

# 转账
aurora token transfer -i aurora -f <from> -t <to> -a 100 -k <privkey>

# 授权
aurora token approve -i aurora -s <spender> -a 500 -k <privkey>

# 代理转账
aurora token transfer-from -i aurora -o <owner> -t <to> -a 50 -k <privkey>

# 销毁
aurora token burn -i aurora -a 100 -k <privkey>

# 查询
aurora token info -i aurora
aurora token balance -i aurora -o <pubkey>
aurora token allowance -i aurora -o <owner> -s <spender>
aurora token history -i aurora -o <pubkey>

# TUI
aurora token tui
```

## 架构分层

```
cmd/token.go              # CLI
app/token/                # Use Cases
domain/token/             # Entity, Event, Service
infra/sqlite/             # Repository
ui/token/                 # TUI
```

## 实现清单

- [ ] domain/token/entity.go
- [ ] domain/token/event.go
- [ ] domain/token/service.go
- [ ] domain/token/errors.go
- [ ] domain/token/validator.go
- [ ] infra/sqlite/token.go
- [ ] infra/sqlite/event_store.go
- [ ] infra/sqlite/allowance.go
- [ ] app/token/create.go
- [ ] app/token/mint.go
- [ ] app/token/transfer.go
- [ ] app/token/approve.go
- [ ] app/token/burn.go
- [ ] app/token/dto.go
- [ ] cmd/token.go
- [ ] ui/token/tui.go
- [ ] 单元测试