# Phase 9: Backup & Restore - Context

**Gathered:** 2026-04-30
**Status:** Ready for planning
**Mode:** Auto-generated (infrastructure phase)

<domain>
## Phase Boundary

Complete backup/restore functionality with verification and safety guarantees:
1. `aurora backup restore --file <path>` restores database from backup
2. Restore performs schema migration if needed
3. Restore creates automatic pre-restore backup
4. `aurora backup verify` validates backup file integrity

</domain>

<decisions>
## Implementation Decisions

### Agent's Discretion
All implementation choices are at the agent's discretion — infrastructure phase.

### Key Design Decisions
- File copy with WAL checkpoint (not JSON export)
- Pre-restore backup always created
- golang-migrate integration for schema versioning

</decisions>

<codebase>
## Existing Code Insights

### Current State
- `internal/infra/backup/backup.go`: JSON-based stub, Restore() returns error
- `cmd/aurora/cmd/backup.go`: Has create and verify commands, no restore

### Database Paths
- blockchain: `data/blockchain.db`
- tokens: `data/tokens.db`

### Requirements
- Use WAL checkpoint before backup
- Atomic file operations (rename, not copy)
- Schema version tracking in metadata

</codebase>

<specifics>
## Specific Ideas

Per research/BACKUP.md:
- Use `database/sql` for SQLite operations
- Use `io.Copy` for file copying
- Include metadata.json with schema version
- SHA-256 checksum for verification

</specifics>

<deferred>
## Deferred Ideas

PITR (point-in-time recovery) - complex, defer to future.

</deferred>