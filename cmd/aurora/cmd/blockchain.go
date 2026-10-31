package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	"github.com/pplmx/aurora/internal/ledger"
	"github.com/pplmx/aurora/internal/records"
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
		if jsonRequested(cmd) {
			// Print the report first, then let the exit code carry the same
			// verdict the table path reports (a script checks $? before fields).
			if err := printJSON(report); err != nil {
				return err
			}
			if !report.Valid {
				return newCLIError("blockchain.invalid")
			}
			return nil
		}

		fmt.Printf("\n⛓  %s\n", i18n.GetText("blockchain.verify_title"))
		fmt.Println("────────────────────────────")
		fmt.Printf("  %s: %d\n", i18n.GetText("blockchain.length"), report.Length)
		if report.Head != "" {
			fmt.Printf("  %s: %s\n", i18n.GetText("blockchain.head"), shortHash(report.Head))
		}
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
		search, err := cmd.Flags().GetString("search")
		if err != nil {
			return err
		}
		chain, err := blockchain.InitWithError()
		if err != nil {
			return wrapCLIError("blockchain.error.init", err)
		}

		search = strings.TrimSpace(search)
		var entries []ledger.BlockEntry
		total := 0
		if search == "" {
			entries = chain.BlockEntriesPage(limit, offset)
			if total = chain.Len() - 1; total < 0 {
				total = 0
			}
		} else {
			needle := strings.ToLower(search)
			entries, total = chain.BlockEntriesPageMatching(limit, offset, func(entry ledger.BlockEntry) bool {
				return blockMatches(entry, needle)
			})
		}
		if jsonRequested(cmd) {
			return printBlocksJSON(entries, total, limit, offset)
		}

		fmt.Printf("\n🧱 %s: %d\n", i18n.GetText("blockchain.blocks_title"), total)
		if len(entries) == 0 {
			// A filtered miss and an offset past the end are different problems
			// from an empty ledger; naming the first write would mislead.
			switch {
			case total > 0:
				fmt.Printf("   "+i18n.GetText("blockchain.blocks_empty_page")+"\n", total)
			case search != "":
				fmt.Printf("   "+i18n.GetText("blockchain.blocks_no_match")+"\n", search)
			default:
				fmt.Println("   " + i18n.GetText("blockchain.blocks_empty"))
			}
			return nil
		}
		fmt.Println("────────────────────────────")
		for _, entry := range entries {
			stamp := time.Unix(entry.Timestamp, 0).Format(blockTimeLayout)
			fmt.Printf("  #%-4d %s  %s\n", entry.Height, stamp, blockLine(entry.Data))
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
	blockchainBlocksCmd.Flags().StringP("search", "s", "", i18n.GetText("blockchain.blocks_search"))
	addJSONFlag(blockchainVerifyCmd)
	addJSONFlag(blockchainBlocksCmd)
}

// blockMatches answers the audit filter with the same three things the window's
// ledger search compares: the stored text, the decoded sentence and the height.
// Searching "transfer" therefore finds a token transfer whose payload spells the
// verb, and "Winners" finds draws even though no payload contains that word.
func blockMatches(entry ledger.BlockEntry, needle string) bool {
	if strings.Contains(strings.ToLower(entry.Data), needle) {
		return true
	}
	if strings.Contains(strings.ToLower(blockLine(entry.Data)), needle) {
		return true
	}
	return strings.Contains(strconv.FormatInt(entry.Height, 10), needle)
}

// blockLine renders one audit payload for the table with the vocabulary the
// window's ledger page uses (module · action · subject, then its facts) instead
// of the stored JSON or pipe line. A payload that matches no known shape prints
// as-is, and `--json` always carries the exact bytes: the decoded sentence is a
// reading aid, never the only copy.
func blockLine(payload string) string {
	record, ok := records.Parse(payload)
	if !ok {
		return payload
	}
	return record.Text()
}

// printBlocksJSON writes the audit page in the same shape the REST endpoint
// uses, so a script parses one document whether it called the CLI or the API.
func printBlocksJSON(entries []ledger.BlockEntry, total, limit, offset int) error {
	type blockEntry struct {
		Height    int64  `json:"height"`
		Timestamp int64  `json:"timestamp"`
		Payload   string `json:"payload"`
	}
	page := struct {
		Blocks []blockEntry `json:"blocks"`
		Total  int          `json:"total"`
		Limit  int          `json:"limit"`
		Offset int          `json:"offset"`
	}{Blocks: make([]blockEntry, 0, len(entries)), Total: total, Limit: limit, Offset: offset}
	for _, entry := range entries {
		page.Blocks = append(page.Blocks, blockEntry{
			Height: entry.Height, Timestamp: entry.Timestamp, Payload: entry.Data,
		})
	}
	return printJSON(page)
}
