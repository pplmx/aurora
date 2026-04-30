# Testing Patterns for Go REST API Handlers and TUI Packages

**Project:** Aurora v1.2 Operational Readiness
**Researched:** 2026-04-30
**Focus:** Improving API handler coverage (7% → 80%+) and adding UI package tests

---

## Executive Summary

Aurora's handlers currently use basic testing patterns with ad-hoc mocks. This document provides **specific, actionable patterns** to achieve 80%+ coverage. The key insight: **test behavior through the handler interface, not implementation details**.

**For Handlers:** Table-driven tests with structured mocks, test both success/error paths, verify response structure.
**For TUI:** Test the model logic separate from the view, use `tea.Model` interface to send `tea.Msg` directly.

---

## 1. HTTP Handler Testing Patterns

### 1.1 Core Testing Setup

**Pattern: Handler Test Helper**

```go
// internal/api/handler/handler_test.go
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testRequest builds a properly configured HTTP request with Chi URL params
func testRequest(t *testing.T, method, path string, body []byte) (*http.Request, *httptest.ResponseRecorder) {
	t.Helper()

	var bodyReader *bytes.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, httptest.NewRecorder()
}

// withChiParams adds Chi URL parameters to the request context
func withChiParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// assertJSONResponse checks response structure without strict equality
func assertJSONResponse(t *testing.T, rr *httptest.ResponseRecorder, wantStatus int) {
	t.Helper()
	assert.Equal(t, wantStatus, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

// parseResponse parses response body into the given struct
func parseResponse(t *testing.T, rr *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), v))
}
```

### 1.2 Mock Repository Patterns

**Current Problem (from lottery_test.go):**
```go
// BAD: Inline mock with no interface validation
type mockLotteryRepo struct{}
func (m *mockLotteryRepo) Save(*lottery.LotteryRecord) error { return nil }
// ...
```

**Pattern: Typed Mocks with Behavior Configuration**

```go
// internal/api/handler/mock_test.go
package handler

import (
	"github.com/pplmx/aurora/internal/domain/lottery"
)

// MockLotteryRepo provides configurable mock behavior for testing
type MockLotteryRepo struct {
	SaveFunc           func(*lottery.LotteryRecord) error
	GetByIDFunc        func(string) (*lottery.LotteryRecord, error)
	GetAllFunc         func() ([]*lottery.LotteryRecord, error)
	GetByBlockHeightFunc func(int64) ([]*lottery.LotteryRecord, error)
}

// Compile-time interface compliance check
var _ lottery.Repository = (*MockLotteryRepo)(nil)

func (m *MockLotteryRepo) Save(r *lottery.LotteryRecord) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(r)
	}
	return nil
}

func (m *MockLotteryRepo) GetByID(id string) (*lottery.LotteryRecord, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(id)
	}
	return nil, nil
}

func (m *MockLotteryRepo) GetAll() ([]*lottery.LotteryRecord, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc()
	}
	return nil, nil
}

func (m *MockLotteryRepo) GetByBlockHeight(height int64) ([]*lottery.LotteryRecord, error) {
	if m.GetByBlockHeightFunc != nil {
		return m.GetByBlockHeightFunc(height)
	}
	return nil, nil
}
```

**Usage in Tests:**
```go
func TestLotteryHandler_Get_Success(t *testing.T) {
	expected := &lottery.LotteryRecord{
		ID:       "lottery-123",
		Seed:     "test-seed",
		Verified: true,
	}

	mock := &MockLotteryRepo{
		GetByIDFunc: func(id string) (*lottery.LotteryRecord, error) {
			assert.Equal(t, "lottery-123", id)
			return expected, nil
		},
	}

	handler := NewLotteryHandler(mock)

	req, rr := testRequest(t, http.MethodGet, "/api/v1/lottery/lottery-123", nil)
	req = withChiParams(req, map[string]string{"id": "lottery-123"})

	handler.Get(rr, req)

	assertJSONResponse(t, rr, http.StatusOK)

	var resp lottery.LotteryRecord
	parseResponse(t, rr, &resp)
	assert.Equal(t, "lottery-123", resp.ID)
}
```

### 1.3 Table-Driven Handler Tests

**Pattern: Comprehensive Scenario Coverage**

