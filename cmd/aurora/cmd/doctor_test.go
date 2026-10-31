package cmd

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/domain/events"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	infraevents "github.com/pplmx/aurora/internal/infra/events"
)

// TestDoctorReportsAHealthyInstallation: the table names every check and the
// document keeps the same five names in one order, so a support script can tell
// which line warned without parsing the localized table.
func TestDoctorReportsAHealthyInstallation(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setTestLocale(t, "en")

		// Point the schema check at the repository's migrations, the way an
		// installed binary sits next to its migrations directory. The temp
		// working directory makes the configured relative path useless, so
		// resolve it from this file's location.
		_, file, _, ok := runtime.Caller(0)
		require.True(t, ok)
		before := viper.GetString("migrate.path")
		viper.Set("migrate.path", filepath.Join(filepath.Dir(file), "..", "..", "..", "migrations"))
		t.Cleanup(func() { viper.Set("migrate.path", before) })

		out, err := runCmd(t, "doctor")
		require.NoError(t, err)
		for _, want := range []string{"Doctor", "Config:", "Database:", "Migrations:", "Ledger:", "Audit outbox:"} {
			assert.Contains(t, out, want)
		}

		out, err = runCmd(t, "doctor", "--json")
		require.NoError(t, err)
		var report struct {
			Checks []struct {
				Name   string `json:"name"`
				Status string `json:"status"`
				Detail string `json:"detail"`
			} `json:"checks"`
			OK bool `json:"ok"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &report), "got %q", out)
		assert.True(t, report.OK)
		var built struct {
			Version string `json:"version"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &built), "got %q", out)
		assert.NotEmpty(t, built.Version, "the document must identify the build it checked")
		require.Len(t, report.Checks, 5)
		assert.Equal(t, "config", report.Checks[0].Name)
		assert.Equal(t, "database", report.Checks[1].Name)
		assert.Equal(t, "migrations", report.Checks[2].Name)
		assert.Equal(t, "ledger", report.Checks[3].Name)
		assert.Equal(t, "outbox", report.Checks[4].Name)
		assert.NotEmpty(t, report.Checks[1].Detail)
		assert.Equal(t, "ok", report.Checks[2].Status,
			"pending migrations are informational; detail: %s", report.Checks[2].Detail)
		assert.Contains(t, report.Checks[3].Detail, "head", "the ledger line names the head hash")
		assert.NotContains(t, report.Checks[3].Detail, "%!", "every format argument must be consumed")
		assert.NotContains(t, report.Checks[4].Detail, "%!")
	})
}

// TestDoctorWarnsOnPendingOutbox: an audit event waiting for the drainer needs
// attention but does not fail the command — the event is not lost.
func TestDoctorWarnsOnPendingOutbox(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setTestLocale(t, "en")

		dbPath := blockchain.DBPath()
		require.NoError(t, os.MkdirAll(filepath.Dir(dbPath), 0o700))
		store, err := infraevents.NewSQLiteEventStore(dbPath)
		require.NoError(t, err)
		require.NoError(t, store.EnqueuePending(
			events.NewBaseEvent("test.audit", "agg-1", []byte(`{"k":1}`))))
		require.NoError(t, store.Close())

		out, err := runCmd(t, "doctor")
		require.NoError(t, err)
		assert.Contains(t, out, "▲ Audit outbox")
		assert.Contains(t, out, "oldest waits",
			"the warning says whether the retry loop is moving, not only that something is parked")

		out, err = runCmd(t, "doctor", "--json")
		require.NoError(t, err)
		var report struct {
			Checks []struct {
				Name   string `json:"name"`
				Status string `json:"status"`
			} `json:"checks"`
			OK bool `json:"ok"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &report), "got %q", out)
		assert.True(t, report.OK)
		for _, check := range report.Checks {
			if check.Name == "outbox" {
				assert.Equal(t, "warn", check.Status)
			}
		}
	})
}

// TestDoctorReportsAnUnavailableDatabase: when the file cannot open, every
// later check still reports a line instead of vanishing, and the command fails.
func TestDoctorReportsAnUnavailableDatabase(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setTestLocale(t, "en")
		// A directory in place of the database file cannot be opened as one.
		dbPath := filepath.Join(t.TempDir(), "aurora.db")
		require.NoError(t, os.MkdirAll(dbPath, 0o700))
		before := viper.GetString("db.path")
		viper.Set("db.path", dbPath)
		t.Cleanup(func() { viper.Set("db.path", before) })

		_, err := runCmd(t, "doctor")
		require.Error(t, err)

		out, err := runCmd(t, "doctor", "--json")
		require.Error(t, err)
		var report struct {
			Checks []struct {
				Name   string `json:"name"`
				Status string `json:"status"`
				Detail string `json:"detail"`
			} `json:"checks"`
			OK bool `json:"ok"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &report), "got %q", out)
		assert.False(t, report.OK)
		require.Len(t, report.Checks, 5, "every line still reports")
		assert.Equal(t, "database", report.Checks[1].Name)
		assert.Equal(t, "fail", report.Checks[1].Status)
		for _, check := range report.Checks[2:] {
			assert.Equal(t, "fail", check.Status, check.Name)
			assert.Contains(t, check.Detail, "database unavailable", check.Name)
		}
	})
}

