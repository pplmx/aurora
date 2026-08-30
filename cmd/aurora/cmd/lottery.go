package cmd

import (
	"encoding/hex"
	"encoding/json"
	"errors"
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

		records, err := repo.GetAll()
		if err != nil {
			return fmt.Errorf("failed to read history: %w", err)
		}

		if len(records) == 0 {
			fmt.Println(i18n.GetText("lottery.no_records"))
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

		// Open the persistent lottery repository. Lookups go through the
		// dedicated lottery_records table (structured draw data that survives
		// restarts) rather than the blockchain's raw blocks.
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

		// Perform deterministic verification. Draws created before v1.31 have
		// no persisted public key, so we fall back to re-running SelectWinners
		// on the stored VRF output and checking the recorded winners match
		// what the deterministic selection function would produce. Draws
		// created from v1.31 onward also carry a VRFPublicKey; when present we
		// additionally verify the proof against that key and surface the
		// decoded key in the output for independent cross-checking.
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
				if record.VRFPublicKey != "" {
					pk, dierr := domainlottery.DecodePublicKey(record.VRFPublicKey)
					if dierr != nil {
						fmt.Println("\n❌ Verification FAILED: stored VRF public key is malformed")
						integrityOK = false
					} else if ok, _ := domainlottery.NewService().VerifyDraw(record, pk); !ok {
						fmt.Println("\n❌ Verification FAILED: VRF proof does not verify against the stored public key")
						integrityOK = false
					} else {
						fmt.Println("\n✅ " + i18n.GetText("lottery.verified"))
					}
				} else {
					fmt.Println("\n✅ " + i18n.GetText("lottery.verified") + " (deterministic winner match; pre-v1.31 draw has no stored key)")
				}
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
		if record.VRFPublicKey != "" {
			fmt.Printf("🔑 VRF Public Key: %s...\n", record.VRFPublicKey[:min(16, len(record.VRFPublicKey))])
		}
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

		// Persist via the SQLite repository so the structured draw record
		// (ID, winners, VRF proof, block height) survives restarts; the
		// blockchain is a separate append-only audit log.
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

			// Audit-trail integrity (TASK-069): refuse to overwrite a draw
			// that already exists locally. The record ID is deterministic
			// (sha256 of seed + VRF output), and Save uses INSERT OR REPLACE
			// keyed on that ID — without this guard, re-importing an export
			// file, or importing a file whose draw collides with an existing
			// record (including the Verified flag and history), would silently
			// replace the stored draw. Skip colliding IDs rather than clobber.
			if existing, gErr := repo.GetByID(record.ID); gErr == nil && existing != nil {
				failed = append(failed, i)
				continue
			} else if gErr != nil && !errors.Is(gErr, domainlottery.ErrNotFound) {
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

		fmt.Println("\n📊 Lottery Statistics")
		fmt.Println("────────────────────────────")
		fmt.Printf("  Total lotteries: %d\n", len(records))
		fmt.Printf("  Database: %s\n", blockchain.DBPath())

		if len(records) > 0 {
			// Report the highest recorded block height across persisted draws.
			var latest int64 = -1
			for _, r := range records {
				if r.BlockHeight > latest {
					latest = r.BlockHeight
				}
			}
			if latest >= 0 {
				fmt.Printf("  Latest block: #%d\n", latest)
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

		// Reset the in-memory chain singleton back to genesis. The chain is
		// created once (InitBlockChain's sync.Once) and is not rebuilt by the
		// DELETEs above, so without this a subsequent `lottery create` would
		// keep building on the stale pre-reset blocks: the new draw would get
		// a block height continuing from the old count and a PrevHash pointing
		// at a block the reset deleted (TASK-071, ISS-063).
		blockchain.GetBlockChain().ResetBlocks()

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
		defer func() { _ = db.Close() }()

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

		fmt.Println("\n📁 Database Info")
		fmt.Println("────────────────────────────")
		fmt.Printf("  Path: %s\n", blockchain.DBPath())
		fmt.Printf("  Total blocks: %d\n", count)
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
	lotteryCmd.AddCommand(tuiCmd)
	lotteryCmd.AddCommand(statsCmd)
	lotteryCmd.AddCommand(resetCmd)
	lotteryCmd.AddCommand(dbInfoCmd)

	createCmd.Flags().StringP("participants", "p", "", i18n.GetText("lottery.participants"))
	createCmd.Flags().StringP("seed", "s", "", i18n.GetText("lottery.seed"))
	// The registered default (3) is only for help text; the run-time default is
	// resolved in RunE so a configured lottery.defaultCount is honored after
	// initConfig loads (a viper read at init() time is always 0 — TASK-108).
	createCmd.Flags().IntP("count", "c", 3, i18n.GetText("lottery.count"))

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
