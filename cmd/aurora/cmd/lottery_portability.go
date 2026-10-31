// Moving lottery records between installations: the JSON export and the
// import that reads one back. Both go through the app layer's portability
// helpers, so the CLI, the desktop's File menu and its drop handler accept and
// produce the same document.
package cmd

import (
	"errors"
	"fmt"

	"github.com/pplmx/aurora/internal/app/lotteryapp"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/pplmx/aurora/internal/infra/store"
	"github.com/pplmx/aurora/internal/utils"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export [file.json]",
	Short: i18n.GetText("lottery.export"),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := args[0]

		// Read from the persistent lottery_records table, not the in-memory
		// chain — see historyCmd for the full rationale.
		repo, err := sqlite.NewLotteryRepository(store.Path())
		if err != nil {
			return wrapCLIError("lottery.error.open_repo", err)
		}
		defer func() { _ = repo.Close() }()

		count, err := lotteryapp.WriteRecordsToFile(repo, filename)
		if err != nil {
			return wrapCLIError("lottery.error.write_file", err)
		}

		if jsonRequested(cmd) {
			// The file is the result; the document says where it landed and how
			// many draws it holds, so a script does not parse the message.
			return printJSON(struct {
				Path     string `json:"path"`
				Exported int    `json:"exported"`
			}{Path: filename, Exported: count})
		}
		fmt.Printf("✅ %s\n", i18n.GetTextF("lottery.exported", count, filename))
		return nil
	},
}

var importCmd = &cobra.Command{
	Use:   "import [file.json]",
	Short: i18n.GetText("lottery.import"),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := args[0]

		data, err := lotteryapp.ReadImportFile(filename)
		if err != nil {
			if errors.Is(err, lotteryapp.ErrImportFileTooLarge) {
				return newCLIError("lottery.error.import_too_large",
					utils.FormatBytes(lotteryapp.MaxImportFileSize))
			}
			return wrapCLIError("lottery.error.read_file", err)
		}

		// Persist via the SQLite repository so the structured draw record
		// (ID, winners, VRF proof, block height) survives restarts; the
		// blockchain is a separate append-only audit log.
		repo, err := sqlite.NewLotteryRepository(store.Path())
		if err != nil {
			return wrapCLIError("lottery.error.open_repo", err)
		}
		defer func() { _ = repo.Close() }()

		// Validate + collision-check + save live in the app layer, so the
		// desktop shell's native Import performs exactly this import instead of
		// a re-implementation that could drift (validation, the TASK-069
		// overwrite guard, partial-failure accounting).
		result, err := lotteryapp.ImportRecords(repo, data)
		if err != nil {
			return wrapCLIError("lottery.error.import_records", err)
		}
		if jsonRequested(cmd) {
			// The document always prints; a partial import still exits non-zero
			// so a script notices the rejected indices.
			if err := printJSON(result); err != nil {
				return err
			}
			if len(result.Rejected) > 0 {
				return newCLIError("lottery.import_partial_error", len(result.Rejected), result.Total)
			}
			return nil
		}

		if len(result.Rejected) == 0 {
			fmt.Printf("✅ %s\n", i18n.GetTextF("lottery.imported", result.Imported))
			return nil
		}
		// Partial failure: surface the problem honestly. Returning a
		// non-nil error also lets CI / shell scripts detect the partial
		// failure via $?, instead of silently treating it as success.
		fmt.Printf("⚠️  %s\n",
			i18n.GetTextF("lottery.import_partial", result.Imported, result.Total, len(result.Rejected)))
		fmt.Printf("   %s\n", i18n.GetTextF("lottery.failed_indices", result.Rejected))
		return newCLIError("lottery.import_partial_error", len(result.Rejected), result.Total)
	},
}
