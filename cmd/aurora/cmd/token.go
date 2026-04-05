package cmd

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"

	tokent "github.com/pplmx/aurora/internal/app/token"
	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/domain/token"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: i18n.GetText("token.cmd"),
	Long:  i18n.GetText("token.cmd"),
}

func init() {
	rootCmd.AddCommand(tokenCmd)

	tokenCmd.AddCommand(tokenCreateCmd)
	tokenCmd.AddCommand(tokenMintCmd)
	tokenCmd.AddCommand(tokenTransferCmd)
	tokenCmd.AddCommand(tokenApproveCmd)
	tokenCmd.AddCommand(tokenBurnCmd)
	tokenCmd.AddCommand(tokenBalanceCmd)
	tokenCmd.AddCommand(tokenAllowanceCmd)
	tokenCmd.AddCommand(tokenHistoryCmd)
	tokenCmd.AddCommand(tokenInfoCmd)
	tokenCmd.AddCommand(tokenTuiCmd)

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

	tokenBurnCmd.Flags().StringP("token", "t", "", i18n.GetText("token.token_id"))
	tokenBurnCmd.Flags().StringP("from", "f", "", i18n.GetText("token.from"))
	tokenBurnCmd.Flags().StringP("amount", "a", "", i18n.GetText("token.amount"))
	tokenBurnCmd.Flags().StringP("private-key", "k", "", i18n.GetText("token.private_key"))
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
	tokenHistoryCmd.Flags().IntP("limit", "l", 50, "Limit results")
	_ = tokenHistoryCmd.MarkFlagRequired("token")
	_ = tokenHistoryCmd.MarkFlagRequired("owner")

	tokenInfoCmd.Flags().StringP("token", "t", "", i18n.GetText("token.token_id"))
	_ = tokenInfoCmd.MarkFlagRequired("token")
}

var tokenCreateCmd = &cobra.Command{
	Use:   "create",
	Short: i18n.GetText("token.create.cmd"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := sqlite.NewTokenRepository(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to create repo: %w", err)
		}
		defer func() { _ = repo.Close() }()

		chain := blockchain.InitBlockChain()
		eventStore, err := sqlite.NewTokenEventStore(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to create event store: %w", err)
		}
		defer func() { _ = eventStore.Close() }()

		service := token.NewService(repo, eventStore, chain)

		name, _ := cmd.Flags().GetString("name")
		symbol, _ := cmd.Flags().GetString("symbol")
		supply, _ := cmd.Flags().GetString("supply")

		totalSupply, err := token.NewAmountFromString(supply)
		if err != nil {
			return fmt.Errorf("invalid supply: %w", err)
		}

		pub, priv, err := ed25519.GenerateKey(nil)
		if err != nil {
			return fmt.Errorf("failed to generate key: %w", err)
		}

		tk, err := service.CreateToken(&token.CreateTokenRequest{
			Name:        name,
			Symbol:      symbol,
			TotalSupply: totalSupply,
			Owner:       token.PublicKey(pub),
		})
		if err != nil {
			return fmt.Errorf("failed to create token: %w", err)
		}

		fmt.Printf(i18n.GetText("token.created"), tk.ID(), tk.Name(), tk.Symbol())
		fmt.Printf("   Owner Public Key: %s\n", b64Encode(pub))
		fmt.Printf("   Owner Private Key: %s (SAVE THIS!)\n", b64Encode(priv))
		return nil
	},
}

var tokenMintCmd = &cobra.Command{
	Use:   "mint",
	Short: i18n.GetText("token.mint.cmd"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := sqlite.NewTokenRepository(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to create repo: %w", err)
		}
		defer func() { _ = repo.Close() }()

		chain := blockchain.InitBlockChain()
		eventStore, err := sqlite.NewTokenEventStore(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to create event store: %w", err)
		}
		defer func() { _ = eventStore.Close() }()

		service := token.NewService(repo, eventStore, chain)
		uc := tokent.NewMintUseCase(service)

		tokenID, _ := cmd.Flags().GetString("token")
		to, _ := cmd.Flags().GetString("to")
		amount, _ := cmd.Flags().GetString("amount")
		privKey, _ := cmd.Flags().GetString("private-key")

		req := &tokent.MintRequest{
			TokenID:    tokenID,
			To:         to,
			Amount:     amount,
			PrivateKey: privKey,
		}

		resp, err := uc.Execute(req)
		if err != nil {
			return fmt.Errorf("failed to mint: %w", err)
		}

		fmt.Println("✅ " + i18n.GetText("token.minted"))
		fmt.Printf("   ID: %s\n", resp.ID)
		fmt.Printf("   To: %s\n", truncateBase64(resp.To))
		fmt.Printf("   Amount: %s\n", resp.Amount)
		return nil
	},
}

