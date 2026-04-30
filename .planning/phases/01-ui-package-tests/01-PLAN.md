# Phase 1 Plan: UI Package Tests

**Phase:** 1/4 — UI Package Tests
**Status:** Ready for execution
**Created:** 2026-04-30

## Objective
Achieve meaningful test coverage for all UI packages using Go standard testing with testify for assertions.

## Requirements
- UI/lottery: 60%+ coverage
- UI/nft: 60%+ coverage  
- UI/oracle: 60%+ coverage
- UI/token: 60%+ coverage
- UI/components: 50%+ coverage

## Tasks

### 01-test-ui-packages.md
Write tests for all UI TUI packages covering:
- Model initialization
- View state transitions
- Message handling (key presses)
- Error state rendering
- Theme/style application

### 02-test-components.md
Write tests for UI components package:
- Theme styles render correctly
- Menu item styling

### 03-run-coverage.md
Run coverage report and verify targets met

## Implementation Approach
1. Use Go standard testing with `testing` package
2. Use `testify/assert` for assertions
3. Mock bubbletea `tea.Msg` types for message handling tests
4. Test view output string patterns for rendering tests
5. Skip blockchain initialization in tests (mock or avoid)
