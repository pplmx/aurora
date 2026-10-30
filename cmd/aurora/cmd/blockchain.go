package cmd

import (
	"fmt"

	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	"github.com/spf13/cobra"
)

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

func init() {
	rootCmd.AddCommand(blockchainCmd)
	blockchainCmd.AddCommand(blockchainVerifyCmd)
}
