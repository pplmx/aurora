package cmd

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"github.com/pplmx/aurora/internal/app/votingapp"
	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	"github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/spf13/cobra"
)

// votingInitMu guards the package-level singletons (votingDB,
// votingRepo, votingService). Without this lock, two concurrent
// callers of getVotingRepo() (e.g. multiple cobra subcommands
// running in parallel, or tests using t.Parallel) both observe
// votingRepo == nil, both call NewVotingRepository, and race on
// the assignment. The assignment itself is racy even when InitDB
// is safe — see internal/infra/blockchain/init.go for the
// underlying DB initialisation fix (Round 19).
var votingInitMu sync.Mutex

func getVotingRepo() (voting.TransactableRepository, error) {
	votingInitMu.Lock()
	defer votingInitMu.Unlock()
	if votingRepo != nil {
		return votingRepo, nil
	}
	db, err := blockchain.InitDB()
	if err != nil {
		return nil, err
	}
	votingDB = db
	repo, err := sqlite.NewVotingRepository(db)
	if err != nil {
		return nil, err
	}
	votingRepo = repo
	return votingRepo, nil
}

// getVotingTxManager returns the transaction manager sharing the voting
// repository's database handle. CastVote's voter claim + vote row + tally
// increment run in one SQLite transaction through it.
func getVotingTxManager() (*sqlite.TxManager, error) {
	if _, err := getVotingRepo(); err != nil {
		return nil, err
	}
	votingInitMu.Lock()
	defer votingInitMu.Unlock()
	if votingTxManager == nil {
		votingTxManager = sqlite.NewTxManager(votingDB)
	}
	return votingTxManager, nil
}

func getVotingService() voting.Service {
	votingInitMu.Lock()
	defer votingInitMu.Unlock()
	if votingService != nil {
		return votingService
	}
	votingService = voting.NewEd25519Service()
	return votingService
}

var (
	votingRepo      voting.TransactableRepository
	votingService   voting.Service
	votingDB        *sql.DB
	votingTxManager *sqlite.TxManager
)

var votingCmd = &cobra.Command{
	Use:   "voting",
	Args:  cobra.NoArgs,
	Short: i18n.GetText("voting.cmd"),
	Long:  i18n.GetText("voting.cmd"),
}

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
		printLocalizedField("common.id", cand.ID)
		printLocalizedField("voting.party_label", cand.Party)
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

		if limit > 0 {
			fmt.Printf("\n📋 %s: %s\n",
				i18n.GetText("voting.candidates"),
				i18n.GetTextF("common.page_with_offset", len(list), offset))
		} else {
			fmt.Println("\n📋 " + i18n.GetText("voting.candidates") + ":")
		}
		if len(list) == 0 {
			fmt.Println("   " + i18n.GetText("voting.none"))
		}
		for _, c := range list {
			fmt.Printf("   - %s [%s] - %s\n", c.Name, c.Party, i18n.GetTextF("voting.votes", c.VoteCount))
			fmt.Printf("     %s: %s\n", i18n.GetText("voting.id"), c.ID)
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
			fmt.Printf("\n👥 %s: %s\n",
				i18n.GetText("voting.page_total"),
				i18n.GetTextF("common.page_with_offset", len(list), offset))
		} else {
			fmt.Println("\n👥 " + i18n.GetText("voting.voters") + ":")
		}
		if len(list) == 0 {
			if limit > 0 || offset > 0 {
				return nil
			}
			fmt.Println("   " + i18n.GetText("voting.none"))
		}
		for _, v := range list {
			status := "✅ " + i18n.GetText("voting.voted")
			if !v.HasVoted {
				status = "⏳ " + i18n.GetText("voting.not_voted")
			}
			fmt.Printf("   - %s [%s]\n", v.Name, status)
			if len(v.PublicKey) > 16 {
				fmt.Printf("     %s: %s...\n", i18n.GetText("voting.public_key_label"), v.PublicKey[:16])
			} else {
				printField("  "+i18n.GetText("voting.public_key_label"), v.PublicKey)
			}
		}
		return nil
	},
}

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
		fmt.Printf("   %s:     %s\n", i18n.GetText("voting.vote_id"), record.ID)
		fmt.Printf("   %s: %d\n", i18n.GetText("common.block_height"), record.BlockHeight)
		return nil
	},
}

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
		printLocalizedField("common.id", session.ID)
		printLocalizedField("common.status", localizedVotingStatus(session.Status))
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

		uc := votingapp.NewListSessionsUseCase(repo)
		list, err := uc.Execute(&votingapp.ListSessionsRequest{Limit: limit, Offset: offset})
		if err != nil {
			return wrapCLIError("voting.error.list_sessions", err)
		}
		if jsonRequested(cmd) {
			if list == nil {
				list = []*votingapp.SessionResponse{}
			}
			return printJSON(list)
		}

		if limit > 0 {
			fmt.Printf("\n🗳️ %s: %s\n",
				i18n.GetText("voting.sessions"),
				i18n.GetTextF("common.page_with_offset", len(list), offset))
		} else {
			fmt.Println("\n🗳️ " + i18n.GetText("voting.sessions") + ":")
		}
		if len(list) == 0 {
			fmt.Println("   " + i18n.GetText("voting.none"))
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

		fmt.Println("\n📊 " + i18n.GetText("voting.results_heading") + ":")
		for _, cand := range resp.Candidates {
			name := cand.ID
			if cand.Name != "" {
				name = fmt.Sprintf("%s (%s)", cand.Name, cand.Party)
			}
			fmt.Printf("   %s: %s\n", name, i18n.GetTextF("voting.votes", cand.VoteCount))
		}
		return nil
	},
}

