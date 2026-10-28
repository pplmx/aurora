package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/pplmx/aurora/internal/app/lotteryapp"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	"github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/pplmx/aurora/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var lotteryCmd = &cobra.Command{
	Use:   "lottery",
	Short: i18n.GetText("lottery.cmd"),
	Long:  i18n.GetText("lottery.long"),
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: i18n.GetText("lottery.create"),
	Example: `  aurora lottery create -p "Alice,Bob,Charlie" -s "random-seed-123"
  aurora lottery create -p "A,B,C,D,E" -s "my-lottery" -c 3`,
	RunE: func(cmd *cobra.Command, args []string) error {
		participantsStr, _ := cmd.Flags().GetString("participants")
		seed, _ := cmd.Flags().GetString("seed")
		// Resolve the winner count at run time, not at flag-registration time
		// (init): the flag was registered with viper.GetInt("lottery.defaultCount")
		// before initConfig/setDefaultConfig ever ran, so the default was always 0
		// and any `lottery create` without an explicit -c failed with "winner count
		// must be positive" (TASK-108, ISS-100). When -c is absent, fall back to the
		// configured lottery.defaultCount (default 3), which IS populated by the
		// time RunE executes.
		count := viper.GetInt("lottery.defaultCount")
		if cmd.Flags().Changed("count") {
			count, _ = cmd.Flags().GetInt("count")
		}

		lotteryRepo, err := sqlite.NewLotteryRepository(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to create lottery repository: %w", err)
		}
		defer func() { _ = lotteryRepo.Close() }()

		chain, err := blockchain.InitWithError()
		if err != nil {
			return fmt.Errorf("failed to initialize blockchain: %w", err)
		}

		uc := lotteryapp.NewCreate(lotteryRepo, chain)

		req := lotteryapp.CreateLotteryRequest{
			Participants: participantsStr,
			Seed:         seed,
			WinnerCount:  count,
		}

		resp, err := uc.Execute(req)
		if err != nil {
			return fmt.Errorf("failed to create lottery: %w", err)
		}

		fmt.Println("\n✅ " + i18n.GetText("lottery.success"))
		fmt.Printf("📋 %s: %s\n", i18n.GetText("lottery.lottery_id"), resp.ID)
		fmt.Printf("🔢 %s: #%d\n", i18n.GetText("lottery.block_height"), resp.BlockHeight)
		fmt.Println("\n🎉 " + i18n.GetText("lottery.winners") + ":")
		for i, w := range resp.Winners {
			// Guard against mismatched slice lengths (could happen with
			// imported data, older DB schemas, or partial writes).
			// Without this check the CLI panics with index-out-of-range.
			addr := i18n.GetText("lottery.no_address")
			if i < len(resp.WinnerAddresses) {
				addr = resp.WinnerAddresses[i]
			}
			fmt.Printf("   %d. %s (%s)\n", i+1, w, addr)
		}
		fmt.Printf("\n🔐 %s: %s...\n", i18n.GetText("lottery.vrf_output"), resp.VRFOutput[:min(16, len(resp.VRFOutput))])
		fmt.Printf("📜 %s: %s...\n", i18n.GetText("lottery.vrf_proof"), resp.VRFProof[:min(16, len(resp.VRFProof))])

		return nil
	},
}

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: i18n.GetText("lottery.history"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read from the persistent lottery_records table rather than the
		// blockchain's raw blocks. The chain is persisted (AddBlock writes to
		// the blocks table and reloads on start), but lottery_records holds
		// the structured draw data (winners, addresses, VRF proof, block
		// height) that raw blocks do not carry, and survives restarts without
		// parsing block data.
		repo, err := sqlite.NewLotteryRepository(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to open lottery repository: %w", err)
		}
		defer func() { _ = repo.Close() }()

		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		if limit < 0 {
			return fmt.Errorf("invalid --limit %d: must be zero or greater", limit)
		}
		if offset < 0 {
			return fmt.Errorf("invalid --offset %d: must be zero or greater", offset)
		}

		records, err := repo.List(limit, offset)
		if err != nil {
			return fmt.Errorf("failed to read history: %w", err)
		}

		if len(records) == 0 {
			if limit > 0 || offset > 0 {
				fmt.Printf("\n📜 %s: %s\n",
					i18n.GetText("lottery.page_total"),
					i18n.GetTextF("common.page_with_offset", 0, offset))
				return nil
			}
			fmt.Println(i18n.GetText("lottery.no_records"))
			return nil
		}

		if limit > 0 {
			fmt.Printf("\n📜 %s: %s\n\n",
				i18n.GetText("lottery.page_total"),
				i18n.GetTextF("common.page_with_offset", len(records), offset))
		} else {
			fmt.Printf("\n📜 %s: %d\n\n", i18n.GetText("lottery.total"), len(records))
		}
		for i, record := range records {
			jsonData, err := record.ToJSON()
			if err != nil {
				continue
			}
			fmt.Printf("%s\n", i18n.GetTextF("lottery.history_item", i+1))
			fmt.Println(jsonData[:min(200, len(jsonData))])
			fmt.Println()
		}
		return nil
	},
}

