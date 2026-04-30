# SQLite Backup/Restore Research

**Project:** Aurora v1.2 Operational Readiness  
**Topic:** BCK-04 Backup Restore Implementation  
**Date:** 2026-04-30  
**Confidence:** MEDIUM-HIGH (based on SQLite/go-sqlite3 docs and Go ecosystem patterns)

---

## Executive Summary

Implementing backup restore for Aurora requires choosing between SQLite's native **online backup API** (via cgo) or **file-based copying**. For a CLI blockchain application with WAL mode enabled, the **file-copy approach with WAL checkpointing** is recommended for simplicity and reliability. Schema versioning is already handled by golang-migrate; restore should integrate with the existing migration system.

---

## 1. SQLite Backup Strategies

### Strategy A: Online Backup API (`sqlite3_backup_*`)

**How it works:** SQLite provides C API functions for online backup that copy pages incrementally without blocking writers.

```go
// Via go-sqlite3's cgo bindings
import "github.com/mattn/go-sqlite3"

// sqlite3_backup_init, sqlite3_backup_step, sqlite3_backup_finish
```

**Pros:**
- Non-blocking reads/writes during backup
- Incremental page-by-page copying
- Works with concurrent database access
- Native SQLite feature, well-tested

**Cons:**
- Requires cgo (already used by go-sqlite3)
- More complex implementation
- Slightly larger memory footprint during backup
- Less common in Go ecosystem (fewer examples)

**Use when:** High availability required, large databases, frequent concurrent access.

### Strategy B: File Copy with WAL Checkpoint

**How it works:** Close WAL, copy main database file, optionally copy WAL file.

```go
func (s *BackupService) Create(ctx context.Context, output string) error {
    // 1. Checkpoint WAL to merge writes into main file
    _, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
    
    // 2. Copy database file
    err = copyFile(dbPath, backupPath)
}
```

**Pros:**
- Simple, well-understood
- No additional dependencies
- Fast for typical database sizes
- Easy to verify (checksum entire file)

**Cons:**
- Brief exclusive lock during checkpoint (~100ms typically)
- Not truly "online" - small window of inconsistency
- Must stop or pause writes for exact point-in-time copy

**Use when:** CLI tool, acceptable brief lock, simplicity preferred.

### Strategy C: JSON Export (Current State)

**How it works:** Current implementation serializes data to JSON.

**Issues with current implementation:**
1. `Create()` doesn't actually dump any tables - just creates empty structure
2. `Restore()` returns "not implemented" stub
3. Doesn't handle schema versioning
4. No actual data serialization

**Recommendation:** Replace with either Strategy A or B, keep JSON format as **optional human-readable export**.

---

## 2. Recommended Approach for Aurora

### Choose: **Strategy B (File Copy with WAL Checkpoint)**

**Rationale:**
1. CLI tool → brief lock acceptable
2. Already using go-sqlite3 (cgo already present)
3. golang-migrate handles schema → restore just needs file copy + migration
4. Simpler to test and verify
5. Consistent with v1.1's "SQLite .backup" direction in research

### Implementation Design

```go
// internal/infra/backup/backup.go

type BackupService struct {
    dbPaths map[string]string  // map of logical name -> file path
    migPath string              // migration directory
}

type BackupResult struct {
    File        string
    Size        int64
    Checksum    string
    Timestamp   time.Time
    SchemaVersion uint
}

// Create performs online backup with WAL checkpoint
func (s *BackupService) Create(ctx context.Context, output string) (*BackupResult, error) {
    for name, path := range s.dbPaths {
        // 1. Checkpoint WAL
        db, err := sql.Open("sqlite3", path)
        if err != nil {
            return nil, fmt.Errorf("open %s: %w", name, err)
        }
        
        // Truncate WAL after checkpointing
        if _, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
            db.Close()
            return nil, fmt.Errorf("checkpoint %s: %w", name, err)
        }
        
        // 2. Copy file
        destPath := filepath.Join(output, name+".db")
        if err := copyFile(path, destPath); err != nil {
            db.Close()
            return nil, fmt.Errorf("copy %s: %w", name, err)
        }
        
        db.Close()
    }
    
    // 3. Export metadata (schema version, timestamp)
    metadata := BackupMetadata{
        Version:   "1.2",
        Timestamp: time.Now().UTC(),
        Databases: slices.SortedMapKeys(s.dbPaths),
        SchemaVersion: s.getCurrentSchemaVersion(),
    }
    
    // 4. Save metadata.json alongside backups
    metaPath := filepath.Join(output, "metadata.json")
    if err := saveJSON(metaPath, metadata); err != nil {
        return nil, err
    }
    
    // 5. Create checksum of entire backup directory
    checksum := s.computeDirectoryChecksum(output)
    
    return &BackupResult{...}, nil
}
```

---

## 3. Schema Versioning and Migration During Restore

### Integration with golang-migrate

Aurora already uses `golang-migrate/v4` with SQLite3 driver. Restore should:

