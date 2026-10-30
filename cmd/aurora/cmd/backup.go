package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/backup"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	"github.com/spf13/cobra"
)

// newCLIBackup builds a backup service over the same SQLite file the rest of
// the CLI uses (blockchain.DBPath(), i.e. ./data/aurora.db relative to the
// cwd). Opening and reading the source owns filesystem-error reporting.
func newCLIBackup() (*backup.BackupService, error) {
	dbPath := blockchain.DBPath()
	return backup.NewBackupService(map[string]string{"aurora": dbPath}), nil
}

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: i18n.GetText("backup.short"),
	Long:  i18n.GetText("backup.long"),
	Example: `  aurora backup create ./backups/backup-20260822   # backup the database
  aurora backup verify ./backups/backup-20260822          # check integrity
  aurora backup restore ./backups/backup-20260822 --confirm`,
}

var backupCreateCmd = &cobra.Command{
	Use:   "create <dir>",
	Short: i18n.GetText("backup.create"),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newCLIBackup()
		if err != nil {
			return err
		}
		res, err := svc.Create(context.Background(), args[0])
		if err != nil {
			return wrapCLIError("backup.error.create", err)
		}
		fmt.Println(i18n.GetTextF("backup.create_done", res.File, res.Size, res.SchemaVersion))
		return nil
	},
}

var backupVerifyCmd = &cobra.Command{
	Use:   "verify <dir>",
	Short: i18n.GetText("backup.verify"),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := newCLIBackup()
		if err != nil {
			return err
		}
		if err := svc.Verify(context.Background(), args[0]); err != nil {
			return wrapCLIError("backup.error.verify", err)
		}
		fmt.Println(i18n.GetText("backup.verify_ok"))
		return nil
	},
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore <dir>",
	Short: i18n.GetText("backup.restore"),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		confirm, _ := cmd.Flags().GetBool("confirm")
		if !confirm {
			return newCLIError("backup.confirm_required")
		}
		svc, err := newCLIBackup()
		if err != nil {
			return err
		}
		if err := svc.Restore(context.Background(), args[0]); err != nil {
			return wrapCLIError("backup.error.restore", err)
		}
		fmt.Println(i18n.GetTextF("backup.restore_done", args[0]))
		// The safety copy is a complete archive, so an undo is one command
		// instead of copying database files by hand; print it only when this
		// restore had a previous database to keep.
		undo := filepath.Clean(args[0]) + ".pre_restore"
		if _, err := os.Stat(filepath.Join(undo, "metadata.json")); err == nil {
			fmt.Println(i18n.GetTextF("backup.restore_undo", undo, undo))
		}
		return nil
	},
}

func init() {
	// -y shorthand matches the canonical addConfirmFlag gate used by token burn,
	// nft burn, migrate down and oracle source delete, so a script standardizing
	// on -y/--confirm works here too (TASK-152, ISS-140).
	backupRestoreCmd.Flags().BoolP("confirm", "y", false, i18n.GetText("backup.confirm_flag"))
	rootCmd.AddCommand(backupCmd)
	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupVerifyCmd)
	backupCmd.AddCommand(backupRestoreCmd)
}
