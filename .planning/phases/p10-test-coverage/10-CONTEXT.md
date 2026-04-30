# Phase 10: Test Coverage - Context

**Gathered:** 2026-04-30
**Status:** Ready for planning
**Mode:** Auto-generated (infrastructure phase)

<domain>
## Phase Boundary

Achieve ≥80% test coverage for API handlers and add TUI package tests:
1. Token API handler tests (≥80% coverage)
2. NFT API handler tests (≥80% coverage)
3. Lottery API handler tests (≥80% coverage)
4. Voting API handler tests (≥80% coverage)
5. Oracle API handler tests (≥80% coverage)
6. TUI package unit tests for model logic

</domain>

<decisions>
## Implementation Decisions

### Agent's Discretion
- Use table-driven tests for handler scenarios
- Create mock repositories for testing
- Test both success and error paths
- Focus on high-coverage items first

</decisions>

<codebase>
## Existing Code Insights

### Current Handler Coverage
- Token handler: low coverage
- NFT handler: low coverage
- Lottery handler: exists, needs improvement
- Voting handler: exists, needs improvement
- Oracle handler: low coverage

### Testing Pattern
Uses httptest.NewRecorder for HTTP testing
Chi router used for URL params

### Key Files
- internal/api/handler/*.go
- internal/ui/*/tui.go (TUI models)

</codebase>

<specifics>
## Specific Ideas

Per research/TESTING.md:
- Create mock repos with function fields
- Use testRequest() helper with Chi params
- Table-driven tests for multiple scenarios
- Test Update() logic separately from View()

</specifics>

<deferred>
## Deferred Ideas

None — infrastructure phase, no scope for new features.

</deferred>