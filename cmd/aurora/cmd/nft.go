package cmd

import (
	"fmt"
	"sync"

	appnft "github.com/pplmx/aurora/internal/app/nft"
	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	nftdomain "github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/sqlite"
	uinftr "github.com/pplmx/aurora/internal/ui/nft"
	"github.com/spf13/cobra"
)

var nftCmd = &cobra.Command{
	Use:   "nft",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("nft.cmd"),
	Long:  i18n.GetText("nft.cmd"),
}

// Lazy DB initialization.
//
// Originally this package opened the SQLite DB in init() and panicked on
// failure — which crashed the entire binary, including unrelated subcommands
// like `aurora lottery history` or `aurora --help`. We now lazily open the
// DB on first use so a misconfigured NFT environment cannot take down the
// rest of the CLI. Errors are reported per-subcommand instead.
var (
	nftRepoOnce sync.Once
	nftRepo     *sqlite.NFTRepository
	nftRepoErr  error
)

func getNFTRepo() (nftdomain.TransactableRepository, error) {
	nftRepoOnce.Do(func() {
		nftRepo, nftRepoErr = sqlite.NewNFTRepository(blockchain.DBPath())
	})
	if nftRepoErr != nil {
		return nil, nftRepoErr
	}
	return nftRepo, nil
}

func nftService() (nftdomain.Service, error) {
	repo, err := getNFTRepo()
	if err != nil {
		return nil, err
	}
	// Multi-step NFT writes (NFT row + audit operation) run in one SQLite
	// transaction; the TxManager must share the repository's pool so the
	// tx-scoped repository and the transaction operate on the same database.
	txManager := sqlite.NewTxManager(nftRepo.GetDB())
	return nftdomain.NewService(repo, txManager), nil
}

func nftChain() *blockchain.BlockChain {
	return blockchain.InitBlockChain()
}

var nftTuiCmd = &cobra.Command{
	Use:   "tui",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("nft.tui.cmd"),
	// RunE (not Run): a TUI failure must propagate to Execute() so the process
	// exits 1 and the error goes to stderr — the old Run: printed to stdout
	// and exited 0, so $?-checking scripts/CI reported success on a failed run
	// (v1.76, ISS-083).
	RunE: func(cmd *cobra.Command, args []string) error {
		return uinftr.RunNFTUI()
	},
}