var tokenTransferCmd = &cobra.Command{
	Use:   "transfer",
	Short: i18n.GetText("token.transfer.cmd"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := sqlite.NewTokenRepository(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to create repo: %w", err)
		}
		defer func() { _ = repo.Close() }()

		chain := blockchain.InitBlockChain()
		eventStore, err := sqlite.NewTokenEventStore(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to create event store: %w", err)
		}
		defer func() { _ = eventStore.Close() }()

		service := token.NewService(repo, eventStore, chain)
		uc := tokent.NewTransferUseCase(service)

		tokenID, _ := cmd.Flags().GetString("token")
		from, _ := cmd.Flags().GetString("from")
		to, _ := cmd.Flags().GetString("to")
		amount, _ := cmd.Flags().GetString("amount")
		privKey, _ := cmd.Flags().GetString("private-key")

		req := &tokent.TransferRequest{
			TokenID:    tokenID,
			From:       from,
			To:         to,
			Amount:     amount,
			PrivateKey: privKey,
		}

		resp, err := uc.Execute(req)
		if err != nil {
			return fmt.Errorf("failed to transfer: %w", err)
		}

		fmt.Println("✅ " + i18n.GetText("token.transferred"))
		fmt.Printf("   ID: %s\n", resp.ID)
		fmt.Printf("   From: %s\n", truncateBase64(resp.From))
		fmt.Printf("   To: %s\n", truncateBase64(resp.To))
		fmt.Printf("   Amount: %s\n", resp.Amount)
		return nil
	},
}

var tokenApproveCmd = &cobra.Command{
	Use:   "approve",
	Short: i18n.GetText("token.approve.cmd"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := sqlite.NewTokenRepository(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to create repo: %w", err)
		}
		defer func() { _ = repo.Close() }()

		chain := blockchain.InitBlockChain()
		eventStore, err := sqlite.NewTokenEventStore(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to create event store: %w", err)
		}
		defer func() { _ = eventStore.Close() }()

		service := token.NewService(repo, eventStore, chain)
		uc := tokent.NewApproveUseCase(service)

		tokenID, _ := cmd.Flags().GetString("token")
		owner, _ := cmd.Flags().GetString("owner")
		spender, _ := cmd.Flags().GetString("spender")
		amount, _ := cmd.Flags().GetString("amount")
		privKey, _ := cmd.Flags().GetString("private-key")

		req := &tokent.ApproveRequest{
			TokenID:    tokenID,
			Owner:      owner,
			Spender:    spender,
			Amount:     amount,
			PrivateKey: privKey,
		}

		resp, err := uc.Execute(req)
		if err != nil {
			return fmt.Errorf("failed to approve: %w", err)
		}

		fmt.Println("✅ " + i18n.GetText("token.approved"))
		fmt.Printf("   ID: %s\n", resp.ID)
		fmt.Printf("   Owner: %s\n", truncateBase64(resp.Owner))
		fmt.Printf("   Spender: %s\n", truncateBase64(resp.Spender))
		fmt.Printf("   Amount: %s\n", resp.Amount)
		return nil
	},
}

var tokenBurnCmd = &cobra.Command{
	Use:   "burn",
	Short: i18n.GetText("token.burn.cmd"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := sqlite.NewTokenRepository(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to create repo: %w", err)
		}
		defer func() { _ = repo.Close() }()

		chain := blockchain.InitBlockChain()
		eventStore, err := sqlite.NewTokenEventStore(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to create event store: %w", err)
		}
		defer func() { _ = eventStore.Close() }()

		service := token.NewService(repo, eventStore, chain)
		uc := tokent.NewBurnUseCase(service)

		tokenID, _ := cmd.Flags().GetString("token")
		from, _ := cmd.Flags().GetString("from")
		amount, _ := cmd.Flags().GetString("amount")
		privKey, _ := cmd.Flags().GetString("private-key")

		req := &tokent.BurnRequest{
			TokenID:    tokenID,
			From:       from,
			Amount:     amount,
			PrivateKey: privKey,
		}

		resp, err := uc.Execute(req)
		if err != nil {
			return fmt.Errorf("failed to burn: %w", err)
		}

		fmt.Println("✅ " + i18n.GetText("token.burned"))
		fmt.Printf("   ID: %s\n", resp.ID)
		fmt.Printf("   From: %s\n", truncateBase64(resp.From))
		fmt.Printf("   Amount: %s\n", resp.Amount)
		return nil
	},
}

var tokenBalanceCmd = &cobra.Command{
	Use:   "balance",
	Short: i18n.GetText("token.balance.cmd"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := sqlite.NewTokenRepository(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to create repo: %w", err)
		}
		defer func() { _ = repo.Close() }()

		chain := blockchain.InitBlockChain()
		eventStore, err := sqlite.NewTokenEventStore(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to create event store: %w", err)
		}
		defer func() { _ = eventStore.Close() }()

		service := token.NewService(repo, eventStore, chain)
		uc := tokent.NewGetBalanceUseCase(service)

		tokenID, _ := cmd.Flags().GetString("token")
		owner, _ := cmd.Flags().GetString("owner")

		req := &tokent.BalanceRequest{
			TokenID: tokenID,
			Owner:   owner,
		}

		resp, err := uc.Execute(req)
		if err != nil {
			return fmt.Errorf("failed to get balance: %w", err)
		}

		fmt.Printf("\n💰 Balance:\n")
		fmt.Printf("   Token: %s\n", resp.TokenID)
		fmt.Printf("   Owner: %s\n", truncateBase64(resp.Owner))
		fmt.Printf("   Balance: %s\n", resp.Amount)
		return nil
	},
}

