package cmd

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pplmx/aurora/internal/app/nftapp"
	"github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	"github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/pplmx/aurora/internal/infra/store"
	"github.com/spf13/cobra"
)

var nftCmd = &cobra.Command{
	Use:   "nft",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("nft.cmd"),
	Long:  i18n.GetText("nft.long"),
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
		nftRepo, nftRepoErr = sqlite.NewNFTRepository(store.Path())
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

func nftChain() (*blockchain.Chain, error) {
	return blockchain.InitWithError()
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
			return wrapCLIError("nft.error.init_service", err)
		}

		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")
		imageURL, _ := cmd.Flags().GetString("image")
		tokenURI, _ := cmd.Flags().GetString("token-uri")
		creator, _ := cmd.Flags().GetString("creator")

		chain, err := nftChain()
		if err != nil {
			return wrapCLIError("nft.error.init_blockchain", err)
		}
		mintUC := nftapp.NewMint(service, chain)
		req := &nftapp.MintNFTRequest{
			Name:        name,
			Description: description,
			ImageURL:    imageURL,
			TokenURI:    tokenURI,
			Creator:     creator,
		}

		result, err := mintUC.Execute(req)
		if err != nil {
			return wrapCLIError("nft.error.mint", err)
		}
		if jsonRequested(cmd) {
			return printJSON(result)
		}

		fmt.Println("✅ " + i18n.GetText("nft.minted"))
		printFields(
			field{i18n.GetText("common.id"), result.ID},
			field{i18n.GetText("common.name"), result.Name},
			field{i18n.GetText("common.owner"), result.Owner},
		)
		fmt.Printf("   %s\n", i18n.Label(i18n.GetText("common.block_height"), fmt.Sprintf("#%d", result.BlockHeight)))
		return nil
	},
}

var transferCmd = &cobra.Command{
	Use:     "transfer",
	Example: `  aurora nft transfer --nft <nft-id> --from <owner-public-key> --to <new-owner-public-key> -k "<owner-private-key>"`,
	Args:    cobra.NoArgs,
	Short:   i18n.GetText("nft.transfer"),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, err := nftService()
		if err != nil {
			return wrapCLIError("nft.error.init_service", err)
		}

		nftID, _ := cmd.Flags().GetString("nft")
		from, _ := cmd.Flags().GetString("from")
		to, _ := cmd.Flags().GetString("to")
		privateKey, _ := cmd.Flags().GetString("private-key")

		chain, err := nftChain()
		if err != nil {
			return wrapCLIError("nft.error.init_blockchain", err)
		}
		transferUC := nftapp.NewTransferNFTUseCase(service, chain)
		req := &nftapp.TransferNFTRequest{
			NFTID:      nftID,
			From:       from,
			To:         to,
			PrivateKey: privateKey,
		}

		result, err := transferUC.Execute(req)
		if err != nil {
			return wrapCLIError("nft.error.transfer", err)
		}
		if jsonRequested(cmd) {
			return printJSON(result)
		}

		fmt.Println("✅ " + i18n.GetText("nft.transferred"))
		printFields(
			field{i18n.GetText("nft.operation_id"), result.ID},
			field{i18n.GetText("common.from"), truncateBase64(result.From)},
			field{i18n.GetText("common.to"), truncateBase64(result.To)},
			field{i18n.GetText("common.block_height"), fmt.Sprintf("#%d", result.BlockHeight)},
		)
		return nil
	},
}

