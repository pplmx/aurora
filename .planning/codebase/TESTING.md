# Testing Patterns

**Analysis Date:** 2026-04-30

## Test Framework

**Test Runner:**
- Go standard `testing` package
- Version: Go 1.26+

**Assertion Libraries:**
- `github.com/stretchr/testify` (assert, require)

**Run Commands:**
```bash
just test              # Run all tests
go test ./... -cover   # With coverage report
just test-coverage     # Generate coverage.out
```

## Test File Organization

**Location:**
- Co-located with implementation: `internal/domain/lottery/entity_test.go`
- E2E in `e2e/` directory: `e2e/lottery_e2e_test.go`

**Naming:**
- Unit tests: `*_test.go`
- E2E tests: `*_e2e_test.go`

**Package Declaration:**
- Same package as implementation (not `_test` suffix)
```go
package lottery  // Not package lottery_test
```

## Test Structure

### Unit Tests

**Basic pattern:**
```go
func TestGetWinners(t *testing.T) {
    record := &LotteryRecord{
        Winners: []string{"Alice", "Bob"},
    }
    got := record.GetWinners()
    if len(got) != 2 || got[0] != "Alice" || got[1] != "Bob" {
        t.Errorf("GetWinners() = %v, want [Alice Bob]", got)
    }
}
```

### Table-Driven Tests

**For multiple test cases:**
```go
func TestValidateParticipantName_Valid(t *testing.T) {
    valid := []string{"Alice", "Bob 123", "test-name", "Name_With", "日本語", "中文"}
    for _, name := range valid {
        err := ValidateParticipantName(name)
        if err != nil {
            t.Errorf("ValidateParticipantName(%q) should not error: %v", name, err)
        }
    }
}

func TestSanitizeString(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"  hello  ", "hello"},
        {"hello\x00world", "helloworld"},
        {"normal text", "normal text"},
    }

    for _, tt := range tests {
        got := SanitizeString(tt.input)
        if got != tt.expected {
            t.Errorf("SanitizeString(%q) = %q, want %q", tt.input, got, tt.expected)
        }
    }
}
```

**With subtests for named cases:**
```go
func TestLotteryRecord_Validate(t *testing.T) {
    tests := []struct {
        name    string
        record  *LotteryRecord
        wantErr bool
    }{
        {
            name: "valid record with seed and participants",
            record: &LotteryRecord{
                Seed:         "test-seed",
                Participants: []string{"Alice", "Bob", "Charlie"},
                Winners:      []string{"Alice"},
            },
            wantErr: false,
        },
        // ... more cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.record.Validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Mocking Patterns

### Manual Interface Mocking

**Define mock in test file:**
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

func (m *mockLotteryRepo) GetAll() ([]*lottery.LotteryRecord, error) {
    return m.records, nil
}

func (m *mockLotteryRepo) GetByBlockHeight(height int64) ([]*lottery.LotteryRecord, error) {
    var result []*lottery.LotteryRecord
    for _, r := range m.records {
        if r.BlockHeight == height {
            result = append(result, r)
        }
    }
    return result, nil
}
```

**Usage in tests:**
```go
func TestCreateLotteryUseCase_Execute(t *testing.T) {
    lotteryRepo := &mockLotteryRepo{}
    blockChain := &mockBlockChain{}

    uc := NewCreateLotteryUseCase(lotteryRepo, blockChain)

    req := CreateLotteryRequest{
        Participants: "Alice,Bob,Charlie",
        Seed:         "test-seed",
        WinnerCount:  2,
    }

    resp, err := uc.Execute(req)
    require.NoError(t, err)
    require.NotNil(t, resp)
    // assertions...
}
```

## Test Fixtures

### DB Test Setup

```go
func setupTokenTestDB(t *testing.T) (*TokenRepository, func()) {
    tmpDir := t.TempDir()
    dbPath := filepath.Join(tmpDir, "test_token.db")

    repo, err := NewTokenRepository(dbPath)
    if err != nil {
        t.Fatalf("Failed to create token repository: %v", err)
    }

    cleanup := func() {
        if repo != nil {
            _ = repo.Close()
        }
        _ = os.RemoveAll(tmpDir)
    }

    return repo, cleanup
}

func TestTokenRepository_GetToken(t *testing.T) {
    repo, cleanup := setupTokenTestDB(t)
    defer cleanup()

    testToken := token.NewToken("TEST", "Test Token", "TEST",
        token.NewAmount(1000000), token.PublicKey([]byte("owner")))

    err := repo.SaveToken(testToken)
    require.NoError(t, err)

    retrieved, err := repo.GetToken("TEST")
    require.NoError(t, err)
    assert.Equal(t, "TEST", retrieved.ID())
}
```

