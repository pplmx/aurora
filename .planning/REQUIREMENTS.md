# Requirements: Aurora v1.4 CLI Command Test Coverage

**Status:** Active
**Milestone:** v1.4 CLI Command Test Coverage
**Last updated:** 2026-08-11

## Overview

`cmd/aurora/cmd` is the primary user-facing surface of Aurora — every
CLI interaction flows through its cobra command tree — yet it sits at
**21.9% statement coverage** (measured 2026-08-11). Only small helpers
and `init()` wiring are tested; every `RunE` body, the lazy
DB/service constructors, and the root config path are untested, so
flag-parsing, output-formatting, and wiring bugs go undetected.

## Requirements

### Coverage

- [ ] **CLI-01**: `cmd/aurora/cmd` package reaches 80%+ test coverage
- [ ] **CLI-02**: Every lottery subcommand tested (`create`, `history`,
      `verify`, `export`, `import`, `stats`, `reset`, `db-info`, `version`)
- [ ] **CLI-03**: Every NFT subcommand tested (`mint`, `transfer`,
      `burn`, `get`, `list`, `history`) — exercises lazy `getNFTRepo` path
- [ ] **CLI-04**: Every oracle subcommand tested (`source add/list/delete/
      enable/disable`, `fetch`, `data`, `latest`, `template list/add`)
- [ ] **CLI-05**: Every token subcommand tested (`create`, `mint`,
      `transfer`, `approve`, `burn`, `balance`, `allowance`, `history`,
      `info`, `tui`) — exercises `newTokenService` constructor
- [ ] **CLI-06**: Every voting subcommand tested (`candidate add/list`,
      `voter register/list`, `vote`, `session create/list/start/end`,
      `results`)
- [ ] **CLI-07**: Root command tested (`Execute`, `initConfig`,
      `setDefaultConfig`, `getGoVersion`)

### Quality

- [ ] **CLI-08**: Tests isolate state (temp dirs, reset of package
      singletons, no reliance on pre-existing `./data`)
- [ ] **CLI-09**: Tests run clean under `go test -race` for the package

## Out of Scope

- `cmd/api` / `cmd/aurora` `main()` bodies (thin process boots; exit
  paths not unit-testable in-process)
- TUI subcommands beyond stub coverage (interactive, need pty harness)
- Rewriting commands to inject output writers (larger refactor than the
  milestone warrants; stdout capture in tests is sufficient)
