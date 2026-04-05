package cmd

import (
	"fmt"
	"strings"

	"github.com/pplmx/aurora/internal/blockchain"
	"github.com/pplmx/aurora/internal/lottery"
	"github.com/spf13/cobra"
)

var lotteryCmd = &cobra.Command{
	Use:   "lottery",
	Short: "VRF-based transparent lottery system",
	Long:  "A verifiable random function based lottery system with blockchain storage",
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new lottery",
	RunE: func(cmd *cobra.Command, args []string) error {
		participantsStr, _ := cmd.Flags().GetString("participants")
		seed, _ := cmd.Flags().GetString("seed")
		count, _ := cmd.Flags().GetInt("count")

		participants := strings.Split(participantsStr, ",")
		for i := range participants {
			participants[i] = strings.TrimSpace(participants[i])
		}
		participants = removeEmpty(participants)

		if len(participants) < count {
			return fmt.Errorf("not enough participants: need at least %d, got %d", count, len(participants))
		}

		if seed == "" {
			return fmt.Errorf("seed cannot be empty")
		}

		pk, sk, err := lottery.GenerateKeyPair()
		if err != nil {
			return fmt.Errorf("failed to generate key pair: %w", err)
		}

		output, proof, err := lottery.VRFProve(sk, []byte(seed))
		if err != nil {
			return fmt.Errorf("failed to compute VRF: %w", err)
		}

		winners := lottery.SelectWinners(output, participants, count)

		winnerAddrs := make([]string, len(winners))
		for i, w := range winners {
			winnerAddrs[i] = lottery.NameToAddress(w)
		}

		record := lottery.CreateLotteryRecord(seed, participants, winners, winnerAddrs, output, proof, 0)

		jsonData, err := record.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to serialize record: %w", err)
		}

		chain := blockchain.InitBlockChain()
		height, err := chain.AddLotteryRecord(jsonData)
		if err != nil {
			return fmt.Errorf("failed to add to blockchain: %w", err)
		}
		record.BlockHeight = height

		fmt.Println("\n✅ Lottery created successfully!")
		fmt.Printf("📋 Lottery ID: %s\n", record.ID)
		fmt.Printf("🔢 Block height: #%d\n", height)
		fmt.Println("\n🎉 Winners:")
		for i, w := range record.Winners {
			fmt.Printf("   %d. %s (%s)\n", i+1, w, record.WinnerAddrs[i])
		}
		fmt.Printf("\n🔐 VRF Output: %s...\n", record.VRFOutput[:min(16, len(record.VRFOutput))])
		fmt.Printf("📜 VRF Proof: %s...\n", record.VRFProof[:min(16, len(record.VRFProof))])

		_ = pk
		return nil
	},
}

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch TUI interface",
	RunE: func(cmd *cobra.Command, args []string) error {
		return lottery.RunLotteryTUI()
	},
}

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show lottery history",
	RunE: func(cmd *cobra.Command, args []string) error {
		chain := blockchain.InitBlockChain()
		records := chain.GetLotteryRecords()

		if len(records) == 0 {
			fmt.Println("No lottery records found.")
			return nil
		}

		fmt.Printf("\n📜 Total lotteries: %d\n\n", len(records))
		for i, data := range records {
			fmt.Printf("--- Lottery #%d ---\n", i+1)
			fmt.Println(data[:min(200, len(data))])
			fmt.Println()
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lotteryCmd)
	lotteryCmd.AddCommand(createCmd)
	lotteryCmd.AddCommand(historyCmd)
	lotteryCmd.AddCommand(tuiCmd)

	createCmd.Flags().StringP("participants", "p", "", "Participant names (comma-separated)")
	createCmd.Flags().StringP("seed", "s", "", "Random seed")
	createCmd.Flags().IntP("count", "c", 3, "Number of winners")

	createCmd.MarkFlagRequired("participants")
	createCmd.MarkFlagRequired("seed")
}

func removeEmpty(s []string) []string {
	result := make([]string, 0, len(s))
	for _, str := range s {
		if str != "" {
			result = append(result, str)
		}
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