```go
func TestLotteryHandler_Create_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    CreateLotteryRequest
		mockSetup      func(*MockLotteryRepo)
		wantStatus     int
		wantErrCode    string
		wantIDContains string
	}{
		{
			name: "valid request returns created",
			requestBody: CreateLotteryRequest{
				Participants: "Alice,Bob,Charlie",
				Seed:         "test-seed-123",
				WinnerCount:  2,
			},
			mockSetup: func(m *MockLotteryRepo) {
				m.SaveFunc = func(r *lottery.LotteryRecord) error {
					r.ID = "generated-id"
					return nil
				}
			},
			wantStatus:     http.StatusOK,
			wantIDContains: "lottery-",
		},
		{
			name: "invalid JSON returns bad request",
			requestBody: CreateLotteryRequest{
				Participants: "",
				Seed:         "",
				WinnerCount:  0,
			},
			mockSetup:   func(m *MockLotteryRepo) {},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_REQUEST",
		},
		{
			name: "repository error returns internal error",
			requestBody: CreateLotteryRequest{
				Participants: "Alice,Bob",
				Seed:         "seed",
				WinnerCount:  1,
			},
			mockSetup: func(m *MockLotteryRepo) {
				m.SaveFunc = func(r *lottery.LotteryRecord) error {
					return errors.New("database error")
				}
			},
			wantStatus:  http.StatusInternalServerError,
			wantErrCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockLotteryRepo{}
			tt.mockSetup(mock)

			// Setup blockchain mock - use interface approach
			blockRepo := &MockBlockRepo{
				AddLotteryRecordFunc: func(data string) (int64, error) {
					return 1, nil
				},
			}

			handler := &LotteryHandler{
				repo: mock,
				// If handler needs blockRepo directly, inject it
			}

			body, _ := json.Marshal(tt.requestBody)
			req, rr := testRequest(t, http.MethodPost, "/api/v1/lottery/create", body)

			handler.Create(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)

			if tt.wantErrCode != "" {
				var resp ErrorResponse
				parseResponse(t, rr, &resp)
				assert.Equal(t, tt.wantErrCode, resp.Code)
			}

			if tt.wantIDContains != "" {
				var resp map[string]interface{}
				parseResponse(t, rr, &resp)
				assert.Contains(t, resp["id"], tt.wantIDContains)
			}
		})
	}
}
```

### 1.4 Testing Error Responses

**Pattern: Structured Error Testing**

```go
func TestTokenHandler_Balance_MissingParams(t *testing.T) {
	handler := NewTokenHandler(nil) // Service not needed for validation

	tests := []struct {
		name          string
		queryParams   string
		wantStatus    int
		wantErrCode   string
	}{
		{
			name:        "missing both params",
			queryParams: "",
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_REQUEST",
		},
		{
			name:        "missing token_id",
			queryParams: "?owner=alice",
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_REQUEST",
		},
		{
			name:        "missing owner",
			queryParams: "?token_id=token-123",
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "INVALID_REQUEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, rr := testRequest(t, http.MethodGet, "/api/v1/token/balance"+tt.queryParams, nil)
			handler.Balance(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)

			var resp ErrorResponse
			parseResponse(t, rr, &resp)
			assert.Equal(t, tt.wantErrCode, resp.Code)
		})
	}
}
```

### 1.5 Integration-Style Tests with httptest.Server

**When to Use:** Testing with actual router, middleware, and full HTTP stack.

```go
func TestLotteryHandler_WithRouter(t *testing.T) {
	mock := &MockLotteryRepo{
		GetAllFunc: func() ([]*lottery.LotteryRecord, error) {
			return []*lottery.LotteryRecord{
				{ID: "lottery-1", Seed: "seed1"},
				{ID: "lottery-2", Seed: "seed2"},
			}, nil
		},
	}

	handler := NewLotteryHandler(mock)

	r := chi.NewRouter()
	handler.Routes(r)

	// Test via actual HTTP client
	server := httptest.NewServer(r)
	defer server.Close()

	client := server.Client()

	resp, err := client.Get(server.URL + "/api/v1/lottery/history")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var results []*lottery.LotteryRecord
	json.NewDecoder(resp.Body).Decode(&results)
	assert.Len(t, results, 2)
}
```

---

## 2. TUI Testing Patterns (BubbleTea)

### 2.1 Core TUI Testing Approach

**Key Insight:** TUI = Model + View + Update. Test the **Model** and **Update** logic, not the **View**.