func localizedVotingStatus(status string) string {
	key := "voting.status." + status
	if translated := i18n.GetText(key); translated != key {
		return translated
	}
	return status
}

func init() {
	rootCmd.AddCommand(votingCmd)

	votingCmd.AddCommand(candidateCmd)
	candidateCmd.AddCommand(candidateAddCmd)
	addJSONFlag(candidateAddCmd)
	candidateCmd.AddCommand(candidateListCmd)
	addJSONFlag(candidateListCmd)

	votingCmd.AddCommand(voterCmd)
	voterCmd.AddCommand(voterRegisterCmd)
	voterCmd.AddCommand(voterListCmd)
	addJSONFlag(voterListCmd)

	votingCmd.AddCommand(voteCmd)
	addJSONFlag(voteCmd)

	votingCmd.AddCommand(sessionCmd)
	sessionCmd.AddCommand(sessionCreateCmd)
	addJSONFlag(sessionCreateCmd)
	sessionCmd.AddCommand(sessionListCmd)
	addJSONFlag(sessionListCmd)
	sessionCmd.AddCommand(sessionStartCmd)
	addJSONFlag(sessionStartCmd)
	sessionCmd.AddCommand(sessionEndCmd)
	addJSONFlag(sessionEndCmd)

	votingCmd.AddCommand(resultsCmd)
	addJSONFlag(resultsCmd)

	candidateAddCmd.Flags().StringP("name", "n", "", i18n.GetText("voting.name"))
	candidateAddCmd.Flags().StringP("party", "p", "", i18n.GetText("voting.party"))
	candidateAddCmd.Flags().StringP("program", "m", "", i18n.GetText("voting.program"))
	_ = candidateAddCmd.MarkFlagRequired("name")
	_ = candidateAddCmd.MarkFlagRequired("party")
	candidateListCmd.Flags().IntP("limit", "l", 0, i18n.GetText("token.list_limit"))
	candidateListCmd.Flags().IntP("offset", "p", 0, i18n.GetText("token.offset"))

	voterRegisterCmd.Flags().StringP("name", "n", "", i18n.GetText("voting.name"))
	_ = voterRegisterCmd.MarkFlagRequired("name")
	voterListCmd.Flags().IntP("limit", "l", 0, i18n.GetText("token.list_limit"))
	voterListCmd.Flags().IntP("offset", "p", 0, i18n.GetText("token.offset"))

	voteCmd.Flags().StringP("voter", "v", "", i18n.GetText("voting.public_key"))
	voteCmd.Flags().StringP("candidate", "c", "", i18n.GetText("voting.candidate_id"))
	voteCmd.Flags().StringP("private-key", "k", "", i18n.GetText("voting.private_key"))
	voteCmd.Flags().StringP("session", "s", "", i18n.GetText("voting.session_id"))
	_ = voteCmd.MarkFlagRequired("voter")
	_ = voteCmd.MarkFlagRequired("candidate")
	_ = voteCmd.MarkFlagRequired("private-key")
	// session is used by the domain before anything else; mark it required so
	// an omission fails fast instead of a misleading "session not found"
	// (TASK-153, ISS-144; sibling voting results already marks it).
	_ = voteCmd.MarkFlagRequired("session")

	sessionCreateCmd.Flags().StringP("title", "t", "", i18n.GetText("voting.title"))
	sessionCreateCmd.Flags().StringP("description", "d", "", i18n.GetText("voting.description"))
	sessionCreateCmd.Flags().StringSliceP("candidates", "c", nil, i18n.GetText("voting.candidate_id"))
	sessionListCmd.Flags().IntP("limit", "l", 0, i18n.GetText("token.list_limit"))
	sessionListCmd.Flags().IntP("offset", "p", 0, i18n.GetText("token.offset"))
	sessionCreateCmd.Flags().Int64P("start-time", "", 0, i18n.GetText("voting.session_start_time"))
	sessionCreateCmd.Flags().Int64P("end-time", "", 0, i18n.GetText("voting.session_end_time"))
	_ = sessionCreateCmd.MarkFlagRequired("title")
	_ = sessionCreateCmd.MarkFlagRequired("candidates")
	// start/end default to 0 and omitting both used to surface a misleading
	// "session end time must be after start time" (INVALID_SESSION_TIME) from
	// the domain; mark them required so an omission fails fast like title.
	_ = sessionCreateCmd.MarkFlagRequired("start-time")
	_ = sessionCreateCmd.MarkFlagRequired("end-time")

	// --session/-s (not --id/-i) so the session selector is spelled the same
	// across session start/end and vote/results (TASK-152, ISS-143).
	sessionStartCmd.Flags().StringP("session", "s", "", i18n.GetText("voting.session_id"))
	_ = sessionStartCmd.MarkFlagRequired("session")

	sessionEndCmd.Flags().StringP("session", "s", "", i18n.GetText("voting.session_id"))
	_ = sessionEndCmd.MarkFlagRequired("session")

	resultsCmd.Flags().StringP("session", "s", "", i18n.GetText("voting.session_id"))
	_ = resultsCmd.MarkFlagRequired("session")
}
