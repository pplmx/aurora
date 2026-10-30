package cmd

import (
	"fmt"

	"github.com/pplmx/aurora/internal/app/tokenapp"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/spf13/cobra"
)

var tokenMintCmd = &cobra.Command{
	Use:   "mint",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("token.mint.cmd"),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, cleanup, err := newTokenService()
		if err != nil {
			return err
		}
		defer cleanup()

		uc := tokenapp.NewMintUseCase(service)

		tokenID, _ := cmd.Flags().GetString("token")
		to, _ := cmd.Flags().GetString("to")
		amount, _ := cmd.Flags().GetString("amount")
		privKey, _ := cmd.Flags().GetString("private-key")

		req := &tokenapp.MintRequest{
			TokenID:    tokenID,
			To:         to,
			Amount:     amount,
			PrivateKey: privKey,
		}

		resp, err := uc.Execute(req)
		if err != nil {
			return wrapCLIError("token.error.mint", err)
		}
		if jsonRequested(cmd) {
			return printJSON(resp)
		}

		fmt.Println("✅ " + i18n.GetText("token.minted"))
		printLocalizedField("common.id", resp.ID)
		printLocalizedField("common.to", truncateBase64(resp.To))
		printLocalizedField("common.amount", resp.Amount)
		return nil
	},
}

var tokenTransferCmd = &cobra.Command{
	Use:   "transfer",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("token.transfer.cmd"),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, cleanup, err := newTokenService()
		if err != nil {
			return err
		}
		defer cleanup()

		uc := tokenapp.NewTransferUseCase(service)

		tokenID, _ := cmd.Flags().GetString("token")
		from, _ := cmd.Flags().GetString("from")
		to, _ := cmd.Flags().GetString("to")
		amount, _ := cmd.Flags().GetString("amount")
		privKey, _ := cmd.Flags().GetString("private-key")

		req := &tokenapp.TransferRequest{
			TokenID:    tokenID,
			From:       from,
			To:         to,
			Amount:     amount,
			PrivateKey: privKey,
		}

		resp, err := uc.Execute(req)
		if err != nil {
			return wrapCLIError("token.error.transfer", err)
		}
		if jsonRequested(cmd) {
			return printJSON(resp)
		}

		fmt.Println("✅ " + i18n.GetText("token.transferred"))
		printLocalizedField("common.id", resp.ID)
		printLocalizedField("common.from", truncateBase64(resp.From))
		printLocalizedField("common.to", truncateBase64(resp.To))
		printLocalizedField("common.amount", resp.Amount)
		return nil
	},
}

var tokenApproveCmd = &cobra.Command{
	Use:   "approve",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("token.approve.cmd"),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, cleanup, err := newTokenService()
		if err != nil {
			return err
		}
		defer cleanup()

		uc := tokenapp.NewApproveUseCase(service)

		tokenID, _ := cmd.Flags().GetString("token")
		owner, _ := cmd.Flags().GetString("owner")
		spender, _ := cmd.Flags().GetString("spender")
		amount, _ := cmd.Flags().GetString("amount")
		privKey, _ := cmd.Flags().GetString("private-key")

		req := &tokenapp.ApproveRequest{
			TokenID:    tokenID,
			Owner:      owner,
			Spender:    spender,
			Amount:     amount,
			PrivateKey: privKey,
		}

		resp, err := uc.Execute(req)
		if err != nil {
			return wrapCLIError("token.error.approve", err)
		}
		if jsonRequested(cmd) {
			return printJSON(resp)
		}

		fmt.Println("✅ " + i18n.GetText("token.approved"))
		printLocalizedField("common.id", resp.ID)
		printLocalizedField("common.owner", truncateBase64(resp.Owner))
		printLocalizedField("common.spender", truncateBase64(resp.Spender))
		printLocalizedField("common.amount", resp.Amount)
		return nil
	},
}

var tokenTransferFromCmd = &cobra.Command{
	Use:   "transfer-from",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("token.transfer_from.cmd"),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, cleanup, err := newTokenService()
		if err != nil {
			return err
		}
		defer cleanup()

		uc := tokenapp.NewTransferFromUseCase(service)

		tokenID, _ := cmd.Flags().GetString("token")
		owner, _ := cmd.Flags().GetString("owner")
		to, _ := cmd.Flags().GetString("to")
		amount, _ := cmd.Flags().GetString("amount")
		spender, _ := cmd.Flags().GetString("spender")
		privateKey, _ := cmd.Flags().GetString("private-key")

		req := &tokenapp.TransferFromRequest{
			TokenID:    tokenID,
			Owner:      owner,
			To:         to,
			Amount:     amount,
			Spender:    spender,
			SpenderKey: privateKey,
		}

		resp, err := uc.Execute(req)
		if err != nil {
			return wrapCLIError("token.error.transfer_from", err)
		}
		if jsonRequested(cmd) {
			return printJSON(resp)
		}

		fmt.Println("✅ " + i18n.GetText("token.transferred"))
		printLocalizedField("common.id", resp.ID)
		printLocalizedField("common.owner", truncateBase64(resp.From))
		printLocalizedField("common.to", truncateBase64(resp.To))
		printLocalizedField("common.spender", truncateBase64(req.Spender))
		printLocalizedField("common.amount", resp.Amount)
		return nil
	},
}

var tokenBurnCmd = &cobra.Command{
	Use:   "burn",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("token.burn.cmd"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Burn permanently destroys tokens; require -y/--confirm first
		// (the same gate every destructive command uses).
		if err := requireConfirm(cmd, "cli.confirm.token_burn"); err != nil {
			return err
		}
		service, cleanup, err := newTokenService()
		if err != nil {
			return err
		}
		defer cleanup()

		uc := tokenapp.NewBurnUseCase(service)

		tokenID, _ := cmd.Flags().GetString("token")
		from, _ := cmd.Flags().GetString("from")
		amount, _ := cmd.Flags().GetString("amount")
		privKey, _ := cmd.Flags().GetString("private-key")

		req := &tokenapp.BurnRequest{
			TokenID:    tokenID,
			From:       from,
			Amount:     amount,
			PrivateKey: privKey,
		}

		resp, err := uc.Execute(req)
		if err != nil {
			return wrapCLIError("token.error.burn", err)
		}
		if jsonRequested(cmd) {
			return printJSON(resp)
		}

		fmt.Println("✅ " + i18n.GetText("token.burned"))
		printLocalizedField("common.id", resp.ID)
		printLocalizedField("common.from", truncateBase64(resp.From))
		printLocalizedField("common.amount", resp.Amount)
		return nil
	},
}
