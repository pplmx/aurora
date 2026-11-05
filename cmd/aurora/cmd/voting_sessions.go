// Sessions and their tally: opening a governance window, moving it through
// its lifecycle, and reading the votes it accepted.
package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pplmx/aurora/internal/app/votingapp"
	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/spf13/cobra"
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("voting.session.cmd"),
}

var sessionCreateCmd = &cobra.Command{
	Use:     "create",
	Args:    cobra.NoArgs,
	Short:   i18n.GetText("voting.session.create"),
	Example: `  aurora voting session create -t "Election 2026" -d "Annual board election" -c cand-1 -c cand-2 -c cand-3`,
	RunE: func(cmd *cobra.Command, args []string) error {
		title, _ := cmd.Flags().GetString("title")
		description, _ := cmd.Flags().GetString("description")
		candidates, _ := cmd.Flags().GetStringSlice("candidates")

		repo, err := getVotingRepo()
		if err != nil {
			return wrapCLIError("voting.error.get_repo", err)
		}

		startTime, _ := cmd.Flags().GetInt64("start-time")
		endTime, _ := cmd.Flags().GetInt64("end-time")

		req := votingapp.CreateSessionRequest{
			Title:        title,
			Description:  description,
			CandidateIDs: candidates,
			StartTime:    startTime,
			EndTime:      endTime,
		}
		uc := votingapp.NewCreateSessionUseCase(repo)
		session, err := uc.Execute(req)
		if err != nil {
			return wrapCLIError("voting.error.create_session", err)
		}
		if jsonRequested(cmd) {
			return printJSON(session)
		}

		fmt.Printf("✅ "+i18n.GetText("voting.session_created")+"\n", session.Title)
		printFields(
			field{i18n.GetText("common.id"), session.ID},
			field{i18n.GetText("common.status"), localizedVotingStatus(session.Status)},
		)
		return nil
	},
}

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("voting.session.list"),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := getVotingRepo()
		if err != nil {
			return wrapCLIError("voting.error.get_repo", err)
		}

		limit, offset, err := pagingFlags(cmd)
		if err != nil {
			return err
		}

		search, _ := cmd.Flags().GetString("search")
		search = strings.TrimSpace(search)

		uc := votingapp.NewListSessionsUseCase(repo)
		list, err := uc.Execute(&votingapp.ListSessionsRequest{
			Search: search, Limit: limit, Offset: offset,
		})
		if err != nil {
			return wrapCLIError("voting.error.list_sessions", err)
		}
		if jsonRequested(cmd) {
			if list == nil {
				list = []*votingapp.SessionResponse{}
			}
			return printJSON(list)
		}

		// An offset alone asks for a page too, so it takes the paged heading:
		// without it, `--offset 9` on a non-empty store read as an empty one. A
		// search is a fourth answer: "Sessions: 0" beside a filter claimed an
		// empty store when the store may be full.
		switch {
		case limit > 0 || offset > 0:
			fmt.Printf("\n🗳️ %s\n", i18n.Label(i18n.GetText("voting.sessions_page"), i18n.GetTextF("common.page_with_offset", len(list), offset)))
		case search != "":
			fmt.Printf("\n🗳️ %s\n", i18n.Label(i18n.GetText("voting.matches"), len(list)))
		default:
			fmt.Printf("\n🗳️ %s\n", i18n.Label(i18n.GetText("voting.sessions_total"), len(list)))
		}
		if len(list) == 0 {
			switch {
			case limit > 0 || offset > 0:
				fmt.Println("   " + i18n.GetText("common.page_empty"))
			case search != "":
				fmt.Printf("   "+i18n.GetText("voting.session_no_match")+"\n", search)
			default:
				fmt.Println("   " + i18n.GetText("common.none"))
			}
			return nil
		}
		for _, s := range list {
			fmt.Printf("   - %s [%s]\n", s.Title, localizedVotingStatus(s.Status))
			printField("  "+i18n.GetText("common.id"), s.ID)
		}
		return nil
	},
}

var sessionStartCmd = &cobra.Command{
	Use:   "start",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("voting.session.start"),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID, _ := cmd.Flags().GetString("session")

		repo, err := getVotingRepo()
		if err != nil {
			return wrapCLIError("voting.error.get_repo", err)
		}

		session, err := repo.GetSession(sessionID)
		if err != nil {
			return wrapCLIError("voting.error.get_session", err)
		}
		if session == nil {
			return newCLIError("voting.session_not_found")
		}

		session.Status = voting.SessionStatusActive
		if err := repo.UpdateSession(session); err != nil {
			return wrapCLIError("voting.error.start_session", err)
		}
		if jsonRequested(cmd) {
			return printJSON(votingapp.SessionResponseOf(session))
		}

		fmt.Println("✅ " + i18n.GetText("voting.session_started"))
		return nil
	},
}

var sessionEndCmd = &cobra.Command{
	Use:   "end",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("voting.session.end"),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID, _ := cmd.Flags().GetString("session")

		repo, err := getVotingRepo()
		if err != nil {
			return wrapCLIError("voting.error.get_repo", err)
		}

		session, err := repo.GetSession(sessionID)
		if err != nil {
			return wrapCLIError("voting.error.get_session", err)
		}
		if session == nil {
			return newCLIError("voting.session_not_found")
		}

		session.Status = voting.SessionStatusEnded
		if err := repo.UpdateSession(session); err != nil {
			return wrapCLIError("voting.error.end_session", err)
		}
		if jsonRequested(cmd) {
			return printJSON(votingapp.SessionResponseOf(session))
		}

		fmt.Println("✅ " + i18n.GetText("voting.session_ended"))
		return nil
	},
}

var resultsCmd = &cobra.Command{
	Use:   "results",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("voting.results"),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID, _ := cmd.Flags().GetString("session")

		repo, err := getVotingRepo()
		if err != nil {
			return wrapCLIError("voting.error.get_repo", err)
		}

		// The app-layer use case owns the tally (roster order, per-session
		// claims, deleted candidates at zero), so the CLI reports exactly what
		// the REST results endpoint returns instead of rebuilding the map.
		resp, err := votingapp.NewGetResultsUseCase(repo).Execute(sessionID)
		if err != nil {
			if errors.Is(err, voting.ErrSessionNotFound) {
				return newCLIError("voting.session_not_found")
			}
			return wrapCLIError("voting.error.get_results", err)
		}
		if jsonRequested(cmd) {
			return printJSON(resp)
		}

		// The heading carries the turn-out, as every other collection heading in
		// the CLI carries its count: a list of per-candidate figures leaves the
		// reader to add them up to answer "how many voted".
		fmt.Printf("\n📊 %s\n", i18n.Label(
			i18n.GetText("voting.results_heading"), voteCountText(resp.TotalVotes)))
		for _, cand := range resp.Candidates {
			name := cand.ID
			if cand.Name != "" {
				name = cand.Name + i18n.Paren(cand.Party)
			}
			fmt.Printf("   %s\n", i18n.Label(name, voteCountText(cand.VoteCount)))
		}
		return nil
	},
}
