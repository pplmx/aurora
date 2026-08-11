# Requirements: Aurora v1.5 Fresh-Install Operations & Coverage Bar

**Status:** Complete ✅
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

- [x] **MIG-01**: `aurora migrate up [N]` applies pending migrations (default: all)
- [x] **MIG-02**: `aurora migrate down [N]` rolls back N steps (default: 1)
- [x] **MIG-03**: `aurora migrate status` prints current version, dirty flag,
      and applied + pending migration versions
- [x] **MIG-04**: DB path = `blockchain.DBPath()` (./data/aurora.db — the same file
      the rest of the CLI addresses; equals config `db.path` in production);
      migration path from `migrate.path` (default `./migrations`) — see decision-migrate-cli-dbpath
- [x] **MIG-05**: subcommand covered by tests — happy + error paths
      (already-migrated, missing migrations dir, invalid N, overrun caps)

### Coverage Bar — Infrastructure

- [x] **COV-01**: `internal/logger` ≥ 80% (**96.6%**)
- [x] **COV-02**: `internal/i18n` ≥ 80% (**97.0%**)
- [x] **COV-03**: `internal/infra/backup` ≥ 80% (**80.5%**)

### Coverage Bar — UI

- [x] **COV-04**: `internal/ui/nft` ≥ 80% (**94.2%**, surfaced issue-nft-tui-transfer-guard)
- [x] **COV-05**: `internal/ui/token` ≥ 80% (**88.6%**)
- [x] **COV-06**: No existing tests weakened or deleted; whole suite green
      under `go test -race ./...`

## Out of Scope

- `cmd/api`/`cmd/aurora` `main()` bodies (thin process boots; exit paths not
  unit-testable in-process)
- Binary-level e2e harness for the migrate command (service-level e2e already
  covers workflows; command coverage is via cobra CLI tests in
  `cmd/aurora/cmd`)
- `task-nft-voting-transactions` accepted debt (no touchpoint trigger this
  milestone)