**Pattern: Direct Model Interaction**

```go
// internal/ui/lottery/tui_test.go
package lottery

import (
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// newTestModel creates a model in a known state for testing
func newTestModel() *model {
	return &model{
		view:              "menu",
		menuIndex:         0,
		showHelp:          false,
		participantsInput: textinput.New(),
		seedInput:         textinput.New(),
		countInput:        textinput.New(),
	}
}

// TestModel_MenuNavigation tests menu navigation
func TestModel_MenuNavigation(t *testing.T) {
	m := newTestModel()

	// Test initial state
	assert.Equal(t, "menu", m.view)
	assert.Equal(t, 0, m.menuIndex)

	// Test key press: down arrow
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})

	assert.Equal(t, 1, m.menuIndex)

	// Test key press: down arrow again
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})

	assert.Equal(t, 2, m.menuIndex) // Max menu index

	// Test key press: down arrow at max - should not exceed
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})

	assert.Equal(t, 2, m.menuIndex)
}

// TestModel_ViewTransitions tests view state changes
func TestModel_ViewTransitions(t *testing.T) {
	m := newTestModel()

	// Navigate to "create" menu item
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Equal(t, "create", m.view)

	// Go back to menu
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	assert.Equal(t, "menu", m.view)
}

// TestModel_QuitFromMenu tests quit behavior
func TestModel_QuitFromMenu(t *testing.T) {
	m := newTestModel()

	// Navigate to exit item (index 2)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})

	assert.Equal(t, 2, m.menuIndex)

	// Press enter to quit
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Cmd should signal quit
	assert.Equal(t, tea.Quit, cmd)
}
```

### 2.2 Testing Input Handling

**Pattern: Text Input in TUI**

```go
func TestModel_CreateView_InputValidation(t *testing.T) {
	m := newTestModel()

	// Navigate to create view
	m.menuIndex = 0
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, "create", m.view)

	// Test input in participants field
	m.participantsInput.Focus()
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})

	assert.Contains(t, m.participantsInput.Value(), "A")
}

// TestModel_ValidationErrors tests error display
func TestModel_ValidationErrors(t *testing.T) {
	m := newTestModel()

	// Navigate to create view with empty inputs
	m.view = "create"
	m.participantsInput.SetValue("Alice")
	m.seedInput.SetValue("")
	m.countInput.SetValue("10")

	// Execute create with insufficient participants
	seed := m.seedInput.Value()
	participants := parseTextArea(m.participantsInput.Value())
	count := 3

	// Simulate the validation from handleCreate
	if len(participants) < count {
		m.err = "参与者人数必须多于获奖人数"
	}

	assert.NotEmpty(t, m.err)
	assert.Equal(t, "参与者人数必须多于获奖人数", m.err)
}
```

### 2.3 View Rendering Tests

**Pattern: Test View Output**

```go
func TestModel_ViewOutput(t *testing.T) {
	m := newTestModel()

	view := m.View()

	// Verify view contains expected elements
	assert.Contains(t, view, "抽奖")
	assert.Contains(t, view, "创建抽奖")
	assert.Contains(t, view, "查看历史")
}

// TestModel_ResultView tests result display
func TestModel_ResultView(t *testing.T) {
	m := newTestModel()
	m.view = "result"

	m.result = &lottery.LotteryRecord{
		ID:     "lottery-123",
		Seed:   "test-seed",
		Winners: []string{"Alice", "Bob"},
		WinnerAddresses: []string{"addr1", "addr2"},
		VRFOutput: "vrf-output-12345678901234567890",
		VRFProof: "vrf-proof-12345678901234567890",
	}

	view := m.View()

	assert.Contains(t, view, "lottery-123")
	assert.Contains(t, view, "Alice")
	assert.Contains(t, view, "Bob")
	assert.Contains(t, view, "vrf-output-12345678901234567890...")
}
```

### 2.4 TUI Test Helper Functions

