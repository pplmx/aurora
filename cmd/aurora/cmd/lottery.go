// The lottery commands that make and check a draw: create, history, verify.
// The record-level operations that move or delete the store live beside it —
// export/import in lottery_portability.go and stats/reset/db-info in
// lottery_admin.go — because a reader looking for "how do I re-check this
// draw" should not have to scroll past the backup and accounting commands.
package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/pplmx/aurora/internal/app/lotteryapp"
	"github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	"github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/pplmx/aurora/internal/infra/store"
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

		lotteryRepo, err := sqlite.NewLotteryRepository(store.Path())
		if err != nil {
			return wrapCLIError("lottery.error.create_repo", err)
		}
		defer func() { _ = lotteryRepo.Close() }()

		chain, err := blockchain.InitWithError()
		if err != nil {
			return wrapCLIError("lottery.error.init_blockchain", err)
		}

		uc := lotteryapp.NewCreate(lotteryRepo, chain)

		req := lotteryapp.CreateLotteryRequest{
			Participants: participantsStr,
			Seed:         seed,
			WinnerCount:  count,
		}

		resp, err := uc.Execute(req)
		if err != nil {
			return wrapCLIError("lottery.error.create", err)
		}
		if jsonRequested(cmd) {
			return printJSON(resp)
		}

		fmt.Println("\n✅ " + i18n.GetText("lottery.success"))
		fmt.Printf("📋 %s\n", i18n.Label(i18n.GetText("lottery.lottery_id"), resp.ID))
		fmt.Printf("🔢 %s\n", i18n.Label(i18n.GetText("lottery.block_height"), fmt.Sprintf("#%d", resp.BlockHeight)))
		fmt.Println("\n🎉 " + i18n.GetText("lottery.winners") + i18n.Colon())
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
		fmt.Printf("\n🔐 %s\n", i18n.Label(i18n.GetText("lottery.vrf_output"), fmt.Sprintf("%s...", resp.VRFOutput[:min(16, len(resp.VRFOutput))])))
		fmt.Printf("📜 %s\n", i18n.Label(i18n.GetText("lottery.vrf_proof"), fmt.Sprintf("%s...", resp.VRFProof[:min(16, len(resp.VRFProof))])))

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
		repo, err := sqlite.NewLotteryRepository(store.Path())
		if err != nil {
			return wrapCLIError("lottery.error.open_repo", err)
		}
		defer func() { _ = repo.Close() }()

		limit, offset, err := pagingFlags(cmd)
		if err != nil {
			return err
		}
		search, _ := cmd.Flags().GetString("search")
		search = strings.TrimSpace(search)

		// The filter runs in storage: a page-local search would only find the
		// draws that happened to land on the requested page.
		var records []*lottery.LotteryRecord
		if search != "" {
			records, err = repo.ListMatching(search, limit, offset)
		} else {
			records, err = repo.List(limit, offset)
		}
		if err != nil {
			return wrapCLIError("lottery.error.read_history", err)
		}
		if jsonRequested(cmd) {
			if records == nil {
				records = []*lottery.LotteryRecord{}
			}
			return printJSON(records)
		}

		if len(records) == 0 {
			if limit > 0 || offset > 0 {
				fmt.Printf("\n📜 %s\n", i18n.Label(i18n.GetText("lottery.page_total"), i18n.GetTextF("common.page_with_offset", 0, offset)))
				return nil
			}
			if search != "" {
				fmt.Printf("   "+i18n.GetText("lottery.history_no_match")+"\n", search)
				return nil
			}
			fmt.Println(i18n.GetText("lottery.no_records"))
			return nil
		}

		if limit > 0 {
			fmt.Printf("\n📜 %s\n", i18n.Label(i18n.GetText("lottery.page_total"), fmt.Sprintf("%s\n", i18n.GetTextF("common.page_with_offset", len(records), offset))))
		} else if search != "" {
			// "Total lotteries" beside a filter would read as the store's
			// count; name what the number is.
			fmt.Printf("\n📜 %s\n", i18n.Label(i18n.GetText("lottery.matches"), fmt.Sprintf("%d\n", len(records))))
		} else {
			fmt.Printf("\n📜 %s\n", i18n.Label(i18n.GetText("lottery.total"), fmt.Sprintf("%d\n", len(records))))
		}
		for i, record := range records {
			// The table is the reading view: one labelled line per auditable
			// field instead of a JSON blob cut at 200 characters. `--json`
			// carries the whole record for a script.
			fmt.Printf("%s\n", i18n.GetTextF("lottery.history_item", i+1))
			printFields(
				field{i18n.GetText("common.id"), record.ID},
				field{i18n.GetText("lottery.seed_label"), record.Seed},
				field{i18n.GetText("lottery.winners"), strings.Join(record.Winners, ", ")},
				field{i18n.GetText("lottery.participants_label"), fmt.Sprint(len(record.Participants))},
				field{i18n.GetText("common.block_height"), fmt.Sprint(record.BlockHeight)},
				field{i18n.GetText("common.timestamp"),
					time.Unix(record.Timestamp, 0).Format(blockTimeLayout)},
			)
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
		repo, err := sqlite.NewLotteryRepository(store.Path())
		if err != nil {
			return wrapCLIError("lottery.error.open_repo", err)
		}
		defer func() { _ = repo.Close() }()

		// Resolution (exact ID → whole block height → substring) and the
		// integrity checks live in the app layer, shared with the desktop
		// shell so both report identical verdicts for the same record.
		report, err := lotteryapp.VerifyInput(repo, input)
		if err != nil {
			return err
		}
		if jsonRequested(cmd) {
			// The document carries the verdict, but the exit code is what a
			// script checks first; a failed check must not read as success.
			if err := printJSON(report); err != nil {
				return err
			}
			if !report.IntegrityOK {
				return lotteryapp.ErrVerificationFailed
			}
			return nil
		}
		record := report.Record

		// Display verification info
		fmt.Println("\n📋 " + i18n.Label(i18n.GetText("lottery.lottery_id"), record.ID))
		fmt.Printf("🔢 %s\n", i18n.Label(i18n.GetText("common.block_height"), fmt.Sprintf("#%d", record.BlockHeight)))
		fmt.Printf("🌱 %s\n", i18n.Label(i18n.GetText("lottery.seed_label"), record.Seed))
		fmt.Printf("👥 %s\n", i18n.Label(i18n.GetText("lottery.participants_label"), len(record.Participants)))
		fmt.Printf("🎉 %s\n", i18n.Label(i18n.GetText("lottery.winners"), len(record.Winners)))

		// Draws created before v1.31 have no persisted public key, so the
		// verdict falls back to a deterministic winner match; later draws also
		// get the cryptographic proof check (see app/lottery.VerifyRecord).
		// The "✅ Verified" line is printed only for a passing record — it must
		// never follow a plain parse, which was a false positive.
		var passed []string
		for _, check := range report.Checks {
			if check.Status == lotteryapp.VerifyFail {
				fmt.Printf("\n❌ %s\n",
					i18n.GetTextF("lottery.verify_failed", i18n.LocalizeVerifyCheckMessage(check.Message)))
				continue
			}
			passed = append(passed, i18n.LocalizeVerifyCheckMessage(check.Message))
		}
		if report.IntegrityOK {
			if report.ProofVerified {
				fmt.Println("\n✅ " + i18n.GetText("lottery.verified"))
			} else {
				fmt.Println("\n✅ " + i18n.GetText("lottery.verified_deterministic"))
			}
			// The verdict says the record passed; the checks say what passed —
			// a proof against the stored key, or the deterministic winner match
			// a pre-v1.31 draw falls back to.
			if len(passed) > 0 {
				fmt.Printf("\n🔍 %s\n", i18n.GetText("lottery.checks")+i18n.Colon())
				for _, message := range passed {
					fmt.Printf("   - %s\n", message)
				}
			}
		}

		fmt.Printf("\n🏆 %s\n", i18n.GetText("lottery.winners")+i18n.Colon())
		for i, w := range record.Winners {
			// Same guard as in createCmd: defend against mismatched slice
			// lengths in imported/corrupted records.
			addr := i18n.GetText("lottery.no_address")
			if i < len(record.WinnerAddresses) {
				addr = record.WinnerAddresses[i]
			}
			fmt.Printf("   %d. %s (%s)\n", i+1, w, addr)
		}
		fmt.Printf("\n🔐 %s\n", i18n.Label(i18n.GetText("lottery.vrf_output"), fmt.Sprintf("%s...", record.VRFOutput[:min(16, len(record.VRFOutput))])))
		fmt.Printf("📜 %s\n", i18n.Label(i18n.GetText("lottery.vrf_proof"), fmt.Sprintf("%s...", record.VRFProof[:min(16, len(record.VRFProof))])))
		if record.VRFPublicKey != "" {
			fmt.Printf("🔑 %s\n", i18n.Label(i18n.GetText("lottery.vrf_public_key"), fmt.Sprintf("%s...", record.VRFPublicKey[:min(16, len(record.VRFPublicKey))])))
		}
		fmt.Printf("⏰ %s\n", i18n.Label(i18n.GetText("common.timestamp"), record.Timestamp))

		if !report.IntegrityOK {
			// Surface the failure to the caller (e.g. shell scripts that
			// check $?) without also printing the success summary.
			return lotteryapp.ErrVerificationFailed
		}
		return nil
	},
}

