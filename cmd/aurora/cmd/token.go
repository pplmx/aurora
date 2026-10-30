package cmd

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"math"
	"strconv"

	"github.com/pplmx/aurora/internal/domain/token"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	infraevents "github.com/pplmx/aurora/internal/infra/events"
	"github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/pplmx/aurora/internal/logger"
	"github.com/spf13/cobra"
)

func newTokenService() (*token.TokenService, func(), error) {
	dbPath := blockchain.DBPath()
	repo, err := sqlite.NewTokenRepository(dbPath)
	if err != nil {
		return nil, nil, wrapCLIError("token.error.create_repo", err)
	}

	eventStore, err := infraevents.NewSQLiteEventStore(dbPath)
	if err != nil {
		_ = repo.Close()
		return nil, nil, wrapCLIError("token.error.create_event_store", err)
	}

	eventReader := sqlite.NewTokenEventReader(eventStore)

	// The CLI must subscribe the audit handler — without a subscriber the
	// SyncEventBus dropped every token event, so CLI mint/transfer/burn/etc.
	// persisted nothing and `token history` was always empty (the v1.73
	// ISS-080 fix wired only the API server; TASK-113, ISS-105).
	//
	// The handler is wired with the event store as its durable outbox
	// (TASK-119, ISS-111): a transiently-failed publish parks the event in
	// pending_events instead of dropping it, and the single DrainOnce below
	// (before the handles close) heals anything the command's own publishes
	// parked — a CLI invocation is short-lived, so a background goroutine is
	// the wrong shape here.
	eventBus := infraevents.NewSyncEventBus()
	eventBus.SubscribeAll(infraevents.NewAuditHandlerWithOutbox(eventStore, eventStore).Handle)

	replay, err := infraevents.NewSQLiteReplayProtection(dbPath)
	if err != nil {
		_ = eventStore.Close()
		_ = repo.Close()
		return nil, nil, wrapCLIError("token.error.create_replay_protection", err)
	}

	chain, err := blockchain.InitWithError()
	if err != nil {
		_ = replay.Close()
		_ = eventStore.Close()
		_ = repo.Close()
		return nil, nil, wrapCLIError("token.error.init_blockchain", err)
	}

	txManager := sqlite.NewTxManager(repo.GetDB())
	service := token.NewService(repo, txManager, eventBus, eventReader, replay, chain)
	cleanup := func() {
		// Deliver any audit events this command's publishes parked in the
		// durable outbox (a transient DB hiccup mid-command). The CLI has no
		// background drainer; one synchronous pass before closing the handles
		// is the right shape (TASK-119, ISS-111).
		if _, err := infraevents.NewOutboxDrainer(eventStore, nil).DrainOnce(); err != nil {
			logger.Error().Err(err).Msg("audit outbox drain failed before closing event store")
		}
		_ = replay.Close()
		_ = eventStore.Close()
		_ = repo.Close()
	}

	return service, cleanup, nil
}

var tokenCmd = &cobra.Command{
	Use:   "token",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("token.cmd"),
	Long:  i18n.GetText("token.cmd"),
}

