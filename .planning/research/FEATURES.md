# Features Research

**Date:** 2026-04-26
**Milestone:** v1.1 Production Hardening

## Feature Analysis

### 1. REST API Server

**Table stakes:**
- CRUD endpoints for lottery, voting, NFT, token operations
- JSON request/response format
- Basic auth or API key authentication

**Differentiators:**
- WebSocket for real-time updates (lottery draws, vote counts)
- OpenAPI/Swagger documentation
- Rate limiting per client

**Expected behaviors:**
- `GET /api/v1/lottery` - list lotteries
- `POST /api/v1/lottery/create` - create lottery
- `GET /api/v1/token/balance/:owner` - check balance
- `POST /api/v1/token/transfer` - transfer tokens

### 2. Oracle Real Data

**Table stakes:**
- Configurable data sources (HTTP APIs)
- Data validation before storage
- Error handling for failed fetches

**Differentiators:**
- Multiple source aggregation
- Data transformation rules
- Historical data query

**Expected behaviors:**
- Fetch from configured URLs
- Parse JSON response
- Store validated data
- Expose via query API

### 3. Web UI

**Table stakes:**
- Login/authentication
- Dashboard showing system state
- Basic CRUD forms

**Differentiators:**
- Real-time lottery drawing visualization
- Live voting tally
- Mobile-responsive design

**Expected behaviors:**
- Browser-based access (not CLI)
- Same operations as CLI
- Better data visualization

### 4. Database Migrations

**Table stakes:**
- Create/alter tables safely
- Version tracking
- Rollback capability

**Differentiators:**
- Zero-downtime migrations
- Data migration scripts
- Migration testing

**Expected behaviors:**
- Version table tracks current state
- `migrate up` / `migrate down` commands
- Schema diff generation

### 5. Backup/Restore

**Table stakes:**
- Full database backup
- Point-in-time restore
- Backup verification

**Differentiators:**
- Incremental backups
- Cloud storage integration
- Encryption at rest

**Expected behaviors:**
- `aurora backup create` - create backup
- `aurora backup restore <file>` - restore
- `aurora backup list` - show backups

## Complexity Notes
- API server: Medium (reuse existing handlers)
- Oracle: Low-Medium (wire up existing HTTP fetcher)
- Web UI: Medium-High (separate frontend)
- Migrations: Low (small schema)
- Backup: Low (SQLite backup straightforward)
