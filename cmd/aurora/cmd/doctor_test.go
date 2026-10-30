package cmd

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/infra/blockchain"
)

// TestDoctorReportsAHealthyInstallation: the table names every check and the
// document keeps the same five names in one order, so a support script can tell
// which line warned without parsing the localized table.
func TestDoctorReportsAHealthyInstallation(t *testing.T) {
	withTempDir(t, func(t *testing.T) {
		setTestLocale(t, "en")

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
		require.Len(t, report.Checks, 5)
		assert.Equal(t, "config", report.Checks[0].Name)
		assert.Equal(t, "database", report.Checks[1].Name)
		assert.Equal(t, "migrations", report.Checks[2].Name)
		assert.Equal(t, "ledger", report.Checks[3].Name)
		assert.Equal(t, "outbox", report.Checks[4].Name)
		assert.NotEmpty(t, report.Checks[1].Detail)
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
