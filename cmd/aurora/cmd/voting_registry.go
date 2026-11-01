// The standing registries a session draws on: candidates and voters.
package cmd

import (
	"fmt"

	"github.com/pplmx/aurora/internal/app/votingapp"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/spf13/cobra"
)

var candidateCmd = &cobra.Command{
	Use:   "candidate",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("voting.candidate.cmd"),
}

var candidateAddCmd = &cobra.Command{
	Use:   "add",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("voting.candidate.add"),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		party, _ := cmd.Flags().GetString("party")
		program, _ := cmd.Flags().GetString("program")

		repo, err := getVotingRepo()
		if err != nil {
			return wrapCLIError("voting.error.get_repo", err)
		}

		req := votingapp.RegisterCandidateRequest{
			Name:    name,
			Party:   party,
			Program: program,
		}
		uc := votingapp.NewRegisterCandidateUseCase(repo)
		cand, err := uc.Execute(req)
		if err != nil {
			return wrapCLIError("voting.error.register_candidate", err)
		}
		if jsonRequested(cmd) {
			return printJSON(cand)
		}

		fmt.Printf("✅ "+i18n.GetText("voting.candidate_added")+"\n", cand.Name)
		printFields(
			field{i18n.GetText("common.id"), cand.ID},
			field{i18n.GetText("voting.party_label"), cand.Party},
		)
		return nil
	},
}

var candidateListCmd = &cobra.Command{
	Use:   "list",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("voting.candidate.list"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := getVotingRepo()
		if err != nil {
			return wrapCLIError("voting.error.get_repo", err)
		}

		limit, offset, err := pagingFlags(cmd)
		if err != nil {
			return err
		}

		uc := votingapp.NewGetCandidatesUseCase(repo)
		list, err := uc.Execute(&votingapp.ListCandidatesRequest{Limit: limit, Offset: offset})
		if err != nil {
			return wrapCLIError("voting.error.list_candidates", err)
		}
		if jsonRequested(cmd) {
			if list == nil {
				list = []*votingapp.CandidateResponse{}
			}
			return printJSON(list)
		}

		// An offset alone asks for a page too, so it takes the paged heading:
		// without it, `--offset 9` on a non-empty store read as an empty one.
		if limit > 0 || offset > 0 {
			fmt.Printf("\n📋 %s\n", i18n.Label(i18n.GetText("voting.candidates_page"), i18n.GetTextF("common.page_with_offset", len(list), offset)))
		} else {
			fmt.Printf("\n📋 %s\n", i18n.Label(i18n.GetText("voting.candidates_total"), len(list)))
		}
		if len(list) == 0 {
			// Three different answers, as Voters gives them: an empty store, a
			// page past the end, and a page the offset skipped past.
			if limit > 0 || offset > 0 {
				fmt.Println("   " + i18n.GetText("common.page_empty"))
				return nil
			}
			fmt.Println("   " + i18n.GetText("common.none"))
		}
		for _, c := range list {
			fmt.Printf("   - %s [%s] - %s\n", c.Name, c.Party, voteCountText(c.VoteCount))
			fmt.Printf("     %s\n", i18n.Label(i18n.GetText("voting.id"), c.ID))
		}
		return nil
	},
}

var voterCmd = &cobra.Command{
	Use:   "voter",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("voting.voter.cmd"),
}

var voterRegisterCmd = &cobra.Command{
	Use:   "register",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("voting.voter.register"),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")

		repo, err := getVotingRepo()
		if err != nil {
			return wrapCLIError("voting.error.get_repo", err)
		}

		req := votingapp.RegisterVoterRequest{
			Name: name,
		}
		uc := votingapp.NewRegisterVoterUseCase(repo)
		resp, err := uc.Execute(req)
		if err != nil {
			return wrapCLIError("voting.error.register_voter", err)
		}

		fmt.Println("✅ " + i18n.GetText("voting.voter_registered"))
		fmt.Printf("\n📣 %s\n   %s\n", i18n.GetText("voting.public_key_share"), resp.PublicKey)
		fmt.Printf("\n🔐 %s\n   %s\n", i18n.GetText("voting.private_key_save"), resp.PrivateKey)
		return nil
	},
}

var voterListCmd = &cobra.Command{
	Use:   "list",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("voting.voter.list"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := getVotingRepo()
		if err != nil {
			return wrapCLIError("voting.error.get_repo", err)
		}

		limit, offset, err := pagingFlags(cmd)
		if err != nil {
			return err
		}

		list, err := votingapp.NewListVotersUseCase(repo).Execute(&votingapp.ListVotersRequest{
			Limit: limit, Offset: offset,
		})
		if err != nil {
			return wrapCLIError("voting.error.list_voters", err)
		}
		if jsonRequested(cmd) {
			if list == nil {
				list = []*votingapp.VoterSummaryResponse{}
			}
			return printJSON(list)
		}

		if limit > 0 || offset > 0 {
			fmt.Printf("\n👥 %s\n", i18n.Label(i18n.GetText("voting.page_total"), i18n.GetTextF("common.page_with_offset", len(list), offset)))
		} else {
			fmt.Printf("\n👥 %s\n", i18n.Label(i18n.GetText("voting.voters_total"), len(list)))
		}
		if len(list) == 0 {
			// Three different answers: an empty store, a page past the end, and
			// a page the offset skipped past. The paged case used to print its
			// heading and nothing else, which reads as a command that failed
			// silently rather than as a page with no rows.
			if limit > 0 || offset > 0 {
				fmt.Println("   " + i18n.GetText("common.page_empty"))
				return nil
			}
			fmt.Println("   " + i18n.GetText("common.none"))
		}
		for _, v := range list {
			status := "✅ " + i18n.GetText("voting.voted")
			if !v.HasVoted {
				status = "⏳ " + i18n.GetText("voting.not_voted")
			}
			fmt.Printf("   - %s [%s]\n", v.Name, status)
			if len(v.PublicKey) > 16 {
				fmt.Printf("     %s\n", i18n.Label(i18n.GetText("voting.public_key_label"), fmt.Sprintf("%s...", v.PublicKey[:16])))
			} else {
				printField("  "+i18n.GetText("voting.public_key_label"), v.PublicKey)
			}
		}
		return nil
	},
}
