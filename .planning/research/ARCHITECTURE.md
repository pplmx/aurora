# Architecture Research

**Date:** 2026-04-26
**Milestone:** v1.1 Production Hardening

## Integration Architecture

### Current Architecture (v1.0)

```
cmd/aurora/           # CLI entry point
├── cmd/              # Cobra commands
internal/
├── domain/           # Business logic
├── app/              # Use cases
├── ui/               # TUI (bubbletea)
├── infra/            # SQLite, HTTP, Events
internal/api/         # API handlers (stub)
```

### Proposed v1.1 Architecture

```
cmd/
├── aurora/           # CLI + TUI
├── api/              # REST API server  ← NEW
web/                  # Web UI static    ← NEW
migrations/           # DB migrations   ← NEW
internal/
├── domain/           # Business logic (unchanged)
├── app/              # Use cases (unchanged)
├── ui/               # TUI (unchanged)
├── infra/            # SQLite, HTTP, Events
│   ├── sqlite/       # + backup utility
│   └── http/         # (enhanced for oracle)
internal/api/         # API handlers
```

## New Components

### API Server (`cmd/api/`)
- Standalone HTTP server
- Reuses `internal/api/handlers`
- Uses existing `internal/app/*` services
- New middleware: auth, rate limiting, logging

### Web UI (`web/`)
- Static HTML/JS/CSS
- HTMX for interactivity
- Alpine.js for reactivity
- Builds to `dist/` for serving
- Can be served by API or standalone

### Migrations (`migrations/`)
- SQL migration files
- Version table: `schema_migrations`
- Tool: `migrate` CLI or Go library
- Hook into build process

### Backup Utility (`internal/infra/backup/`)
- SQLite backup via `.backup` command
- JSON export for portability
- File compression option
- Verification step

## Build Order Recommendation

1. **Database Migrations** (foundation - other features need it)
2. **API Server** (core feature - highest priority)
3. **Oracle Real Data** (wire up existing fetcher)
4. **Web UI** (UI layer - depends on API)
5. **Backup/Restore** (ops feature - last)

## Data Flow Changes

### API Flow
```
HTTP Request → Gin Router → Middleware → Handler → App Service → Domain → Repository → SQLite
```

### Web UI Flow
```
Browser → HTMX → API Server → Same handlers as CLI
```

## Configuration Changes

```toml
[api]
host = "0.0.0.0"
port = 8080
auth = "api-key"  # or "basic"

[web]
enabled = true
dir = "./web/dist"

[migrations]
dir = "./migrations"
auto = true  # auto-run on startup

[backup]
dir = "./backups"
format = "json"  # or "sqlite"
```
