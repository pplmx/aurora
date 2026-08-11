# Graph Schema Reference

The graph is a JSON document at `.planning/graph.json`. State lives there; this file only documents the contract. Round every write with an optimistic lock based on `updated_at` (retry + diff-merge on conflict; evidence nodes are append-only so they never conflict).

## Nodes

Every node has: `id`, `type`, `status`, `created_at`, `updated_at`, `confidence` (0–1).

| type | purpose |
|------|---------|
| `component` | module / service / file-level entity |
| `issue` | identified problem: bug, risk, debt (carries `severity` 0–1) |
| `hypothesis` | unverified root-cause guess (carries `confidence` 0–1) |
| `evidence` | a specific observation supporting/refuting a hypothesis (cite commit hash / test name / file:line) |
| `decision` | a made choice, with `rationale` and `alternatives_rejected`; never deleted |
| `change` | an actual code modification, linked to its commit hash |
| `task` | an actionable next step, carrying `priority_score` |

## Edges

Directed, semantically typed — no untyped "related" edges.

| edge | meaning |
|------|---------|
| `depends_on` | task→task or component→component |
| `causes` | issue→issue (mark root-cause vs symptom); also issue→task ("derived") and issue→decision ("accepted") |
| `blocks` | task→task |
| `validates` / `refutes` | evidence→hypothesis |
| `resolves` | change→issue (or change→task) |
| `supersedes` | decision→decision (records evolution; the older decision is kept, not overwritten) |

## Lifecycle

`status` values: `active`, `stale`, `resolved`, `superseded`, `abandoned`.

- A hypothesis/task untouched for ~10 rounds is marked `stale` (kept for audit); EVALUATE skips stale nodes unless new evidence reactivates them.
- Decisions are never deleted — only superseded.
- Run a consistency check (orphaned nodes, cyclic `depends_on`, long-open `blocks` edges) every ~50 commits or weekly, as a concrete task.

## Scoring

`priority_score = category_weight × severity × confidence × (1/√effort) × unlock_factor`

- `category_weight`: correctness/security 10, stability/key-bug 8, core feature 6, performance 5, test quality 4, maintainability 3, developer-experience 2, docs 1 (adjustable per repo).
- `severity`: impact × trigger probability (0–1).
- `confidence`: how much `validates` evidence backs the linked root-cause; unverified hypotheses are discounted.
- `effort`: estimated cost, so trivial high-weight work doesn't dominate.
- `unlock_factor`: value/count of downstream tasks unlocked.

Switch threshold: only switch from the working task when a new one scores ≥ 1.5× the current; record the reason as a `decision`.

## Counting: Deep-Dive Budget

- ≤ 3 new task nodes per deep-dive round (more → scope didn't converge; merge/classify first).
- Single commit diff ≤ 300 lines by default (larger refactors need a `decision` note).
- If two consecutive deep-dive rounds yield only tasks below the 3.0 threshold, stop deep-diving and evalute stop conditions.

## Concurrency

- Optimistic lock node writes on `updated_at`; on conflict retry and diff-merge — never overwrite blindly.
- Task node `status=in_progress` + `owner=instance_id` is the distributed lock before executing on a component; timeout 30 min.
- Evidence nodes are append-only — always record new observations as new evidence nodes.
