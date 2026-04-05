package cmd

import (
	"fmt"

	appnft "github.com/pplmx/aurora/internal/app/nft"
	nftdomain "github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/infra/sqlite"
	uinftr "github.com/pplmx/aurora/internal/ui/nft"
	"github.com/spf13/cobra"
)

var nftCmd = &cobra.Command{
	Use:   "nft",
	Short: "NFT system",
	Long:  "Ed25519 signature based NFT system",
}

func init() {
	rootCmd.AddCommand(nftCmd)

	repo, err := sqlite.NewNFTRepository("data/aurora.db")
	if err != nil {
		panic(fmt.Errorf("failed to initialize NFT repository: %w", err))
	}
	defer repo.Close()

	service := nftdomain.NewService(repo)

	mintUC := appnft.NewMintNFTUseCase(service)
	transferUC := appnft.NewTransferNFTUseCase(service)
	burnUC := appnft.NewBurnNFTUseCase(service)
	getUC := appnft.NewGetNFTUseCase(service)
	listUC := appnft.NewListNFTsByOwnerUseCase(service)
	historyUC := appnft.NewGetNFTOperationsUseCase(service)

	nftTuiCmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch TUI interface",
		Run: func(cmd *cobra.Command, args []string) {
			if err := uinftr.RunNFTUI(); err != nil {
				fmt.Println("Error:", err)
			}
		},
	}
	nftCmd.AddCommand(nftTuiCmd)

	mintCmd := &cobra.Command{
		Use:   "mint",
		Short: "Mint a new NFT",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			description, _ := cmd.Flags().GetString("description")
			imageURL, _ := cmd.Flags().GetString("image")
			tokenURI, _ := cmd.Flags().GetString("token-uri")
			creator, _ := cmd.Flags().GetString("creator")

			req := &appnft.MintNFTRequest{
				Name:        name,
				Description: description,
				ImageURL:    imageURL,
				TokenURI:    tokenURI,
				Creator:     creator,
			}

			result, err := mintUC.Execute(req)
			if err != nil {
				return fmt.Errorf("failed to mint NFT: %w", err)
			}

			fmt.Println("✅ NFT minted successfully!")
			fmt.Printf("   ID: %s\n", result.ID)
			fmt.Printf("   Name: %s\n", result.Name)
			fmt.Printf("   Owner: %s\n", result.Owner)
			fmt.Printf("   Block Height: #%d\n", result.BlockHeight)
			return nil
		},
	}
	nftCmd.AddCommand(mintCmd)
	mintCmd.Flags().StringP("name", "n", "", "NFT name")
	mintCmd.Flags().StringP("description", "d", "", "NFT description")
	mintCmd.Flags().StringP("image", "i", "", "Image URL")
	mintCmd.Flags().StringP("token-uri", "t", "", "Token URI")
	mintCmd.Flags().StringP("creator", "c", "", "Creator public key (Base64)")
	_ = mintCmd.MarkFlagRequired("name")
	_ = mintCmd.MarkFlagRequired("creator")

	transferCmd := &cobra.Command{
		Use:   "transfer",
		Short: "Transfer NFT ownership",
		RunE: func(cmd *cobra.Command, args []string) error {
			nftID, _ := cmd.Flags().GetString("nft")
			from, _ := cmd.Flags().GetString("from")
			to, _ := cmd.Flags().GetString("to")
			privateKey, _ := cmd.Flags().GetString("private-key")

			req := &appnft.TransferNFTRequest{
				NFTID:      nftID,
				From:       from,
				To:         to,
				PrivateKey: privateKey,
			}

			result, err := transferUC.Execute(req)
			if err != nil {
				return fmt.Errorf("failed to transfer NFT: %w", err)
			}

			fmt.Println("✅ NFT transferred successfully!")
			fmt.Printf("   Operation ID: %s\n", result.ID)
			fmt.Printf("   From: %s\n", truncateBase64(result.From))
			fmt.Printf("   To: %s\n", truncateBase64(result.To))
			fmt.Printf("   Block Height: #%d\n", result.BlockHeight)
			return nil
		},
	}
	nftCmd.AddCommand(transferCmd)
	transferCmd.Flags().StringP("nft", "i", "", "NFT ID")
	transferCmd.Flags().StringP("from", "f", "", "From public key (Base64)")
	transferCmd.Flags().StringP("to", "", "", "To public key (Base64)")
	transferCmd.Flags().StringP("private-key", "k", "", "From private key (Base64)")
	_ = transferCmd.MarkFlagRequired("nft")
	_ = transferCmd.MarkFlagRequired("from")
	_ = transferCmd.MarkFlagRequired("to")
	_ = transferCmd.MarkFlagRequired("private-key")

	burnCmd := &cobra.Command{
		Use:   "burn",
		Short: "Burn an NFT",
		RunE: func(cmd *cobra.Command, args []string) error {
			nftID, _ := cmd.Flags().GetString("nft")
			owner, _ := cmd.Flags().GetString("owner")
			privateKey, _ := cmd.Flags().GetString("private-key")

			req := &appnft.BurnNFTRequest{
				NFTID:      nftID,
				Owner:      owner,
				PrivateKey: privateKey,
			}

			if err := burnUC.Execute(req); err != nil {
				return fmt.Errorf("failed to burn NFT: %w", err)
			}

			fmt.Println("✅ NFT burned successfully!")
			return nil
		},
	}
	nftCmd.AddCommand(burnCmd)
	burnCmd.Flags().StringP("nft", "i", "", "NFT ID")
	burnCmd.Flags().StringP("owner", "o", "", "Owner public key (Base64)")
	burnCmd.Flags().StringP("private-key", "k", "", "Owner private key (Base64)")
	_ = burnCmd.MarkFlagRequired("nft")
	_ = burnCmd.MarkFlagRequired("owner")
	_ = burnCmd.MarkFlagRequired("private-key")

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get NFT by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			nftID, _ := cmd.Flags().GetString("id")

			result, err := getUC.Execute(nftID)
			if err != nil {
				return fmt.Errorf("failed to get NFT: %w", err)
			}

			fmt.Println("\n🎨 NFT Details:")
			fmt.Printf("   ID: %s\n", result.ID)
			fmt.Printf("   Name: %s\n", result.Name)
			fmt.Printf("   Description: %s\n", result.Description)
			fmt.Printf("   Image URL: %s\n", result.ImageURL)
			fmt.Printf("   Token URI: %s\n", result.TokenURI)
			fmt.Printf("   Creator: %s\n", result.Creator)
			fmt.Printf("   Owner: %s\n", result.Owner)
			fmt.Printf("   Block Height: #%d\n", result.BlockHeight)
			return nil
		},
	}
	nftCmd.AddCommand(getCmd)
	getCmd.Flags().StringP("id", "i", "", "NFT ID")
	_ = getCmd.MarkFlagRequired("id")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List NFTs by owner",
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, _ := cmd.Flags().GetString("owner")

			results, err := listUC.Execute(owner)
			if err != nil {
				return fmt.Errorf("failed to list NFTs: %w", err)
			}

			fmt.Printf("\n🎨 NFTs owned: %d\n", len(results))
			if len(results) == 0 {
				fmt.Println("   (none)")
			}
			for _, n := range results {
				fmt.Printf("   - %s (%s)\n", n.ID, n.Name)
			}
			return nil
		},
	}
	nftCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("owner", "o", "", "Owner public key (Base64)")
	_ = listCmd.MarkFlagRequired("owner")

	historyCmd := &cobra.Command{
		Use:   "history",
		Short: "Get NFT operation history",
		RunE: func(cmd *cobra.Command, args []string) error {
			nftID, _ := cmd.Flags().GetString("nft")

			results, err := historyUC.Execute(nftID)
			if err != nil {
				return fmt.Errorf("failed to get history: %w", err)
			}

			fmt.Printf("\n📜 Operations: %d\n", len(results))
			if len(results) == 0 {
				fmt.Println("   (none)")
			}
			for _, op := range results {
				fmt.Printf("   - %s @ Block #%d\n", op.Type, op.BlockHeight)
			}
			return nil
		},
	}
	nftCmd.AddCommand(historyCmd)
	historyCmd.Flags().StringP("nft", "i", "", "NFT ID")
	_ = historyCmd.MarkFlagRequired("nft")
}

func truncateBase64(s string) string {
	if len(s) > 20 {
		return s[:20] + "..."
	}
	return s
}