// TestDoctorFailsOnBrokenChain: the doctor's exit code is the machine-readable
// verdict, and the ledger check must carry the failure instead of a green line.
func TestDoctorFailsOnBrokenChain(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setTestLocale(t, "en")
		// Keep the database outside ./data: ResetForTest removes that directory.
		dbPath := filepath.Join(t.TempDir(), "chain.db")
		before := viper.GetString("db.path")
		viper.Set("db.path", dbPath)
		t.Cleanup(func() { viper.Set("db.path", before) })

		_, err := runCmd(t, "doctor")
		require.NoError(t, err)

		blockchain.ResetForTest()
		t.Cleanup(blockchain.ResetForTest)

		db, err := sql.Open("sqlite", dbPath)
		require.NoError(t, err)
		_, err = db.Exec("UPDATE blocks SET data = 'tampered' WHERE height = 0")
		require.NoError(t, err)
		require.NoError(t, db.Close())

		_, err = runCmd(t, "doctor")
		require.Error(t, err)

		out, err := runCmd(t, "doctor", "--json")
		require.Error(t, err)
		var report struct {
			Checks []struct {
				Name   string `json:"name"`
				Status string `json:"status"`
			} `json:"checks"`
			OK bool `json:"ok"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &report), "got %q", out)
		assert.False(t, report.OK)
		var ledgerStatus string
		for _, check := range report.Checks {
			if check.Name == "ledger" {
				ledgerStatus = check.Status
			}
		}
		assert.Equal(t, "fail", ledgerStatus)
	})
}

// TestDoctorWarnsOnAWorldReadableConfig: the configuration can hold the API key,
// so a mode that lets another user read it is worth a warning — the command
// still ran, so it is not a failure.
func TestDoctorWarnsOnAWorldReadableConfig(t *testing.T) {
	setTestLocale(t, "en")

	dir := t.TempDir()

	open := filepath.Join(dir, "open.toml")
	require.NoError(t, os.WriteFile(open, []byte("[log]\nlevel = \"error\"\n"), 0o600))
	require.NoError(t, os.Chmod(open, 0o644))
	mode, readable := configReadableByOthers(open)
	require.True(t, readable, "0644 lets another user read the file")
	assert.Equal(t, "0644", mode)

	private := filepath.Join(dir, "private.toml")
	require.NoError(t, os.WriteFile(private, []byte("[log]\nlevel = \"error\"\n"), 0o600))
	_, readable = configReadableByOthers(private)
	assert.False(t, readable, "0600 is the mode the log file uses too")

	_, readable = configReadableByOthers(filepath.Join(dir, "missing.toml"))
	assert.False(t, readable, "a missing file is not a permission warning")
}