var burnCmd = &cobra.Command{
	Use:     "burn",
	Example: `  aurora nft burn --nft <nft-id> --owner <owner-public-key> -k "<owner-private-key>" --confirm`,
	Args:    cobra.NoArgs,
	Short:   i18n.GetText("nft.burn"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Burning permanently destroys an NFT; require -y/--confirm first
		// (the same gate every destructive command uses).
		if err := requireConfirm(cmd, "cli.confirm.nft_burn"); err != nil {
			return err
		}
		service, err := nftService()
		if err != nil {
			return wrapCLIError("nft.error.init_service", err)
		}

		nftID, _ := cmd.Flags().GetString("nft")
		owner, _ := cmd.Flags().GetString("owner")
		privateKey, _ := cmd.Flags().GetString("private-key")

		chain, err := nftChain()
		if err != nil {
			return wrapCLIError("nft.error.init_blockchain", err)
		}
		burnUC := nftapp.NewBurnNFTUseCase(service, chain)
		req := &nftapp.BurnNFTRequest{
			NFTID:      nftID,
			Owner:      owner,
			PrivateKey: privateKey,
		}

		if err := burnUC.Execute(req); err != nil {
			return wrapCLIError("nft.error.burn", err)
		}
		if jsonRequested(cmd) {
			// The REST burn endpoint answers {"status":"burned"}; match it so a
			// caller parses one shape whichever surface burned the asset.
			return printJSON(map[string]string{"status": "burned"})
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
			return wrapCLIError("nft.error.init_service", err)
		}

		nftID, _ := cmd.Flags().GetString("nft")

		getUC := nftapp.NewGet(service)
		result, err := getUC.Execute(nftID)
		if err != nil {
			return wrapCLIError("nft.error.get", err)
		}
		if jsonRequested(cmd) {
			return printJSON(result)
		}

		fmt.Printf("\n🎨 %s\n", i18n.GetText("nft.details_heading")+i18n.Colon())
		var fields []field
		fields = addLocalizedField(fields, "common.id", result.ID)
		fields = addLocalizedField(fields, "common.name", result.Name)
		// Metadata is optional: an NFT minted without it keeps the fields out of
		// the block instead of printing empty labels.
		fields = addLocalizedField(fields, "nft.description_label", result.Description)
		fields = addLocalizedField(fields, "nft.image_url", result.ImageURL)
		fields = addLocalizedField(fields, "nft.token_uri", result.TokenURI)
		fields = addLocalizedField(fields, "nft.creator_label", result.Creator)
		fields = addLocalizedField(fields, "common.owner", result.Owner)
		fields = addField(fields, i18n.GetText("common.block_height"), fmt.Sprintf("#%d", result.BlockHeight))
		printFields(fields...)
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
			return wrapCLIError("nft.error.init_service", err)
		}

		owner, _ := cmd.Flags().GetString("owner")
		search, _ := cmd.Flags().GetString("search")
		search = strings.TrimSpace(search)
		// One read answers one question. Answering the owner and dropping the
		// term would print rows the operator did not ask for, and the count
		// would still say "matches".
		if owner != "" && search != "" {
			return fmt.Errorf("%s", i18n.GetText("nft.list_owner_or_search"))
		}

		// The CLI lists the full collection (0,0 = unbounded); only the REST
		// list is paged/bounded (TASK-101, ISS-093).
		var results []*nftapp.NFTResponse
		switch {
		case owner != "":
			results, err = nftapp.NewListByOwner(service).Execute(
				&nftapp.ListNFTsByOwnerRequest{Owner: owner})
		case search != "":
			// The term goes to storage, so it finds an asset whose row is on an
			// older page rather than filtering the list already read.
			results, err = nftapp.NewListAll(service).Execute(
				&nftapp.ListNFTsRequest{Search: search})
		default:
			// Without an owner or a term this is the whole collection: an owner
			// key is what a caller who did not mint the asset does not have.
			results, err = nftapp.NewListAll(service).Execute(&nftapp.ListNFTsRequest{})
		}
		if err != nil {
			return wrapCLIError("nft.error.list", err)
		}
		if jsonRequested(cmd) {
			if results == nil {
				results = []*nftapp.NFTResponse{}
			}
			return printJSON(results)
		}

		var label string
		switch {
		case owner != "":
			label = i18n.GetText("nft.list_owned")
		case search != "":
			// "All NFTs" beside a filter would read as the store's count; name
			// what the number is, as `token list` and `lottery history` do.
			label = i18n.GetText("nft.matches")
		default:
			label = i18n.GetText("nft.list_all")
		}
		fmt.Printf("\n🎨 %s\n", i18n.Label(label, len(results)))
		if len(results) == 0 {
			// A term that matches nothing is not an empty collection: saying
			// "none" beside a filter would send the operator off to mint what
			// already exists.
			if search != "" && owner == "" {
				fmt.Printf("   "+i18n.GetText("nft.no_match")+"\n", search)
			} else {
				fmt.Println("   " + i18n.GetText("common.none"))
			}
		}
		// The row carries the holder as well as the identity: `token list` shows an
		// owner and an NFT's holder decides every later write, so a list that gave
		// only an id and a name left the reader to look each asset up one by one.
		for _, n := range results {
			printFields(
				field{i18n.GetText("common.id"), n.ID},
				field{i18n.GetText("common.name"), n.Name},
				field{i18n.GetText("common.owner"), n.Owner},
				field{i18n.GetText("common.block_height"), fmt.Sprintf("#%d", n.BlockHeight)},
			)
			fmt.Println()
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
			return wrapCLIError("nft.error.init_service", err)
		}

		nftID, _ := cmd.Flags().GetString("nft")
		limit, offset, err := pagingFlags(cmd)
		if err != nil {
			return err
		}

		historyUC := nftapp.NewGetNFTOperationsUseCase(service)
		results, err := historyUC.Execute(&nftapp.GetNFTOperationsRequest{
			NFTID: nftID, Limit: limit, Offset: offset,
		})
		if err != nil {
			return wrapCLIError("nft.error.history", err)
		}
		if jsonRequested(cmd) {
			if results == nil {
				results = []*nftapp.OperationResponse{}
			}
			return printJSON(results)
		}

		// An offset alone asks for a page too, as it does in the other lists.
		// Each mode names what its number counts: the page's rows, or the whole
		// history. "Operations: 1 (offset 0)" said neither.
		if limit > 0 || offset > 0 {
			fmt.Printf("\n📜 %s\n", i18n.Label(i18n.GetText("nft.operations_page"), i18n.GetTextF("common.page_with_offset", len(results), offset)))
		} else {
			fmt.Printf("\n📜 %s\n", i18n.Label(i18n.GetText("nft.operations_total"), len(results)))
		}
		if len(results) == 0 {
			// Three different answers: an empty history, a page past the end,
			// and a page the offset skipped past.
			if limit > 0 || offset > 0 {
				fmt.Println("   " + i18n.GetText("common.page_empty"))
				return nil
			}
			fmt.Println("   " + i18n.GetText("common.none"))
		}
		for _, op := range results {
			// A history is read for when an operation happened: without the time
			// the row dates nothing, and an asset's whole past reads as a list of
			// block numbers.
			fmt.Printf("   - %s\n", i18n.GetTextF(
				"nft.operation_line",
				localizedNFTOperationType(op.Type),
				op.BlockHeight,
				time.Unix(op.Timestamp, 0).Format(blockTimeLayout),
			))
			// The counterparty is its own labelled line: a mint has no sender and
			// a burn has no recipient, so folding it into the line above would
			// need two formats for the same fact.
			counterpart, label := op.To, "nft.holder_after"
			if counterpart == "" {
				counterpart, label = op.From, "nft.holder_before"
			}
			if counterpart != "" {
				fmt.Printf("     %s\n", i18n.Label(i18n.GetText(label), truncateBase64(counterpart)))
			}
		}
		return nil
	},
}

