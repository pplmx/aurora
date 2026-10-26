package deskapp

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/infra/sqlite"
)

// TestShellUsesTheInjectedLotteryRepository pins the one-shot file actions to
// the runtime's repository. The fallback still exists for a shell built before
// the runtime, but production must not open a second SQLite writer.
func TestShellUsesTheInjectedLotteryRepository(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "aurora.db")
	repo, err := sqlite.NewLotteryRepository(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	shell := NewShell(About{DBPath: dbPath})
	shell.SetLotteryRepository(repo)

	got, closeRepo, err := shell.lotteryRepository()
	require.NoError(t, err)
	require.Same(t, repo, got)
	require.NotPanics(t, closeRepo, "the container owns the handle; the action must not close it")
}
