package cmd

import (
	"fmt"

	"github.com/pplmx/aurora/internal/app/tokenapp"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/spf13/cobra"
)

var tokenBalanceCmd = &cobra.Command{
	Use:   "balance",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("token.balance.cmd"),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, cleanup, err := newTokenService()
		if err != nil {
			return err
		}
		defer cleanup()

		uc := tokenapp.NewGetBalanceUseCase(service)

		tokenID, _ := cmd.Flags().GetString("token")
		owner, _ := cmd.Flags().GetString("owner")

		req := &tokenapp.BalanceRequest{
			TokenID: tokenID,
			Owner:   owner,
		}

		resp, err := uc.Execute(req)
		if err != nil {
			return wrapCLIError("token.error.get_balance", err)
		}
		if jsonRequested(cmd) {
			return printJSON(resp)
		}

		fmt.Printf("\n💰 %s:\n", i18n.GetText("token.balance_heading"))
		printLocalizedField("common.token", resp.TokenID)
		printLocalizedField("common.owner", truncateBase64(resp.Owner))
		printLocalizedField("common.balance", resp.Amount)
		return nil
	},
}

var tokenAllowanceCmd = &cobra.Command{
	Use:   "allowance",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("token.allowance.cmd"),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, cleanup, err := newTokenService()
		if err != nil {
			return err
		}
		defer cleanup()

		uc := tokenapp.NewGetAllowanceUseCase(service)

		tokenID, _ := cmd.Flags().GetString("token")
		owner, _ := cmd.Flags().GetString("owner")
		spender, _ := cmd.Flags().GetString("spender")

		req := &tokenapp.AllowanceRequest{
			TokenID: tokenID,
			Owner:   owner,
			Spender: spender,
		}

		resp, err := uc.Execute(req)
		if err != nil {
			return wrapCLIError("token.error.get_allowance", err)
		}
		if jsonRequested(cmd) {
			return printJSON(resp)
		}

		fmt.Printf("\n🔑 %s:\n", i18n.GetText("token.allowance_heading"))
		printLocalizedField("common.token", resp.TokenID)
		printLocalizedField("common.owner", truncateBase64(resp.Owner))
		printLocalizedField("common.spender", truncateBase64(resp.Spender))
		printLocalizedField("common.allowance", resp.Amount)
		return nil
	},
}

var tokenHistoryCmd = &cobra.Command{
	Use:   "history",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("token.history.cmd"),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, cleanup, err := newTokenService()
		if err != nil {
			return err
		}
		defer cleanup()

		uc := tokenapp.NewGetHistoryUseCase(service)

		tokenID, _ := cmd.Flags().GetString("token")
		owner, _ := cmd.Flags().GetString("owner")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		// Mirrors the API handler caps (maxHistoryLimit=100,
		// maxHistoryOffset=1000) so the CLI cannot drive an unbounded scan.
		if limit <= 0 {
			limit = 50
		} else if limit > 100 {
			limit = 100
		}
		if offset < 0 {
			offset = 0
		} else if offset > 1000 {
			offset = 1000
		}

		req := &tokenapp.HistoryRequest{
			TokenID: tokenID,
			Owner:   owner,
			Limit:   limit,
			Offset:  offset,
		}

		transfers, err := uc.Execute(req)
		if err != nil {
			return wrapCLIError("token.error.get_history", err)
		}
		if jsonRequested(cmd) {
			if transfers == nil {
				transfers = []tokenapp.TransferResponse{}
			}
			return printJSON(transfers)
		}

		fmt.Printf("\n📜 %s: %d\n", i18n.GetText("token.history_heading"), len(transfers))
		if len(transfers) == 0 {
			fmt.Println("   " + i18n.GetText("token.no_history"))
		}
		for _, t := range transfers {
			fmt.Printf("   - %s\n", i18n.GetTextF(
				"token.history_entry",
				truncateBase64(t.From),
				truncateBase64(t.To),
				t.Amount,
				t.BlockHeight,
			))
		}
		return nil
	},
}