func init() {
	rootCmd.AddCommand(tokenCmd)

	tokenCmd.AddCommand(tokenCreateCmd)
	tokenCmd.AddCommand(tokenMintCmd)
	tokenCmd.AddCommand(tokenTransferCmd)
	tokenCmd.AddCommand(tokenApproveCmd)
	tokenCmd.AddCommand(tokenTransferFromCmd)
	tokenCmd.AddCommand(tokenBurnCmd)
	tokenCmd.AddCommand(tokenBalanceCmd)
	tokenCmd.AddCommand(tokenAllowanceCmd)
	tokenCmd.AddCommand(tokenHistoryCmd)
	tokenCmd.AddCommand(tokenInfoCmd)
	tokenCmd.AddCommand(tokenListCmd)
	addJSONFlag(tokenListCmd)

	tokenCreateCmd.Flags().StringP("name", "n", "", i18n.GetText("token.name"))
	tokenCreateCmd.Flags().StringP("symbol", "s", "", i18n.GetText("token.symbol"))
	tokenCreateCmd.Flags().StringP("supply", "t", "", i18n.GetText("token.supply"))
	tokenCreateCmd.Flags().StringP("decimals", "d", "8", i18n.GetText("token.decimals"))
	_ = tokenCreateCmd.MarkFlagRequired("name")
	_ = tokenCreateCmd.MarkFlagRequired("symbol")
	_ = tokenCreateCmd.MarkFlagRequired("supply")

	tokenMintCmd.Flags().StringP("token", "t", "", i18n.GetText("token.token_id"))
	tokenMintCmd.Flags().StringP("to", "o", "", i18n.GetText("token.to"))
	tokenMintCmd.Flags().StringP("amount", "a", "", i18n.GetText("token.amount"))
	tokenMintCmd.Flags().StringP("private-key", "k", "", i18n.GetText("token.private_key"))
	_ = tokenMintCmd.MarkFlagRequired("token")
	_ = tokenMintCmd.MarkFlagRequired("to")
	_ = tokenMintCmd.MarkFlagRequired("amount")
	_ = tokenMintCmd.MarkFlagRequired("private-key")

	tokenTransferCmd.Flags().StringP("token", "t", "", i18n.GetText("token.token_id"))
	tokenTransferCmd.Flags().StringP("from", "f", "", i18n.GetText("token.from"))
	tokenTransferCmd.Flags().StringP("to", "o", "", i18n.GetText("token.to"))
	tokenTransferCmd.Flags().StringP("amount", "a", "", i18n.GetText("token.amount"))
	tokenTransferCmd.Flags().StringP("private-key", "k", "", i18n.GetText("token.private_key"))
	_ = tokenTransferCmd.MarkFlagRequired("token")
	_ = tokenTransferCmd.MarkFlagRequired("from")
	_ = tokenTransferCmd.MarkFlagRequired("to")
	_ = tokenTransferCmd.MarkFlagRequired("amount")
	_ = tokenTransferCmd.MarkFlagRequired("private-key")

	tokenApproveCmd.Flags().StringP("token", "t", "", i18n.GetText("token.token_id"))
	tokenApproveCmd.Flags().StringP("owner", "o", "", i18n.GetText("token.owner"))
	tokenApproveCmd.Flags().StringP("spender", "s", "", i18n.GetText("token.spender"))
	tokenApproveCmd.Flags().StringP("amount", "a", "", i18n.GetText("token.amount"))
	tokenApproveCmd.Flags().StringP("private-key", "k", "", i18n.GetText("token.private_key"))
	_ = tokenApproveCmd.MarkFlagRequired("token")
	_ = tokenApproveCmd.MarkFlagRequired("owner")
	_ = tokenApproveCmd.MarkFlagRequired("spender")
	_ = tokenApproveCmd.MarkFlagRequired("amount")
	_ = tokenApproveCmd.MarkFlagRequired("private-key")

	tokenTransferFromCmd.Flags().StringP("token", "t", "", i18n.GetText("token.token_id"))
	tokenTransferFromCmd.Flags().StringP("owner", "o", "", i18n.GetText("token.owner"))
	tokenTransferFromCmd.Flags().String("to", "", i18n.GetText("token.to"))
	tokenTransferFromCmd.Flags().StringP("amount", "a", "", i18n.GetText("token.amount"))
	tokenTransferFromCmd.Flags().StringP("spender", "s", "", i18n.GetText("token.spender"))
	// --private-key (not --spender-key) so the Base64 signer secret is named
	// like every other signing command (TASK-152, ISS-141).
	tokenTransferFromCmd.Flags().StringP("private-key", "k", "", i18n.GetText("token.private_key"))
	_ = tokenTransferFromCmd.MarkFlagRequired("token")
	_ = tokenTransferFromCmd.MarkFlagRequired("owner")
	_ = tokenTransferFromCmd.MarkFlagRequired("to")
	_ = tokenTransferFromCmd.MarkFlagRequired("amount")
	_ = tokenTransferFromCmd.MarkFlagRequired("spender")
	_ = tokenTransferFromCmd.MarkFlagRequired("private-key")

	tokenBurnCmd.Flags().StringP("token", "t", "", i18n.GetText("token.token_id"))
	tokenBurnCmd.Flags().StringP("from", "f", "", i18n.GetText("token.from"))
	tokenBurnCmd.Flags().StringP("amount", "a", "", i18n.GetText("token.amount"))
	tokenBurnCmd.Flags().StringP("private-key", "k", "", i18n.GetText("token.private_key"))
	addConfirmFlag(tokenBurnCmd, "cli.confirm.token_burn")
	_ = tokenBurnCmd.MarkFlagRequired("token")
	_ = tokenBurnCmd.MarkFlagRequired("from")
	_ = tokenBurnCmd.MarkFlagRequired("amount")
	_ = tokenBurnCmd.MarkFlagRequired("private-key")

	tokenBalanceCmd.Flags().StringP("token", "t", "", i18n.GetText("token.token_id"))
	tokenBalanceCmd.Flags().StringP("owner", "o", "", i18n.GetText("token.owner"))
	_ = tokenBalanceCmd.MarkFlagRequired("token")
	_ = tokenBalanceCmd.MarkFlagRequired("owner")

	tokenAllowanceCmd.Flags().StringP("token", "t", "", i18n.GetText("token.token_id"))
	tokenAllowanceCmd.Flags().StringP("owner", "o", "", i18n.GetText("token.owner"))
	tokenAllowanceCmd.Flags().StringP("spender", "s", "", i18n.GetText("token.spender"))
	_ = tokenAllowanceCmd.MarkFlagRequired("token")
	_ = tokenAllowanceCmd.MarkFlagRequired("owner")
	_ = tokenAllowanceCmd.MarkFlagRequired("spender")

	tokenHistoryCmd.Flags().StringP("token", "t", "", i18n.GetText("token.token_id"))
	tokenHistoryCmd.Flags().StringP("owner", "o", "", i18n.GetText("token.owner"))
	tokenHistoryCmd.Flags().IntP("limit", "l", 50, i18n.GetText("token.limit"))
	tokenHistoryCmd.Flags().IntP("offset", "p", 0, i18n.GetText("token.offset"))
	_ = tokenHistoryCmd.MarkFlagRequired("token")

	tokenInfoCmd.Flags().StringP("token", "t", "", i18n.GetText("token.token_id"))
	_ = tokenInfoCmd.MarkFlagRequired("token")

	tokenListCmd.Flags().IntP("limit", "l", 0, i18n.GetText("token.list_limit"))
	tokenListCmd.Flags().IntP("offset", "p", 0, i18n.GetText("token.offset"))
}

