# Event System Test Enhancement Design

Date: 2026-04-19
Status: Approved
Version: 1.0

## Overview

Add comprehensive tests for the event-driven architecture to achieve:
1. **Reliability** - Ensure correct behavior under concurrency and error conditions
2. **Regression Protection** - Prevent refactoring from breaking existing functionality
3. **Documentation** - Tests as executable specifications of system behavior

## Coverage Goals

| Package | Current | Target |
|---------|---------|--------|
| internal/domain/events | 85.9% | 90%+ |
| internal/infra/events | 87.0% | 92%+ |
| internal/domain/token | 64.7% | 80%+ |
| internal/app/token | 91.9% | 93%+ |

## Test File Structure

```
internal/
├── domain/events/
│   ├── event_test.go           # Event interface tests
│   ├── types_token_test.go     # Token event type tests
│   ├── types_nft_test.go       # NFT event type tests
│   ├── types_voting_test.go    # Voting event type tests
│   ├── types_lottery_test.go   # Lottery event type tests
│   └── types_oracle_test.go    # Oracle event type tests
├── infra/events/
│   ├── bus_test.go             # EventBus tests
│   ├── async_bus_test.go       # AsyncEventBus concurrency tests
│   ├── handlers_test.go        # Handler tests
│   ├── event_store_test.go     # EventStore tests
│   ├── replay_test.go          # ReplayProtection tests
│   └── integration_test.go     # Full flow integration tests
├── domain/token/
│   └── *_test.go               # Add event-related tests
└── app/token/
    └── *_test.go               # Add event integration tests
```

## Test Scenarios

### 1. Happy Path (10+ tests)

**Goal:** Verify correct event flow from creation to storage to retrieval.

```go
// Core flow: Create → Publish → Store → Retrieve → Verify
func TestEventFlow_Transfer(t *testing.T) {
    event := NewTokenTransferEvent(tokenID, from, to, amount, nonce)
    _ = eventBus.Publish(event)
    time.Sleep(50 * time.Millisecond) // Wait for async
    
    stored, _ := eventStore.GetByAggregate(string(tokenID))
    require.Len(t, stored, 1)
    require.Equal(t, event.ID(), stored[0].ID())
}
```

**Test Cases:**
- `TestEventFlow_TokenTransfer` - Token 转账事件完整流程
- `TestEventFlow_TokenMint` - Token 铸造事件完整流程
- `TestEventFlow_TokenBurn` - Token 销毁事件完整流程
- `TestEventFlow_TokenApprove` - Token 授权事件完整流程
- `TestEventFlow_NFTMint` - NFT 铸造事件完整流程
- `TestEventFlow_NFTTransfer` - NFT 转账事件完整流程
- `TestEventFlow_NFTBurn` - NFT 销毁事件完整流程
- `TestEventFlow_VotingCreate` - 投票创建事件完整流程
- `TestEventFlow_VotingVote` - 投票事件完整流程
- `TestEventFlow_LotteryCreate` - 抽奖创建事件完整流程
- `TestEventFlow_LotteryDraw` - 抽奖开奖事件完整流程
- `TestEventFlow_OracleFetch` - 预言机获取数据事件完整流程
- `TestEventFlow_MultipleEvents` - 多个事件顺序发布
- `TestEventFlow_MultipleAggregates` - 多个聚合根独立存储

### 2. Concurrency Safety (5+ tests)

**Goal:** Verify correct behavior under concurrent access.

```go
// 50 goroutines publishing simultaneously
func TestEventBus_ConcurrentPublish(t *testing.T) {
    var count int32
    seen := make(map[string]bool)
    
    bus.SubscribeAll(func(e events.Event) error {
        atomic.AddInt32(&count, 1)
        return nil
    })
    
    var wg sync.WaitGroup
    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _ = bus.Publish(events.NewBaseEvent("test", "agg", nil))
        }()
    }
    wg.Wait()
    
    require.Equal(t, int32(50), atomic.LoadInt32(&count))
}
```

**Test Cases:**
- `TestEventBus_ConcurrentPublish` - 50 goroutines 同时发布，无丢失无重复
- `TestEventBus_ConcurrentSubscribe` - 10 goroutines 同时订阅，事件不丢失
- `TestEventBus_ConcurrentUnsubscribe` - 订阅后立即取消，行为确定
- `TestReplayProtection_ConcurrentNonce` - 多 goroutine 并发读写 nonce
- `TestAsyncEventBus_ConcurrentClose` - 并发调用 Close 安全

### 3. Error Recovery (5+ tests)

**Goal:** Verify graceful handling of failures.

