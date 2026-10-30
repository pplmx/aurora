package cmd

import (
	"database/sql"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	"github.com/pplmx/aurora/internal/utils"
)

// Doctor status values. A warning keeps the command's exit code at zero (the
// data is readable, something just wants attention); a failure makes it
// non-zero, like the verify commands.
const (
	doctorOK   = "ok"
	doctorWarn = "warn"
	doctorFail = "fail"
)

// doctorCheck is one line of the self-check. Name is a stable English key (the
// table localizes it, the JSON keeps it stable for scripts).
type doctorCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

// doctorCmd answers the first support question — "is this installation
// healthy?" — in one pass instead of db-info + blockchain verify + migrate
// status, and reports the result as data (--json) or a short table.
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("doctor.short"),
	RunE: func(cmd *cobra.Command, args []string) error {
		checks := runDoctor()
		healthy := true
		for _, check := range checks {
			if check.Status == doctorFail {
				healthy = false
			}
		}

		if jsonRequested(cmd) {
			if err := printJSON(struct {
				Checks []doctorCheck `json:"checks"`
				OK     bool          `json:"ok"`
			}{Checks: checks, OK: healthy}); err != nil {
				return err
			}
		} else {
			fmt.Println("\n🩺 " + i18n.GetText("doctor.title"))
			fmt.Println("────────────────────────────")
			for _, check := range checks {
				mark := "✔"
				switch check.Status {
				case doctorWarn:
					mark = "▲"
				case doctorFail:
					mark = "✖"
				}
				fmt.Printf("  %s %s: %s\n", mark, i18n.GetText("doctor.check."+check.Name), check.Detail)
			}
		}

		if !healthy {
			return newCLIError("doctor.failed")
		}
		return nil
	},
}

// runDoctor collects the checks in a fixed order so two runs (and the JSON)
// agree. It never mutates data beyond opening the configured database.
func runDoctor() []doctorCheck {
	checks := make([]doctorCheck, 0, 5)

	configDetail := i18n.GetText("doctor.detail.defaults")
	if used := viper.ConfigFileUsed(); used != "" {
		configDetail = used
	}
	checks = append(checks, doctorCheck{Name: "config", Status: doctorOK, Detail: configDetail})

	path := blockchain.DBPath()
	db, err := blockchain.InitDB()
	if err != nil {
		checks = append(checks, doctorCheck{Name: "database", Status: doctorFail, Detail: err.Error()})
		return appendUnavailable(checks, "migrations", "ledger", "outbox")
	}
	checks = append(checks, doctorCheck{
		Name: "database", Status: doctorOK,
		Detail: fmt.Sprintf("%s (%s)", path, utils.FormatBytes(utils.SQLiteDiskUsage(path))),
	})

	checks = append(checks, doctorMigrationsCheck())
	checks = append(checks, doctorLedgerCheck())
	checks = append(checks, doctorOutboxCheck(db))
	return checks
}

// doctorMigrationsCheck reads the schema version without applying anything; a
// dirty version is a hard failure because migrations cannot be re-run over it.
// An unreadable migrations source is a warning: a CLI-only installation never
// needs the directory, so it must not fail the whole report.
func doctorMigrationsCheck() doctorCheck {
	migrator, err := newCLIMigrator()
	if err != nil {
		return doctorCheck{Name: "migrations", Status: doctorWarn, Detail: err.Error()}
	}
	defer func() { _ = migrator.Close() }()

	status, err := migrator.Status()
	if err != nil {
		return doctorCheck{Name: "migrations", Status: doctorWarn, Detail: err.Error()}
	}
	if status.Dirty {
		return doctorCheck{Name: "migrations", Status: doctorFail,
			Detail: i18n.GetTextF("doctor.detail.migrations_dirty", status.Current)}
	}
	detail := i18n.GetTextF("doctor.detail.migrations", status.Current, len(status.PendingVersions))
	if len(status.PendingVersions) > 0 {
		return doctorCheck{Name: "migrations", Status: doctorWarn, Detail: detail}
	}
	return doctorCheck{Name: "migrations", Status: doctorOK, Detail: detail}
}

// doctorLedgerCheck re-runs the same integrity verification the blockchain
// page and `blockchain verify` use.
func doctorLedgerCheck() doctorCheck {
	chain, err := blockchain.InitWithError()
	if err != nil {
		return doctorCheck{Name: "ledger", Status: doctorFail, Detail: err.Error()}
	}
	report := chain.VerifyIntegrity()
	if !report.Valid {
		return doctorCheck{Name: "ledger", Status: doctorFail,
			Detail: i18n.GetTextF("doctor.detail.ledger_broken",
				report.FirstBrokenIndex, report.FirstBrokenReason, report.Length)}
	}
	return doctorCheck{Name: "ledger", Status: doctorOK,
		Detail: i18n.GetTextF("doctor.detail.ledger_ok", report.Length)}
}

// doctorOutboxCheck reports events still waiting for the API drainer. The CLI
// drains synchronously before closing, so a backlog here means an earlier run
// could not deliver and the next API start will retry.
func doctorOutboxCheck(db *sql.DB) doctorCheck {
	var pending int
	err := db.QueryRow("SELECT COUNT(*) FROM pending_events").Scan(&pending)
	if err != nil && !isNoSuchTable(err) {
		return doctorCheck{Name: "outbox", Status: doctorFail, Detail: err.Error()}
	}
	detail := i18n.GetTextF("doctor.detail.outbox", pending)
	if pending > 0 {
		return doctorCheck{Name: "outbox", Status: doctorWarn, Detail: detail}
	}
	return doctorCheck{Name: "outbox", Status: doctorOK, Detail: detail}
}

// appendUnavailable marks the checks that cannot run without a database, so the
// table still lists every line and the JSON stays the same length.
func appendUnavailable(checks []doctorCheck, names ...string) []doctorCheck {
	detail := i18n.GetText("doctor.detail.no_database")
	for _, name := range names {
		checks = append(checks, doctorCheck{Name: name, Status: doctorFail, Detail: detail})
	}
	return checks
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	addJSONFlag(doctorCmd)
}
