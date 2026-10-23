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

func requireMode(t *testing.T, path string, want os.FileMode, message string) {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, want, info.Mode().Perm(), "%s: %s", path, message)
}
