# Roadmap: Aurora v1.5 Fresh-Install Operations & Coverage Bar

**Status:** In Progress
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
- [ ] `aurora migrate up/status/down` registered under root
- [ ] Real checkout migrations apply via the command against a fresh temp DB
- [ ] Cobra CLI tests cover output + error paths (`no migrations`, invalid N,
      already-migrated); `-race` green
- [ ] Graph: `task-migrate-cli-subcommand` → resolved; `issue-cli-no-migrate-subcommand` updated

---

## Phase 2: Coverage Bar — Infrastructure

**Goal:** Remaining infra packages meet the 80% bar.

**Requirements:** COV-01 .. COV-03

**Success Criteria:**
- [ ] `internal/logger` ≥ 80% (55.2% → …)
- [ ] `internal/i18n` ≥ 80% (65.2% → …)
- [ ] `internal/infra/backup` ≥ 80% (73.8% → …)
- [ ] Graph: `task-coverage-bar-infra` → resolved

---

## Phase 3: Coverage Bar — UI

**Goal:** Remaining UI packages meet the 80% bar using the established
TUI state-machine test pattern.

**Requirements:** COV-04 .. COV-06

**Success Criteria:**
- [ ] `internal/ui/nft` ≥ 80% (66.7% → …)
- [ ] `internal/ui/token` ≥ 80% (76.6% → …)
- [ ] No weakened tests; `go test -race ./...` green
- [ ] Graph: `task-coverage-bar-ui` → resolved

---

## Completion Gate

- [ ] `go test -race ./...` green
- [ ] Every package ≥ 80% except documented thin boots
  (`cmd/aurora`, `cmd/api` main bodies)
- [ ] `aurora migrate` exercised end-to-end against real migrations on a fresh DB
- [ ] v1.4 archived ✓; graph updated with evidence + change nodes
