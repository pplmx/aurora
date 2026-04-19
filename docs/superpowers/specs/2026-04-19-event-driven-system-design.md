# Event-Driven System Design

Date: 2026-04-19
Status: Approved
Version: 1.0

## Overview

Implement an event-driven architecture across all modules (Token, NFT, Voting, Lottery, Oracle) to enable:
1. Unified event storage and audit trail
2. Decoupled module communication via publish/subscribe
3. Pluggable handlers for audit, statistics, and webhooks

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Application Layer                       │
│  TokenService(repo, eventBus, replay, chain)                │
│  NFTService(repo, eventBus)                                 │
│  VotingService(repo, eventBus)                              │
│  LotteryService(repo, eventBus)                             │
│  OracleService(repo, eventBus)                              │
└─────────────────────────┬───────────────────────────────────┘
                          │ Publish(Event)
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                    CompositeEventBus                        │
│                                                             │
│  SyncEventBus ──► [AuditHandler]                           │
│                  ──► [StatsHandler]  (all-or-nothing)      │
│                  ──► [WebhookHandler]                       │
│                           │                                 │
│  AsyncEventBus ───────────┴── (non-blocking goroutine)     │
│                                                             │
│  PluginBus ──────────────────────────── (extensible)       │
└─────────────────────────────────────────────────────────────┘
```

## File Structure

```
internal/domain/events/
├── event.go         # Event interface + BaseEvent
├── types.go         # Module event types
└── errors.go        # Error definitions

internal/infra/events/
├── bus.go           # EventBus, SyncEventBus, CompositeEventBus
├── async_bus.go     # Channel-based async bus
├── plugin_bus.go    # Pluggable extension interface
├── event_store.go   # SQLite EventStore implementation
├── handlers.go      # AuditHandler, StatsHandler, WebhookHandler
└── replay.go        # SQLite ReplayProtection implementation

internal/infra/sqlite/
└── event_store.go   # Deprecated, to be removed after migration
```

## Core Interfaces

### Event Interface

```go
type Event interface {
    EventType() string      // e.g., "token.transfer"
    Module() string         // e.g., "token", "nft"
    AggregateID() string    // Aggregate root ID
    Timestamp() time.Time
    Payload() []byte        // JSON serialized data
}

type BaseEvent struct {
    id          string
    eventType   string
    module      string
    aggID       string
    timestamp   time.Time
    payload     []byte
}
```

### Payload-First Design

Events are constructed with typed fields for convenience, but `Payload()` is the source of truth:

```go
type TokenTransferEvent struct {
    *BaseEvent
}

func NewTokenTransferEvent(...) *TokenTransferEvent {
    payload := map[string]interface{}{
        "from":   base64.StdEncoding.EncodeToString(from),
        "to":     base64.StdEncoding.EncodeToString(to),
        "amount": amount.String(),
        "nonce":  nonce,
    }
    data, _ := json.Marshal(payload)
    return &TokenTransferEvent{
        BaseEvent: NewBaseEvent("token.transfer", tokenID, data),
    }
}

func (e *TokenTransferEvent) From() ([]byte, error) {
    var m map[string]interface{}
    if err := json.Unmarshal(e.Payload(), &m); err != nil {
        return nil, err
    }
    fromB64, ok := m["from"].(string)
    if !ok {
        return nil, ErrInvalidPayload
    }
    return base64.StdEncoding.DecodeString(fromB64)
}
```

**原则：accessor 方法返回 error，不吞没错误。**
```

### Event Types

| Module  | Event Type              | Fields                                      |
|---------|-------------------------|---------------------------------------------|
| token   | token.transfer          | from, to, amount, nonce, signature          |
| token   | token.mint              | to, amount                                  |
| token   | token.burn              | from, amount                                |
| token   | token.approve           | owner, spender, amount                      |
| nft     | nft.mint                | owner, metadata                             |
| nft     | nft.transfer            | from, to                                    |
| nft     | nft.burn                | from                                        |
| voting  | voting.created          | proposer, proposal                          |
| voting  | voting.vote             | voter, choice                               |
| lottery | lottery.created         | participants, winner_count                  |
| lottery | lottery.drawn           | winners, proof                              |
| oracle  | oracle.data_fetched     | source, data                                |

