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

// TestExportKeepsAnExistingFileMode pins the writer's contract on the export
// path: a destination the operator already set keeps its bits, so a deliberate
// 0644 stays 0644 — only a fresh file gets the owner-only default.
func TestExportKeepsAnExistingFileMode(t *testing.T) {
	repo := &mockLotteryRepo{records: []*lottery.LotteryRecord{
		{ID: "draw-1", Seed: "abcdef", Winners: []string{"A"}, Participants: []string{"A", "B"}},
	}}

	path := filepath.Join(t.TempDir(), "chosen-export.json")
	require.NoError(t, os.WriteFile(path, []byte("[]"), 0o600))
	require.NoError(t, os.Chmod(path, 0o640))

	_, err := WriteRecordsToFile(repo, path)
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o640), info.Mode().Perm(),
		"the operator's mode must survive a re-export")
}
