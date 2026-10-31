// The ballot: casting one signed vote in one session.
package cmd

import (
	"fmt"

	"github.com/pplmx/aurora/internal/app/votingapp"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	"github.com/spf13/cobra"
)

var voteCmd = &cobra.Command{
	Use:   "vote",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("voting.vote"),
	RunE: func(cmd *cobra.Command, args []string) error {
		voterPK, _ := cmd.Flags().GetString("voter")
		candidateID, _ := cmd.Flags().GetString("candidate")
		privKey, _ := cmd.Flags().GetString("private-key")
		sessionID, _ := cmd.Flags().GetString("session")

		repo, err := getVotingRepo()
		if err != nil {
			return wrapCLIError("voting.error.get_repo", err)
		}

		txManager, err := getVotingTxManager()
		if err != nil {
			return wrapCLIError("voting.error.get_tx_manager", err)
		}

		service := getVotingService()
		chain, err := blockchain.InitWithError()
		if err != nil {
			return wrapCLIError("voting.error.init_blockchain", err)
		}

		req := votingapp.CastVoteRequest{
			VoterPublicKey: voterPK,
			CandidateID:    candidateID,
			PrivateKey:     privKey,
			SessionID:      sessionID,
		}
		uc := votingapp.NewCastVoteUseCase(repo, service, txManager)
		uc.SetChain(chain)
		record, err := uc.Execute(req)
		if err != nil {
			return wrapCLIError("voting.error.cast_vote", err)
		}
		if jsonRequested(cmd) {
			return printJSON(record)
		}

		fmt.Println("✅ " + i18n.GetText("voting.vote_cast"))
		fmt.Printf("   %s\n", i18n.Label(i18n.GetText("voting.vote_id"), record.ID))
		fmt.Printf("   %s\n", i18n.Label(i18n.GetText("common.block_height"), record.BlockHeight))
		return nil
	},
}
