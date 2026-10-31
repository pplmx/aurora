// The voting command group: the shared repository/service singletons the
// subcommands use, the group itself, and the registration table in init.
// The commands themselves live by registry — candidates and voters in
// voting_registry.go, the ballot in voting_ballot.go, sessions and their tally
// in voting_sessions.go — the same arrangement the lottery and token groups use.
package cmd

import (
	"database/sql"
	"sync"

	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/pplmx/aurora/internal/i18n"
	"github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/pplmx/aurora/internal/infra/store"
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
	db, err := store.OpenShared()
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