```go
// Handler failure stops chain (all-or-nothing)
func TestEventBus_HandlerChainFailure(t *testing.T) {
    var order []string
    bus := NewSyncEventBus()
    
    bus.SubscribeAll(func(e events.Event) error {
        order = append(order, "handler1")
        return nil
    })
    bus.SubscribeAll(func(e events.Event) error {
        order = append(order, "handler2")
        return errors.New("fail")
    })
    bus.SubscribeAll(func(e events.Event) error {
        order = append(order, "handler3") // Should not execute
        return nil
    })
    
    _ = bus.Publish(events.NewBaseEvent("test", "agg", nil))
    require.Equal(t, []string{"handler1", "handler2"}, order)
}
```

**Test Cases:**
- `TestEventBus_EventStoreError` - EventStore 返回错误时 Publish 返回错误
- `TestEventBus_HandlerChainAllOrNothing` - Handler 失败阻止后续 Handler 执行
- `TestAsyncEventBus_ChannelFull` - Channel 满时返回错误而非阻塞
- `TestAsyncEventBus_CloseWhilePublishing` - Close 时正在处理的事件完成后再退出
- `TestCompositeEventBus_SyncBlocksAsync` - SyncBus 错误阻止 AsyncBus/PluginBus

### 4. Edge Cases (5+ tests)

**Goal:** Cover boundary conditions and unusual inputs.

```go
// Duplicate event ID rejected
func TestEventStore_DuplicateID(t *testing.T) {
    e := events.NewBaseEvent("test", "agg", nil)
    
    err1 := eventStore.Save(e)
    err2 := eventStore.Save(e)
    
    require.NoError(t, err1)
    require.Error(t, err2) // Duplicate key
}

// Empty query returns empty slice
func TestEventStore_GetEmpty(t *testing.T) {
    events, err := eventStore.GetByType("nonexistent", 10)
    require.NoError(t, err)
    require.Len(t, events, 0)
}
```

**Test Cases:**
- `TestEventStore_GetByType_Empty` - 不存在类型返回空列表
- `TestEventStore_GetByModule_Empty` - 不存在模块返回空列表
- `TestEventStore_GetAggregate_Empty` - 不存在聚合根返回空列表
- `TestEventStore_DuplicateID_Replace` - 重复 ID 使用 REPLACE 覆盖
- `TestReplayProtection_NonSequentialNonce` - 非递增 nonce 被覆盖
- `TestEventStore_GetByType_Limit` - Limit 参数正确限制返回数量
- `TestEventStore_GetByType_Order` - 结果按时间戳降序排列

## Mock Implementations

### MockEventStore

```go
type MockEventStore struct {
    mu     sync.RWMutex
    events []events.Event
    err    error
}

func (m *MockEventStore) Save(e events.Event) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    if m.err != nil {
        return m.err
    }
    m.events = append(m.events, e)
    return nil
}

func (m *MockEventStore) GetByType(eventType string, limit int) ([]events.Event, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    // ... filter and return
}

func (m *MockEventStore) GetByModule(module string, limit int) ([]events.Event, error)
func (m *MockEventStore) GetByAggregate(aggID string) ([]events.Event, error)
func (m *MockEventStore) Reset() { m.events = nil }
```

### MockReplayProtection

```go
type MockReplayProtection struct {
    mu     sync.RWMutex
    nonces map[string]uint64
    err    error
}

func (m *MockReplayProtection) GetLastNonce(tokenID string, owner []byte) (uint64, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    if m.err != nil {
        return 0, m.err
    }
    key := tokenID + "|" + string(owner)
    return m.nonces[key], nil
}

func (m *MockReplayProtection) SaveNonce(tokenID string, owner []byte, nonce uint64) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    if m.err != nil {
        return m.err
    }
    m.nonces[tokenID+"|"+string(owner)] = nonce
    return nil
}
```

## Running Tests

```bash
# Unit tests
go test ./internal/domain/events/... -v -cover
go test ./internal/infra/events/... -v -cover

# With race detection
go test ./internal/infra/events/... -race -cover

# Integration tests
go test ./internal/infra/events/... -v -tags=integration

# All tests with coverage
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Implementation Order

1. Add mock implementations for stores and services
2. Add Happy Path tests for all event types
3. Add concurrency tests (with -race flag)
4. Add error recovery tests
5. Add edge case tests
6. Run full coverage and adjust

## Acceptance Criteria

- [ ] `domain/events`: 14+ Happy Path tests, coverage ≥90%
- [ ] `infra/events`: 10+ Concurrency tests, all pass with `-race`
- [ ] `infra/events`: 5+ Error Recovery tests
- [ ] `infra/events`: 7+ Edge Case tests
- [ ] `domain/token` + `app/token`: Event integration tests added
- [ ] All tests pass, no flaky tests (verify with 3x run)