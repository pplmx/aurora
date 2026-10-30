package cmd

import (
	"fmt"
	"time"

	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	"github.com/spf13/cobra"
)

// blockTimeLayout is the CLI's audit timestamp format: local time, minute
// precision, no zone suffix (an auditor comparing two machines reads the local
// clock, and the exact epoch value stays in the payload record).
const blockTimeLayout = "2006-01-02 15:04:05"

var blockchainCmd = &cobra.Command{
	Use:   "blockchain",
	Short: i18n.GetText("blockchain.cmd"),
	Long:  i18n.GetText("blockchain.long"),
}

// verifyCmd reports the local chain's integrity. The API exposes the same
// verdict on /blockchain/verify and the desktop draws it on the Blockchain
// page; an operator on a machine without either checks it here.
var blockchainVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: i18n.GetText("blockchain.verify"),
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		chain, err := blockchain.InitWithError()
		if err != nil {
			return wrapCLIError("blockchain.error.init", err)
		}
		report := chain.VerifyIntegrity()

		fmt.Printf("\n⛓  %s\n", i18n.GetText("blockchain.verify_title"))
		fmt.Println("────────────────────────────")
		fmt.Printf("  %s: %d\n", i18n.GetText("blockchain.length"), report.Length)
		if report.Valid {
			fmt.Printf("  %s: %s\n", i18n.GetText("blockchain.state"), i18n.GetText("blockchain.valid"))
			return nil
		}
		reason := report.FirstBrokenReason
		if reason == "" {
			reason = i18n.GetText("blockchain.no_reason")
		}
		fmt.Printf("  %s: %s\n", i18n.GetText("blockchain.state"),
			i18n.GetTextF("blockchain.broken_at", report.FirstBrokenIndex, reason))
		// A verification command must fail the process: a script that checks the
		// exit code cannot read the printed verdict.
		return newCLIError("blockchain.invalid")
	},
}

// blocksCmd prints the recorded payloads newest first. The payloads are the
// exact text each module appended (base64 keys for the signature-bearing
// writes), so the output can be piped to grep/jq without escaping binary.
var blockchainBlocksCmd = &cobra.Command{
	Use:   "blocks",
	Short: i18n.GetText("blockchain.blocks"),
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, offset, err := pagingFlags(cmd)
		if err != nil {
			return err
		}
		chain, err := blockchain.InitWithError()
		if err != nil {
			return wrapCLIError("blockchain.error.init", err)
		}

		entries := chain.BlockEntriesPage(limit, offset)
		total := chain.Len() - 1
		if total < 0 {
			total = 0
		}
		fmt.Printf("\n🧱 %s: %d\n", i18n.GetText("blockchain.blocks_title"), total)
		if len(entries) == 0 {
			fmt.Println("   " + i18n.GetText("blockchain.blocks_empty"))
			return nil
		}
		fmt.Println("────────────────────────────")
		for _, entry := range entries {
			stamp := time.Unix(entry.Timestamp, 0).Format(blockTimeLayout)
			fmt.Printf("  #%-4d %s  %s\n", entry.Height, stamp, entry.Data)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(blockchainCmd)
	blockchainCmd.AddCommand(blockchainVerifyCmd)
	blockchainCmd.AddCommand(blockchainBlocksCmd)
	blockchainBlocksCmd.Flags().IntP("limit", "l", 20, i18n.GetText("blockchain.blocks_limit"))
	blockchainBlocksCmd.Flags().IntP("offset", "p", 0, i18n.GetText("blockchain.blocks_offset"))
}
