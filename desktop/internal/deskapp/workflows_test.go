package deskapp

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/desktop/internal/nativeui"
)

// TestWorkflowTokenCreateMintRefresh walks the token task through the real
// application layer and the real page: create a token, mint to its owner, and
// press the header Refresh. The list must read the store the CLI reads, not a
// fake; the individual form handlers are covered by nativeui's view tests.
func TestWorkflowTokenCreateMintRefresh(t *testing.T) {
	app := startWorkflowApp(t)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Token")
	tester.Frame()
	require.False(t, tester.HasText("Workflow Coin"), "the store starts empty")

	owner, ownerPriv, err := app.Services().GenerateKeyPair()
	require.NoError(t, err)
	created, err := app.Services().CreateToken("Workflow Coin", "WFC", "1000", owner, 8)
	require.NoError(t, err)
	require.NoError(t, app.Services().MintToken(created.ID, owner, "250", ownerPriv))

	require.NoError(t, tester.Click("Refresh"))
	tester.Frame()
	require.True(t, tester.HasText("Workflow Coin"),
		"Refresh must list the token created in the same database")

	balance, err := app.Services().TokenBalance(created.ID, owner)
	require.NoError(t, err)
	require.Equal(t, "1250", balance, "the owner starts with the supply and receives the mint")
}

// TestWorkflowVotingSessionAndBallot walks the voting task: register a voter
// and candidate, create and start a session, cast a signed ballot, and read
// the tally. The Setup and Results stages must be looking at the same store.
func TestWorkflowVotingSessionAndBallot(t *testing.T) {
	app := startWorkflowApp(t)
	tester := ui.NewTester(app.View, 1280, 6000)
	app.ShowPage("Voting")
	tester.Frame()

	voter, err := app.Services().RegisterVoter("Alice")
	require.NoError(t, err)
	candidate, err := app.Services().RegisterCandidate("Bob", "Greens", "Free transit")
	require.NoError(t, err)
	now := time.Now().Unix()
	session, err := app.Services().CreateSession(
		"Budget 2027", "Approve the budget", []string{candidate.ID}, now-60, now+3600)
	require.NoError(t, err)

	require.NoError(t, tester.Click("Refresh"))
	tester.Frame()
	require.True(t, tester.HasText("Budget 2027"), "the session list must show the created session")
	require.True(t, tester.HasText("Bob"), "the candidate list must show the registered candidate")

	require.NoError(t, app.Services().StartSession(session.ID))
	require.NoError(t, app.Services().CastVote(
		session.ID, candidate.ID, voter.PublicKey, voter.PrivateKey))

	results, err := app.Services().SessionResults(session.ID)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, 1, results[0].Votes, "the signed ballot must count for the candidate")
}

// TestWorkflowOracleSourceRefresh walks the oracle task's first move: adding a
// source and seeing it in Sources after Refresh. The fetch itself depends on
// an external HTTP endpoint and is covered by the scheduler and health tests.
func TestWorkflowOracleSourceRefresh(t *testing.T) {
	app := startWorkflowApp(t)
	tester := ui.NewTester(app.View, 1280, 5000)
	app.ShowPage("Oracle")
	tester.Frame()
	require.False(t, tester.HasText("workflow-feed"))

	src, err := app.Services().AddOracleSource(nativeui.SourceInput{
		Name:     "workflow-feed",
		URL:      "https://aurora-workflow.invalid/price",
		Interval: 60,
	})
	require.NoError(t, err)
	require.NotEmpty(t, src.ID)

	require.NoError(t, tester.Click("Refresh"))
	tester.Frame()
	require.True(t, tester.HasText("workflow-feed"),
		"the Sources stage must list the source added to the real database")
}

// startWorkflowApp boots the real desktop runtime against a temporary database
// and returns its window state. It is the integration seam the workflow tests
// share.
func startWorkflowApp(t *testing.T) *nativeui.App {
	t.Helper()
	resetState(t)
	viperKey(t)
	viper.Set("log.path", filepath.Join(t.TempDir(), "logs"))

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	rt, err := Start(ctx, Options{DataDir: t.TempDir()})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, rt.Close()) })

	app := rt.App()
	require.NotNil(t, app)
	return app
}
