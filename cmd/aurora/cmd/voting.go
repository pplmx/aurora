package cmd

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"

	votingapp "github.com/pplmx/aurora/internal/app/voting"
	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/pplmx/aurora/internal/i18n"
	votingrepo "github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/spf13/cobra"
)

// votingInitMu guards the package-level singletons (votingDB,
// votingRepo, votingService). Without this lock, two concurrent
// callers of getVotingRepo() (e.g. multiple cobra subcommands
// running in parallel, or tests using t.Parallel) both observe
// votingRepo == nil, both call NewVotingRepository, and race on
// the assignment. The assignment itself is racy even when InitDB
// is safe — see internal/domain/blockchain/init.go for the
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
	repo, err := votingrepo.NewVotingRepository(db)
	if err != nil {
		return nil, err
	}
	votingRepo = repo
	return votingRepo, nil
}

// getVotingTxManager returns the transaction manager sharing the voting
// repository's database handle. CastVote's voter claim + vote row + tally
// increment run in one SQLite transaction through it.
func getVotingTxManager() (*votingrepo.TxManager, error) {
	if _, err := getVotingRepo(); err != nil {
		return nil, err
	}
	votingInitMu.Lock()
	defer votingInitMu.Unlock()
	if votingTxManager == nil {
		votingTxManager = votingrepo.NewTxManager(votingDB)
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
	votingTxManager *votingrepo.TxManager
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
			return fmt.Errorf("failed to get repository: %w", err)
		}

		req := votingapp.RegisterCandidateRequest{
			Name:    name,
			Party:   party,
			Program: program,
		}
		uc := votingapp.NewRegisterCandidateUseCase(repo)
		cand, err := uc.Execute(req)
		if err != nil {
			return fmt.Errorf("failed to register candidate: %w", err)
		}

		fmt.Printf("✅ "+i18n.GetText("voting.candidate_added")+"\n", cand.Name)
		fmt.Printf("   ID: %s\n", cand.ID)
		fmt.Printf("   Party: %s\n", cand.Party)
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
			return fmt.Errorf("failed to get repository: %w", err)
		}

		uc := votingapp.NewGetCandidatesUseCase(repo)
		list, err := uc.Execute()
		if err != nil {
			return fmt.Errorf("failed to list candidates: %w", err)
		}

		fmt.Println("\n📋 Candidates:")
		if len(list) == 0 {
			fmt.Println("   (none)")
		}
		for _, c := range list {
			fmt.Printf("   - %s [%s] - %d votes\n", c.Name, c.Party, c.VoteCount)
			fmt.Printf("     ID: %s\n", c.ID)
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
			return fmt.Errorf("failed to get repository: %w", err)
		}

		req := votingapp.RegisterVoterRequest{
			Name: name,
		}
		uc := votingapp.NewRegisterVoterUseCase(repo)
		resp, err := uc.Execute(req)
		if err != nil {
			return fmt.Errorf("failed to register voter: %w", err)
		}

		fmt.Println("✅ " + i18n.GetText("voting.voter_registered"))
		fmt.Printf("\n📣 Public Key (share this for verification):\n   %s\n", resp.PublicKey)
		fmt.Printf("\n🔐 Private Key (SAVE THIS SECURELY!):\n   %s\n", resp.PrivateKey)
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
			return fmt.Errorf("failed to get repository: %w", err)
		}

		list, err := repo.ListVoters()
		if err != nil {
			return fmt.Errorf("failed to list voters: %w", err)
		}

		fmt.Println("\n👥 Voters:")
		if len(list) == 0 {
			fmt.Println("   (none)")
		}
		for _, v := range list {
			status := "✅ voted"
			if !v.HasVoted {
				status = "⏳ not voted"
			}
			fmt.Printf("   - %s [%s]\n", v.Name, status)
			if len(v.PublicKey) > 16 {
				fmt.Printf("     Public Key: %s...\n", v.PublicKey[:16])
			} else {
				fmt.Printf("     Public Key: %s\n", v.PublicKey)
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
			return fmt.Errorf("failed to get repository: %w", err)
		}

		txManager, err := getVotingTxManager()
		if err != nil {
			return fmt.Errorf("failed to get transaction manager: %w", err)
		}

		service := getVotingService()
		blockchain.InitBlockChain()

		req := votingapp.CastVoteRequest{
			VoterPublicKey: voterPK,
			CandidateID:    candidateID,
			PrivateKey:     privKey,
			SessionID:      sessionID,
		}
		uc := votingapp.NewCastVoteUseCase(repo, service, txManager)
		uc.SetChain(blockchain.GetBlockChain())
		record, err := uc.Execute(req)
		if err != nil {
			return fmt.Errorf("failed to cast vote: %w", err)
		}

		fmt.Println("✅ " + i18n.GetText("voting.vote_cast"))
		fmt.Printf("   Vote ID:     %s\n", record.ID)
		fmt.Printf("   Block Height: %d\n", record.BlockHeight)
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
			return fmt.Errorf("failed to get repository: %w", err)
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
			return fmt.Errorf("failed to create session: %w", err)
		}

		fmt.Printf("✅ "+i18n.GetText("voting.session_created")+"\n", session.Title)
		fmt.Printf("   ID: %s\n", session.ID)
		fmt.Printf("   Status: %s\n", session.Status)
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
			return fmt.Errorf("failed to get repository: %w", err)
		}

		list, err := repo.ListSessions()
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}

		fmt.Println("\n🗳️ Voting Sessions:")
		if len(list) == 0 {
			fmt.Println("   (none)")
		}
		for _, s := range list {
			fmt.Printf("   - %s [%s]\n", s.Title, s.Status)
			fmt.Printf("     ID: %s\n", s.ID)
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
			return fmt.Errorf("failed to get repository: %w", err)
		}

		session, err := repo.GetSession(sessionID)
		if err != nil {
			return fmt.Errorf("failed to get session: %w", err)
		}
		if session == nil {
			return fmt.Errorf("session not found")
		}

		session.Status = "active"
		if err := repo.UpdateSession(session); err != nil {
			return fmt.Errorf("failed to start session: %w", err)
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
			return fmt.Errorf("failed to get repository: %w", err)
		}

		session, err := repo.GetSession(sessionID)
		if err != nil {
			return fmt.Errorf("failed to get session: %w", err)
		}
		if session == nil {
			return fmt.Errorf("session not found")
		}

		session.Status = "ended"
		if err := repo.UpdateSession(session); err != nil {
			return fmt.Errorf("failed to end session: %w", err)
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
			return fmt.Errorf("failed to get repository: %w", err)
		}

		session, err := repo.GetSession(sessionID)
		if err != nil {
			return fmt.Errorf("failed to get session: %w", err)
		}
		if session == nil {
			return fmt.Errorf("session not found")
		}

		results := make(map[string]int)
		for _, cid := range session.Candidates {
			cand, err := repo.GetCandidate(cid)
			if errors.Is(err, votingrepo.ErrNotFound) {
				// The roster references a candidate that no longer exists —
				// show the bare id at 0 votes instead of hiding it (TASK-154).
				results[cid] = 0
				continue
			}
			if err != nil {
				return fmt.Errorf("failed to load candidate %s for results: %w", cid, err)
			}
			if cand != nil {
				results[fmt.Sprintf("%s (%s)", cand.Name, cand.Party)] = cand.VoteCount
			} else {
				results[cid] = 0
			}
		}

		fmt.Println("\n📊 Results:")
		for name, count := range results {
			fmt.Printf("   %s: %d votes\n", name, count)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(votingCmd)

	votingCmd.AddCommand(candidateCmd)
	candidateCmd.AddCommand(candidateAddCmd)
	candidateCmd.AddCommand(candidateListCmd)

	votingCmd.AddCommand(voterCmd)
	voterCmd.AddCommand(voterRegisterCmd)
	voterCmd.AddCommand(voterListCmd)

	votingCmd.AddCommand(voteCmd)

	votingCmd.AddCommand(sessionCmd)
	sessionCmd.AddCommand(sessionCreateCmd)
	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionStartCmd)
	sessionCmd.AddCommand(sessionEndCmd)

	votingCmd.AddCommand(resultsCmd)

	candidateAddCmd.Flags().StringP("name", "n", "", i18n.GetText("voting.name"))
	candidateAddCmd.Flags().StringP("party", "p", "", i18n.GetText("voting.party"))
	candidateAddCmd.Flags().StringP("program", "m", "", i18n.GetText("voting.program"))
	_ = candidateAddCmd.MarkFlagRequired("name")
	_ = candidateAddCmd.MarkFlagRequired("party")

	voterRegisterCmd.Flags().StringP("name", "n", "", i18n.GetText("voting.name"))
	_ = voterRegisterCmd.MarkFlagRequired("name")

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
	sessionCreateCmd.Flags().Int64P("start-time", "", 0, i18n.GetText("voting.session_start_time"))
	sessionCreateCmd.Flags().Int64P("end-time", "", 0, i18n.GetText("voting.session_end_time"))
	_ = sessionCreateCmd.MarkFlagRequired("title")
	_ = sessionCreateCmd.MarkFlagRequired("candidates")

	// --session/-s (not --id/-i) so the session selector is spelled the same
	// across session start/end and vote/results (TASK-152, ISS-143).
	sessionStartCmd.Flags().StringP("session", "s", "", i18n.GetText("voting.session_id"))
	_ = sessionStartCmd.MarkFlagRequired("session")

	sessionEndCmd.Flags().StringP("session", "s", "", i18n.GetText("voting.session_id"))
	_ = sessionEndCmd.MarkFlagRequired("session")

	resultsCmd.Flags().StringP("session", "s", "", i18n.GetText("voting.session_id"))
	_ = resultsCmd.MarkFlagRequired("session")
}
