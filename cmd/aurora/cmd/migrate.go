package cmd

import (
	"fmt"
	"strconv"
	"strings"

	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/migrate"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// migrationSteps parses the optional [N] step-count argument. def is the
// default used when the argument is omitted (up: all, down: 1). A present
// but invalid count (non-integer, or < 1) is a usage error, not a silent
// no-op — "migrate up 0" must not be misread as "apply everything".
func migrationSteps(args []string, def int) (int, error) {
	if len(args) == 0 {
		return def, nil
	}
	n, err := strconv.Atoi(args[0])
	if err != nil || n < 1 {
		return 0, fmt.Errorf("invalid migration count %q: must be a positive integer", args[0])
	}
	return n, nil
}

// newCLIMigrator builds a migrator against the same SQLite file the rest of
// the CLI uses (blockchain.DBPath(), i.e. ./data/aurora.db relative to the
// cwd) plus the configured migration source (migrate.path, default
// ./migrations).
func newCLIMigrator() (*migrate.Migrator, error) {
	dbPath := blockchain.DBPath()
	if dbPath == "" {
		// DBPath returns "" only when the ./data directory cannot be
		// created — passing "" to migrate.New would silently create a
		// junk file literally named "?_foreign_keys=ON" in the cwd.
		return nil, fmt.Errorf("failed to resolve database path: data directory unavailable")
	}

	migPath := viper.GetString("migrate.path")
	if migPath == "" {
		migPath = "./migrations"
	}
	m, err := migrate.New(dbPath, migPath)
	if err != nil {
		return nil, fmt.Errorf("migrator init failed: %w", err)
	}
	return m, nil
}

// joinVersions renders a []uint as "1, 2" or the i18n "(none)" placeholder.
func joinVersions(versions []uint) string {
	if len(versions) == 0 {
		return i18n.GetText("migrate.none")
	}
	parts := make([]string, len(versions))
	for i, v := range versions {
		parts[i] = strconv.FormatUint(uint64(v), 10)
	}
	return strings.Join(parts, ", ")
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: i18n.GetText("migrate.short"),
	Long:  i18n.GetText("migrate.long"),
	Example: `  aurora migrate up        # apply all pending migrations
  aurora migrate up 2      # apply only the next 2
  aurora migrate status    # show current / applied / pending
  aurora migrate down 1    # roll back one step (default)`,
}

var migrateUpCmd = &cobra.Command{
	Use:   "up [N]",
	Short: i18n.GetText("migrate.up"),
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		steps, err := migrationSteps(args, 0)
		if err != nil {
			return err
		}

		m, err := newCLIMigrator()
		if err != nil {
			return err
		}
		defer func() { _ = m.Close() }()

		before, err := m.Version()
		if err != nil {
			return err
		}

		// Cap an explicit N to the number of genuinely pending migrations.
		// golang-migrate's Steps(N) applies what it can and then errors on the
		// overrun ("limit 3 short" / "file does not exist"), so "up 5" with
		// two pending would apply both and still exit non-zero. We instead
		// apply min(N, pending) as a plain success.
		if steps > 0 {
			st, err := m.Status()
			if err != nil {
				return err
			}
			if len(st.PendingVersions) < steps {
				steps = len(st.PendingVersions)
			}
			if steps == 0 {
				fmt.Println(i18n.GetTextF("migrate.already", before))
				return nil
			}
		}

		version, err := m.Up(steps)
		if err != nil {
			return err
		}

		if version == before && before > 0 {
			fmt.Println(i18n.GetTextF("migrate.already", version))
		} else {
			fmt.Println(i18n.GetTextF("migrate.up_done", version))
		}
		return nil
	},
}

var migrateDownCmd = &cobra.Command{
	Use:   "down [N]",
	Short: i18n.GetText("migrate.down"),
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Rolling back drops schema/data; require -y/--confirm first.
		if err := requireConfirm(cmd, "rolls back and drops schema"); err != nil {
			return err
		}
		steps, err := migrationSteps(args, 1)
		if err != nil {
			return err
		}

		m, err := newCLIMigrator()
		if err != nil {
			return err
		}
		defer func() { _ = m.Close() }()

		before, err := m.Version()
		if err != nil {
			return err
		}

		// Cap N to the applied count so "down 5" on a version-2 DB rolls back
		// both steps and reports success instead of hitting golang-migrate's
		// overrun error after the rollback already happened (which previously
		// also misprinted "nothing to roll back").
		if steps > int(before) {
			steps = int(before)
		}
		if steps == 0 {
			fmt.Println(i18n.GetText("migrate.none_to_rollback"))
			return nil
		}

		version, err := m.Down(steps)
		if err != nil {
			return err
		}

		fmt.Println(i18n.GetTextF("migrate.down_done", version))
		return nil
	},
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: i18n.GetText("migrate.status"),
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := newCLIMigrator()
		if err != nil {
			return err
		}
		defer func() { _ = m.Close() }()

		st, err := m.Status()
		if err != nil {
			return err
		}

		fmt.Println("\n" + i18n.GetText("migrate.status_header"))
		fmt.Println(i18n.GetTextF("migrate.current_version", st.Current))
		fmt.Println(i18n.GetTextF("migrate.dirty", st.Dirty))
		fmt.Println(i18n.GetTextF("migrate.applied", joinVersions(st.Applied)))
		fmt.Println(i18n.GetTextF("migrate.pending", joinVersions(st.PendingVersions)))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateDownCmd)
	migrateCmd.AddCommand(migrateStatusCmd)
	// Rolling back is destructive; require -y/--confirm first.
	addConfirmFlag(migrateDownCmd, "Roll back the schema (requires --confirm)")
}
