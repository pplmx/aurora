# Roadmap: Aurora v1.3 Quality & Documentation

**Status:** Active
**Milestone:** v1.3 Quality & Documentation
**Phases:** 1-4
**Started:** 2026-04-30

## Overview

Comprehensive test coverage across UI, handler, and E2E layers with improved documentation.

## Phase 1: UI Package Tests

**Goal:** Achieve meaningful test coverage for all UI packages

**Requirements:** UI-01, UI-02, UI-03, UI-04, UI-05

**Success Criteria:**
- [ ] UI/lottery package ≥60% coverage
- [ ] UI/nft package ≥60% coverage
- [ ] UI/oracle package ≥60% coverage
- [ ] UI/token package ≥60% coverage
- [ ] UI/components package ≥50% coverage
- [ ] Theme and styling tests included
- [ ] TUI state machine tests where applicable

**Key Deliverables:**
- `internal/ui/*/*_test.go` files
- Component rendering tests
- State transition tests

---

## Phase 2: Handler Tests

**Goal:** Achieve 80%+ coverage for API handlers

**Requirements:** HND-01, HND-02, HND-03, HND-04

**Success Criteria:**
- [ ] API/handler package ≥80% coverage
- [ ] API package ≥60% coverage
- [ ] All error responses tested (404, 400, 401)
- [ ] Auth middleware with mock API keys
- [ ] Request validation tested
- [ ] JSON response format verified

**Key Deliverables:**
- `internal/api/handler/*_test.go` expanded
- `internal/api/*_test.go` integration tests
- Auth middleware test helpers

---

## Phase 3: E2E Tests

**Goal:** Comprehensive end-to-end workflow tests

**Requirements:** E2E-01, E2E-02, E2E-03, E2E-04, E2E-05, E2E-06

**Success Criteria:**
- [ ] Lottery create → view → history workflow
- [ ] Voting create → vote → tally workflow
- [ ] NFT mint → transfer → list workflow
- [ ] Token mint → transfer → balance workflow
- [ ] Oracle fetch → query workflow
- [ ] Error recovery scenarios tested
- [ ] Proper test isolation and cleanup

**Key Deliverables:**
- `e2e/*_test.go` workflows expanded
- Error injection tests
- Cleanup verification

---

## Phase 4: Documentation

**Goal:** Consistent and helpful CLI documentation

**Requirements:** DOC-01, DOC-02, DOC-03

**Success Criteria:**
- [ ] All subcommands have help text
- [ ] Help includes usage examples
- [ ] Root command lists available modules
- [ ] Consistent help format across commands

**Key Deliverables:**
- Updated command help strings
- Example output in help
- Root command overview

---

## Phase Summary

| Phase | Focus | Requirements | Criteria |
|-------|-------|--------------|----------|
| 1 | UI Tests | UI-01 to UI-05 | 5 packages covered |
| 2 | Handler Tests | HND-01 to HND-04 | 80% handler coverage |
| 3 | E2E Tests | E2E-01 to E2E-06 | All workflows covered |
| 4 | Documentation | DOC-01 to DOC-03 | Help improvements |

## Coverage Targets

| Package | Current | Phase 1 | Phase 2 |
|---------|---------|---------|---------|
| UI/lottery | 0% | 60%+ | - |
| UI/nft | 0% | 60%+ | - |
| UI/oracle | 0% | 60%+ | - |
| UI/token | 0% | 60%+ | - |
| UI/components | 0% | 50%+ | - |
| API/handler | 39.3% | - | 80%+ |
| API | 2.7% | - | 60%+ |

---

_For current project status, see .planning/STATE.md_