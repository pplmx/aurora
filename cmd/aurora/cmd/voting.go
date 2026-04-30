package cmd

import (
	"database/sql"
	"fmt"

	votingapp "github.com/pplmx/aurora/internal/app/voting"
	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/pplmx/aurora/internal/i18n"
	votingrepo "github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/spf13/cobra"
)

func initVotingDB() (*sql.DB, error) {
	if votingDB != nil {
		return votingDB, nil
	}
	db, err := blockchain.InitDB()
	if err != nil {
		return nil, err
	}
	votingDB = db
	return db, nil
}

func getVotingRepo() (voting.Repository, error) {
	if votingRepo != nil {
		return votingRepo, nil
	}
	db, err := initVotingDB()
	if err != nil {
		return nil, err
	}
	votingRepo = votingrepo.NewVotingRepository(db)
	return votingRepo, nil
}

func getVotingService() voting.Service {
	if votingService != nil {
		return votingService
	}
	votingService = voting.NewEd25519Service()
	return votingService
}

var (
	votingDB      *sql.DB
	votingRepo    voting.Repository
	votingService voting.Service
)

var votingCmd = &cobra.Command{
	Use:   "voting",
	Short: i18n.GetText("voting.cmd"),
	Long:  i18n.GetText("voting.cmd"),
}

var candidateCmd = &cobra.Command{
	Use:   "candidate",
	Short: i18n.GetText("voting.candidate.cmd"),
}

var candidateAddCmd = &cobra.Command{
	Use:   "add",
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

		fmt.Printf("✅ Candidate registered: %s\n", cand.Name)
		fmt.Printf("   ID: %s\n", cand.ID)
		fmt.Printf("   Party: %s\n", cand.Party)
		return nil
	},
}

var candidateListCmd = &cobra.Command{
	Use:   "list",
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
	Short: i18n.GetText("voting.voter.cmd"),
}

var voterRegisterCmd = &cobra.Command{
	Use:   "register",
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

		fmt.Println("✅ Voter registered successfully!")
		fmt.Printf("\n📣 Public Key (share this for verification):\n   %s\n", resp.PublicKey)
		fmt.Printf("\n🔐 Private Key (SAVE THIS SECURELY!):\n   %s\n", resp.PrivateKey)
		return nil
	},
}

var voterListCmd = &cobra.Command{
	Use:   "list",
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
	Short: i18n.GetText("voting.vote"),
	RunE: func(cmd *cobra.Command, args []string) error {
		voterPK, _ := cmd.Flags().GetString("voter")
		candidateID, _ := cmd.Flags().GetString("candidate")
		privKey, _ := cmd.Flags().GetString("private-key")

		repo, err := getVotingRepo()
		if err != nil {
			return fmt.Errorf("failed to get repository: %w", err)
		}

		service := getVotingService()
		blockchain.InitBlockChain()

		req := votingapp.CastVoteRequest{
			VoterPublicKey: voterPK,
			CandidateID:    candidateID,
			PrivateKey:     privKey,
		}
		uc := votingapp.NewCastVoteUseCase(repo, service)
		record, err := uc.Execute(req)
		if err != nil {
			return fmt.Errorf("failed to cast vote: %w", err)
		}

		fmt.Println("✅ Vote cast successfully!")
		fmt.Printf("   Vote ID:     %s\n", record.ID)
		fmt.Printf("   Block Height: %d\n", record.BlockHeight)
		return nil
	},
}

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: i18n.GetText("voting.session.cmd"),
}

var sessionCreateCmd = &cobra.Command{
	Use:     "create",
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

		req := votingapp.CreateSessionRequest{
			Title:        title,
			Description:  description,
			CandidateIDs: candidates,
			StartTime:    0,
			EndTime:      0,
		}
		uc := votingapp.NewCreateSessionUseCase(repo)
		session, err := uc.Execute(req)
		if err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}

		fmt.Printf("✅ Session created: %s\n", session.Title)
		fmt.Printf("   ID: %s\n", session.ID)
		fmt.Printf("   Status: %s\n", session.Status)
		return nil
	},
}

var sessionListCmd = &cobra.Command{
	Use:   "list",
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
	Short: i18n.GetText("voting.session.start"),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID, _ := cmd.Flags().GetString("id")

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

		fmt.Println("✅ Session started!")
		return nil
	},
}

var sessionEndCmd = &cobra.Command{
	Use:   "end",
	Short: i18n.GetText("voting.session.end"),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID, _ := cmd.Flags().GetString("id")

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

		fmt.Println("✅ Session ended!")
		return nil
	},
}

var resultsCmd = &cobra.Command{
	Use:   "results",
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
			if err != nil {
				continue
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
	_ = voteCmd.MarkFlagRequired("voter")
	_ = voteCmd.MarkFlagRequired("candidate")
	_ = voteCmd.MarkFlagRequired("private-key")

	sessionCreateCmd.Flags().StringP("title", "t", "", i18n.GetText("voting.title"))
	sessionCreateCmd.Flags().StringP("description", "d", "", i18n.GetText("voting.description"))
	sessionCreateCmd.Flags().StringSliceP("candidates", "c", nil, i18n.GetText("voting.candidate_id"))
	_ = sessionCreateCmd.MarkFlagRequired("title")
	_ = sessionCreateCmd.MarkFlagRequired("candidates")

	sessionStartCmd.Flags().StringP("id", "i", "", i18n.GetText("voting.session_id"))
	_ = sessionStartCmd.MarkFlagRequired("id")

	sessionEndCmd.Flags().StringP("id", "i", "", i18n.GetText("voting.session_id"))
	_ = sessionEndCmd.MarkFlagRequired("id")

	resultsCmd.Flags().StringP("session", "s", "", i18n.GetText("voting.session_id"))
	_ = resultsCmd.MarkFlagRequired("session")
}
