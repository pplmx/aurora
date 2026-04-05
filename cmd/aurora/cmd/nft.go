package cmd

import (
	"encoding/base64"
	"fmt"

	"github.com/pplmx/aurora/internal/blockchain"
	"github.com/pplmx/aurora/internal/nft"
	"github.com/spf13/cobra"
)

var nftCmd = &cobra.Command{
	Use:   "nft",
	Short: "NFT system",
	Long:  "Ed25519 signature based NFT system",
}

var mintCmd = &cobra.Command{
	Use:   "mint",
	Short: "Mint a new NFT",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")
		imageURL, _ := cmd.Flags().GetString("image")
		tokenURI, _ := cmd.Flags().GetString("token-uri")
		creatorPubB64, _ := cmd.Flags().GetString("creator")

		creatorPub, err := base64.StdEncoding.DecodeString(creatorPubB64)
		if err != nil {
			return fmt.Errorf("invalid creator public key: %w", err)
		}

		chain := blockchain.InitBlockChain()
		nftItem, err := nft.MintNFT(name, description, imageURL, tokenURI, creatorPub, chain)
		if err != nil {
			return fmt.Errorf("failed to mint NFT: %w", err)
		}

		fmt.Println("✅ NFT minted successfully!")
		fmt.Printf("   ID: %s\n", nftItem.ID)
		fmt.Printf("   Name: %s\n", nftItem.Name)
		fmt.Printf("   Owner: %s\n", nftItem.Owner)
		fmt.Printf("   Block Height: #%d\n", nftItem.BlockHeight)
		return nil
	},
}

var transferCmd = &cobra.Command{
	Use:   "transfer",
	Short: "Transfer NFT ownership",
	RunE: func(cmd *cobra.Command, args []string) error {
		nftID, _ := cmd.Flags().GetString("nft")
		fromPubB64, _ := cmd.Flags().GetString("from")
		toPubB64, _ := cmd.Flags().GetString("to")
		fromPrivB64, _ := cmd.Flags().GetString("private-key")

		fromPub, err := base64.StdEncoding.DecodeString(fromPubB64)
		if err != nil {
			return fmt.Errorf("invalid from public key: %w", err)
		}

		toPub, err := base64.StdEncoding.DecodeString(toPubB64)
		if err != nil {
			return fmt.Errorf("invalid to public key: %w", err)
		}

		fromPriv, err := base64.StdEncoding.DecodeString(fromPrivB64)
		if err != nil {
			return fmt.Errorf("invalid private key: %w", err)
		}

		chain := blockchain.InitBlockChain()
		op, err := nft.TransferNFT(nftID, fromPub, fromPriv, toPub, chain)
		if err != nil {
			return fmt.Errorf("failed to transfer NFT: %w", err)
		}

		fmt.Println("✅ NFT transferred successfully!")
		fmt.Printf("   Operation ID: %s\n", op.ID)
		fmt.Printf("   From: %s\n", truncateBase64(op.From))
		fmt.Printf("   To: %s\n", truncateBase64(op.To))
		fmt.Printf("   Block Height: #%d\n", op.BlockHeight)
		return nil
	},
}

var burnCmd = &cobra.Command{
	Use:   "burn",
	Short: "Burn an NFT",
	RunE: func(cmd *cobra.Command, args []string) error {
		nftID, _ := cmd.Flags().GetString("nft")
		ownerPubB64, _ := cmd.Flags().GetString("owner")
		ownerPrivB64, _ := cmd.Flags().GetString("private-key")

		ownerPub, err := base64.StdEncoding.DecodeString(ownerPubB64)
		if err != nil {
			return fmt.Errorf("invalid owner public key: %w", err)
		}

		ownerPriv, err := base64.StdEncoding.DecodeString(ownerPrivB64)
		if err != nil {
			return fmt.Errorf("invalid private key: %w", err)
		}

		chain := blockchain.InitBlockChain()
		err = nft.BurnNFT(nftID, ownerPub, ownerPriv, chain)
		if err != nil {
			return fmt.Errorf("failed to burn NFT: %w", err)
		}

		fmt.Println("✅ NFT burned successfully!")
		return nil
	},
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get NFT by ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		nftID, _ := cmd.Flags().GetString("id")

		nftItem, err := nft.GetNFTByID(nftID)
		if err != nil {
			return fmt.Errorf("failed to get NFT: %w", err)
		}
		if nftItem == nil {
			return fmt.Errorf("NFT not found")
		}

		fmt.Println("\n🎨 NFT Details:")
		fmt.Printf("   ID: %s\n", nftItem.ID)
		fmt.Printf("   Name: %s\n", nftItem.Name)
		fmt.Printf("   Description: %s\n", nftItem.Description)
		fmt.Printf("   Image URL: %s\n", nftItem.ImageURL)
		fmt.Printf("   Token URI: %s\n", nftItem.TokenURI)
		fmt.Printf("   Creator: %s\n", nftItem.Creator)
		fmt.Printf("   Owner: %s\n", nftItem.Owner)
		fmt.Printf("   Block Height: #%d\n", nftItem.BlockHeight)
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List NFTs by owner",
	RunE: func(cmd *cobra.Command, args []string) error {
		ownerPubB64, _ := cmd.Flags().GetString("owner")

		ownerPub, err := base64.StdEncoding.DecodeString(ownerPubB64)
		if err != nil {
			return fmt.Errorf("invalid owner public key: %w", err)
		}

		nfts, err := nft.GetNFTsByOwner(ownerPub)
		if err != nil {
			return fmt.Errorf("failed to list NFTs: %w", err)
		}

		fmt.Printf("\n🎨 NFTs owned: %d\n", len(nfts))
		if len(nfts) == 0 {
			fmt.Println("   (none)")
		}
		for _, n := range nfts {
			fmt.Printf("   - %s (%s)\n", n.ID, n.Name)
		}
		return nil
	},
}

var nftHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Get NFT operation history",
	RunE: func(cmd *cobra.Command, args []string) error {
		nftID, _ := cmd.Flags().GetString("nft")

		ops, err := nft.GetNFTOperations(nftID)
		if err != nil {
			return fmt.Errorf("failed to get history: %w", err)
		}

		fmt.Printf("\n📜 Operations: %d\n", len(ops))
		if len(ops) == 0 {
			fmt.Println("   (none)")
		}
		for _, op := range ops {
			fmt.Printf("   - %s @ Block #%d\n", op.Operation, op.BlockHeight)
		}
		return nil
	},
}

func truncateBase64(s string) string {
	if len(s) > 20 {
		return s[:20] + "..."
	}
	return s
}

var nftTuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch TUI interface",
	Run: func(cmd *cobra.Command, args []string) {
		storage := nft.NewNFTStorage()
		nft.SetNFTStorage(storage)
		if err := nft.RunNFTUI(storage); err != nil {
			fmt.Println("Error:", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(nftCmd)

	nftCmd.AddCommand(nftTuiCmd)
	nftCmd.AddCommand(mintCmd)
	mintCmd.Flags().StringP("name", "n", "", "NFT name")
	mintCmd.Flags().StringP("description", "d", "", "NFT description")
	mintCmd.Flags().StringP("image", "i", "", "Image URL")
	mintCmd.Flags().StringP("token-uri", "t", "", "Token URI")
	mintCmd.Flags().StringP("creator", "c", "", "Creator public key (Base64)")
	_ = mintCmd.MarkFlagRequired("name")
	_ = mintCmd.MarkFlagRequired("creator")

	nftCmd.AddCommand(transferCmd)
	transferCmd.Flags().StringP("nft", "i", "", "NFT ID")
	transferCmd.Flags().StringP("from", "f", "", "From public key (Base64)")
	transferCmd.Flags().StringP("to", "", "", "To public key (Base64)")
	transferCmd.Flags().StringP("private-key", "k", "", "From private key (Base64)")
	_ = transferCmd.MarkFlagRequired("nft")
	_ = transferCmd.MarkFlagRequired("from")
	_ = transferCmd.MarkFlagRequired("to")
	_ = transferCmd.MarkFlagRequired("private-key")

	nftCmd.AddCommand(burnCmd)
	burnCmd.Flags().StringP("nft", "i", "", "NFT ID")
	burnCmd.Flags().StringP("owner", "o", "", "Owner public key (Base64)")
	burnCmd.Flags().StringP("private-key", "k", "", "Owner private key (Base64)")
	_ = burnCmd.MarkFlagRequired("nft")
	_ = burnCmd.MarkFlagRequired("owner")
	_ = burnCmd.MarkFlagRequired("private-key")

	nftCmd.AddCommand(getCmd)
	getCmd.Flags().StringP("id", "i", "", "NFT ID")
	_ = getCmd.MarkFlagRequired("id")

	nftCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("owner", "o", "", "Owner public key (Base64)")
	listCmd.MarkFlagRequired("owner")

	nftCmd.AddCommand(nftHistoryCmd)
	nftHistoryCmd.Flags().StringP("nft", "i", "", "NFT ID")
	nftHistoryCmd.MarkFlagRequired("nft")
}
