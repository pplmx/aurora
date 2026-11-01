// Inspecting and resetting the lottery store: the counters, the destructive
// reset, and the database facts an operator quotes in a support thread.
package cmd

import (
	"fmt"

	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	"github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/pplmx/aurora/internal/infra/store"
	"github.com/pplmx/aurora/internal/logger"
	"github.com/pplmx/aurora/internal/utils"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: i18n.GetText("lottery.stats"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read from the persistent lottery_records table (the same source as
		// history/verify/export/import), NOT the in-memory chain. The chain
		// only reflects draws added as blocks; imported draws write
		// lottery_records and never reach the chain, so counting the chain
		// made stats disagree with history (TASK-070, ISS-062).
		repo, err := sqlite.NewLotteryRepository(store.Path())
		if err != nil {
			return wrapCLIError("lottery.error.open_repo", err)
		}
		defer func() { _ = repo.Close() }()

		records, err := repo.GetAll()
		if err != nil {
			return wrapCLIError("lottery.error.read_history", err)
		}

		var latest int64 = -1
		for _, r := range records {
			if r.BlockHeight > latest {
				latest = r.BlockHeight
			}
		}
		if jsonRequested(cmd) {
			out := struct {
				Total       int    `json:"total"`
				DBPath      string `json:"db_path"`
				LatestBlock *int64 `json:"latest_block,omitempty"`
			}{Total: len(records), DBPath: store.Path()}
			if latest >= 0 {
				out.LatestBlock = &latest
			}
			return printJSON(out)
		}

		fmt.Println("\n📊 " + i18n.GetText("lottery.statistics"))
		fmt.Println("────────────────────────────")
		fields := []field{
			{i18n.GetText("lottery.total"), fmt.Sprint(len(records))},
			{i18n.GetText("lottery.db_path"), store.Path()},
		}
		if latest >= 0 {
			fields = append(fields, field{i18n.GetText("lottery.latest_block"), fmt.Sprintf("#%d", latest)})
		}
		printFields(fields...)

		return nil
	},
}

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: i18n.GetText("lottery.reset"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// The destructive reset did NOT run when the gate is missing — surface
		// that as an error (non-zero exit) instead of printing a warning and
		// exiting 0, like the other destructive commands. A script that forgot
		// the gate must be able to detect the refusal.
		if err := requireConfirm(cmd, "cli.confirm.lottery_reset"); err != nil {
			return err
		}

		db, err := store.OpenShared()
		if err != nil {
			return wrapCLIError("lottery.error.init_db", err)
		}
		// Deliberately no db.Close(): InitDB returns the process-wide
		// singleton under a sync.Once, and closing it without resetting the
		// once poisons that singleton — any later InitDB() call in the same
		// process (a composable command flow or a test) gets an already-
		// closed handle. Every other CLI command (history/verify/create/...)
		// and cmd/api (Close once at shutdown) let the handle live until the
		// process exits, which is exactly right for this single-command
		// process (TASK-262, ISS-258).

		// A lottery lives in two stores: the chain (`blocks` table) and the
		// persistent history (`lottery_records` table, which historyCmd,
		// verifyCmd, exportCmd and importCmd read). Deleting only `blocks`
		// left `lottery_records` intact, so after a reset, `lottery history`
		// still listed every draw — contradicting the "delete ALL lottery
		// records!" confirmation. Clear both.
		//
		// On a brand-new DB neither table exists yet (created lazily by
		// Init / the repository / migrations), so a DELETE against
		// a missing table is "no such table" — tolerated, not an error.
		for _, stmt := range []string{
			"DELETE FROM blocks WHERE height > 0",
			"DELETE FROM lottery_vrf_keys",
			"DELETE FROM lottery_records",
		} {
			if _, err := db.Exec(stmt); err != nil && !isNoSuchTable(err) {
				return wrapCLIError("lottery.error.reset", err)
			}
		}

		// Reset the in-memory chain singleton back to genesis. The chain is
		// created once (Init's sync.Once) and is not rebuilt by the
		// DELETEs above, so without this a subsequent `lottery create` would
		// keep building on the stale pre-reset blocks: the new draw would get
		// a block height continuing from the old count and a PrevHash pointing
		// at a block the reset deleted (TASK-071, ISS-063).
		chain, err := blockchain.InitWithError()
		if err != nil {
			return wrapCLIError("lottery.error.init_blockchain_after_reset", err)
		}
		chain.ResetBlocks()

		logger.Info().Msg("Database reset successfully")
		fmt.Println("✅ " + i18n.GetText("lottery.reset_done"))
		return nil
	},
}

var dbInfoCmd = &cobra.Command{
	Use:   "db-info",
	Short: i18n.GetText("lottery.db_info"),
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := store.OpenShared()
		if err != nil {
			return wrapCLIError("lottery.error.init_db", err)
		}
		// No db.Close() here either — same singleton-ownership reasoning as
		// the reset command above (TASK-262, ISS-258).

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM blocks WHERE height > 0").Scan(&count)
		switch {
		case err == nil:
			// ok
		case isNoSuchTable(err):
			// A brand-new DB has no blocks table yet; report 0 truthfully
			// instead of surfacing a "no such table" failure (TASK-154,
			// mirroring lottery reset's tolerance).
			count = 0
		default:
			return wrapCLIError("lottery.error.count_blocks", err)
		}

		path := store.Path()
		size := utils.SQLiteDiskUsage(path)
		if jsonRequested(cmd) {
			return printJSON(struct {
				Path      string `json:"path"`
				SizeBytes int64  `json:"size_bytes"`
				Blocks    int    `json:"blocks"`
			}{Path: path, SizeBytes: size, Blocks: count})
		}

		fmt.Println("\n📁 " + i18n.GetText("lottery.db_info_title"))
		fmt.Println("────────────────────────────")
		printFields(
			field{i18n.GetText("lottery.db_path"), path},
			field{i18n.GetText("lottery.db_size"), utils.FormatBytes(size)},
			field{i18n.GetText("lottery.total_blocks"), fmt.Sprint(count)},
		)
		return nil
	},
}
