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

// appliedSchemaVersion queries the schema_migrations table directly to
// cross-check what the CLI command actually wrote, independent of the
// command's own status output. The runner deletes the row when fully rolled
// back to version 0, so a missing row means 0.
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

// indexExists reports whether the given index exists in ./data/aurora.db.
func indexExists(t *testing.T, name string) bool {
	t.Helper()
	db := openTestAuroraDB(t)
	var n int
	require.NoError(t, db.QueryRow(
		"SELECT count(*) FROM sqlite_master WHERE type='index' AND name=?",
		name,
	).Scan(&n))
	return n > 0
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
		assert.Contains(t, out, "Current version: 6", "all migrations applied")

		assert.Equal(t, 6, appliedSchemaVersion(t))
		assert.True(t, tableExists(t, "candidates"), "migration 000001 should create candidates")
		assert.True(t, tableExists(t, "token_transfers"), "migration 000002 should create token_transfers")
		assert.True(t, tableExists(t, "vote_sessions"), "migration 000003 should add vote_sessions")
		assert.True(t, tableExists(t, "lottery_vrf_keys"), "migration 000004 should add lottery_vrf_keys")
		assert.True(t, indexExists(t, "idx_oracle_data_source_ts"), "migration 000005 should add the ordered oracle index")
		assert.True(t, indexExists(t, "idx_nft_ops_nft_ts"), "migration 000005 should add the ordered NFT index")
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
		assert.False(t, tableExists(t, "vote_sessions"), "step 2 must not apply 000003")

		out, err = runCmd(t, "migrate", "up", "1")
		require.NoError(t, err)
		assert.Contains(t, out, "Current version: 3")
		assert.Equal(t, 3, appliedSchemaVersion(t))

		out, err = runCmd(t, "migrate", "up", "1")
		require.NoError(t, err)
		assert.Contains(t, out, "Current version: 4")
		assert.Equal(t, 4, appliedSchemaVersion(t))
		assert.True(t, tableExists(t, "lottery_vrf_keys"), "step 4 applies 000004")
		assert.False(t, indexExists(t, "idx_oracle_data_source_ts"), "step 4 must not apply 000005")

		out, err = runCmd(t, "migrate", "up", "1")
		require.NoError(t, err)
		assert.Contains(t, out, "Current version: 5")
		assert.Equal(t, 5, appliedSchemaVersion(t))
		assert.True(t, indexExists(t, "idx_oracle_data_source_ts"), "step 5 applies 000005")

		// Step 6 adds the token listing index. A fresh database gets the same
		// index from the repository's own bootstrap, so the migrated and the
		// fresh schema must agree.
		out, err = runCmd(t, "migrate", "up", "1")
		require.NoError(t, err)
		assert.Contains(t, out, "Current version: 6")
		assert.Equal(t, 6, appliedSchemaVersion(t))
		assert.True(t, indexExists(t, "idx_tokens_created"), "step 6 applies 000006")
	})
}

func TestMigrateUp_AlreadyApplied(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setMigrationsPathForTest(t)

		_, err := runCmd(t, "migrate", "up")
		require.NoError(t, err)

		out, err := runCmd(t, "migrate", "up")
		require.NoError(t, err, "re-running up on a migrated DB is a no-op, not an error")
		assert.Contains(t, out, "Current version: 6")
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
		assert.Contains(t, out, "Current version: 6")
		assert.Contains(t, out, "Applied: 1, 2, 3, 4, 5")
		assert.Contains(t, out, "Pending: (none)")
	})
}

func TestMigrateDown_StepsAndBottom(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setMigrationsPathForTest(t)

		_, err := runCmd(t, "migrate", "up")
		require.NoError(t, err)

		out, err := runCmd(t, "migrate", "down", "1", "--confirm")
		require.NoError(t, err)
		assert.Contains(t, out, "Current version: 5")
		assert.Equal(t, 5, appliedSchemaVersion(t))
		assert.False(t, indexExists(t, "idx_tokens_created"), "rolling back 000006 must drop its listing index")

		out, err = runCmd(t, "migrate", "down", "1", "--confirm")
		require.NoError(t, err)
		assert.Contains(t, out, "Current version: 4")
		assert.Equal(t, 4, appliedSchemaVersion(t))
		assert.False(t, indexExists(t, "idx_oracle_data_source_ts"), "rolling back 000005 must drop its ordered index")
		assert.False(t, indexExists(t, "idx_nft_ops_nft_ts"), "rolling back 000005 must drop its ordered index")
		assert.True(t, tableExists(t, "lottery_vrf_keys"), "rolling back 000005 must not touch 000004")

		// Roll down one step at a time until the bottom.
		out, err = runCmd(t, "migrate", "down", "1", "--confirm")
		require.NoError(t, err)
		assert.Contains(t, out, "Current version: 3")
		assert.Equal(t, 3, appliedSchemaVersion(t))
		assert.False(t, tableExists(t, "lottery_vrf_keys"), "rolling back 000004 must drop its side table")

		out, err = runCmd(t, "migrate", "down", "1", "--confirm")
		require.NoError(t, err)
		assert.Contains(t, out, "Current version: 2")
		assert.Equal(t, 2, appliedSchemaVersion(t))

		out, err = runCmd(t, "migrate", "down", "1", "--confirm")
		require.NoError(t, err)
		assert.Contains(t, out, "Current version: 1")
		assert.Equal(t, 1, appliedSchemaVersion(t))

		out, err = runCmd(t, "migrate", "down", "--confirm")
		require.NoError(t, err)
		assert.Contains(t, out, "Current version: 0")
		assert.Equal(t, 0, appliedSchemaVersion(t))

		// down while already at the bottom is a friendly no-op, not an error.
		out, err = runCmd(t, "migrate", "down", "--confirm")
		require.NoError(t, err)
		assert.Contains(t, out, "No migrations to roll back")
	})
}