var verifyCmd = &cobra.Command{
	Use:   "verify [lottery-id or block-height]",
	Short: i18n.GetText("lottery.verify"),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		input := args[0]

		// Open the persistent lottery repository. Lookups go through the
		// dedicated lottery_records table (structured draw data that survives
		// restarts) rather than the blockchain's raw blocks.
		repo, err := sqlite.NewLotteryRepository(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to open lottery repository: %w", err)
		}
		defer func() { _ = repo.Close() }()

		// Resolution (exact ID → whole block height → substring) and the
		// integrity checks live in the app layer, shared with the desktop
		// shell so both report identical verdicts for the same record.
		report, err := lotteryapp.VerifyInput(repo, input)
		if err != nil {
			return err
		}
		record := report.Record

		// Display verification info
		fmt.Println("\n📋 " + i18n.GetText("lottery.lottery_id") + ": " + record.ID)
		fmt.Printf("🔢 %s: #%d\n", i18n.GetText("common.block_height"), record.BlockHeight)
		fmt.Printf("🌱 %s: %s\n", i18n.GetText("lottery.seed_label"), record.Seed)
		fmt.Printf("👥 %s: %d\n", i18n.GetText("lottery.participants_label"), len(record.Participants))
		fmt.Printf("🎉 %s: %d\n", i18n.GetText("lottery.winners"), len(record.Winners))

		// Draws created before v1.31 have no persisted public key, so the
		// verdict falls back to a deterministic winner match; later draws also
		// get the cryptographic proof check (see app/lottery.VerifyRecord).
		// The "✅ Verified" line is printed only for a passing record — it must
		// never follow a plain parse, which was a false positive.
		for _, check := range report.Checks {
			if check.Status == lotteryapp.VerifyFail {
				fmt.Printf("\n❌ %s\n", i18n.GetTextF("lottery.verify_failed", check.Message))
			}
		}
		if report.IntegrityOK {
			if report.ProofVerified {
				fmt.Println("\n✅ " + i18n.GetText("lottery.verified"))
			} else {
				fmt.Println("\n✅ " + i18n.GetText("lottery.verified_deterministic"))
			}
		}

		fmt.Printf("\n🏆 %s:\n", i18n.GetText("lottery.winners"))
		for i, w := range record.Winners {
			// Same guard as in createCmd: defend against mismatched slice
			// lengths in imported/corrupted records.
			addr := i18n.GetText("lottery.no_address")
			if i < len(record.WinnerAddresses) {
				addr = record.WinnerAddresses[i]
			}
			fmt.Printf("   %d. %s (%s)\n", i+1, w, addr)
		}
		fmt.Printf("\n🔐 %s: %s...\n", i18n.GetText("lottery.vrf_output"), record.VRFOutput[:min(16, len(record.VRFOutput))])
		fmt.Printf("📜 %s: %s...\n", i18n.GetText("lottery.vrf_proof"), record.VRFProof[:min(16, len(record.VRFProof))])
		if record.VRFPublicKey != "" {
			fmt.Printf("🔑 %s: %s...\n", i18n.GetText("lottery.vrf_public_key"), record.VRFPublicKey[:min(16, len(record.VRFPublicKey))])
		}
		fmt.Printf("⏰ %s: %d\n", i18n.GetText("common.timestamp"), record.Timestamp)

		if !report.IntegrityOK {
			// Surface the failure to the caller (e.g. shell scripts that
			// check $?) without also printing the success summary.
			return lotteryapp.ErrVerificationFailed
		}
		return nil
	},
}

