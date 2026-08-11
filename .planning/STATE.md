# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-11)

**Core value:** Complete, production-ready blockchain toolkit with comprehensive test coverage and operational tooling
**Current focus:** v1.5 Fresh-Install Operations & Coverage Bar (IN PROGRESS)

## Current Position

Phase: v1.5 Phases 1-3 (migrate CLI impl done; coverage bar complete)
Plan: [ROADMAP.md](ROADMAP.md) — 3 phases (migrate CLI, coverage-bar infra, coverage-bar UI)
Status: Phases 2-3 complete; Phase 1 waits on code review before commit
Last activity: 2026-08-11 — v1.4 completed; v1.5 kicked off

Progress: [██████████░] 90%

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
| `task-migrate-cli-subcommand` — `aurora migrate up/status/down` | 1.91 | in review → commit |
| `task-coverage-bar-infra` — logger **96.6%**, i18n **97.0%**, backup **80.5%** | 0.98 | ✅ resolved |
| `task-coverage-bar-ui` — ui/nft **94.2%**, ui/token **88.6%** | 0.85 | ✅ resolved |

Deferred: `task-nft-voting-transactions` (0.28, accepted debt).

## Coverage Snapshot (2026-08-11)

Coverage bar MET (2026-08-11): logger 96.6%, i18n 97.0%, backup 80.5%,
ui/nft 94.2%, ui/token 88.6%, app 100%, cmd/aurora/cmd 86.6%.

Documented thin boots (out of scope): `cmd/aurora`, `cmd/api` (0%);
`e2e` is a test-only package (0% statements, suite green).

## Session Continuity

Last session: 2026-08-11 — v1.4 CLI coverage milestone complete
Next: v1.5 Phase 1 — implement `aurora migrate` subcommand + tests
