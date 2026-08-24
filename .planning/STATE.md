# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-11)

**Core value:** Complete, production-ready blockchain toolkit with comprehensive test coverage and operational tooling
**Current focus:** v1.69 rate-limit spoof-bypass hardening complete

## Current Position

Phase: v1.5+ Continuous Deep-Dive Loop
Plan: Incremental milestones tracked in the RIL graph and git history
Status: v1.24–v1.69 complete (web/API/CLI parity, security hardening, observability, integrity, collision + extraction hardening, concurrency atomicity, event/state atomicity, rate-limit spoof hardening)
Last activity: 2026-08-24 — v1.69 closed: per-client rate limiting is keyed
  on the true socket peer captured by the new PeerIP middleware BEFORE chi's
  RealIP rewrites r.RemoteAddr from X-Forwarded-For / X-Real-IP /
  True-Client-IP; forwarded headers are believed only for peers on the new
  api.rateLimit.trustedProxies allow-list (default empty, fail-safe), so
  rotating spoofed headers no longer grants a fresh budget (TASK-083,
  ISS-073, CHG-081 / c2ee489). TASK-082 left active by the v1.68 close was
  also marked resolved (bookkeeping).
  RIL graph at round 79.

Progress: continuous loop — every resolved milestone advanced the graph;
  recent deep-dives closed a CRITICAL CORS/key-exfiltration flaw (v1.64), a
  baseline test-suite regression (v1.64), a silent backup data-loss path
  (v1.65), an NFT audit-history collapse (v1.66), a non-atomic nonce claim
  that broke under a real SQLite pool (v1.67), a phantom-event leak on
  token transaction rollback (v1.68), and a rate-limit spoof bypass via
  chi RealIP trusting client-supplied forwarded headers (v1.69).

## Milestone History (recent)

| Version | Focus | Result |
| ------- | ----- | ------ |
| v1.64 | CORS cross-origin key-exfiltration hardening | ✅ done |
| v1.65 | Backup self-overwrite guard | ✅ done |
| v1.66 | NFT operation audit-trail collapse | ✅ done |
| v1.67 | Atomic ClaimNextNonce under a real connection pool | ✅ done |
| v1.68 | No phantom events on token tx rollback | ✅ done |
| v1.69 | Rate-limit spoof bypass via trusted-proxy allow-list | ✅ done |

## Session Continuity

Last session: 2026-08-24 — v1.69 rate-limit spoof-bypass hardening complete
Next: continue graph-engineering deep-dive for the next milestone (backlog:
  backlog exhausted — fresh deep-dive to surface the next issue/task)