### Event Store Setup

```go
func setupEventStore(t *testing.T) (*SQLiteEventStore, func()) {
    tmpFile, err := os.CreateTemp("", "event_store_test_*.db")
    require.NoError(t, err)
    _ = tmpFile.Close()

    store, err := NewSQLiteEventStore(tmpFile.Name())
    require.NoError(t, err)

    cleanup := func() {
        _ = store.Close()
        _ = os.Remove(tmpFile.Name())
    }
    return store, cleanup
}
```

## Testify Assertions

### require (Fatal)

Stops test immediately on failure:
```go
require.NoError(t, err)
require.NotNil(t, resp)
require.Len(t, results, 2)
require.Equal(t, "expected", actual)
```

### assert (Non-fatal)

Continues test execution:
```go
assert.Equal(t, http.StatusOK, rr.Code)
assert.NotEqual(t, http.StatusBadRequest, rr.Code)
assert.Contains(t, output, "expected string")
```

## E2E Tests

**Location:** `e2e/*_e2e_test.go`
**Package:** `test` (shared package for integration)

```go
package test

import (
    "testing"

    blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
    lottery "github.com/pplmx/aurora/internal/domain/lottery"
)

func TestLotteryE2E_FullFlow(t *testing.T) {
    blockchain.ResetForTest()

    participants := []string{"Alice", "Bob", "Charlie", "David", "Eve"}
    seed := "e2e-test-seed-123"
    count := 3

    _, sk, err := lottery.GenerateKeyPair()
    if err != nil {
        t.Fatalf("GenerateKeyPair failed: %v", err)
    }

    output, proof, err := lottery.VRFProve(sk, []byte(seed))
    // ...
}
```

**Test isolation:**
- `blockchain.ResetForTest()` resets global state before each test

## HTTP Handler Tests

```go
func TestLotteryHandler_Create_InvalidRequest(t *testing.T) {
    handler := &LotteryHandler{}

    req := httptest.NewRequest(http.MethodPost, "/api/v1/lottery/create",
        bytes.NewBufferString("invalid json"))
    rr := httptest.NewRecorder()

    handler.Create(rr, req)

    assert.Equal(t, http.StatusBadRequest, rr.Code)

    var resp ErrorResponse
    err := json.Unmarshal(rr.Body.Bytes(), &resp)
    assert.NoError(t, err)
    assert.Equal(t, "INVALID_REQUEST", resp.Code)
}
```

## Coverage Requirements

**Current coverage by module:**

| Module | Domain | App |
|--------|--------|-----|
| Lottery | 93.3% | 87.1% |
| Voting | 87.5% | 76.8% |
| NFT | 93.8% | 88.1% |
| Token | 90.3% | 91.9% |
| Oracle | 76.1% | 94.5% |

**Command:**
```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

**Low coverage areas:**
- `internal/api` (3.6% - handler coverage)
- `internal/api/handler` (9.5%)
- `internal/domain/blockchain` (30.9%)
- `internal/infra/sqlite` (49.1%)

## Test Naming Conventions

**Pattern:** `Test<Subject>_<Scenario>`

Examples:
- `TestGetWinners`
- `TestLotteryRecord_ToJSON`
- `TestLotteryRecord_ToJSON_Invalid`
- `TestCreateLotteryUseCase_Execute`
- `TestCreateLotteryUseCase_InvalidInput`
- `TestLotteryE2E_FullFlow`
- `TestSQLiteEventStore_Save`

## Error Testing Patterns

**Expected error:**
```go
func TestValidateSeed_TooShort(t *testing.T) {
    err := ValidateSeed("ab")
    if err == nil {
        t.Error("ValidateSeed() 'ab' should error")
    }
}
```

**With subtests:**
```go
func TestCreateLotteryUseCase_InvalidInput(t *testing.T) {
    tests := []struct {
        name    string
        req     CreateLotteryRequest
        wantErr bool
    }{
        {
            name: "empty participants",
            req: CreateLotteryRequest{
                Participants: "",
                Seed:         "seed",
                WinnerCount:  1,
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := uc.Execute(tt.req)
            if (err != nil) != tt.wantErr {
                t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Pre-commit Hooks

**Configuration:** `.pre-commit-config.yaml`

Go hooks:
- `gofmt` - Format code
- `goimports` - Organize imports
- `go vet` - Static analysis
- `golangci-lint` - Full lint suite (5m timeout, errcheck/staticcheck disabled)

Other hooks:
- `commitizen` - Conventional commits
- `end-of-file-fixer` - Normalize line endings
- `trailing-whitespace` - Remove trailing whitespace
- `check-yaml`, `check-toml` - Validate config files
- `rumdl` - Markdown linting

---

*Testing analysis: 2026-04-30*
