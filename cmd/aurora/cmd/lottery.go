package cmd

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	lotteryapp "github.com/pplmx/aurora/internal/app/lottery"
	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	domainlottery "github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/pplmx/aurora/internal/logger"
	uilottery "github.com/pplmx/aurora/internal/ui/lottery"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var lotteryCmd = &cobra.Command{
	Use:   "lottery",
	Short: i18n.GetText("lottery.tui.title"),
	Long:  i18n.GetText("lottery.create"),
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: i18n.GetText("lottery.create"),
	Example: `  aurora lottery create -p "Alice,Bob,Charlie" -s "random-seed-123"
  aurora lottery create -p "A,B,C,D,E" -s "my-lottery" -c 3`,
	RunE: func(cmd *cobra.Command, args []string) error {
		participantsStr, _ := cmd.Flags().GetString("participants")
		seed, _ := cmd.Flags().GetString("seed")
		count, _ := cmd.Flags().GetInt("count")

		lotteryRepo, err := sqlite.NewLotteryRepository(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to create lottery repository: %w", err)
		}
		defer func() { _ = lotteryRepo.Close() }()

		blockChain := blockchain.InitBlockChain()

		uc := lotteryapp.NewCreateLotteryUseCase(lotteryRepo, blockChain)

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
		fmt.Printf("📋 Lottery ID: %s\n", resp.ID)
		fmt.Printf("🔢 Block height: #%d\n", resp.BlockHeight)
		fmt.Println("\n🎉 Winners:")
		for i, w := range resp.Winners {
			// Guard against mismatched slice lengths (could happen with
			// imported data, older DB schemas, or partial writes).
			// Without this check the CLI panics with index-out-of-range.
			addr := "(no address)"
			if i < len(resp.WinnerAddresses) {
				addr = resp.WinnerAddresses[i]
			}
			fmt.Printf("   %d. %s (%s)\n", i+1, w, addr)
		}
		fmt.Printf("\n🔐 VRF Output: %s...\n", resp.VRFOutput[:min(16, len(resp.VRFOutput))])
		fmt.Printf("📜 VRF Proof: %s...\n", resp.VRFProof[:min(16, len(resp.VRFProof))])

		return nil
	},
}

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: i18n.GetText("lottery.tui"),
	RunE: func(cmd *cobra.Command, args []string) error {
		return uilottery.RunLotteryTUI()
	},
}

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: i18n.GetText("lottery.history"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read from the persistent lottery_records table, not the in-memory
		// chain. The chain's in-memory append is intentionally not flushed
		// back to the blocks table (per AddBlock's current contract), so
		// reading GetLotteryRecords() returns empty across process
		// boundaries — which the lottery_records table was specifically
		// designed to prevent.
		repo, err := sqlite.NewLotteryRepository(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to open lottery repository: %w", err)
		}
		defer func() { _ = repo.Close() }()

		records, err := repo.GetAll()
		if err != nil {
			return fmt.Errorf("failed to read history: %w", err)
		}

		if len(records) == 0 {
			fmt.Println("No lottery records found.")
			return nil
		}

		fmt.Printf("\n📜 Total lotteries: %d\n\n", len(records))
		for i, record := range records {
			jsonData, err := record.ToJSON()
			if err != nil {
				continue
			}
			fmt.Printf("--- Lottery #%d ---\n", i+1)
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

		// Open the persistent lottery repository. We can't rely on
		// the in-memory chain for lookup because AddBlock doesn't flush
		// back to the blocks table, so reads across processes (or even
		// separate command invocations) would return empty.
		repo, err := sqlite.NewLotteryRepository(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to open lottery repository: %w", err)
		}
		defer func() { _ = repo.Close() }()

		var record *domainlottery.LotteryRecord

		// Disambiguation between "lottery-id" and "block-height".
		//
		// A lottery ID is a 16-hex-char hash prefix (see CreateLotteryRecord),
		// which may begin with a decimal digit (e.g. "07551e0cac54a4cf"), and
		// may even consist entirely of digits. Trying to parse the height first
		// via fmt.Sscanf(input, "%d", &height) is wrong on two counts:
		//  1. Sscanf accepts a PARTIAL numeric prefix, so an ID that starts
		//     with a digit is swallowed as a (bogus) height.
		//  2. A numeric-prefix ID (or an all-digit ID) would never reach the
		//     ID lookup at all, so "verify <valid-id>" would report
		//     "lottery not found" for an existing draw.
		//
		// So we check the exact ID match first (the common case: users paste
		// an ID from create/history), then treat the input as a height only
		// if the ENTIRE input is a decimal integer, and only then fall back to
		// a substring ID/seed match for convenience.
		record, err = repo.GetByID(input)
		if err != nil {
			// Not an exact ID match. If the whole input is a plain decimal
			// integer, it is a block height.
			if height, perr := strconv.ParseInt(input, 10, 64); perr == nil {
				records, rerr := repo.GetByBlockHeight(height)
				if rerr != nil {
					return fmt.Errorf("failed to read by block height: %w", rerr)
				}
				if len(records) == 0 {
					return fmt.Errorf("lottery not found: %s", input)
				}
				record = records[0]
			} else {
				// Fall back to a substring match so partial IDs work the
				// way they did before this command was rewritten to read
				// from the persistent store.
				all, getAllErr := repo.GetAll()
				if getAllErr != nil {
					return fmt.Errorf("failed to read history: %w", getAllErr)
				}
				for _, r := range all {
					if strings.Contains(r.ID, input) || strings.Contains(r.Seed, input) {
						record = r
						err = nil
						break
					}
				}
				if err != nil {
					return fmt.Errorf("lottery not found: %s", input)
				}
			}
		}

		// Display verification info
		fmt.Println("\n📋 " + i18n.GetText("lottery.lottery_id") + ": " + record.ID)
		fmt.Printf("🔢 Block Height: #%d\n", record.BlockHeight)
		fmt.Printf("🌱 Seed: %s\n", record.Seed)
		fmt.Printf("👥 Participants: %d\n", len(record.Participants))
		fmt.Printf("🎉 Winners: %d\n", len(record.Winners))

		// Perform deterministic verification. We can't re-verify the VRF
		// proof without the public key (which we deliberately do not store
		// per draw), but we CAN re-run SelectWinners on the stored VRF
		// output and check that the recorded winners match what the
		// deterministic selection function would produce.
		//
		// Note on the "✅ Verified" UX: previously this command printed
		// "✅ Lottery Record Verified!" after a plain JSON parse — that
		// was a false positive. We now actually check the record's
		// integrity and report honest results.
		integrityOK := true
		vrfOutputBytes, err := hex.DecodeString(record.VRFOutput)
		if err != nil {
			fmt.Println("\n❌ Verification FAILED: VRF output is not valid hex")
			integrityOK = false
		} else if _, err := hex.DecodeString(record.VRFProof); err != nil {
			fmt.Println("\n❌ Verification FAILED: VRF proof is not valid hex")
			integrityOK = false
		} else {
			expected := domainlottery.SelectWinners(vrfOutputBytes, record.Participants, len(record.Winners))
			if !sameStringSet(expected, record.Winners) {
				fmt.Println("\n❌ Verification FAILED: stored winners do not match the VRF output")
				fmt.Println("   Expected:", expected)
				fmt.Println("   Stored:  ", record.Winners)
				integrityOK = false
			} else if len(record.Winners) != len(record.WinnerAddresses) {
				fmt.Println("\n⚠️  Stored record has mismatched winner/address slices (possible data corruption)")
				integrityOK = false
			} else {
				fmt.Println("\n✅ " + i18n.GetText("lottery.verified"))
			}
		}

		fmt.Println("\n🏆 Winners:")
		for i, w := range record.Winners {
			// Same guard as in createCmd: defend against mismatched slice
			// lengths in imported/corrupted records.
			addr := "(no address)"
			if i < len(record.WinnerAddresses) {
				addr = record.WinnerAddresses[i]
			}
			fmt.Printf("   %d. %s (%s)\n", i+1, w, addr)
		}
		fmt.Printf("\n🔐 VRF Output: %s...\n", record.VRFOutput[:min(16, len(record.VRFOutput))])
		fmt.Printf("📜 VRF Proof: %s...\n", record.VRFProof[:min(16, len(record.VRFProof))])
		fmt.Printf("⏰ Timestamp: %d\n", record.Timestamp)

		if !integrityOK {
			// Surface the failure to the caller (e.g. shell scripts that
			// check $?) without also printing the success summary.
			return fmt.Errorf("lottery record failed integrity check")
		}
		return nil
	},
}

