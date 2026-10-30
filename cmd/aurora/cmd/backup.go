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
		if jsonRequested(cmd) {
			return printJSON(res)
		}
		fmt.Println(i18n.GetTextF("backup.create_done", res.Dir, res.Size, res.SchemaVersion))
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
		if jsonRequested(cmd) {
			// No details beyond "it verified": the metadata and content hashes
			// were checked, or the command would have failed.
			return printJSON(struct {
				Path   string `json:"path"`
				Status string `json:"status"`
			}{Path: args[0], Status: "verified"})
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
		if err := requireConfirm(cmd, "cli.confirm.backup_restore"); err != nil {
			return err
		}
		svc, err := newCLIBackup()
		if err != nil {
			return err
		}
		if err := svc.Restore(context.Background(), args[0]); err != nil {
			return wrapCLIError("backup.error.restore", err)
		}
		// The safety archive is part of the result, not just a hint: a script
		// that restores needs to know the undo directory exists.
		undo := filepath.Clean(args[0]) + ".pre_restore"
		if jsonRequested(cmd) {
			out := struct {
				Path   string `json:"path"`
				Status string `json:"status"`
				Undo   string `json:"undo,omitempty"`
			}{Path: args[0], Status: "restored"}
			if _, err := os.Stat(filepath.Join(undo, "metadata.json")); err == nil {
				out.Undo = undo
			}
			return printJSON(out)
		}

		fmt.Println(i18n.GetTextF("backup.restore_done", args[0]))
		// The safety copy is a complete archive, so an undo is one command
		// instead of copying database files by hand; print it only when this
		// restore had a previous database to keep.
		if _, err := os.Stat(filepath.Join(undo, "metadata.json")); err == nil {
			fmt.Println(i18n.GetTextF("backup.restore_undo", undo, undo))
		}
		return nil
	},
}

func init() {
	// One gate for every destructive command: same -y/--confirm flag, same
	// localized refusal, same exit code (TASK-152, ISS-140).
	addConfirmFlag(backupRestoreCmd, "cli.confirm.backup_restore")
	rootCmd.AddCommand(backupCmd)
	backupCmd.AddCommand(backupCreateCmd)
	addJSONFlag(backupCreateCmd)
	backupCmd.AddCommand(backupVerifyCmd)
	addJSONFlag(backupVerifyCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	addJSONFlag(backupRestoreCmd)
}