### EventBus Interface

```go
type EventBus interface {
    Publish(Event) error
    Subscribe(eventType string, handler Handler) func() // returns unsubscribe
    SubscribeAll(handler Handler) func()
}

type Handler func(Event) error
```

### CompositeEventBus (执行编排)

```go
type CompositeEventBus struct {
    syncBus   EventBus   // 同步执行 handlers
    asyncBus  EventBus   // 异步执行 handlers (goroutine + channel)
    pluginBus EventBus   // 可插拔扩展点
}

func (b *CompositeEventBus) Publish(e Event) error {
    if err := b.syncBus.Publish(e); err != nil {
        return err  // all-or-nothing: 同步失败则停止
    }
    b.asyncBus.Publish(e)  // 异步不阻塞，不返回错误
    b.pluginBus.Publish(e) // 插件不阻塞，不返回错误
    return nil
}
```

**执行顺序：SyncBus (阻塞) → AsyncBus (非阻塞) → PluginBus (非阻塞)**
```

### ReplayProtection Interface

```go
type ReplayProtection interface {
    GetLastNonce(tokenID string, owner []byte) (uint64, error)
    SaveNonce(tokenID string, owner []byte, nonce uint64) error
}
```

### EventStore Interface

```go
type EventStore interface {
    Save(event Event) error
    GetByType(eventType string, limit int) ([]Event, error)
    GetByModule(module string, limit int) ([]Event, error)
    GetByAggregate(aggID string) ([]Event, error)
}
```

## Database Schema

```sql
CREATE TABLE events (
    id          TEXT PRIMARY KEY,
    event_type  TEXT NOT NULL,
    module      TEXT NOT NULL,
    agg_id      TEXT NOT NULL,
    payload     BLOB NOT NULL,
    timestamp   INTEGER NOT NULL
);

CREATE INDEX idx_events_type ON events(event_type);
CREATE INDEX idx_events_module ON events(module);
CREATE INDEX idx_events_agg ON events(agg_id);
CREATE INDEX idx_events_timestamp ON events(timestamp DESC);
```

## Error Handling

- **all-or-nothing**: If any handler fails, the entire publish chain stops
- **Handler execution order**: Deterministic, follows registration order
- **First failure wins**: Subsequent handlers are not executed on error

## Implementation Order

### Phase 1: Core Infrastructure
1. Create `internal/domain/events/` with interfaces (`Event`, `ReplayProtection`)
2. Create `internal/infra/events/bus.go` with `EventBus`, `SyncEventBus`
3. Create `internal/infra/events/event_store.go` with SQLite implementation
4. Create `internal/infra/events/handlers.go` with `AuditHandler`

### Phase 2: Async & Plugin Support
5. Create `internal/infra/events/async_bus.go` with channel-based async
6. Create `internal/infra/events/plugin_bus.go` with extension interface
7. Create `CompositeEventBus` orchestrating all buses
8. Add `StatsHandler` and `WebhookHandler`

### Phase 3: Module Migration
9. Migrate Token events to new system, inject EventBus + ReplayProtection
10. Migrate NFT, Voting, Lottery, Oracle modules
11. Add data migration script for existing events
12. Remove legacy `internal/infra/sqlite/event_store.go`

### Phase 4: Testing & Polish
13. Add unit tests for all event components (>85% coverage)
14. Add integration tests for handler chain
15. Update module tests to mock new interfaces

## Service Constructor Changes

```go
// Before
func NewService(repo Repository, eventStore EventStore, chain blockchain.BlockWriter) *TokenService

// After
func NewService(repo Repository, eventBus EventBus, replay ReplayProtection, chain blockchain.BlockWriter) *TokenService
```

## Migration Strategy

1. **Parallel Run**: New system writes to both old and new stores during transition
2. **Event Migration**: One-time script copies existing events to new schema
3. **Backward Compatibility**: Keep legacy `TokenEventStore` until all events migrated
4. **Rollback Plan**: Feature flag to disable new system if issues arise

## Test Coverage Targets

| Component          | Target Coverage |
|--------------------|-----------------|
| domain/events/     | 90%             |
| infra/events/bus   | 85%             |
| infra/events/store | 85%             |
| infra/events/async | 80%             |
| Handlers           | 85%             |