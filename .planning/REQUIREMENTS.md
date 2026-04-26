# Requirements: Aurora v1.1 Production Hardening

## Goals & Success Criteria

1. **Production-ready API** — REST endpoints for all modules
2. **Real data oracle** — Wire up HTTP fetcher, remove mock
3. **Web interface** — Browser-based UI for operations
4. **Database migrations** — Version tracking, safe schema changes
5. **Backup/restore** — Data safety and recovery

## Requirements

### API Server

- [ ] **API-01**: REST endpoints for lottery operations
- [ ] **API-02**: REST endpoints for voting operations
- [ ] **API-03**: REST endpoints for NFT operations
- [ ] **API-04**: REST endpoints for token operations
- [ ] **API-05**: JSON request/response format
- [ ] **API-06**: API key authentication
- [ ] **API-07**: Standard error responses

### Oracle

- [ ] **ORA-01**: Wire up HTTP fetcher for real data
- [ ] **ORA-02**: Data source configuration
- [ ] **ORA-03**: Response validation
- [ ] **ORA-04**: Error handling for failed fetches

### Web UI

- [ ] **WEB-01**: Dashboard showing system state
- [ ] **WEB-02**: Lottery operations interface
- [ ] **WEB-03**: Voting interface
- [ ] **WEB-04**: Responsive design
- [ ] **WEB-05**: HTMX for interactivity

### Database Migrations

- [ ] **MIG-01**: Migration version tracking
- [ ] **MIG-02**: Up/Down migration support
- [ ] **MIG-03**: Migration CLI command
- [ ] **MIG-04**: Automatic migration on startup

### Backup/Restore

- [ ] **BCK-01**: Full database backup
- [ ] **BCK-02**: JSON export format
- [ ] **BCK-03**: Backup verification
- [ ] **BCK-04**: Backup restore command

## Out of Scope

- OAuth/auth providers (API key sufficient for now)
- GraphQL (REST is sufficient)
- Incremental backups (full backup only)
- Cloud storage (local filesystem)
- PostgreSQL migration (stay with SQLite)

## Constraints

- Quality over speed
- Test-first approach
- Maintain backward compatibility
- Reuse existing patterns
- API reuses CLI handlers (don't duplicate logic)