// sameStringSet returns true if a and b contain the same elements,
// regardless of order. SelectWinners returns winners in the order they
// were drawn from the VRF stream, but the stored record could be reordered
// in transit; we want to compare sets, not ordered lists.
func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	count := make(map[string]int, len(a))
	for _, s := range a {
		count[s]++
	}
	for _, s := range b {
		count[s]--
		if count[s] < 0 {
			return false
		}
	}
	return true
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

		records, err := repo.GetAll()
		if err != nil {
			return fmt.Errorf("failed to read records: %w", err)
		}

		output, err := json.MarshalIndent(records, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal: %w", err)
		}

		if err := os.WriteFile(filename, output, 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}

		fmt.Printf("✅ Exported %d lottery records to %s\n", len(records), filename)
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

		var records []domainlottery.LotteryRecord
		if err := json.Unmarshal(data, &records); err != nil {
			return fmt.Errorf("failed to parse file: %w", err)
		}

		// Persist via the SQLite repository, not the in-memory chain —
		// AddBlock doesn't flush back to the blocks table, so writing
		// through the chain would lose the records on the next process
		// start.
		repo, err := sqlite.NewLotteryRepository(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to open lottery repository: %w", err)
		}
		defer func() { _ = repo.Close() }()

		imported := 0
		var failed []int // indices of records that failed to import

		for i, record := range records {
			// Validate the record before accepting it. An imported file is
			// untrusted input — we must not let bad data corrupt the chain.
			if err := record.Validate(); err != nil {
				failed = append(failed, i)
				continue
			}
			if err := repo.Save(&record); err != nil {
				failed = append(failed, i)
				continue
			}
			imported++
		}

		if len(failed) == 0 {
			fmt.Printf("✅ Imported %d lottery records\n", imported)
			return nil
		}
		// Partial failure: surface the problem honestly. Returning a
		// non-nil error also lets CI / shell scripts detect the partial
		// failure via $?, instead of silently treating it as success.
		fmt.Printf("⚠️  Imported %d of %d lottery records (failed: %d)\n",
			imported, len(records), len(failed))
		fmt.Printf("   Failed record indices (0-based): %v\n", failed)
		return fmt.Errorf("import partially failed: %d of %d records rejected", len(failed), len(records))
	},
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: i18n.GetText("lottery.stats"),
	RunE: func(cmd *cobra.Command, args []string) error {
		chain := blockchain.InitBlockChain()
		records := chain.GetLotteryRecords()

		fmt.Println("\n📊 Lottery Statistics")
		fmt.Println("────────────────────────────")
		fmt.Printf("  Total lotteries: %d\n", len(records))
		fmt.Printf("  Database: %s\n", blockchain.DBPath())

		if len(records) > 0 {
			fmt.Printf("  Latest block: #%d\n", len(chain.Blocks)-1)
		}

		return nil
	},
}

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: i18n.GetText("lottery.reset"),
	RunE: func(cmd *cobra.Command, args []string) error {
		confirm, _ := cmd.Flags().GetBool("yes")
		if !confirm {
			fmt.Println("⚠️  This will delete ALL lottery records!")
			fmt.Println("   Use --yes to confirm")
			return nil
		}

		db, err := blockchain.InitDB()
		if err != nil {
			return fmt.Errorf("failed to init db: %w", err)
		}
		defer func() { _ = db.Close() }()

		// A lottery lives in two stores: the chain (`blocks` table) and the
		// persistent history (`lottery_records` table, which historyCmd,
		// verifyCmd, exportCmd and importCmd read). Deleting only `blocks`
		// left `lottery_records` intact, so after a reset, `lottery history`
		// still listed every draw — contradicting the "delete ALL lottery
		// records!" confirmation. Clear both.
		//
		// On a brand-new DB neither table exists yet (created lazily by
		// InitBlockChain / the repository / migrations), so a DELETE against
		// a missing table is "no such table" — tolerated, not an error.
		for _, stmt := range []string{
			"DELETE FROM blocks WHERE height > 0",
			"DELETE FROM lottery_records",
		} {
			if _, err := db.Exec(stmt); err != nil && !isNoSuchTable(err) {
				return fmt.Errorf("failed to reset: %w", err)
			}
		}

		logger.Info().Msg("Database reset successfully")
		fmt.Println("✅ Database reset complete!")
		return nil
	},
}

