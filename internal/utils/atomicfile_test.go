package utils

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteFileAtomic_ReplacesContentAndKeepsMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "export.json")
	require.NoError(t, os.WriteFile(path, []byte(`["old"]`), 0o600))
	require.NoError(t, os.Chmod(path, 0o640))

	require.NoError(t, WriteFileAtomic(path, []byte(`["new"]`), 0o644))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, `["new"]`, string(data))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o640), info.Mode().Perm(),
		"replacing an export must keep the file's permissions")

	entries, err := os.ReadDir(filepath.Dir(path))
	require.NoError(t, err)
	assert.Len(t, entries, 1, "the temporary sibling must not be left behind")
}

// TestWriteFileAtomic_KeepsTheOldExportWhenPromotionFails is the regression
// for the direct os.WriteFile implementation: it truncated the destination
// before writing, so a failure left the user's previous export empty.
func TestWriteFileAtomic_KeepsTheOldExportWhenPromotionFails(t *testing.T) {
	original := renameFile
	t.Cleanup(func() { renameFile = original })
	renameFile = func(string, string) error { return errors.New("disk full") }

	path := filepath.Join(t.TempDir(), "export.json")
	require.NoError(t, os.WriteFile(path, []byte(`["old"]`), 0o644))

	err := WriteFileAtomic(path, []byte(`["new"]`), 0o644)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disk full")

	data, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	assert.Equal(t, `["old"]`, string(data), "a failed export must leave the previous document intact")

	entries, readDirErr := os.ReadDir(filepath.Dir(path))
	require.NoError(t, readDirErr)
	assert.Len(t, entries, 1, "a failed export must remove its temporary sibling")
}

func TestWriteFileAtomic_WritesThroughASymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires extra privileges on Windows")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "real.json")
	link := filepath.Join(dir, "link.json")
	require.NoError(t, os.WriteFile(target, []byte(`["old"]`), 0o644))
	require.NoError(t, os.Symlink(target, link))

	require.NoError(t, WriteFileAtomic(link, []byte(`["new"]`), 0o644))

	info, err := os.Lstat(link)
	require.NoError(t, err)
	assert.NotZero(t, info.Mode()&os.ModeSymlink, "the symlink itself must survive")

	data, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, `["new"]`, string(data))
}

func TestWriteFileAtomic_RefusesADirectory(t *testing.T) {
	dir := t.TempDir()
	require.Error(t, WriteFileAtomic(dir, []byte("x"), 0o644))
}
