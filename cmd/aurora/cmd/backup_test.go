package cmd

import (
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackupCmd_Registered(t *testing.T) {
	resetCliForTest()
	names := make(map[string]bool, len(backupCmd.Commands()))
	for _, c := range backupCmd.Commands() {
		names[c.Name()] = true
	}
	assert.True(t, names["create"], "backup should register create")
	assert.True(t, names["verify"], "backup should register verify")
	assert.True(t, names["restore"], "backup should register restore")

	rootNames := map[string]*cobra.Command{}
	for _, c := range rootCmd.Commands() {
		rootNames[c.Name()] = c
	}
	require.Contains(t, rootNames, "backup", "root should register backup")
}

func TestBackupCreateVerify_RoundTrip(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		// Materialise ./data/aurora.db with the real schema (the source file
		// the backup command backs up).
		runMigrations(t)

		backupDir := filepath.Join(t.TempDir(), "bk1")

		out, err := runCmd(t, "backup", "create", backupDir)
		require.NoError(t, err)
		assert.Contains(t, out, "Backup created")

		assert.FileExists(t, filepath.Join(backupDir, "aurora.db"))
		assert.FileExists(t, filepath.Join(backupDir, "metadata.json"))

		out, err = runCmd(t, "backup", "verify", backupDir)
		require.NoError(t, err)
		assert.Contains(t, out, "Backup verified")
	})
}

func TestBackupRestore_RequiresConfirm(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		runMigrations(t)
		backupDir := filepath.Join(t.TempDir(), "bk2")

		_, err := runCmd(t, "backup", "create", backupDir)
		require.NoError(t, err)

		// Without an explicit --confirm the destructive restore is refused.
		_, err = runCmd(t, "backup", "restore", backupDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "confirm")

		// With --confirm it proceeds.
		out, err := runCmd(t, "backup", "restore", backupDir, "--confirm")
		require.NoError(t, err)
		assert.Contains(t, out, "restored")
	})
}

func TestBackupRestore_NegativeYShorthand(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		runMigrations(t)
		backupDir := filepath.Join(t.TempDir(), "bk-y")

		_, err := runCmd(t, "backup", "create", backupDir)
		require.NoError(t, err)

		// -y is the canonical destructive-op shorthand; backup restore must
		// accept it like token/nft burn and the other gates (TASK-152).
		out, err := runCmd(t, "backup", "restore", backupDir, "-y")
		require.NoError(t, err)
		assert.Contains(t, out, "restored")
	})
}
