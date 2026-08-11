# Requirements: Aurora v1.5 Fresh-Install Operations & Coverage Bar

**Status:** In Progress
**Milestone:** v1.5 Fresh-Install Operations & Coverage Bar
**Last updated:** 2026-08-11

## Overview

Two goals.

1. **Fresh-install operations.** The migration engine (`internal/infra/migrate`)
   is fully implemented and tested (81.5% coverage; real checkout migrations
   apply cleanly), yet no CLI surface runs it: `migrate.autoRun` defaults to
   `false` and there is **no `aurora migrate` subcommand** — so the
   voting/NFT/token tables from `migrations/` are unreachable from the CLI on a
   fresh install, and `voting`, `nft`, and `token` commands all fail with
   `no such table`. v1.1's requirements mark `MIG-03 Migration CLI command` as
   complete, but it was never actually shipped in this tree. This milestone
   restores that documented surface.

2. **Coverage bar.** Five packages sit below the project's 80% quality bar:
   `internal/logger` (55.2%), `internal/i18n` (65.2%),
   `internal/infra/backup` (73.8%), `internal/ui/nft` (66.7%),
   `internal/ui/token` (76.6%). Lift all of them to ≥80% with genuine tests.

## Requirements

### Migrate CLI

- [ ] **MIG-01**: `aurora migrate up [N]` applies pending migrations (default: all)
- [ ] **MIG-02**: `aurora migrate down [N]` rolls back N steps (default: 1)
- [ ] **MIG-03**: `aurora migrate status` prints current version, dirty flag,
      and applied + pending migration versions
- [ ] **MIG-04**: DB path resolution matches root (`db.path`, else
      `data.dir/aurora.db`, else `~/.aurora/data/aurora.db`); migration path
      from `migrate.path` (default `./migrations`)
- [ ] **MIG-05**: subcommand covered by tests — happy + error paths
      (already-migrated, missing migrations dir, invalid N)

### Coverage Bar — Infrastructure

- [ ] **COV-01**: `internal/logger` ≥ 80% (55.2% baseline)
- [ ] **COV-02**: `internal/i18n` ≥ 80% (65.2% baseline)
- [ ] **COV-03**: `internal/infra/backup` ≥ 80% (73.8% baseline)

### Coverage Bar — UI

- [ ] **COV-04**: `internal/ui/nft` ≥ 80% (66.7% baseline)
- [ ] **COV-05**: `internal/ui/token` ≥ 80% (76.6% baseline)
- [ ] **COV-06**: No existing tests weakened or deleted; whole suite green
      under `go test -race ./...`

## Out of Scope

- `cmd/api`/`cmd/aurora` `main()` bodies (thin process boots; exit paths not
  unit-testable in-process)
- Binary-level e2e harness for the migrate command (service-level e2e already
  covers workflows; command coverage is via cobra CLI tests in
  `cmd/aurora/cmd`)
- `task-nft-voting-transactions` accepted debt (no touchpoint trigger this
  milestone)
