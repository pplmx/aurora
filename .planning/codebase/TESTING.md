# Testing Patterns

**Analysis Date:** 2026-04-26

## Test Framework

**Language:** Go 1.26+

**Standard Library:**
- `testing` - Core testing package
- `testify` - Enhanced assertions (`github.com/stretchr/testify`)

**Test Database:**
- SQLite in-memory for repository tests

**Coverage:** See AGENTS.md for module coverage targets

## Run Commands

### justfile Recipes

```bash
just test              # Run all tests
just test-coverage     # Run with coverage report
```

### Standard go test

```bash
go test ./...                       # All tests
go test ./internal/domain/... -v    # Domain layer
go test ./internal/app/... -v       # App layer
go test ./e2e/ -v                   # E2E tests
go test ./... -cover                # With coverage
go test ./... -coverprofile=coverage.out  # Coverage file
```

## Test File Organization

### Location Pattern

**Unit tests:** Co-located with source files
```
internal/domain/lottery/
├── service.go
├── service_test.go      # Unit tests
├── entity.go
└── entity_test.go       # Entity tests

internal/app/lottery/
├── usecase.go
└── usecase_test.go      # Use case tests
```

**E2E tests:** Separate `e2e/` directory
```
e2e/
├── lottery_e2e_test.go
├── voting_e2e_test.go
├── token_e2e_test.go
├── nft_e2e_test.go
└── oracle_e2e_test.go
```

### Naming Convention

- Test files: `*_test.go`
- Test functions: `Test<Subject>_<Scenario>` (e.g., `TestLotteryService_DrawWinners`)

## Test Structure

### Package Declaration

**Unit tests:** Same package as source
```go
package lottery
```

**E2E tests:** `test` package
```go
package test
```

### Test Function Pattern

```go
func Test<Subject>_<Action>(t *testing.T) {
    // Arrange: Set up test data
    service := NewService()
    participants := []string{"Alice", "Bob", "Charlie"}

    // Act: Execute the function under test
    winners, _, _, _, err := service.DrawWinners(participants, "seed", 1)

    // Assert: Verify results
    if err != nil {
        t.Fatalf("DrawWinners failed: %v", err)
    }

    if len(winners) != 1 {
        t.Errorf("Expected 1 winner, got %d", len(winners))
    }
}
```

### Error Test Pattern

```go
func TestLotteryService_DrawWinners_InvalidInput(t *testing.T) {
    service := NewService()

    _, _, _, _, err := service.DrawWinners([]string{}, "seed", 1)
    if err == nil {
        t.Fatal("Expected error for empty participants")
    }
}
```

### Table-Driven Tests

```go
func TestLotteryE2E_AddressConversion(t *testing.T) {
    tests := []struct {
        name     string
        wantLen  int
        wantPref string
    }{
        {"Alice", 42, "0x"},
        {"Bob", 42, "0x"},
        {"中文", 42, "0x"},
        {"", 42, "0x"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := lottery.NameToAddress(tt.name)
            if len(got) != tt.wantLen {
                t.Errorf("len = %d, want %d", len(got), tt.wantLen)
            }
        })
    }
}
```

## Mocking Patterns

### Interface-based Mocks

Define mock structs that implement domain interfaces:

```go
type mockLotteryRepo struct {
    records []*lottery.LotteryRecord
}

func (m *mockLotteryRepo) Save(record *lottery.LotteryRecord) error {
    m.records = append(m.records, record)
    return nil
}

func (m *mockLotteryRepo) GetByID(id string) (*lottery.LotteryRecord, error) {
    for _, r := range m.records {
        if r.ID == id {
            return r, nil
        }
    }
    return nil, nil
}
```

### Inline Test Doubles

For E2E tests with in-memory implementations:

```go
type inMemoryTokenRepo struct {
    tokens    map[token.TokenID]*token.Token
    balances  map[string]*token.Amount
    approvals map[string]*token.Approval
}

func newInMemoryTokenRepo() *inMemoryTokenRepo {
    return &inMemoryTokenRepo{
        tokens:    make(map[token.TokenID]*token.Token),
        balances:  make(map[string]*token.Amount),
        approvals: make(map[string]*token.Approval),
    }
}
```

### Blockchain Reset

E2E tests reset blockchain state:

```go
func TestLotteryE2E_FullFlow(t *testing.T) {
    blockchain.ResetForTest()
    // ... test code
}
```

## Test Fixtures

### Database Fixtures

In-memory SQLite setup for repository tests:

```go
func setupVotingTestDB(t *testing.T) (*VotingRepository, func()) {
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatalf("Failed to open database: %v", err)
    }

    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS votes (...)
    `)
    if err != nil {
        t.Fatalf("Failed to create tables: %v", err)
    }

    repo := NewVotingRepository(db)
    cleanup := func() { db.Close() }
    return repo, cleanup
}
```

### Test Data Builders

Inline construction in tests:

```go
record := &lottery.LotteryRecord{
    ID:              "test-id-123",
    Seed:            "test-seed",
    Participants:    []string{"A", "B", "C"},
    Winners:         []string{"A"},
    WinnerAddresses: []string{"0xabc123"},
    VRFProof:        "proof-data",
    VRFOutput:       "output-data",
    BlockHeight:     1,
    Timestamp:       1234567890,
}
```

## Assertion Patterns

### Using testify/require

```go
import "github.com/stretchr/testify/require"

func TestExample(t *testing.T) {
    require.NotNil(t, resp)
    require.Equal(t, 2, len(resp.Winners))
    require.NoError(t, err)
}
```

### Standard testing assertions

```go
// Fatal on failure (stops test)
if err != nil {
    t.Fatalf("Execute failed: %v", err)
}

// Error on failure (continues test)
if len(winners) != count {
    t.Errorf("Expected %d winners, got %d", count, len(winners))
}

// Boolean checks
if !found {
    t.Error("Should have found the element")
}

// Nil checks
if record == nil {
    t.Fatal("Record should not be nil")
}
```

## E2E Test Patterns

### Full Flow Tests

E2E tests verify end-to-end scenarios:

```go
func TestLotteryE2E_FullFlow(t *testing.T) {
    blockchain.ResetForTest()

    participants := []string{"Alice", "Bob", "Charlie", "David", "Eve"}
    seed := "e2e-test-seed-123"

    _, sk, err := lottery.GenerateKeyPair()
    // ... generate proof

    winners := lottery.SelectWinners(output, participants, count)
    // ... verify winners

    chain := blockchain.InitBlockChain()
    height, err := chain.AddLotteryRecord(jsonData)
    // ... verify on blockchain
}
```

### Integration Tests

Test multiple components working together:

```go
func TestTokenE2E_Transfer(t *testing.T) {
    blockchain.ResetForTest()
    eventStore := infraevents.NewInMemoryStore()
    // ... set up components
    // ... execute operations
    // ... verify state and events
}
```

## Coverage Targets

From AGENTS.md:

| Module  | Domain | App   |
|---------|--------|-------|
| Lottery | 73.8%  | 87.1% |
| Voting  | 87.5%  | 81.4% |
| NFT     | 72.7%  | 88.1% |
| Token   | 70.5%  | 91.9% |
| Oracle  | 76.1%  | 65.8% |

**Overall goal:** Maintain or improve coverage when adding features

## Test Categories

### Domain Tests (`internal/domain/*/ *_test.go`)

- Pure business logic
- Validation rules
- Entity methods
- Service layer logic

### Application Tests (`internal/app/*/ *_test.go`)

- Use case orchestration
- Mocked dependencies
- Request/response validation
- Error handling paths

### Infrastructure Tests (`internal/infra/*/ *_test.go`)

- Repository implementations
- Database operations
- Event handlers
- HTTP fetchers

### E2E Tests (`e2e/*_e2e_test.go`)

- Full integration
- Real dependencies
- End-to-end flows
- Blockchain interactions

## Common Test Patterns

### Testing Edge Cases

```go
func TestSelectWinners_EdgeCases(t *testing.T) {
    // Single participant
    winners := lottery.SelectWinners([]byte{0x01}, []string{"OnlyOne"}, 1)
    if len(winners) != 1 {
        t.Error("Should return single participant")
    }

    // Empty participants
    winners = lottery.SelectWinners([]byte{0x01}, []string{}, 1)
    if len(winners) != 0 {
        t.Error("Should return empty")
    }

    // Count equals participants
    winners = lottery.SelectWinners([]byte{0x01}, []string{"A", "B"}, 2)
    if len(winners) != 2 {
        t.Error("Should return all")
    }
}
```

### Determinism Tests

```go
func TestVRF_Determinism(t *testing.T) {
    _, sk, _ := lottery.GenerateKeyPair()
    seed := "deterministic-test"

    output1, _, _ := lottery.VRFProve(sk, []byte(seed))
    output2, _, _ := lottery.VRFProve(sk, []byte(seed))

    if string(output1) != string(output2) {
        t.Error("Same key and seed should produce same output")
    }
}
```

### State Isolation

Each test resets shared state:
```go
func TestXxx(t *testing.T) {
    blockchain.ResetForTest()  // Reset blockchain
    // ... test code
}
```

---

*Testing analysis: 2026-04-26*