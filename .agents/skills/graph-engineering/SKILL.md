---
name: graph-engineering
description: Use when advancing this repository through its long-running autonomous engineering loop — deciding what to work on next, recording findings/issues/decisions into the typed engineering graph, scoring task priority, or when a fresh session must load the active-task backlog from `.planning/graph.json`. Also use before any deep-dive or refactor round in aurora that should leave an auditable decision trail.
---

# Graph Engineering (Autonomous Loop)

## Overview

Aurora's autonomous engineering loop is driven by a typed knowledge graph at `.planning/graph.json` (full schema in `graph-schema.md`). Every round runs **OBSERVE → MODEL → EVALUATE → SELECT → EXECUTE → VERIFY → LEARN**. The graph is the durable cross-session memory: read it at session start, write to it before finishing a round. Findings that exist only in chat are lost.

## When to Use / Not Use

**Use when:**

- Deciding what to work on next in this repo.
- Recording a bug/risk (issue), a root-cause guess (hypothesis), a concrete observation (evidence), a decision, a change, or an actionable task.
- Continuing a session: load ACTIVE tasks and their 1–2 hop subgraph first.
- Wrapping up a round: confirm the stop conditions against the graph.

**Not for:** one-off edits that need no cross-session continuation.

## The Loop

1. **OBSERVE** — git status/log/diff, code and data flow, tests/coverage, config, roadmap. Look beyond isolated TODOs: understand components, APIs, runtime behavior.
2. **MODEL** — read/write the graph following `graph-schema.md`. Never keep findings as free-text notes only.
3. **EVALUATE** — score every task: `priority_score = category_weight × severity × confidence × (1/√effort) × unlock_factor`.
4. **SELECT** — pick the highest-scoring ACTIVE task; switch only when a new task scores ≥ 1.5× the current one, and record the switch as a decision.
5. **EXECUTE** — repo-local, git-revertible work only. Lock the task node first (`status=in_progress`, `owner=instance_id`).
6. **VERIFY** — build, vet, `go test -race ./...`. Never delete, skip, or weaken tests. If a fix can't be verified, roll back and mark the root-cause hypothesis refuted with evidence.
7. **LEARN** — append nodes/edges; evidence is append-only; decisions are never deleted (supersede instead). Reference node ids in commit messages so code and graph cross-trace.

## Hard Rules

- A hypothesis with no `validates`/`refutes` evidence is never treated as fact.
- Evidence nodes are append-only (new evidence over editing old nodes).
- Two agents never execute the same component concurrently — the task lock is the distributed lock; 30 min without an update releases it.
- Stale nodes (untouched ~10 rounds) are skipped by EVALUATE, never deleted.

## Stop Conditions

A session/round stops only when **all** hold:

1. No ACTIVE task with `priority_score` above the threshold (3.0).
2. Every high-severity issue is resolved, or covered by an explicit decision recording the deferral reason.
3. Last VERIFY is fully green.
4. Two consecutive deep-dive rounds produced nothing above threshold.
5. Graph consistency check (dangling edges / cycles) is clean.

## Common Mistakes

- Skipping the MODEL step and leaving findings as prose → future sessions can't load them.
- Writing the graph as untyped notes instead of typed nodes/edges.
- Executing on a shared component without taking the task lock.
- Scoring tasks without evidence behind the root-cause confidence.
- Committing without referencing the node ids.

Read `graph-schema.md` for the full node/edge types, statuses, scoring weights, and concurrency semantics.
