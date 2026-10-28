package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveFinalSymlink_FollowsDanglingChain(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "latest.json")
	middle := filepath.Join(dir, "current.json")
	target := filepath.Join(dir, "real", "new.json")
	if err := os.Symlink(middle, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Symlink(filepath.Join("real", "new.json"), middle); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	resolved, err := ResolveFinalSymlink(link)
	require.NoError(t, err)
	require.Equal(t, target, resolved)
}

func TestResolveFinalSymlink_UsesRealParentForRelativeTarget(t *testing.T) {
	root := t.TempDir()
	realParent := filepath.Join(root, "real", "current", "v2")
	realTarget := filepath.Join(root, "real", "current", "sibling", "new.json")
	logicalParent := filepath.Join(root, "logical")
	require.NoError(t, os.MkdirAll(realParent, 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(realTarget), 0o755))
	if err := os.Symlink(realParent, logicalParent); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Symlink(filepath.Join("..", "sibling", "new.json"),
		filepath.Join(realParent, "link.json")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	resolved, err := ResolveFinalSymlink(filepath.Join(logicalParent, "link.json"))
	require.NoError(t, err)
	require.Equal(t, realTarget, resolved)
}

func TestResolveFinalSymlink_RejectsLoop(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.json")
	second := filepath.Join(dir, "second.json")
	if err := os.Symlink(second, first); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Symlink(first, second); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	_, err := ResolveFinalSymlink(first)
	require.ErrorContains(t, err, "too many levels of symbolic links")
}
