package backup

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestVerifyRejectsArchiveForDifferentDatabaseSet keeps Verify from reporting
// a backup as usable when Restore must reject it before touching live data.
func TestVerifyRejectsArchiveForDifferentDatabaseSet(t *testing.T) {
	dir := t.TempDir()
	backupDir := filepath.Join(dir, "backup")
	makeValidBackup(t, backupDir, "old")

	svc := NewBackupService(map[string]string{
		"current": filepath.Join(dir, "live", "current.db"),
	})
	err := svc.Verify(context.Background(), backupDir)
	require.ErrorContains(t, err, `backup database "old" is not configured`)
	require.ErrorContains(t, svc.Restore(context.Background(), backupDir),
		`backup database "old" is not configured`)
}

// TestVerifyRejectsDuplicateDatabaseNames proves Verify checks the archive's
// logical set as well as its files. Restore treats a duplicate as ambiguous and
// refuses it, so Verify must not call the same metadata complete.
func TestVerifyRejectsDuplicateDatabaseNames(t *testing.T) {
	dir := t.TempDir()
	backupDir := filepath.Join(dir, "backup")
	makeValidBackup(t, backupDir, "main")
	writeMeta(t, backupDir, "main", "main")

	svc := NewBackupService(map[string]string{
		"main": filepath.Join(dir, "live", "main.db"),
	})
	err := svc.Verify(context.Background(), backupDir)
	require.ErrorContains(t, err, `backup metadata lists database "main" more than once`)
}