// TestMigrateUp_OverrunCapsAtPending guards the CLI's step cap: "up 5" with
// fewer pending migrations must apply all and succeed, not apply-then-error.
func TestMigrateUp_OverrunCapsAtPending(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setMigrationsPathForTest(t)

		out, err := runCmd(t, "migrate", "up", "6")
		require.NoError(t, err, "up with N larger than pending must succeed")
		assert.Contains(t, out, "Current version: 6")
		assert.Equal(t, 6, appliedSchemaVersion(t))
	})
}

// TestMigrateDown_OverrunRollsBackAll guards the mirror case: "down 5" on a
// fully migrated DB must roll everything back and report success — previously it
// rolled back both steps, then errored and misprinted "nothing to roll back".
func TestMigrateDown_OverrunRollsBackAll(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setMigrationsPathForTest(t)

		_, err := runCmd(t, "migrate", "up")
		require.NoError(t, err)

		out, err := runCmd(t, "migrate", "down", "6", "--confirm")
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
			// down passes --confirm so the test reaches the validator: the
			// confirm gate is checked first and would otherwise report
			// "pass --confirm" instead of the invalid-count message.
			{args: []string{"migrate", "down", "0", "--confirm"}, msg: "invalid"},
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

func TestMigrateDown_RequiresConfirm(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setMigrationsPathForTest(t)
		_, err := runCmd(t, "migrate", "up")
		require.NoError(t, err)

		// Without --confirm the rollback is refused with a non-zero exit and
		// the schema is left untouched.
		_, err = runCmd(t, "migrate", "down", "1")
		require.Error(t, err, "down without --confirm must be refused")
		assert.Contains(t, err.Error(), "--confirm")
		assert.Equal(t, 6, appliedSchemaVersion(t), "no migration may roll back without confirmation")
	})
}

// TestMigrateRepair_UnsticksADirtyDatabase covers the recovery path end to
// end: a failed migration leaves the database refusing both directions, and
// `migrate repair` is the only way back without deleting every record.
func TestMigrateRepair_UnsticksADirtyDatabase(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setMigrationsPathForTest(t)

		_, err := runCmd(t, "migrate", "up")
		require.NoError(t, err)

		// Mark the database dirty, as a failed migration would.
		db := openTestAuroraDB(t)
		_, err = db.Exec(`UPDATE schema_migrations SET dirty = 1`)
		require.NoError(t, err)

		// Both directions refuse, which is the state this command exists for.
		_, err = runCmd(t, "migrate", "up")
		require.ErrorContains(t, err, "dirty")
		_, err = runCmd(t, "migrate", "down", "--confirm")
		require.ErrorContains(t, err, "dirty")

		// Repair without the confirmation flag refuses, like every other
		// command that changes recorded schema state.
		_, err = runCmd(t, "migrate", "repair")
		require.ErrorContains(t, err, "confirm")

		out, err := runCmd(t, "migrate", "repair", "--confirm")
		require.NoError(t, err)
		assert.Contains(t, out, "cleared at version 6")

		// The database is usable again, at the version it was at.
		out, err = runCmd(t, "migrate", "status")
		require.NoError(t, err)
		assert.Contains(t, out, "Dirty: false")
		assert.Contains(t, out, "Current version: 6")

		_, err = runCmd(t, "migrate", "up")
		require.NoError(t, err, "up must work again after a repair")

		// A second repair reports that there was nothing to do.
		out, err = runCmd(t, "migrate", "repair", "--confirm")
		require.NoError(t, err)
		assert.Contains(t, out, "not dirty")
	})
}