var tokenCreateCmd = &cobra.Command{
	Use:   "create",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("token.create.cmd"),
	Example: `  aurora token create -n "MyToken" -s "MTK" --supply 1000000
  aurora token create -n "GamingCoin" -s "GAME" --supply 1000000000`,
	RunE: func(cmd *cobra.Command, args []string) error {
		service, cleanup, err := newTokenService()
		if err != nil {
			return err
		}
		defer cleanup()

		name, _ := cmd.Flags().GetString("name")
		symbol, _ := cmd.Flags().GetString("symbol")
		supply, _ := cmd.Flags().GetString("supply")
		decimalsFlag, _ := cmd.Flags().GetString("decimals")
		decimals, err := parseTokenDecimals(decimalsFlag)
		if err != nil {
			return err
		}

		totalSupply, err := token.NewAmountFromString(supply)
		if err != nil {
			return wrapCLIError("token.error.invalid_supply", err)
		}

		pub, priv, err := ed25519.GenerateKey(nil)
		if err != nil {
			return wrapCLIError("token.error.generate_key", err)
		}

		tk, err := service.CreateToken(&token.CreateTokenRequest{
			Name:        name,
			Symbol:      symbol,
			TotalSupply: totalSupply,
			Owner:       token.PublicKey(pub),
			Decimals:    &decimals,
		})
		if err != nil {
			return wrapCLIError("token.error.create", err)
		}

		fmt.Printf(i18n.GetText("token.created"), tk.ID(), tk.Name(), tk.Symbol())
		printLocalizedField("token.owner_public_key", b64Encode(pub))
		fmt.Printf("   %s\n", i18n.GetTextF("token.owner_private_key_saved", b64Encode(priv)))
		return nil
	},
}