// isNoSuchTable reports whether err is SQLite's "no such table" error. The
// destructive reset and db-info tolerate it because blocks and
// lottery_records are created lazily and may not exist on a brand-new DB.
func isNoSuchTable(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no such table")
}

func init() {
	rootCmd.AddCommand(lotteryCmd)
	lotteryCmd.AddCommand(createCmd)
	addJSONFlag(createCmd)
	lotteryCmd.AddCommand(historyCmd)
	addJSONFlag(historyCmd)
	lotteryCmd.AddCommand(verifyCmd)
	addJSONFlag(verifyCmd)
	lotteryCmd.AddCommand(exportCmd)
	addJSONFlag(exportCmd)
	lotteryCmd.AddCommand(importCmd)
	addJSONFlag(importCmd)
	lotteryCmd.AddCommand(statsCmd)
	addJSONFlag(statsCmd)
	lotteryCmd.AddCommand(resetCmd)
	lotteryCmd.AddCommand(dbInfoCmd)
	addJSONFlag(dbInfoCmd)

	createCmd.Flags().StringP("participants", "p", "", i18n.GetText("lottery.participants"))
	createCmd.Flags().StringP("seed", "s", "", i18n.GetText("lottery.seed"))
	// The registered default (3) is only for help text; the run-time default is
	// resolved in RunE so a configured lottery.defaultCount is honored after
	// initConfig loads (a viper read at init() time is always 0 — TASK-108).
	// The effective default is lottery.defaultCount, resolved in RunE; the flag
	// keeps the zero value so pflag does not print a "(default 3)" that would
	// contradict a configured value.
	createCmd.Flags().IntP("count", "c", 0, i18n.GetText("lottery.count"))
	historyCmd.Flags().IntP("limit", "l", 0, i18n.GetText("token.list_limit"))
	historyCmd.Flags().IntP("offset", "p", 0, i18n.GetText("token.offset"))
	historyCmd.Flags().StringP("search", "s", "", i18n.GetText("lottery.history_search"))

	// The reset takes the same -y/--confirm gate as every other destructive
	// command, instead of a private spelling with a second alias.
	addConfirmFlag(resetCmd, "cli.confirm.lottery_reset")

	_ = createCmd.MarkFlagRequired("participants")
	_ = createCmd.MarkFlagRequired("seed")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
