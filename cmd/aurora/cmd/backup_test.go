package cmd

import (
	"encoding/json"
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

		// The same pair through --json: the create document names the file it
		// wrote and the verify document carries the target and the verdict.
		jsonDir := filepath.Join(t.TempDir(), "bk1-json")
		out, err = runCmd(t, "backup", "create", jsonDir, "--json")
		require.NoError(t, err)
		var created struct {
			Dir           string `json:"dir"`
			Size          int64  `json:"size"`
			SchemaVersion uint   `json:"schema_version"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &created), "got %q", out)
		assert.DirExists(t, created.Dir)
		assert.Equal(t, jsonDir, created.Dir)
		assert.Positive(t, created.Size)
		assert.NotZero(t, created.SchemaVersion)

		out, err = runCmd(t, "backup", "verify", jsonDir, "--json")
		require.NoError(t, err)
		var verified struct {
			Path   string `json:"path"`
			Status string `json:"status"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &verified), "got %q", out)
		assert.Equal(t, jsonDir, verified.Path)
		assert.Equal(t, "verified", verified.Status)
	})
}

// TestBackupRestore_JSON: a restore reports where it read the archive, the
// verdict, and the undo directory it kept for the previous database.
func TestBackupRestore_JSON(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		runMigrations(t)
		backupDir := filepath.Join(t.TempDir(), "bk-restore")
		_, err := runCmd(t, "backup", "create", backupDir)
		require.NoError(t, err)

		out, err := runCmd(t, "backup", "restore", backupDir, "--confirm", "--json")
		require.NoError(t, err)
		var restored struct {
			Path   string `json:"path"`
			Status string `json:"status"`
			Undo   string `json:"undo"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &restored), "got %q", out)
		assert.Equal(t, backupDir, restored.Path)
		assert.Equal(t, "restored", restored.Status)
		assert.NotEmpty(t, restored.Undo, "the previous database is kept as an undo archive")
		assert.DirExists(t, restored.Undo)
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
