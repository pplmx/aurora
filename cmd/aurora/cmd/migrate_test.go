package cmd

import (
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setMigrationsPathForTest points the migrate command at the repo's real
// migrations/ directory (repoMigrationsDir resolves from the test file
// location, immune to the temp cwd). Without this the command would look
// at ./migrations relative to the (temp) cwd and fail to find sources.
func setMigrationsPathForTest(t *testing.T) {
	t.Helper()
	viper.Set("migrate.path", repoMigrationsDir())
}

// appliedSchemaVersion queries the golang-migrate version table directly to
// cross-check what the CLI command actually wrote, independent of the
// command's own status output. golang-migrate deletes the schema_migrations
// row when fully rolled back to version 0, so a missing row means 0.
func appliedSchemaVersion(t *testing.T) (version int) {
	t.Helper()
	db := openTestAuroraDB(t)
	err := db.QueryRow("SELECT version FROM schema_migrations").Scan(&version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0
		}
		require.NoError(t, err)
	}
	return version
}

// tableExists reports whether the given table exists in ./data/aurora.db.
func tableExists(t *testing.T, name string) bool {
	t.Helper()
	db := openTestAuroraDB(t)
	var n int
	require.NoError(t, db.QueryRow(
		"SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?",
		name,
	).Scan(&n))
	return n > 0
}

func TestMigrateCmd_Registered(t *testing.T) {
	resetCliForTest()
	sub := migrateCmd.Commands()
	names := make(map[string]bool, len(sub))
	for _, c := range sub {
		names[c.Name()] = true
	}
	assert.True(t, names["up"], "migrate should register up")
	assert.True(t, names["down"], "migrate should register down")
	assert.True(t, names["status"], "migrate should register status")

	// also reachable from rootCmd
	rootNames := map[string]*cobra.Command{}
	for _, c := range rootCmd.Commands() {
		rootNames[c.Name()] = c
	}
	require.Contains(t, rootNames, "migrate", "root should register migrate")
}

func TestMigrateUp_FreshDB(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setMigrationsPathForTest(t)

		out, err := runCmd(t, "migrate", "up")
		require.NoError(t, err)
		assert.Contains(t, out, "Current version: 2", "all migrations applied")

		assert.Equal(t, 2, appliedSchemaVersion(t))
		assert.True(t, tableExists(t, "candidates"), "migration 000001 should create candidates")
		assert.True(t, tableExists(t, "token_transfers"), "migration 000002 should create token_transfers")
	})
}

func TestMigrateUp_WithSteps(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setMigrationsPathForTest(t)

		out, err := runCmd(t, "migrate", "up", "1")
		require.NoError(t, err)
		assert.Contains(t, out, "Current version: 1")
		assert.Equal(t, 1, appliedSchemaVersion(t))
		assert.True(t, tableExists(t, "candidates"), "step 1 applies 000001")
		assert.False(t, tableExists(t, "token_transfers"), "step 1 must not apply 000002")

		out, err = runCmd(t, "migrate", "up", "1")
		require.NoError(t, err)
		assert.Contains(t, out, "Current version: 2")
		assert.Equal(t, 2, appliedSchemaVersion(t))
	})
}

func TestMigrateUp_AlreadyApplied(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setMigrationsPathForTest(t)

		_, err := runCmd(t, "migrate", "up")
		require.NoError(t, err)

		out, err := runCmd(t, "migrate", "up")
		require.NoError(t, err, "re-running up on a migrated DB is a no-op, not an error")
		assert.Contains(t, out, "Current version: 2")
	})
}

func TestMigrateStatus_FreshDB(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setMigrationsPathForTest(t)

		out, err := runCmd(t, "migrate", "status")
		require.NoError(t, err)
		assert.Contains(t, out, "Current version: 0")
		assert.Contains(t, out, "Dirty: false")
		assert.Contains(t, out, "Applied: (none)")
		assert.Contains(t, out, "Pending:")
		assert.Contains(t, out, "1", "pending lists the first unapplied migration")
	})
}

