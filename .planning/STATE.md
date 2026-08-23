# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-11)

**Core value:** Complete, production-ready blockchain toolkit with comprehensive test coverage and operational tooling
**Current focus:** v1.15 Oracle Scheduled Fetching complete; exploring next milestone

## Current Position

Phase: v1.5 Complete (migrate CLI + coverage bar)
Plan: [v1.6 ROADMAP](milestones/v1.6-ROADMAP.md) — Interactive Surface Parity
Status: v1.5–v1.15 complete (Oracle Scheduled Fetching: interval scheduler + restart-safe last-fetch seeding)
Last activity: 2026-08-23 — v1.8–v1.13 closed; deep-dive rounds fixed a
  blockchain concurrency bug and hardened the metrics middleware
  (Flusher/Hijacker/Push/Unwrap); opened v1.14 Observability Enrichment and
  delivered TASK-020 per-module request metrics

Progress: v1.5 [██████████] 100%; v1.6 [█████░░░░░] 50%

## v1.4 Completion Summary

| Phase | Focus                   | Result                               |
| ----- | ----------------------- | ------------------------------------ |
| 1     | Oracle + Root + Helpers | ✅ covered                           |
| 2     | Lottery Commands        | ✅ covered                           |
| 3     | Token + NFT Commands    | ✅ covered                           |
| 4     | Voting Commands         | ✅ covered                           |
| —     | Final coverage          | ✅ 21.9% → **86.3%** (`-race` green) |

v1.4 also surfaced and fixed latent product bugs: migrations never applied,
voting CLI flags broken (session/vote), lottery reset kept history,
lottery verify ID-vs-height confusion.

## v1.5 Backlog (from graph)

| Task                                                                           | Priority | Status                |
| ------------------------------------------------------------------------------ | -------- | --------------------- |
| `task-migrate-cli-subcommand` — `aurora migrate up/status/down`                | 1.91     | ✅ resolved (4e3c9ba) |
| `task-coverage-bar-infra` — logger **96.6%**, i18n **97.0%**, backup **80.5%** | 0.98     | ✅ resolved           |
| `task-coverage-bar-ui` — ui/nft **94.2%**, ui/token **88.6%**                  | 0.85     | ✅ resolved           |

Deferred: `task-nft-voting-transactions` (0.28, accepted debt).

All v1.5 phases complete (2026-08-11). Only documented thin boots
(cmd/aurora, cmd/api mains) and the e2e test-only package remain under
80%; every deliverable package is ≥80%.

## Coverage Snapshot (2026-08-11)

Coverage bar MET (2026-08-11): logger 96.6%, i18n 97.0%, backup 80.5%,
ui/nft 94.2%, ui/token 88.6%, app 100%, cmd/aurora/cmd 86.6%.

Documented thin boots (out of scope): `cmd/aurora`, `cmd/api` (0%);
`e2e` is a test-only package (0% statements, suite green).

## Session Continuity

Last session: 2026-08-11 — v1.4 CLI coverage milestone complete
Next: evaluate new milestone (current accepted-debt backlog is below threshold)