```go
// Test helpers for TUI
package lottery

import (
	"github.com/charmbracelet/bubbletea"
)

// keyPress creates a tea.KeyMsg for testing
func keyPress(key string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}

// keyArrow creates an arrow key message
func keyArrow(arrowType tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: arrowType}
}

// enterKey creates an enter key message
func enterKey() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyEnter}
}

// escapeKey creates an escape key message
func escapeKey() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyEsc}
}

// ctrlC creates a ctrl+c message
func ctrlC() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyCtrlC}
}

// windowSize creates a window size message
func windowSize(width, height int) tea.WindowSizeMsg {
	return tea.WindowSizeMsg{Width: width, Height: height}
}

// Test helper using these
func TestModel_KeyboardShortcuts(t *testing.T) {
	m := newTestModel()

	// Test number keys for menu selection
	m, _ = m.Update(keyPress("1"))
	assert.Equal(t, 0, m.menuIndex)

	m, _ = m.Update(keyPress("2"))
	assert.Equal(t, 1, m.menuIndex)

	m, _ = m.Update(keyPress("3"))
	assert.Equal(t, 2, m.menuIndex)
}
```

---

## 3. Anti-Patterns - What NOT to Do

### 3.1 Don't Test Private Functions Directly

```go
// BAD: Testing parseTextArea directly couples tests to implementation
func TestParseTextArea(t *testing.T) {
	result := parseTextArea("A\nB\nC")
	assert.Equal(t, []string{"A", "B", "C"}, result)
}

// GOOD: Test through public interface - the model behavior
func TestModel_ParticipantsParsing(t *testing.T) {
	m := newTestModel()
	m.view = "create"

	// Simulate user input
	m.participantsInput.SetValue("Alice\nBob\nCharlie")

	// The actual parsing happens in handleCreate
	participants := parseTextArea(m.participantsInput.Value())
	assert.Len(t, participants, 3)
}
```

### 3.2 Don't Use Sleep for Timing

```go
// BAD: Fragile, timing-dependent
func TestTUI_AsyncUpdate(t *testing.T) {
	m := NewModel()
	go func() {
		time.Sleep(100 * time.Millisecond)
		m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	}()
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, "result", m.view)
}

// GOOD: Use channels or synchronous testing
func TestTUI_DirectUpdate(t *testing.T) {
	m := newTestModel()
	m.result = &lottery.LotteryRecord{ID: "test"}

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Equal(t, "menu", m.view) // Back to menu after viewing result
	assert.Nil(t, m.result)
}
```

### 3.3 Don't Test Implementation Details

```go
// BAD: Testing internal state that could change
func TestModel_InternalState(t *testing.T) {
	m := &model{
		view: "create",
		// These are implementation details
		chain: blockchain.InitBlockChain(),
	}
	assert.NotNil(t, m.chain)
}

// GOOD: Test observable behavior and outputs
func TestModel_CreateProducesResult(t *testing.T) {
	m := newTestModel()
	m.view = "create"
	m.seedInput.SetValue("seed")
	m.countInput.SetValue("1")
	// ... set up enough for handleCreate

	m.handleCreate()

	assert.Equal(t, "result", m.view)
	assert.NotNil(t, m.result)
	assert.NotEmpty(t, m.successMsg)
}
```

### 3.4 Don't Ignore Error Path Testing

```go
// BAD: Only testing happy paths
func TestLotteryHandler(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		// Only tests success
	})

	// Missing: error cases!
}

// GOOD: Comprehensive test coverage
func TestLotteryHandler(t *testing.T) {
	t.Run("create valid", func(t *testing.T) { /* ... */ })
	t.Run("create invalid JSON", func(t *testing.T) { /* ... */ })
	t.Run("create missing fields", func(t *testing.T) { /* ... */ })
	t.Run("create repo error", func(t *testing.T) { /* ... */ })
	t.Run("get existing", func(t *testing.T) { /* ... */ })
	t.Run("get not found", func(t *testing.T) { /* ... */ })
	t.Run("history empty", func(t *testing.T) { /* ... */ })
	t.Run("history with data", func(t *testing.T) { /* ... */ })
}
```

---

## 4. Test Organization Patterns

### 4.1 Directory Structure

```
internal/
├── api/
│   └── handler/
│       ├── lottery.go
│       ├── lottery_test.go       # Handler tests
│       ├── token.go
│       ├── token_test.go
│       ├── mock_test.go          # Shared mocks
│       └── testhelpers_test.go   # Test utilities
└── ui/
    └── lottery/
        ├── tui.go
        └── tui_test.go
```

### 4.2 Shared Mock Pattern