1. **Detect target schema version** from backup metadata
2. **Detect current schema version** from live database (if exists)
3. **Run migrations** to align versions

```go
func (s *BackupService) Restore(ctx context.Context, backupPath, targetDBPath string) error {
    // 1. Read backup metadata
    meta, err := readMetadata(filepath.Join(backupPath, "metadata.json"))
    if err != nil {
        return fmt.Errorf("read metadata: %w", err)
    }
    
    // 2. If target database exists, check version compatibility
    migrator, err := migrate.New(targetDBPath, s.migPath)
    if err != nil {
        return fmt.Errorf("create migrator: %w", err)
    }
    defer migrator.Close()
    
    currentVersion, _, _ := migrator.Version()
    targetVersion := meta.SchemaVersion
    
    if currentVersion > 0 && currentVersion != targetVersion {
        // Option A: Migrate down to backup version, restore, migrate up
        // Option B: Refuse restore, require fresh database
        // Option C: Always restore to fresh database
        
        // Recommendation: Option C for safety
        return fmt.Errorf("target database has schema v%d, backup is v%d. Use --fresh to overwrite",
            currentVersion, targetVersion)
    }
    
    // 3. Copy backup files
    for _, dbName := range meta.Databases {
        src := filepath.Join(backupPath, dbName+".db")
        dest := s.dbPaths[dbName]
        
        if err := atomicRename(src, dest); err != nil {
            return fmt.Errorf("restore %s: %w", dbName, err)
        }
    }
    
    // 4. Run migrations if needed (forward migrations)
    if targetVersion < s.getLatestSchemaVersion() {
        migrator.Up()
    }
    
    return nil
}
```

### Migration Safety Rules

| Scenario | Action |
|----------|--------|
| Fresh database | Just copy backup |
| Same schema version | Direct copy |
| Older backup to newer DB | Refuse (too risky) |
| Newer backup to older DB | Copy + run migrations |

---

## 4. Point-in-Time Recovery (PITR)

### Considerations for Blockchain System

Aurora stores blockchain blocks with `height` as sequence. PITR would allow recovery to specific block height.

**Current state:** Aurora stores blockchain data, but PITR implementation is complex and not recommended for v1.2.

**Simplified approach for v1.2:**
- Backup captures complete database at timestamp
- No incremental WAL-based PITR
- Users can take snapshots before critical operations

**Future enhancement:**
- Store backup snapshots with block height metadata
- Implement WAL-based incremental backups
- Add `RestoreToHeight(ctx, backupPath, height)` 

### WAL Mode Consideration

Aurora uses `PRAGMA journal_mode=WAL`. With file-copy backup:
- Checkpoint first (`PRAGMA wal_checkpoint(TRUNCATE)`) ensures all data is in main file
- Without checkpoint, backup captures committed + some uncommitted WAL data

---

## 5. Verification of Restored Backups

### Verification Levels

| Level | What it checks | Implementation |
|-------|---------------|----------------|
| **L1: File integrity** | Backup file exists, readable | `os.Stat()`, read test |
| **L2: Checksum** | File not corrupted | SHA-256 of entire file |
| **L3: Schema** | Tables/columns match expected | `SELECT * FROM sqlite_master` |
| **L4: Data** | Row counts, sample data | Query counts, spot-check values |
| **L5: Functional** | App works after restore | Run domain tests against restored DB |

### Verification Implementation

```go
func (s *BackupService) VerifyBackup(backupPath string) error {
    // L1: File exists
    if _, err := os.Stat(backupPath); err != nil {
        return fmt.Errorf("backup not found: %w", err)
    }
    
    // L2: Checksum (if backup was created with checksum)
    meta, _ := readMetadata(filepath.Join(backupPath, "metadata.json"))
    if meta != nil && meta.Checksum != "" {
        computed := s.computeDirectoryChecksum(backupPath)
        if computed != meta.Checksum {
            return fmt.Errorf("checksum mismatch: expected %s, got %s", 
                meta.Checksum, computed)
        }
    }
    
    // L3: Open and verify schema
    dbPath := filepath.Join(backupPath, "aurora.db") // or whatever main db name
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return fmt.Errorf("cannot open backup: %w", err)
    }
    defer db.Close()
    
    // Check expected tables exist
    expectedTables := []string{"blocks", "lottery_records", "votes", "nfts", "tokens"}
    for _, table := range expectedTables {
        var count int
        err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
        if err != nil || count == 0 {
            return fmt.Errorf("missing table: %s", table)
        }
    }
    
    return nil
}
```

### Restore Verification

