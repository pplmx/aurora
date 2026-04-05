package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pplmx/aurora/internal/blockchain"
	"github.com/pplmx/aurora/internal/lottery"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

var verifyCmd = &cobra.Command{
	Use:   "verify [lottery-id or block-height]",
	Short: "Verify a lottery result",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		input := args[0]
		chain := blockchain.InitBlockChain()

		var record *lottery.LotteryRecord

		// Try to parse as block height first
		var height int64
		if _, err := fmt.Sscanf(input, "%d", &height); err == nil {
			// It's a number, treat as block height
			data, err := chain.GetBlockData(height)
			if err != nil {
				return fmt.Errorf("failed to get block: %w", err)
			}
			record = &lottery.LotteryRecord{}
			if err := json.Unmarshal([]byte(data), record); err != nil {
				return fmt.Errorf("failed to parse record: %w", err)
			}
		} else {
			// Try to find by ID
			records := chain.GetLotteryRecords()
			found := false
			for _, data := range records {
				if strings.Contains(data, input) {
					record = &lottery.LotteryRecord{}
					if err := json.Unmarshal([]byte(data), record); err == nil {
						found = true
						break
					}
				}
			}
			if !found {
				return fmt.Errorf("lottery not found: %s", input)
			}
		}

		// Display verification info
		fmt.Println("\n✅ Lottery Record Verified!")
		fmt.Printf("📋 ID: %s\n", record.ID)
		fmt.Printf("🔢 Block Height: #%d\n", record.BlockHeight)
		fmt.Printf("🌱 Seed: %s\n", record.Seed)
		fmt.Printf("👥 Participants: %d\n", len(record.Participants))
		fmt.Printf("🎉 Winners: %d\n", len(record.Winners))
		fmt.Println("\n🏆 Winners:")
		for i, w := range record.Winners {
			fmt.Printf("   %d. %s (%s)\n", i+1, w, record.WinnerAddrs[i])
		}
		fmt.Printf("\n🔐 VRF Output: %s...\n", record.VRFOutput[:min(16, len(record.VRFOutput))])
		fmt.Printf("📜 VRF Proof: %s...\n", record.VRFProof[:min(16, len(record.VRFProof))])
		fmt.Printf("⏰ Timestamp: %d\n", record.Timestamp)

		return nil
	},
}

var exportCmd = &cobra.Command{
	Use:   "export [file.json]",
	Short: "Export lottery history to JSON file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := args[0]
		chain := blockchain.InitBlockChain()
		records := chain.GetLotteryRecords()

		var lotteryRecords []*lottery.LotteryRecord
		for _, data := range records {
			var record lottery.LotteryRecord
			if err := json.Unmarshal([]byte(data), &record); err == nil {
				lotteryRecords = append(lotteryRecords, &record)
			}
		}

		output, err := json.MarshalIndent(lotteryRecords, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal: %w", err)
		}

		if err := os.WriteFile(filename, output, 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}

		fmt.Printf("✅ Exported %d lottery records to %s\n", len(lotteryRecords), filename)
		return nil
	},
}

var importCmd = &cobra.Command{
	Use:   "import [file.json]",
	Short: "Import lottery records from JSON file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := args[0]

		data, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		var records []lottery.LotteryRecord
		if err := json.Unmarshal(data, &records); err != nil {
			return fmt.Errorf("failed to parse file: %w", err)
		}

		chain := blockchain.InitBlockChain()
		imported := 0

		for _, record := range records {
			jsonData, err := record.ToJSON()
			if err != nil {
				continue
			}
			if _, err := chain.AddLotteryRecord(jsonData); err == nil {
				imported++
			}
		}

		fmt.Printf("✅ Imported %d lottery records\n", imported)
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

	createCmd.Flags().StringP("participants", "p", "", "Participant names (comma-separated)")
	createCmd.Flags().StringP("seed", "s", "", "Random seed")
	createCmd.Flags().IntP("count", "c", viper.GetInt("lottery.default_count"), "Number of winners")

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
