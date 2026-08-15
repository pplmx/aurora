# Roadmap: Aurora v1.5 Fresh-Install Operations & Coverage Bar

**Status:** Complete ✅
**Milestone:** v1.5 Fresh-Install Operations & Coverage Bar
**Phases:** 1-3
**Started:** 2026-08-11
**Requirements:** [REQUIREMENTS.md](REQUIREMENTS.md)

## Overview

Make a fresh install actually usable (`aurora migrate` CLI — restore the
v1.1-documented MIG-03 surface) and lift every package over the 80% quality
bar.

## Phase 1: Migrate CLI Command

**Goal:** A fresh install can create the full schema from the CLI.

**Requirements:** MIG-01 .. MIG-05

**Success Criteria:**

- [x] `aurora migrate up/status/down` registered under root
- [x] Real checkout migrations apply via the command against a fresh temp DB
      (binary e2e: up 5 caps at 2 pending, down 5 to base)
- [x] Cobra CLI tests cover output + error paths (`no migrations`, invalid N,
      already-migrated, overrun caps); `-race` green
- [x] Graph: `task-migrate-cli-subcommand` → resolved; `issue-cli-no-migrate-subcommand` resolved (4e3c9ba)

---

## Phase 2: Coverage Bar — Infrastructure

**Goal:** Remaining infra packages meet the 80% bar.

**Requirements:** COV-01 .. COV-03

**Success Criteria:**

- [x] `internal/logger` ≥ 80% (**96.6%**)
- [x] `internal/i18n` ≥ 80% (**97.0%**)
- [x] `internal/infra/backup` ≥ 80% (**80.5%**)
- [x] Graph: `task-coverage-bar-infra` → resolved (f60a206)

---

## Phase 3: Coverage Bar — UI

**Goal:** Remaining UI packages meet the 80% bar using the established
TUI state-machine test pattern.

**Requirements:** COV-04 .. COV-06

**Success Criteria:**

- [x] `internal/ui/nft` ≥ 80% (**94.2%**; also fixed issue-nft-tui-transfer-guard)
- [x] `internal/ui/token` ≥ 80% (**88.6%**)
- [ ] No weakened tests; `go test -race ./...` green
- [x] Graph: `task-coverage-bar-ui` → resolved (d9867bf, aa15d46)

---

## Completion Gate

- [x] `go test -race ./...` green
- [x] Every package ≥ 80% except documented thin boots
      (cmd/aurora 0%, cmd/api 0% mains; e2e test-only 0%)
  (`cmd/aurora`, `cmd/api` main bodies)
- [x] `aurora migrate` exercised end-to-end against real migrations on a fresh DB
- [x] v1.4 archived ✓; graph updated with evidence + change nodes