func localizedNFTOperationType(opType string) string {
	key := "nft.operation." + opType
	if translated := i18n.GetText(key); translated != key {
		return translated
	}
	return opType
}

func init() {
	rootCmd.AddCommand(nftCmd)

	nftCmd.AddCommand(mintCmd)
	addJSONFlag(mintCmd)
	nftCmd.AddCommand(transferCmd)
	addJSONFlag(transferCmd)
	nftCmd.AddCommand(burnCmd)
	addJSONFlag(burnCmd)
	nftCmd.AddCommand(getCmd)
	addJSONFlag(getCmd)
	nftCmd.AddCommand(listCmd)
	addJSONFlag(listCmd)
	nftCmd.AddCommand(nftHistoryCmd)
	addJSONFlag(nftHistoryCmd)

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
	listCmd.Flags().StringP("search", "s", "", i18n.GetText("nft.list_search"))

	nftHistoryCmd.Flags().StringP("nft", "i", "", i18n.GetText("nft.nft_id"))
	nftHistoryCmd.Flags().IntP("limit", "l", 0, i18n.GetText("common.list_limit"))
	nftHistoryCmd.Flags().IntP("offset", "p", 0, i18n.GetText("token.offset"))
	_ = nftHistoryCmd.MarkFlagRequired("nft")
}

func truncateBase64(s string) string {
	if len(s) > 20 {
		return s[:20] + "..."
	}
	return s
}
