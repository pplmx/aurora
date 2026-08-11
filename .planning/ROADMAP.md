# Roadmap: Aurora v1.4 CLI Command Test Coverage

**Status:** Active
**Milestone:** v1.4 CLI Command Test Coverage
**Phases:** 1-4
**Started:** 2026-08-11

## Overview

Lift `cmd/aurora/cmd` from 21.9% to 80%+ coverage by testing every
cobra subcommand across lottery, NFT, oracle, token, voting and root.

## Phase 1: Oracle + Root + Helpers

**Goal:** Fast wins — in-memory-repo commands plus pure helpers.

**Requirements:** CLI-04, CLI-07

**Success Criteria:**
- [ ] Oracle commands: source add/list/delete/enable/disable, fetch, data, latest, template list/add
- [ ] Root helpers: `getGoVersion`, `setDefaultConfig` defaults, `initConfig` fallback path
- [ ] Command-tree structure assertion (all five module commands under root)

---

## Phase 2: Lottery Commands

**Goal:** DB-backed lottery commands, isolated in temp dirs.

**Requirements:** CLI-02, CLI-08

**Success Criteria:**
- [ ] create (happy + validation errors)
- [ ] history (empty + populated)
- [ ] verify (by height / id / substring / not found / corrupted record)
- [ ] export / import
- [ ] stats / reset / db-info / version

---

## Phase 3: Token + NFT Commands

**Goal:** Service-constructor and DB-backed commands.

**Requirements:** CLI-03, CLI-05

**Success Criteria:**
- [ ] Token create/mint/transfer/approve/burn/balance/allowance/history/info/tui
- [ ] `newTokenService` error path (forced UNIQUE/dir failure)
- [ ] NFT mint/transfer/burn/get/list/history via lazy `getNFTRepo`

---

## Phase 4: Voting Commands

**Goal:** Voting subcommands including singleton-reset safety.

**Requirements:** CLI-06

**Success Criteria:**
- [ ] candidate add/list
- [ ] voter register/list
- [ ] vote
- [ ] session create/list/start/end + results
- [ ] Concurrent `getVotingRepo`/`getVotingService` under `-race`

---

## Completion Gate

- [ ] `go test -race ./cmd/aurora/cmd/...` green
- [ ] Package coverage ≥ 80%
- [ ] v1.3 archive preserved; graph updated with evidence + change nodes
