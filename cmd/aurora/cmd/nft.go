package cmd

import (
	"fmt"
	"sync"

	"github.com/pplmx/aurora/internal/app/nftapp"
	"github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	"github.com/pplmx/aurora/internal/infra/sqlite"
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

func getNFTRepo() (nft.TransactableRepository, error) {
	nftRepoOnce.Do(func() {
		nftRepo, nftRepoErr = sqlite.NewNFTRepository(blockchain.DBPath())
	})
	if nftRepoErr != nil {
		return nil, nftRepoErr
	}
	return nftRepo, nil
}

func nftService() (nft.Service, error) {
	repo, err := getNFTRepo()
	if err != nil {
		return nil, err
	}
	// Multi-step NFT writes (NFT row + audit operation) run in one SQLite
	// transaction; the TxManager must share the repository's pool so the
	// tx-scoped repository and the transaction operate on the same database.
	txManager := sqlite.NewTxManager(nftRepo.GetDB())
	return nft.NewService(repo, txManager), nil
}

func nftChain() *blockchain.Chain {
	return blockchain.Init()
}

var mintCmd = &cobra.Command{
	Use:   "mint",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("nft.mint"),
	Example: `  aurora nft mint -n "MyNFT" -d "A unique digital asset" -c "creator-pubkey"
  aurora nft mint -n "GameItem #1" -d "Rare sword" -c "player-key" --image "https://example.com/item.png"`,
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

		mintUC := nftapp.NewMint(service, nftChain())
		req := &nftapp.MintNFTRequest{
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

		fmt.Println("✅ " + i18n.GetText("nft.minted"))
		printField("ID", result.ID)
		printField("Name", result.Name)
		printField("Owner", result.Owner)
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

		transferUC := nftapp.NewTransferNFTUseCase(service, nftChain())
		req := &nftapp.TransferNFTRequest{
			NFTID:      nftID,
			From:       from,
			To:         to,
			PrivateKey: privateKey,
		}

		result, err := transferUC.Execute(req)
		if err != nil {
			return fmt.Errorf("failed to transfer NFT: %w", err)
		}

		fmt.Println("✅ " + i18n.GetText("nft.transferred"))
		printField("Operation ID", result.ID)
		printField("From", truncateBase64(result.From))
		printField("To", truncateBase64(result.To))
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

		burnUC := nftapp.NewBurnNFTUseCase(service, nftChain())
		req := &nftapp.BurnNFTRequest{
			NFTID:      nftID,
			Owner:      owner,
			PrivateKey: privateKey,
		}

		if err := burnUC.Execute(req); err != nil {
			return fmt.Errorf("failed to burn NFT: %w", err)
		}

		fmt.Println("✅ " + i18n.GetText("nft.burned"))
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

		getUC := nftapp.NewGet(service)
		result, err := getUC.Execute(nftID)
		if err != nil {
			return fmt.Errorf("failed to get NFT: %w", err)
		}

		fmt.Println("\n🎨 NFT Details:")
		printField("ID", result.ID)
		printField("Name", result.Name)
		printField("Description", result.Description)
		printField("Image URL", result.ImageURL)
		printField("Token URI", result.TokenURI)
		printField("Creator", result.Creator)
		printField("Owner", result.Owner)
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
		var results []*nftapp.NFTResponse
		if owner == "" {
			// Without an owner this is the whole collection: an owner key is
			// what a caller who did not mint the asset does not have.
			results, err = nftapp.NewListAll(service).Execute(&nftapp.ListNFTsRequest{})
		} else {
			results, err = nftapp.NewListByOwner(service).Execute(
				&nftapp.ListNFTsByOwnerRequest{Owner: owner})
		}
		if err != nil {
			return fmt.Errorf("failed to list NFTs: %w", err)
		}

		label := "NFTs owned"
		if owner == "" {
			label = "NFTs"
		}
		fmt.Printf("\n🎨 %s: %d\n", label, len(results))
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
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		if limit < 0 {
			return fmt.Errorf("invalid --limit %d: must be zero or greater", limit)
		}
		if offset < 0 {
			return fmt.Errorf("invalid --offset %d: must be zero or greater", offset)
		}

		historyUC := nftapp.NewGetNFTOperationsUseCase(service)
		results, err := historyUC.Execute(&nftapp.GetNFTOperationsRequest{
			NFTID: nftID, Limit: limit, Offset: offset,
		})
		if err != nil {
			return fmt.Errorf("failed to get history: %w", err)
		}

		if limit > 0 {
			fmt.Printf("\n📜 Operations: %d (offset %d)\n", len(results), offset)
		} else {
			fmt.Printf("\n📜 Operations: %d\n", len(results))
		}
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

	nftCmd.AddCommand(mintCmd)
	nftCmd.AddCommand(transferCmd)
	nftCmd.AddCommand(burnCmd)
	nftCmd.AddCommand(getCmd)
	nftCmd.AddCommand(listCmd)
	nftCmd.AddCommand(nftHistoryCmd)

	mintCmd.Flags().StringP("name", "n", "", i18n.GetText("nft.name"))
	mintCmd.Flags().StringP("description", "d", "", i18n.GetText("nft.description"))
	// --image with NO -i shorthand: -i is the NFT-ID selector on get/transfer/
	// burn/history (TASK-152, ISS-142), so leaving -i on mint's image flag made
	// `nft mint -i <id>` (typed by analogy with get) silently set an image URL
	// instead of an NFT ID. Dropping the shorthand makes -i mean NFT ID across
	// the whole family (H1 shorthand-overload audit).
	mintCmd.Flags().String("image", "", i18n.GetText("nft.image_url"))
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

	// The owner is optional: without it the command lists every NFT, which is
	// the only way to reach an asset whose owner key the caller never had.
	listCmd.Flags().StringP("owner", "o", "", i18n.GetText("nft.owner"))

	nftHistoryCmd.Flags().StringP("nft", "i", "", i18n.GetText("nft.nft_id"))
	nftHistoryCmd.Flags().IntP("limit", "l", 0, i18n.GetText("token.list_limit"))
	nftHistoryCmd.Flags().IntP("offset", "p", 0, i18n.GetText("token.offset"))
	_ = nftHistoryCmd.MarkFlagRequired("nft")
}

func truncateBase64(s string) string {
	if len(s) > 20 {
		return s[:20] + "..."
	}
	return s
}
