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

// TestWriteFileAtomic_SyncsParentDirectoryAfterPromotion proves the rename is
// not reported as durable while it may still be lost with the directory entry.
func TestWriteFileAtomic_SyncsParentDirectoryAfterPromotion(t *testing.T) {
	original := syncDirectory
	t.Cleanup(func() { syncDirectory = original })

	dir := t.TempDir()
	path := filepath.Join(dir, "export.json")
	synced := ""
	syncDirectory = func(got string) error {
		synced = got
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, `["new"]`, string(data),
			"the directory must be flushed only after the complete file is visible")
		return nil
	}

	require.NoError(t, WriteFileAtomic(path, []byte(`["new"]`), 0o644))
	assert.Equal(t, dir, synced)
}

// TestWriteFileAtomic_ReportsDirectorySyncFailure is the fault-injection
// regression for returning success when only the file, not its rename, is
// durable. The complete new document remains visible, but callers receive the
// durability error instead of reporting a successful export.
func TestWriteFileAtomic_ReportsDirectorySyncFailure(t *testing.T) {
	original := syncDirectory
	t.Cleanup(func() { syncDirectory = original })
	syncErr := errors.New("injected directory sync failure")
	syncDirectory = func(string) error { return syncErr }

	dir := t.TempDir()
	path := filepath.Join(dir, "export.json")
	require.NoError(t, os.WriteFile(path, []byte(`["old"]`), 0o644))

	err := WriteFileAtomic(path, []byte(`["new"]`), 0o644)
	require.ErrorIs(t, err, syncErr)

	data, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	assert.Equal(t, `["new"]`, string(data),
		"a post-rename sync failure must not leave a truncated export")

	entries, readDirErr := os.ReadDir(dir)
	require.NoError(t, readDirErr)
	assert.Len(t, entries, 1, "no temporary sibling may survive the failure")
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

func TestWriteFileAtomic_CreatesTargetOfDanglingSymlink(t *testing.T) {
	dir := t.TempDir()
	targetDir := filepath.Join(dir, "real")
	target := filepath.Join(targetDir, "new.json")
	link := filepath.Join(dir, "latest.json")
	require.NoError(t, os.MkdirAll(targetDir, 0o755))
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	require.NoError(t, WriteFileAtomic(link, []byte(`["new"]`), 0o644))

	info, err := os.Lstat(link)
	require.NoError(t, err)
	assert.NotZero(t, info.Mode()&os.ModeSymlink, "the symlink itself must survive")
	data, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, `["new"]`, string(data))
}

func TestWriteFileAtomic_ResolvesRelativeTargetFromRealParent(t *testing.T) {
	root := t.TempDir()
	realParent := filepath.Join(root, "real", "current", "v2")
	realTarget := filepath.Join(root, "real", "current", "sibling", "new.json")
	logicalParent := filepath.Join(root, "logical")
	require.NoError(t, os.MkdirAll(realParent, 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(realTarget), 0o755))
	if err := os.Symlink(realParent, logicalParent); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	realLink := filepath.Join(realParent, "latest.json")
	if err := os.Symlink(filepath.Join("..", "sibling", "new.json"), realLink); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	logicalLink := filepath.Join(logicalParent, "latest.json")
	require.NoError(t, WriteFileAtomic(logicalLink, []byte(`["new"]`), 0o644))

	info, err := os.Lstat(realLink)
	require.NoError(t, err)
	assert.NotZero(t, info.Mode()&os.ModeSymlink, "the symlink itself must survive")
	data, err := os.ReadFile(realTarget)
	require.NoError(t, err)
	assert.Equal(t, `["new"]`, string(data))
}

func TestWriteFileAtomic_RefusesADirectory(t *testing.T) {
	dir := t.TempDir()
	require.Error(t, WriteFileAtomic(dir, []byte("x"), 0o644))
}
