# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-11)

**Core value:** Complete, production-ready blockchain toolkit with comprehensive test coverage and operational tooling
**Current focus:** v1.5 Fresh-Install Operations & Coverage Bar (IN PROGRESS)

## Current Position

Phase: v1.5 Phase 1 (Migrate CLI Command)
Plan: [ROADMAP.md](ROADMAP.md) — 3 phases (migrate CLI, coverage-bar infra, coverage-bar UI)
Status: Milestone v1.5 started; Phase 1 beginning
Last activity: 2026-08-11 — v1.4 completed; v1.5 kicked off

Progress: [██░░░░░░░░] 10%

## v1.4 Completion Summary

| Phase | Focus | Result |
|-------|-------|--------|
| 1 | Oracle + Root + Helpers | ✅ covered |
| 2 | Lottery Commands | ✅ covered |
| 3 | Token + NFT Commands | ✅ covered |
| 4 | Voting Commands | ✅ covered |
| — | Final coverage | ✅ 21.9% → **86.3%** (`-race` green) |

v1.4 also surfaced and fixed latent product bugs: migrations never applied,
voting CLI flags broken (session/vote), lottery reset kept history,
lottery verify ID-vs-height confusion.

## v1.5 Backlog (from graph)

| Task | Priority | Status |
|------|----------|--------|
| `task-migrate-cli-subcommand` — `aurora migrate up/status/down` | 1.91 | active |
| `task-coverage-bar-infra` — logger/i18n/backup → ≥80% | 0.98 | active |
| `task-coverage-bar-ui` — ui/nft + ui/token → ≥80% | 0.85 | active |

Deferred: `task-nft-voting-transactions` (0.28, accepted debt).

## Coverage Snapshot (2026-08-11)

Below the 80% bar: `internal/logger` 55.2%, `internal/i18n` 65.2%,
`internal/infra/backup` 73.8%, `internal/ui/nft` 66.7%,
`internal/ui/token` 76.6%.

Documented thin boots (out of scope): `cmd/aurora`, `cmd/api` (0%);
`e2e` is a test-only package (0% statements, suite green).

## Session Continuity

Last session: 2026-08-11 — v1.4 CLI coverage milestone complete
Next: v1.5 Phase 1 — implement `aurora migrate` subcommand + tests
