//go:build !windows

package lotteryapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/domain/lottery"
)

// TestExportWritesAnOwnerOnlyFile: the document lists participant names and
// their draws, so the file is as private as the database it came from; sharing
// it is an explicit copy, not the default mode.
func TestExportWritesAnOwnerOnlyFile(t *testing.T) {
	repo := &mockLotteryRepo{records: []*lottery.LotteryRecord{
		{ID: "draw-1", Seed: "abcdef", Winners: []string{"A"}, Participants: []string{"A", "B"}},
	}}

	path := filepath.Join(t.TempDir(), "export.json")
	count, err := WriteRecordsToFile(repo, path)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

// TestExportTightensAnExistingFile: WriteFileAtomic keeps an existing file's
// bits, so a document exported before the owner-only rule would stay readable by
// other accounts; re-exporting it must tighten the mode, not preserve it.
func TestExportTightensAnExistingFile(t *testing.T) {
	repo := &mockLotteryRepo{records: []*lottery.LotteryRecord{
		{ID: "draw-1", Seed: "abcdef", Winners: []string{"A"}, Participants: []string{"A", "B"}},
	}}

	path := filepath.Join(t.TempDir(), "old-export.json")
	require.NoError(t, os.WriteFile(path, []byte("[]"), 0o600))
	require.NoError(t, os.Chmod(path, 0o644))

	_, err := WriteRecordsToFile(repo, path)
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm(),
		"a re-export must not leave the old 0644 behind")
}