var exportCmd = &cobra.Command{
	Use:   "export [file.json]",
	Short: i18n.GetText("lottery.export"),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := args[0]

		// Read from the persistent lottery_records table, not the in-memory
		// chain — see historyCmd for the full rationale.
		repo, err := sqlite.NewLotteryRepository(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to open lottery repository: %w", err)
		}
		defer func() { _ = repo.Close() }()

		count, err := lotteryapp.WriteRecordsToFile(repo, filename)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
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

		data, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		// Persist via the SQLite repository so the structured draw record
		// (ID, winners, VRF proof, block height) survives restarts; the
		// blockchain is a separate append-only audit log.
		repo, err := sqlite.NewLotteryRepository(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to open lottery repository: %w", err)
		}
		defer func() { _ = repo.Close() }()

		// Validate + collision-check + save live in the app layer, so the
		// desktop shell's native Import performs exactly this import instead of
		// a re-implementation that could drift (validation, the TASK-069
		// overwrite guard, partial-failure accounting).
		result, err := lotteryapp.ImportRecords(repo, data)
		if err != nil {
			return fmt.Errorf("failed to import records: %w", err)
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
		return fmt.Errorf(i18n.GetText("lottery.import_partial_error"), len(result.Rejected), result.Total)
	},
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: i18n.GetText("lottery.stats"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read from the persistent lottery_records table (the same source as
		// history/verify/export/import), NOT the in-memory chain. The chain
		// only reflects draws added as blocks; imported draws write
		// lottery_records and never reach the chain, so counting the chain
		// made stats disagree with history (TASK-070, ISS-062).
		repo, err := sqlite.NewLotteryRepository(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to open lottery repository: %w", err)
		}
		defer func() { _ = repo.Close() }()

		records, err := repo.GetAll()
		if err != nil {
			return fmt.Errorf("failed to read history: %w", err)
		}

		fmt.Println("\n📊 " + i18n.GetText("lottery.statistics"))
		fmt.Println("────────────────────────────")
		fmt.Printf("  %s: %d\n", i18n.GetText("lottery.total"), len(records))
		fmt.Printf("  %s: %s\n", i18n.GetText("lottery.db_path"), blockchain.DBPath())

		if len(records) > 0 {
			// Report the highest recorded block height across persisted draws.
			var latest int64 = -1
			for _, r := range records {
				if r.BlockHeight > latest {
					latest = r.BlockHeight
				}
			}
			if latest >= 0 {
				fmt.Printf("  %s: #%d\n", i18n.GetText("lottery.latest_block"), latest)
			}
		}

		return nil
	},
}

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: i18n.GetText("lottery.reset"),
	RunE: func(cmd *cobra.Command, args []string) error {
		yes, _ := cmd.Flags().GetBool("yes")
		confirm, _ := cmd.Flags().GetBool("confirm")
		if !yes && !confirm {
			// The destructive reset did NOT run — surface that as an error
			// (non-zero exit) instead of printing a warning and exiting 0,
			// matching backup restore --confirm (TASK-100, ISS-092). A
			// script that forgot the gate must be able to detect the refusal.
			// --yes (legacy) and --confirm both count (TASK-152, ISS-140). The
			// refusal shares the localized cli.confirm.required template with
			// the other five destructive commands (TASK-186, ISS-182).
			return fmt.Errorf(i18n.GetText("cli.confirm.required"), i18n.GetText("cli.confirm.lottery_reset"))
		}

		db, err := blockchain.InitDB()
		if err != nil {
			return fmt.Errorf("failed to init db: %w", err)
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
				return fmt.Errorf("failed to reset: %w", err)
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
			return fmt.Errorf("failed to initialize blockchain after reset: %w", err)
		}
		chain.ResetBlocks()

		logger.Info().Msg("Database reset successfully")
		fmt.Println("✅ " + i18n.GetText("lottery.reset_done"))
		return nil
	},
}

// isNoSuchTable reports whether err is SQLite's "no such table" error.
// lottery reset tolerates it because blocks and lottery_records are created
// lazily and may not exist on a brand-new DB.
func isNoSuchTable(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no such table")
}

var dbInfoCmd = &cobra.Command{
	Use:   "db-info",
	Short: i18n.GetText("lottery.db_info"),
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := blockchain.InitDB()
		if err != nil {
			return fmt.Errorf("failed to init db: %w", err)
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
			return fmt.Errorf("failed to count blocks: %w", err)
		}

		fmt.Println("\n📁 " + i18n.GetText("lottery.db_info_title"))
		fmt.Println("────────────────────────────")
		fmt.Printf("  %s: %s\n", i18n.GetText("lottery.db_path"), blockchain.DBPath())
		fmt.Printf("  %s: %d\n", i18n.GetText("lottery.total_blocks"), count)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lotteryCmd)
	lotteryCmd.AddCommand(createCmd)
	lotteryCmd.AddCommand(historyCmd)
	lotteryCmd.AddCommand(verifyCmd)
	lotteryCmd.AddCommand(exportCmd)
	lotteryCmd.AddCommand(importCmd)
	lotteryCmd.AddCommand(statsCmd)
	lotteryCmd.AddCommand(resetCmd)
	lotteryCmd.AddCommand(dbInfoCmd)

	createCmd.Flags().StringP("participants", "p", "", i18n.GetText("lottery.participants"))
	createCmd.Flags().StringP("seed", "s", "", i18n.GetText("lottery.seed"))
	// The registered default (3) is only for help text; the run-time default is
	// resolved in RunE so a configured lottery.defaultCount is honored after
	// initConfig loads (a viper read at init() time is always 0 — TASK-108).
	createCmd.Flags().IntP("count", "c", 3, i18n.GetText("lottery.count"))
	historyCmd.Flags().IntP("limit", "l", 0, i18n.GetText("token.list_limit"))
	historyCmd.Flags().IntP("offset", "p", 0, i18n.GetText("token.offset"))

	resetCmd.Flags().BoolP("yes", "y", false, i18n.GetText("lottery.yes"))
	// --confirm without a shorthand: the -y shorthand is already taken by
	// --yes, but the canonical destructive gate must also work here so all six
	// destructive commands accept --confirm (TASK-152, ISS-140). Its help
	// renders the shared localized template with the reset action phrase
	// (TASK-186, ISS-182).
	resetCmd.Flags().Bool("confirm", false, fmt.Sprintf(i18n.GetText("cli.confirm.flag"), i18n.GetText("cli.confirm.lottery_reset")))

	_ = createCmd.MarkFlagRequired("participants")
	_ = createCmd.MarkFlagRequired("seed")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
