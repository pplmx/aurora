package cmd

import (
	"encoding/base64"
	"fmt"

	"github.com/pplmx/aurora/internal/blockchain"
	"github.com/pplmx/aurora/internal/voting"
	"github.com/spf13/cobra"
)

var votingCmd = &cobra.Command{
	Use:   "voting",
	Short: "Ed25519 signature based transparent voting system",
	Long:  "A secure voting system with Ed25519 signatures and blockchain storage",
}

var candidateCmd = &cobra.Command{
	Use:   "candidate",
	Short: "Candidate management",
}

var candidateAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a candidate",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		party, _ := cmd.Flags().GetString("party")
		program, _ := cmd.Flags().GetString("program")

		cand, err := voting.RegisterCandidate(name, party, program)
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
	Short: "List candidates",
	RunE: func(cmd *cobra.Command, args []string) error {
		list, err := voting.ListCandidates()
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
	Short: "Voter management",
}

var voterRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new voter",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")

		pub, priv, err := voting.RegisterVoter(name)
		if err != nil {
			return fmt.Errorf("failed to register voter: %w", err)
		}

		fmt.Println("✅ Voter registered successfully!")
		fmt.Printf("\n📣 Public Key (share this for verification):\n   %s\n",
			base64.StdEncoding.EncodeToString(pub))
		fmt.Printf("\n🔐 Private Key (SAVE THIS SECURELY!):\n   %s\n",
			base64.StdEncoding.EncodeToString(priv))
		return nil
	},
}

var voterListCmd = &cobra.Command{
	Use:   "list",
	Short: "List voters",
	RunE: func(cmd *cobra.Command, args []string) error {
		list, err := voting.ListVoters()
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
			fmt.Printf("     Public Key: %s...\n", v.PublicKey[:min(16, len(v.PublicKey))])
		}
		return nil
	},
}

var voteCmd = &cobra.Command{
	Use:   "vote",
	Short: "Cast a vote",
	RunE: func(cmd *cobra.Command, args []string) error {
		voterPK, _ := cmd.Flags().GetString("voter")
		candidateID, _ := cmd.Flags().GetString("candidate")
		privKey, _ := cmd.Flags().GetString("private-key")

		pubBytes, err := base64.StdEncoding.DecodeString(voterPK)
		if err != nil {
			return fmt.Errorf("invalid voter public key: %w", err)
		}

		privBytes, err := base64.StdEncoding.DecodeString(privKey)
		if err != nil {
			return fmt.Errorf("invalid private key: %w", err)
		}

		chain := blockchain.InitBlockChain()
		record, err := voting.CastVote(pubBytes, candidateID, privBytes, chain)
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
	Short: "Voting session management",
}

var sessionCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a voting session",
	RunE: func(cmd *cobra.Command, args []string) error {
		title, _ := cmd.Flags().GetString("title")
		description, _ := cmd.Flags().GetString("description")
		candidates, _ := cmd.Flags().GetStringSlice("candidates")

		session, err := voting.CreateSession(title, description, candidates, 0, 0)
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
	Short: "List voting sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		list, err := voting.ListSessions()
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
	Short: "Start a voting session",
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID, _ := cmd.Flags().GetString("id")
		if err := voting.StartSession(sessionID); err != nil {
			return fmt.Errorf("failed to start session: %w", err)
		}
		fmt.Println("✅ Session started!")
		return nil
	},
}

var sessionEndCmd = &cobra.Command{
	Use:   "end",
	Short: "End a voting session",
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID, _ := cmd.Flags().GetString("id")
		if err := voting.EndSession(sessionID); err != nil {
			return fmt.Errorf("failed to end session: %w", err)
		}
		fmt.Println("✅ Session ended!")
		return nil
	},
}

var resultsCmd = &cobra.Command{
	Use:   "results",
	Short: "Get voting results",
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID, _ := cmd.Flags().GetString("session")

		results, err := voting.GetSessionResults(sessionID)
		if err != nil {
			return fmt.Errorf("failed to get results: %w", err)
		}

		fmt.Println("\n📊 Results:")
		for cid, count := range results {
			cand, _ := voting.GetCandidate(cid)
			name := cid
			if cand != nil {
				name = fmt.Sprintf("%s (%s)", cand.Name, cand.Party)
			}
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

	candidateAddCmd.Flags().StringP("name", "n", "", "Candidate name")
	candidateAddCmd.Flags().StringP("party", "p", "", "Candidate party")
	candidateAddCmd.Flags().StringP("program", "m", "", "Candidate program")
	candidateAddCmd.MarkFlagRequired("name")
	candidateAddCmd.MarkFlagRequired("party")

	voterRegisterCmd.Flags().StringP("name", "n", "", "Voter name")
	voterRegisterCmd.MarkFlagRequired("name")

	voteCmd.Flags().StringP("voter", "v", "", "Voter public key (base64)")
	voteCmd.Flags().StringP("candidate", "c", "", "Candidate ID")
	voteCmd.Flags().StringP("private-key", "k", "", "Voter private key (base64)")
	voteCmd.MarkFlagRequired("voter")
	voteCmd.MarkFlagRequired("candidate")
	voteCmd.MarkFlagRequired("private-key")

	sessionCreateCmd.Flags().StringP("title", "t", "", "Session title")
	sessionCreateCmd.Flags().StringP("description", "d", "", "Session description")
	sessionCreateCmd.Flags().StringSliceP("candidates", "c", nil, "Candidate IDs (can repeat)")
	sessionCreateCmd.MarkFlagRequired("title")
	sessionCreateCmd.MarkFlagRequired("candidates")

	sessionStartCmd.Flags().StringP("id", "i", "", "Session ID")
	sessionStartCmd.MarkFlagRequired("id")

	sessionEndCmd.Flags().StringP("id", "i", "", "Session ID")
	sessionEndCmd.MarkFlagRequired("id")

	resultsCmd.Flags().StringP("session", "s", "", "Session ID")
	resultsCmd.MarkFlagRequired("session")
}
