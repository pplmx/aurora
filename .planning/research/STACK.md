# Stack Research

**Date:** 2026-04-26
**Milestone:** v1.1 Production Hardening

## Stack Additions for v1.1

### REST API Server

| Library | Purpose | Version |
|---------|---------|---------|
| gin-gonic/gin | HTTP router | v1.9+ |
| gorilla/mux | Alternative router | v1.8+ |
| chi | Lightweight router | v5+ |

**Recommendation:** Use existing `internal/api/` handlers with gin for routing.

### Web UI

| Library | Purpose | Version |
|---------|---------|---------|
| Vanilla HTML/JS | Simple option | - |
| HTMX | Interactive without SPA | 1.9+ |
| Alpine.js | Lightweight reactivity | 3.x |
| React/Vue | Full SPA | Latest |

**Recommendation:** HTMX + Alpine.js for minimal footprint, matches CLI aesthetic.

### Database Migrations

| Library | Purpose | Version |
|---------|---------|---------|
| golang-migrate/migrate | CLI migrations | v4+ |
| pressly/goose | SQL-first migrations | v3+ |
| rubenv/sql-migrate | XML/SQL migrations | - |

**Recommendation:** golang-migrate for Go-native integration.

### Backup/Restore

| Approach | Description |
|----------|-------------|
| SQLite backup | `sqlite3 .backup` command |
| pg_dump style | Custom export to JSON |
| tar + compression | Bundle db + state |

**Recommendation:** JSON export for portability, tar.gz for full backup.

## What's NOT Being Added
- PostgreSQL (keep SQLite for now)
- GraphQL (REST is sufficient)
- gRPC (overkill for this project)

## Integration Points
- API routes: `cmd/api/` (already exists)
- Web static: `internal/ui/web/` or `web/` directory
- Migrations: `migrations/` directory
- Config: Use existing Viper config