// isNoSuchTable reports whether err is SQLite's "no such table" error.
// lottery reset tolerates it because blocks and lottery_records are created
// lazily and may not exist on a brand-new DB.
func isNoSuchTable(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no such table")
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Aurora - VRF Lottery System")
		fmt.Println("Version: 0.0.1")
		fmt.Println("Go Version:", getGoVersion())
		return nil
	},
}

var dbInfoCmd = &cobra.Command{
	Use:   "db-info",
	Short: "Show database information",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := blockchain.InitDB()
		if err != nil {
			return fmt.Errorf("failed to init db: %w", err)
		}
		defer func() { _ = db.Close() }()

		var count int
		_ = db.QueryRow("SELECT COUNT(*) FROM blocks WHERE height > 0").Scan(&count)

		fmt.Println("\n📁 Database Info")
		fmt.Println("────────────────────────────")
		fmt.Printf("  Path: %s\n", blockchain.DBPath())
		fmt.Printf("  Total blocks: %d\n", count)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lotteryCmd)
	rootCmd.AddCommand(versionCmd)
	lotteryCmd.AddCommand(createCmd)
	lotteryCmd.AddCommand(historyCmd)
	lotteryCmd.AddCommand(verifyCmd)
	lotteryCmd.AddCommand(exportCmd)
	lotteryCmd.AddCommand(importCmd)
	lotteryCmd.AddCommand(tuiCmd)
	lotteryCmd.AddCommand(statsCmd)
	lotteryCmd.AddCommand(resetCmd)
	lotteryCmd.AddCommand(dbInfoCmd)

	createCmd.Flags().StringP("participants", "p", "", i18n.GetText("lottery.participants"))
	createCmd.Flags().StringP("seed", "s", "", i18n.GetText("lottery.seed"))
	createCmd.Flags().IntP("count", "c", viper.GetInt("lottery.defaultCount"), i18n.GetText("lottery.count"))

	resetCmd.Flags().BoolP("yes", "y", false, i18n.GetText("lottery.yes"))

	_ = createCmd.MarkFlagRequired("participants")
	_ = createCmd.MarkFlagRequired("seed")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func getGoVersion() string {
	return "1.26+"
}
