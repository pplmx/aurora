//go:build !windows

package backup

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackupService_CreateKeepsNewBackupSetPrivate(t *testing.T) {
	previousUmask := syscall.Umask(0o022)
	t.Cleanup(func() { syscall.Umask(previousUmask) })

	dir := t.TempDir()
	source := filepath.Join(dir, "source", "aurora.db")
	makeTestDB(t, source)

	output := filepath.Join(dir, "backup")
	svc := NewBackupService(map[string]string{"aurora": source})
	_, err := svc.Create(context.Background(), output)
	require.NoError(t, err)

	requireMode(t, output, 0o700,
		"a backup contains the live database, including Oracle request headers")
	for _, name := range []string{"aurora.db", "metadata.json"} {
		requireMode(t, filepath.Join(output, name), 0o600,
			"every file in a private backup set must stay owner-only")
	}
}

func TestBackupService_CreatePreservesAnExistingOutputDirectory(t *testing.T) {
	previousUmask := syscall.Umask(0o022)
	t.Cleanup(func() { syscall.Umask(previousUmask) })

	dir := t.TempDir()
	source := filepath.Join(dir, "source", "aurora.db")
	makeTestDB(t, source)

	output := filepath.Join(dir, "managed-backups")
	require.NoError(t, os.Mkdir(output, 0o750))
	require.NoError(t, os.Chmod(output, 0o750))

	svc := NewBackupService(map[string]string{"aurora": source})
	_, err := svc.Create(context.Background(), output)
	require.NoError(t, err)

	requireMode(t, output, 0o750,
		"an explicitly managed backup directory must keep its existing permissions")
	for _, name := range []string{"aurora.db", "metadata.json"} {
		requireMode(t, filepath.Join(output, name), 0o600,
			"the files themselves must remain owner-only inside a shared parent")
	}
}

func TestBackupService_RestoreKeepsDatabaseCopiesPrivate(t *testing.T) {
	previousUmask := syscall.Umask(0o022)
	t.Cleanup(func() { syscall.Umask(previousUmask) })

	for _, tc := range []struct {
		name         string
		existingMode os.FileMode
	}{
		{name: "new staging directory"},
		{name: "legacy staging directory", existingMode: 0o755},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			live := filepath.Join(dir, "data", "aurora.db")
			makeTestDB(t, live)

			svc := NewBackupService(map[string]string{"aurora": live})
			backupDir := filepath.Join(dir, "backup")
			_, err := svc.Create(context.Background(), backupDir)
			require.NoError(t, err)

			preRestoreDir := backupDir + ".pre_restore"
			if tc.existingMode != 0 {
				require.NoError(t, os.Mkdir(preRestoreDir, tc.existingMode))
				require.NoError(t, os.Chmod(preRestoreDir, tc.existingMode))
			}

			require.NoError(t, svc.Restore(context.Background(), backupDir))

			requireMode(t, preRestoreDir, 0o700,
				"the undo snapshot directory contains a complete copy of the live database")
			requireMode(t, filepath.Join(preRestoreDir, "aurora.db"), 0o600,
				"the undo snapshot can contain Oracle request headers")
			requireMode(t, live, 0o600,
				"restore must not weaken the live database's owner-only mode")
		})
	}
}

func TestBackupService_RestoreRejectsSymlinkPreRestoreDir(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "data", "aurora.db")
	makeTestDB(t, live)

	svc := NewBackupService(map[string]string{"aurora": live})
	backupDir := filepath.Join(dir, "backup")
	_, err := svc.Create(context.Background(), backupDir)
	require.NoError(t, err)

	target := filepath.Join(dir, "unrelated")
	require.NoError(t, os.Mkdir(target, 0o755))
	require.NoError(t, os.Chmod(target, 0o755))
	require.NoError(t, os.Symlink(target, backupDir+".pre_restore"))

	err = svc.Restore(context.Background(), backupDir)
	require.ErrorContains(t, err, "symlink")
	requireMode(t, target, 0o755,
		"restore must not chmod or write through a planted staging-directory symlink")
}

func requireMode(t *testing.T, path string, want os.FileMode, message string) {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, want, info.Mode().Perm(), "%s: %s", path, message)
}
