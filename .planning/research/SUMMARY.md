# Research Summary

**Date:** 2026-04-26
**Milestone:** v1.1 Production Hardening

## Key Findings

### Stack Additions
| Feature | Recommended Stack |
|---------|-------------------|
| REST API | gin-gonic/gin |
| Web UI | HTMX + Alpine.js |
| Migrations | golang-migrate |
| Backup | SQLite `.backup` + JSON export |

### Feature Table Stakes
- **API**: CRUD endpoints, JSON, basic auth
- **Oracle**: Configurable sources, validation, error handling
- **Web**: Dashboard, forms, real-time updates
- **Migrations**: Version tracking, up/down, rollback
- **Backup**: Create, restore, verify

### Watch Out For
1. **Don't replicate logic** — API reuses CLI handlers
2. **Start simple auth** — API key, not OAuth
3. **HTMX not SPA** — Keep frontend minimal
4. **Always test rollback** — Migration down
5. **Verify backups** — Test restore

### Build Order
1. Migrations (foundation)
2. API Server (core feature)
3. Oracle Real Data
4. Web UI
5. Backup/Restore

### Complexity
- API Server: Medium
- Oracle: Low-Medium
- Web UI: Medium-High
- Migrations: Low
- Backup: Low