var tokenListCmd = &cobra.Command{
	Use:   "list",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("token.list.cmd"),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, cleanup, err := newTokenService()
		if err != nil {
			return err
		}
		defer cleanup()

		limit, offset, err := pagingFlags(cmd)
		if err != nil {
			return err
		}

		tokens, err := service.ListTokens(limit, offset)
		if err != nil {
			return wrapCLIError("token.error.list", err)
		}
		if jsonRequested(cmd) {
			out := make([]tokenJSON, 0, len(tokens))
			for _, tk := range tokens {
				out = append(out, tokenJSON{
					ID:          string(tk.ID()),
					Name:        tk.Name(),
					Symbol:      tk.Symbol(),
					TotalSupply: tk.TotalSupply().String(),
					Decimals:    tk.Decimals(),
					Owner:       b64Encode(tk.Owner()),
				})
			}
			return printJSON(out)
		}
		if len(tokens) == 0 {
			fmt.Println(i18n.GetText("token.no_tokens"))
			return nil
		}

		if limit > 0 {
			fmt.Printf("\n🪙 %s: %s\n\n",
				i18n.GetText("token.list_page"),
				i18n.GetTextF("common.page_with_offset", len(tokens), offset))
		} else {
			fmt.Printf("\n🪙 %s: %d\n\n", i18n.GetText("token.total"), len(tokens))
		}
		for _, tk := range tokens {
			fmt.Println("---")
			printLocalizedField("common.id", string(tk.ID()))
			printLocalizedField("common.name", tk.Name())
			printLocalizedField("common.symbol", tk.Symbol())
			printLocalizedField("common.total_supply", tk.TotalSupply().String())
			fmt.Printf("   %s: %d\n", i18n.GetText("common.decimals"), tk.Decimals())
			printLocalizedField("common.owner", b64Encode(tk.Owner()))
		}
		return nil
	},
}

var tokenInfoCmd = &cobra.Command{
	Use:   "info",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("token.info.cmd"),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, cleanup, err := newTokenService()
		if err != nil {
			return err
		}
		defer cleanup()

		tokenID, _ := cmd.Flags().GetString("token")

		tk, err := service.GetTokenInfo(token.TokenID(tokenID))
		if err != nil {
			return wrapCLIError("token.error.get_info", err)
		}
		if tk == nil {
			return newCLIError("token.not_found")
		}

		fmt.Printf("\n🪙 %s:\n", i18n.GetText("token.info_heading"))
		printLocalizedField("common.id", string(tk.ID()))
		printLocalizedField("common.name", tk.Name())
		printLocalizedField("common.symbol", tk.Symbol())
		printLocalizedField("common.total_supply", tk.TotalSupply().String())
		fmt.Printf("   %s: %d\n", i18n.GetText("common.decimals"), tk.Decimals())
		printLocalizedField("common.owner", b64Encode(tk.Owner()))
		fmt.Printf("   %s: %s\n", i18n.GetText("token.mintable"), localizedBool(tk.IsMintable()))
		fmt.Printf("   %s: %s\n", i18n.GetText("token.burnable"), localizedBool(tk.IsBurnable()))
		return nil
	},
}

func localizedBool(value bool) string {
	if value {
		return i18n.GetText("common.true")
	}
	return i18n.GetText("common.false")
}

func b64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// parseTokenDecimals parses the create --decimals string (default "8" when
// the flag is unset) into the domain's int8 range. Garbage, negatives, and
// values beyond int8 are explicit usage errors instead of being silently
// ignored — the previous behavior where every value was dropped and the
// token always got the default (TASK-099, ISS-091).
func parseTokenDecimals(s string) (int8, error) {
	d, err := strconv.Atoi(s)
	if err != nil {
		return 0, newCLIError("token.error.invalid_decimals_integer", s)
	}
	if d < 0 || d > math.MaxInt8 {
		return 0, newCLIError("token.error.invalid_decimals_range", d, math.MaxInt8)
	}
	return int8(d), nil
}

// tokenJSON mirrors the REST token list so a script parses one shape whether it
// called the CLI or the API.
type tokenJSON struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	TotalSupply string `json:"total_supply"`
	Decimals    int8   `json:"decimals"`
	Owner       string `json:"owner"`
}