var mintCmd = &cobra.Command{
	Use:   "mint",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("nft.mint"),
	Example: `  aurora nft mint -n "MyNFT" -d "A unique digital asset" -c "creator-pubkey"
  aurora nft mint -n "GameItem #1" -d "Rare sword" -c "player-key" -i "https://example.com/item.png"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := nftService()
		if err != nil {
			return fmt.Errorf("failed to initialize NFT service: %w", err)
		}

		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")
		imageURL, _ := cmd.Flags().GetString("image")
		tokenURI, _ := cmd.Flags().GetString("token-uri")
		creator, _ := cmd.Flags().GetString("creator")

		mintUC := appnft.NewMintNFTUseCase(service, nftChain())
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

var transferCmd = &cobra.Command{
	Use:   "transfer",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("nft.transfer"),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := nftService()
		if err != nil {
			return fmt.Errorf("failed to initialize NFT service: %w", err)
		}

		nftID, _ := cmd.Flags().GetString("nft")
		from, _ := cmd.Flags().GetString("from")
		to, _ := cmd.Flags().GetString("to")
		privateKey, _ := cmd.Flags().GetString("private-key")

		transferUC := appnft.NewTransferNFTUseCase(service, nftChain())
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

var burnCmd = &cobra.Command{
	Use:   "burn",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("nft.burn"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Burning permanently destroys an NFT; require -y/--confirm first
		// (matches backup restore --confirm / lottery reset --yes).
		if err := requireConfirm(cmd, "cli.confirm.nft_burn"); err != nil {
			return err
		}
		service, err := nftService()
		if err != nil {
			return fmt.Errorf("failed to initialize NFT service: %w", err)
		}

		nftID, _ := cmd.Flags().GetString("nft")
		owner, _ := cmd.Flags().GetString("owner")
		privateKey, _ := cmd.Flags().GetString("private-key")

		burnUC := appnft.NewBurnNFTUseCase(service, nftChain())
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

var getCmd = &cobra.Command{
	Use:   "get",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("nft.get"),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := nftService()
		if err != nil {
			return fmt.Errorf("failed to initialize NFT service: %w", err)
		}

		nftID, _ := cmd.Flags().GetString("nft")

		getUC := appnft.NewGetNFTUseCase(service)
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

var listCmd = &cobra.Command{
	Use:   "list",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("nft.list"),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := nftService()
		if err != nil {
			return fmt.Errorf("failed to initialize NFT service: %w", err)
		}

		owner, _ := cmd.Flags().GetString("owner")

		// The CLI lists the full collection (0,0 = unbounded); only the REST
		// list is paged/bounded (TASK-101, ISS-093).
		listUC := appnft.NewListNFTsByOwnerUseCase(service)
		results, err := listUC.Execute(&appnft.ListNFTsByOwnerRequest{Owner: owner})
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

var nftHistoryCmd = &cobra.Command{
	Use:   "history",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("nft.history"),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := nftService()
		if err != nil {
			return fmt.Errorf("failed to initialize NFT service: %w", err)
		}

		nftID, _ := cmd.Flags().GetString("nft")

		historyUC := appnft.NewGetNFTOperationsUseCase(service)
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

func init() {
	rootCmd.AddCommand(nftCmd)

	nftCmd.AddCommand(nftTuiCmd)
	nftCmd.AddCommand(mintCmd)
	nftCmd.AddCommand(transferCmd)
	nftCmd.AddCommand(burnCmd)
	nftCmd.AddCommand(getCmd)
	nftCmd.AddCommand(listCmd)
	nftCmd.AddCommand(nftHistoryCmd)

	mintCmd.Flags().StringP("name", "n", "", i18n.GetText("nft.name"))
	mintCmd.Flags().StringP("description", "d", "", i18n.GetText("nft.description"))
	mintCmd.Flags().StringP("image", "i", "", i18n.GetText("nft.image_url"))
	mintCmd.Flags().StringP("token-uri", "t", "", i18n.GetText("nft.token_uri"))
	mintCmd.Flags().StringP("creator", "c", "", i18n.GetText("nft.creator"))
	_ = mintCmd.MarkFlagRequired("name")
	_ = mintCmd.MarkFlagRequired("creator")

	transferCmd.Flags().StringP("nft", "i", "", i18n.GetText("nft.nft_id"))
	transferCmd.Flags().StringP("from", "f", "", i18n.GetText("nft.from"))
	transferCmd.Flags().StringP("to", "", "", i18n.GetText("nft.to"))
	transferCmd.Flags().StringP("private-key", "k", "", i18n.GetText("nft.private_key"))
	_ = transferCmd.MarkFlagRequired("nft")
	_ = transferCmd.MarkFlagRequired("from")
	_ = transferCmd.MarkFlagRequired("to")
	_ = transferCmd.MarkFlagRequired("private-key")

	burnCmd.Flags().StringP("nft", "i", "", i18n.GetText("nft.nft_id"))
	burnCmd.Flags().StringP("owner", "o", "", i18n.GetText("nft.owner"))
	burnCmd.Flags().StringP("private-key", "k", "", i18n.GetText("nft.private_key"))
	addConfirmFlag(burnCmd, "cli.confirm.nft_burn")
	_ = burnCmd.MarkFlagRequired("nft")
	_ = burnCmd.MarkFlagRequired("owner")
	_ = burnCmd.MarkFlagRequired("private-key")

	// --nft (not --id) so the NFT selector is spelled the same across get,
	// transfer, burn and history (TASK-152, ISS-142); -i matches them too.
	getCmd.Flags().StringP("nft", "i", "", i18n.GetText("nft.nft_id"))
	_ = getCmd.MarkFlagRequired("nft")

	listCmd.Flags().StringP("owner", "o", "", i18n.GetText("nft.owner"))
	_ = listCmd.MarkFlagRequired("owner")

	nftHistoryCmd.Flags().StringP("nft", "i", "", i18n.GetText("nft.nft_id"))
	_ = nftHistoryCmd.MarkFlagRequired("nft")
}

func truncateBase64(s string) string {
	if len(s) > 20 {
		return s[:20] + "..."
	}
	return s
}