```go
// internal/api/handler/mock_test.go
package handler

// MockTokenService implements token.Service for testing
type MockTokenService struct {
	CreateTokenFunc   func(req *tokenapp.CreateTokenRequest) (*tokenapp.CreateTokenResponse, error)
	MintFunc          func(req *tokenapp.MintRequest) (*tokenapp.TokenOperationResponse, error)
	TransferFunc      func(req *tokenapp.TransferRequest) (*tokenapp.TokenOperationResponse, error)
	BurnFunc          func(req *tokenapp.BurnRequest) (*tokenapp.TokenOperationResponse, error)
	GetBalanceFunc    func(req *tokenapp.BalanceRequest) (*tokenapp.BalanceResponse, error)
	GetHistoryFunc    func(req *tokenapp.HistoryRequest) (*tokenapp.HistoryResponse, error)
}

var _ token.Service = (*MockTokenService)(nil)

// Implement all methods...
```

### 4.3 Golden File Pattern for View Testing

```go
// For TUI view snapshots
func TestLotteryView_Golden(t *testing.T) {
	m := newTestModel()
	m.view = "menu"
	m.menuIndex = 1

	view := m.View()

	goldenPath := "testdata/lottery_menu.golden"
	if *update {
		os.WriteFile(goldenPath, []byte(view), 0644)
	}

	golden, err := os.ReadFile(goldenPath)
	require.NoError(t, err)

	assert.Equal(t, string(golden), view)
}
```

---

## 5. Coverage Strategy for Aurora Handlers

### 5.1 Priority 1: High-Impact Handlers (Target: 90%+)

| Handler | Critical Paths | Test Count Target |
|---------|----------------|-------------------|
| `TokenHandler.Create` | Validation → Service → Response | 8 tests |
| `TokenHandler.Transfer` | Auth → Validation → Service → Events | 10 tests |
| `NFTHandler.Mint` | Validation → Service → Blockchain | 10 tests |
| `LotteryHandler.Create` | Validation → VRF → Blockchain → Save | 8 tests |

### 5.2 Priority 2: Query Handlers (Target: 80%+)

| Handler | Critical Paths | Test Count Target |
|---------|----------------|-------------------|
| `TokenHandler.Balance` | Param validation → Service → Response | 5 tests |
| `TokenHandler.History` | Pagination → Service → Response | 5 tests |
| `LotteryHandler.Get` | Not found → Found | 3 tests |
| `LotteryHandler.History` | Empty → With data | 3 tests |

### 5.3 Quick Win Tests (Add These First)

```go
// Test all handlers return correct content-type
func TestAllHandlers_ContentType(t *testing.T) {
	handlers := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"token create", tokenHandler.Create},
		{"token mint", tokenHandler.Mint},
		{"lottery create", lotteryHandler.Create},
		// ... add all handlers
	}

	for _, tt := range handlers {
		t.Run(tt.name, func(t *testing.T) {
			req, rr := testRequest(t, http.MethodPost, "/test", []byte("{}"))
			tt.handler(rr, req)
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
		})
	}
}
```

---

## 6. Test Execution Commands

```bash
# Run handler tests with coverage
go test -cover -coverprofile=handler_coverage.out ./internal/api/handler/...

# Run TUI tests
go test ./internal/ui/...

# Run with verbose output
go test -v ./internal/api/handler/...

# Run specific test file
go test -v -run TestTokenHandler_Create_TableDriven ./internal/api/handler/...

# Generate HTML coverage report
go tool cover -html=handler_coverage.out -o coverage.html

# Run all tests with race detection
go test -race ./...
```

---

## 7. Key Takeaways

1. **Use `httptest.NewRecorder()`** for all handler tests - captures response for assertions
2. **Create typed mock structs** with function fields for flexible behavior configuration
3. **Test through the public interface** - handlers expose `http.HandlerFunc`, test at that level
4. **Table-driven tests** reduce boilerplate and ensure comprehensive scenario coverage
5. **For TUI: test `Update()` and `View()` separately** - model logic is testable without rendering
6. **Verify response structure**, not just status codes - use JSON parsing helpers
7. **Add cleanup functions** with `t.Cleanup()` for resource management

---

## Sources

- Go Standard Library: `net/http/httptest` (pkg.go.dev)
- Go Testing Patterns: golang-testing skill
- Chi Router: github.com/go-chi/chi/_examples
- BubbleTea: charmbracelet/bubbletea testing patterns
- Testify: github.com/stretchr/testify