var tokenAllowanceCmd = &cobra.Command{
	Use:   "allowance",
	Short: i18n.GetText("token.allowance.cmd"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := sqlite.NewTokenRepository(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to create repo: %w", err)
		}
		defer func() { _ = repo.Close() }()

		chain := blockchain.InitBlockChain()
		eventStore, err := sqlite.NewTokenEventStore(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to create event store: %w", err)
		}
		defer func() { _ = eventStore.Close() }()

		service := token.NewService(repo, eventStore, chain)
		uc := tokent.NewGetAllowanceUseCase(service)

		tokenID, _ := cmd.Flags().GetString("token")
		owner, _ := cmd.Flags().GetString("owner")
		spender, _ := cmd.Flags().GetString("spender")

		req := &tokent.AllowanceRequest{
			TokenID: tokenID,
			Owner:   owner,
			Spender: spender,
		}

		resp, err := uc.Execute(req)
		if err != nil {
			return fmt.Errorf("failed to get allowance: %w", err)
		}

		fmt.Printf("\n🔑 Allowance:\n")
		fmt.Printf("   Token: %s\n", resp.TokenID)
		fmt.Printf("   Owner: %s\n", truncateBase64(resp.Owner))
		fmt.Printf("   Spender: %s\n", truncateBase64(resp.Spender))
		fmt.Printf("   Allowance: %s\n", resp.Amount)
		return nil
	},
}

var tokenHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: i18n.GetText("token.history.cmd"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := sqlite.NewTokenRepository(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to create repo: %w", err)
		}
		defer func() { _ = repo.Close() }()

		chain := blockchain.InitBlockChain()
		eventStore, err := sqlite.NewTokenEventStore(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to create event store: %w", err)
		}
		defer func() { _ = eventStore.Close() }()

		service := token.NewService(repo, eventStore, chain)
		uc := tokent.NewGetHistoryUseCase(service)

		tokenID, _ := cmd.Flags().GetString("token")
		owner, _ := cmd.Flags().GetString("owner")
		limit, _ := cmd.Flags().GetInt("limit")

		req := &tokent.HistoryRequest{
			TokenID: tokenID,
			Owner:   owner,
			Limit:   limit,
		}

		resp, err := uc.Execute(req)
		if err != nil {
			return fmt.Errorf("failed to get history: %w", err)
		}

		fmt.Printf("\n📜 Transfer History: %d\n", len(resp.Transfers))
		if len(resp.Transfers) == 0 {
			fmt.Println("   " + i18n.GetText("token.no_history"))
		}
		for _, t := range resp.Transfers {
			fmt.Printf("   - From: %s -> To: %s | Amount: %s\n",
				truncateBase64(t.From), truncateBase64(t.To), t.Amount)
		}
		return nil
	},
}

var tokenInfoCmd = &cobra.Command{
	Use:   "info",
	Short: i18n.GetText("token.info.cmd"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := sqlite.NewTokenRepository(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to create repo: %w", err)
		}
		defer func() { _ = repo.Close() }()

		chain := blockchain.InitBlockChain()
		eventStore, err := sqlite.NewTokenEventStore(blockchain.DBPath())
		if err != nil {
			return fmt.Errorf("failed to create event store: %w", err)
		}
		defer func() { _ = eventStore.Close() }()

		service := token.NewService(repo, eventStore, chain)

		tokenID, _ := cmd.Flags().GetString("token")

		tk, err := service.GetTokenInfo(token.TokenID(tokenID))
		if err != nil {
			return fmt.Errorf("failed to get token info: %w", err)
		}
		if tk == nil {
			return fmt.Errorf("%s", i18n.GetText("token.not_found"))
		}

		fmt.Printf("\n🪙 Token Info:\n")
		fmt.Printf("   ID: %s\n", tk.ID())
		fmt.Printf("   Name: %s\n", tk.Name())
		fmt.Printf("   Symbol: %s\n", tk.Symbol())
		fmt.Printf("   Total Supply: %s\n", tk.TotalSupply().String())
		fmt.Printf("   Decimals: %d\n", tk.Decimals())
		fmt.Printf("   Owner: %s\n", b64Encode(tk.Owner()))
		fmt.Printf("   Mintable: %v\n", tk.IsMintable())
		fmt.Printf("   Burnable: %v\n", tk.IsBurnable())
		return nil
	},
}

var tokenTuiCmd = &cobra.Command{
	Use:   "tui",
	Short: i18n.GetText("token.tui.cmd"),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(i18n.GetText("token.tui.title"))
		fmt.Println("TUI not implemented yet")
		return nil
	},
}

func b64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