After restore, run health checks:
```go
func (s *BackupService) VerifyRestored(dbPath string) error {
    // 1. Verify database opens
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return err
    }
    defer db.Close()
    
    // 2. Run migrations (verify schema is correct)
    if err := s.runMigrations(dbPath); err != nil {
        return fmt.Errorf("migration failed: %w", err)
    }
    
    // 3. Quick data sanity check
    var blockCount int
    db.QueryRow("SELECT COUNT(*) FROM blocks").Scan(&blockCount)
    
    // 4. Verify blockchain consistency (if blocks exist)
    if blockCount > 0 {
        var genesisHash string
        db.QueryRow("SELECT hash FROM blocks WHERE height = 0").Scan(&genesisHash)
        if genesisHash == "" {
            return fmt.Errorf("invalid genesis block")
        }
    }
    
    return nil
}
```

---

## 6. Error Handling and Rollback

### Critical Safety Rules

1. **Never overwrite production without confirmation**
2. **Always create backup of current state before restore**
3. **Use atomic operations (rename, not copy)**
4. **Verify before declaring success**

### Rollback Strategy

```go
func (s *BackupService) RestoreWithRollback(ctx context.Context, backupPath, targetPath string) error {
    // 1. Create pre-restore backup (safety net)
    preRestorePath := targetPath + ".pre_restore_bak"
    if _, err := os.Stat(targetPath); err == nil {
        // Target exists - backup current state first
        if err := copyFile(targetPath, preRestorePath); err != nil {
            return fmt.Errorf("pre-restore backup failed: %w (aborting restore)", err)
        }
        defer func() {
            // Clean up pre-restore backup after successful restore
            if err == nil {
                os.Remove(preRestorePath)
            }
        }()
    }
    
    // 2. Attempt restore
    err := s.Restore(ctx, backupPath, targetPath)
    if err != nil {
        // 3. Rollback on failure
        if _, statErr := os.Stat(preRestorePath); statErr == nil {
            // Restore the pre-restore backup
            rollbackErr := copyFile(preRestorePath, targetPath)
            if rollbackErr != nil {
                return fmt.Errorf("restore failed: %w; rollback also failed: %v", err, rollbackErr)
            }
            return fmt.Errorf("restore failed: %w (rolled back to previous state)", err)
        }
        return fmt.Errorf("restore failed: %w", err)
    }
    
    // 4. Verify restored state
    if err := s.VerifyRestored(targetPath); err != nil {
        // Rollback
        if _, statErr := os.Stat(preRestorePath); statErr == nil {
            copyFile(preRestorePath, targetPath)
        }
        return fmt.Errorf("verification failed: %w (rolled back)", err)
    }
    
    return nil
}
```

### Error Categories

| Error | Action |
|-------|--------|
| Backup file missing/corrupt | Return error, no destructive action |
| Schema version mismatch | Return error with guidance, no overwrite |
| Migration failure during restore | Rollback to pre-restore backup |
| Verification failure | Rollback to pre-restore backup |
| Partial restore (some DBs succeed) | Rollback all, return composite error |

---

## 7. CLI Command Design

Based on existing patterns in AGENTS.md:

```bash
# Create backup
./aurora backup create -o ./backups/backup-20260430.tar.gz

# List backups
./aurora backup list -d ./backups/

# Verify backup integrity
./aurora backup verify ./backups/backup-20260430.tar.gz

# Restore (requires --confirm flag for safety)
./aurora backup restore ./backups/backup-20260430.tar.gz --confirm

# Restore to specific location
./aurora backup restore ./backups/backup-20260430.tar.gz --target ./data/ --confirm

# Restore with rollback (fresh database)
./aurora backup restore ./backups/backup-20260430.tar.gz --fresh --confirm
```

---

## 8. Recommended Implementation Phases

### Phase 1: Basic File Copy Backup
- Implement `Create()` with WAL checkpoint + file copy
- Add schema version tracking to backup metadata
- Keep JSON export as optional format

### Phase 2: Restore with Migration
- Implement `Restore()` that copies files + runs migrations
- Add pre-restore safety backup
- Add rollback on failure

### Phase 3: Verification
- Implement `Verify()` for backup files
- Add `VerifyRestored()` for post-restore health checks
- Add CLI commands

---

## 9. Sources

| Source | Confidence | Relevance |
|--------|------------|-----------|
| SQLite Online Backup API (sqlite.org) | HIGH | Primary reference |
| go-sqlite3 GitHub (cgo bindings) | HIGH | Implementation details |
| golang-migrate documentation | HIGH | Migration integration |
| Aurora internal/infra/migrate/ | HIGH | Existing implementation |
| Aurora migrations/*.sql | HIGH | Schema understanding |

---

## 10. Summary Recommendations

| Decision | Recommendation | Rationale |
|----------|---------------|-----------|
| **Backup method** | File copy with WAL checkpoint | Simple, reliable, adequate for CLI tool |
| **Backup format** | Binary .db files + metadata.json | Fast, complete, verifiable |
| **Schema handling** | golang-migrate integration | Already in use, handles versioning |
| **PITR** | Not in scope for v1.2 | Complex, defer to future |
| **Verification** | Multi-level (file → schema → data) | Catches corruption early |
| **Rollback** | Pre-restore backup always | Safety before destructive ops |

---

*Next step: Implement backup service following Phase 1-3 approach above.*