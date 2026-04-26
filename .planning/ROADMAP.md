# Roadmap: Aurora v1.1 Production Hardening

## Phase 1: Database Migrations
**Goal**: Foundation for schema changes and safe upgrades

**Requirements**: MIG-01, MIG-02, MIG-03, MIG-04

**Success Criteria**:
- [ ] `aurora migrate status` shows current version
- [ ] `aurora migrate up` applies pending migrations
- [ ] `aurora migrate down` rolls back one version
- [ ] `aurora migrate` runs on startup (if configured)
- [ ] Migration tests exist and pass

**Complexity**: Low
**Approach**: Use golang-migrate with SQL files

---

## Phase 2: REST API Server
**Goal**: Programmatic access to all Aurora modules

**Requirements**: API-01, API-02, API-03, API-04, API-05, API-06, API-07

**Success Criteria**:
- [ ] GET/POST endpoints for lottery (create, list, history)
- [ ] GET/POST endpoints for voting (create proposal, cast vote)
- [ ] GET/POST endpoints for NFT (mint, transfer, list)
- [ ] GET/POST endpoints for token (transfer, balance, history)
- [ ] JSON error responses with consistent format
- [ ] API key authentication working
- [ ] API tests cover all endpoints

**Complexity**: Medium
**Approach**: Reuse internal/app/* handlers, add gin router

---

## Phase 3: Oracle Real Data
**Goal**: Replace mock data with actual HTTP fetching

**Requirements**: ORA-01, ORA-02, ORA-03, ORA-04

**Success Criteria**:
- [ ] Oracle fetches real data from configured sources
- [ ] Source URLs configurable per source
- [ ] Response validation before storage
- [ ] Graceful error handling (timeout, HTTP errors)
- [ ] Tests with mock HTTP server

**Complexity**: Low-Medium
**Approach**: Wire up existing HTTP fetcher, add config

---

## Phase 4: Web UI
**Goal**: Browser-based interface for Aurora operations

**Requirements**: WEB-01, WEB-02, WEB-03, WEB-04, WEB-05

**Success Criteria**:
- [ ] Dashboard page with system overview
- [ ] Lottery interface (create, view)
- [ ] Voting interface (create, vote)
- [ ] Responsive design (mobile-friendly)
- [ ] HTMX for dynamic content (no full reload)
- [ ] Web UI served by API or standalone

**Complexity**: Medium-High
**Approach**: HTML + HTMX + Alpine.js, served statically

---

## Phase 5: Backup/Restore
**Goal**: Data safety and recovery capability

**Requirements**: BCK-01, BCK-02, BCK-03, BCK-04

**Success Criteria**:
- [ ] `aurora backup create` creates backup file
- [ ] JSON export format for portability
- [ ] `aurora backup verify` checks integrity
- [ ] `aurora backup restore` restores from file
- [ ] Backup tests exist

**Complexity**: Low
**Approach**: SQLite .backup + JSON export

---

## Timeline

| Phase | Focus | Priority Order |
|-------|-------|----------------|
| Phase 1 | Migrations | 1st - Foundation |
| Phase 2 | API Server | 2nd - Core |
| Phase 3 | Oracle | 3rd - Fix |
| Phase 4 | Web UI | 4th - User-facing |
| Phase 5 | Backup | 5th - Ops |

---

## Next Steps
Run `/gsd-plan-phase 1` to start Phase 1