func TestMigrateStatus_AfterUp(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setMigrationsPathForTest(t)

		_, err := runCmd(t, "migrate", "up")
		require.NoError(t, err)

		out, err := runCmd(t, "migrate", "status")
		require.NoError(t, err)
		assert.Contains(t, out, "Current version: 2")
		assert.Contains(t, out, "Applied: 1, 2")
		assert.Contains(t, out, "Pending: (none)")
	})
}

func TestMigrateDown_StepsAndBottom(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setMigrationsPathForTest(t)

		_, err := runCmd(t, "migrate", "up")
		require.NoError(t, err)

		out, err := runCmd(t, "migrate", "down", "1")
		require.NoError(t, err)
		assert.Contains(t, out, "Current version: 1")
		assert.Equal(t, 1, appliedSchemaVersion(t))

		// down twice reaches the bottom (version 0).
		out, err = runCmd(t, "migrate", "down")
		require.NoError(t, err)
		assert.Contains(t, out, "Current version: 0")
		assert.Equal(t, 0, appliedSchemaVersion(t))

		// down while already at the bottom is a friendly no-op, not an error.
		out, err = runCmd(t, "migrate", "down")
		require.NoError(t, err)
		assert.Contains(t, out, "No migrations to roll back")
	})
}

// TestMigrateUp_OverrunCapsAtPending guards against golang-migrate's
// Steps(N) overrun: "up 5" with only 2 pending must apply both and succeed,
// not apply-then-error ("limit 3 short" / "file does not exist").
func TestMigrateUp_OverrunCapsAtPending(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setMigrationsPathForTest(t)

		out, err := runCmd(t, "migrate", "up", "5")
		require.NoError(t, err, "up with N larger than pending must succeed")
		assert.Contains(t, out, "Current version: 2")
		assert.Equal(t, 2, appliedSchemaVersion(t))
	})
}

// TestMigrateDown_OverrunRollsBackAll guards the mirror case: "down 5" on a
// version-2 DB must roll everything back and report success — previously it
// rolled back both steps, then errored and misprinted "nothing to roll back".
func TestMigrateDown_OverrunRollsBackAll(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setMigrationsPathForTest(t)

		_, err := runCmd(t, "migrate", "up")
		require.NoError(t, err)

		out, err := runCmd(t, "migrate", "down", "5")
		require.NoError(t, err, "down with N larger than applied must succeed")
		assert.Contains(t, out, "Current version: 0")
		assert.Equal(t, 0, appliedSchemaVersion(t))
		assert.NotContains(t, out, "No migrations to roll back", "a real rollback must not be reported as a no-op")
	})
}

func TestMigrate_InvalidCounts(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setMigrationsPathForTest(t)

		// Non-integer and zero are rejected by the command's own validator
		// with a clear message.
		for _, tc := range []struct {
			args []string
			msg  string
		}{
			{args: []string{"migrate", "up", "abc"}, msg: "invalid"},
			{args: []string{"migrate", "up", "0"}, msg: "invalid"},
			{args: []string{"migrate", "down", "0"}, msg: "invalid"},
		} {
			_, err := runCmd(t, tc.args...)
			require.Error(t, err, "args %v should error", tc.args)
			assert.Contains(t, strings.ToLower(err.Error()), tc.msg)
		}

		// A negative count is swallowed by cobra's flag parser (interpreted
		// as a shorthand flag) before reaching the validator — the important
		// contract is that the command rejects it rather than apply/roll back.
		for _, args := range [][]string{
			{"migrate", "up", "-1"},
			{"migrate", "down", "-2"},
		} {
			_, err := runCmd(t, args...)
			require.Error(t, err, "args %v should error", args)
		}
	})
}

func TestMigrate_MissingMigrationDir(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		viper.Set("migrate.path", filepath.Join(t.TempDir(), "no-such-dir"))

		_, err := runCmd(t, "migrate", "up")
		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "migration source")
	})
}
