package deskapp

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/pplmx/aurora/internal/app/oracleapp"
	"github.com/pplmx/aurora/internal/app/tokenapp"
	"github.com/pplmx/aurora/internal/app/votingapp"
	"github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/pplmx/aurora/internal/domain/repoerr"
	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/pplmx/aurora/internal/infra/blockchain"
	"github.com/stretchr/testify/require"
)

type dashboardLotteryCounter struct {
	lottery.Repository
	count int
}

func (r dashboardLotteryCounter) Count() (int, error) { return r.count, nil }

func (r dashboardLotteryCounter) GetAll() ([]*lottery.LotteryRecord, error) {
	panic("dashboard must not materialise lottery history")
}

func (r dashboardLotteryCounter) List(int, int) ([]*lottery.LotteryRecord, error) {
	panic("dashboard must not materialise lottery history")
}

type dashboardVotingCounter struct {
	voting.TransactableRepository
	counts voting.Counts
}

func (r dashboardVotingCounter) Counts() (voting.Counts, error) { return r.counts, nil }

func (r dashboardVotingCounter) ListCandidates(int, int) ([]*voting.Candidate, error) {
	panic("dashboard must not materialise candidates")
}

func (r dashboardVotingCounter) ListSessions(int, int) ([]*voting.Session, error) {
	panic("dashboard must not materialise sessions")
}

type dashboardOracleCounter struct {
	oracle.Repository
	count int
}

func (r dashboardOracleCounter) CountSources() (int, error) { return r.count, nil }

func (r dashboardOracleCounter) ListSources(int, int) ([]*oracle.DataSource, error) {
	panic("dashboard must not materialise Oracle sources")
}

type voterSummaryRepo struct {
	voting.TransactableRepository
	rows       []*voting.Voter
	lastLimit  int
	lastOffset int
}

func (r *voterSummaryRepo) ListVoters(limit, offset int) ([]*voting.Voter, error) {
	r.lastLimit, r.lastOffset = limit, offset
	return r.rows, nil
}

func TestVoterListRowsOmitTheRegistrationPrivateKey(t *testing.T) {
	repo := &voterSummaryRepo{rows: []*voting.Voter{{
		PublicKey: "pk-alice",
		Name:      "Alice",
		HasVoted:  true,
	}}}
	services := &nativeServices{votingRepo: repo}

	rows, err := services.Voters(25, 50)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, 25, repo.lastLimit)
	require.Equal(t, 50, repo.lastOffset)
	require.Equal(t, "pk-alice", rows[0].PublicKey)
	require.Equal(t, "Alice", rows[0].Name)
	require.True(t, rows[0].HasVoted)
	require.Empty(t, rows[0].PrivateKey, "list reads must not carry a registration secret")
}

type candidateNamesRepo struct {
	voting.TransactableRepository
	candidates map[string]*voting.Candidate
	lookupErr  map[string]error
	lookups    []string
}

func (r *candidateNamesRepo) GetCandidate(id string) (*voting.Candidate, error) {
	r.lookups = append(r.lookups, id)
	if err := r.lookupErr[id]; err != nil {
		return nil, err
	}
	if candidate, ok := r.candidates[id]; ok {
		return candidate, nil
	}
	return nil, repoerr.ErrNotFound
}

func TestCandidateNamesResolveSessionRosterOutsideTheCurrentPage(t *testing.T) {
	repo := &candidateNamesRepo{
		candidates: map[string]*voting.Candidate{
			"cand-101": {ID: "cand-101", Name: "Alice"},
			"cand-102": {ID: "cand-102", Name: "Bob"},
		},
		lookupErr: map[string]error{"cand-deleted": repoerr.ErrNotFound},
	}
	services := &nativeServices{votingRepo: repo}

	names, err := services.CandidateNames([]string{
		"cand-101", "cand-deleted", "cand-102", "cand-101", "cand-deleted",
	})
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"cand-101": "Alice",
		"cand-102": "Bob",
	}, names)
	require.Equal(t, []string{"cand-101", "cand-deleted", "cand-102"}, repo.lookups,
		"the adapter should de-duplicate a roster without hiding a deleted candidate")
}

func TestCandidateNamesReturnARealStorageFailure(t *testing.T) {
	boom := errors.New("database is locked")
	services := &nativeServices{votingRepo: &candidateNamesRepo{
		lookupErr: map[string]error{"cand-101": boom},
	}}

	names, err := services.CandidateNames([]string{"cand-101"})
	require.ErrorIs(t, err, boom)
	require.Nil(t, names, "a partial roster must not be presented as a successful read")
}

func TestDashboardUsesAggregateCountsWithoutMaterialisingCollections(t *testing.T) {
	services := &nativeServices{
		lotteryRepo: dashboardLotteryCounter{count: 4},
		votingRepo: dashboardVotingCounter{counts: voting.Counts{
			Candidates: 3,
			Votes:      17,
			Sessions:   2,
		}},
		oracleRepo: dashboardOracleCounter{count: 5},
	}

	stats, err := services.Dashboard()
	require.NoError(t, err)
	require.Equal(t, 4, stats.Lotteries)
	require.Equal(t, 3, stats.Candidates)
	require.Equal(t, 17, stats.Votes)
	require.Equal(t, 2, stats.Sessions)
	require.Equal(t, 5, stats.Oracle)
	require.Equal(t, "broken", stats.Integrity, "a nil chain is not a verified ledger")
}

// TestGenerateKeyPairIsAUsableEd25519Pair checks the one service the desktop
// added for its forms: the two halves must be the right size and belong
// together, because a token owner key that cannot sign would only fail later,
// on the first mint.
func TestGenerateKeyPairIsAUsableEd25519Pair(t *testing.T) {
	services := &nativeServices{}
	pubText, privText, err := services.GenerateKeyPair()
	require.NoError(t, err)

	pub, err := base64.StdEncoding.DecodeString(pubText)
	require.NoError(t, err)
	require.Len(t, pub, ed25519.PublicKeySize)

	priv, err := base64.StdEncoding.DecodeString(privText)
	require.NoError(t, err)
	require.Len(t, priv, ed25519.PrivateKeySize)

	message := []byte("aurora")
	signature := ed25519.Sign(ed25519.PrivateKey(priv), message)
	require.True(t, ed25519.Verify(ed25519.PublicKey(pub), message, signature),
		"the public half must verify what the private half signs")
}

// TestHealthRowsCarrySchedulerCounters pins the adapter the Feed Health card
// reads. It used to return (nil, nil) unconditionally, so the card could only
// ever say "(no feed activity yet)" while the REST endpoint showed counters.
func TestHealthRowsCarrySchedulerCounters(t *testing.T) {
	row := healthRow(oracleapp.SourceStat{
		SourceID: "s1", Name: "btc-price", Attempts: 7, Successes: 5, Failures: 2,
		LastSuccessAt: 1_700_000_000, LastError: "upstream 503",
	})
	require.Equal(t, "s1", row.SourceID)
	require.Equal(t, "btc-price", row.Name)
	require.Equal(t, 7, row.Attempts)
	require.Equal(t, 5, row.Successes)
	require.Equal(t, 2, row.Failures)
	require.Equal(t, int64(1_700_000_000), row.LastSuccess)
	require.Equal(t, "upstream 503", row.LastError)
}

// TestOracleHealthWithoutASchedulerIsEmpty keeps the adapter nil-safe: a test
// or a runtime built before StartOracleScheduler must report no activity, not
// panic on a nil provider.
func TestOracleHealthWithoutASchedulerIsEmpty(t *testing.T) {
	rows, err := (&nativeServices{}).OracleHealth()
	require.NoError(t, err)
	require.Empty(t, rows)
}

func TestOracleSourceLookupCarriesTheActionRow(t *testing.T) {
	repo := oracle.NewInmemRepo()
	require.NoError(t, repo.SaveSource(&oracle.DataSource{
		ID: "old-source", Name: "Historic feed", URL: "https://example.com/price",
		Type: "json", Enabled: false, Interval: 60, Headers: `{"Authorization":"Bearer x"}`,
	}))
	services := &nativeServices{oracleRepo: repo}

	row, err := services.OracleSource("old-source")
	require.NoError(t, err)
	require.Equal(t, "old-source", row.ID)
	require.Equal(t, "Historic feed", row.Name)
	require.Equal(t, "https://example.com/price", row.URL)
	require.False(t, row.Enabled)
	require.Equal(t, 60, row.Interval)
	require.True(t, row.HasHeaders)

	_, err = services.OracleSource("missing")
	require.ErrorIs(t, err, oracle.ErrSourceNotFound)
}

// TestTransferRowsCarryNonces pins the second half of the token history
// mapping: the use case now returns a nonce, and the view row must keep it or
// the desktop column still renders 0.
func TestTransferRowsCarryNonces(t *testing.T) {
	row := transferRow(tokenapp.TransferResponse{
		From: "alice", To: "bob", Amount: "12", Nonce: 9,
	})
	require.Equal(t, "alice", row.From)
	require.Equal(t, "bob", row.To)
	require.Equal(t, "12", row.Amount)
	require.Equal(t, uint64(9), row.Nonce)
}

// TestSessionRowsCarryDescriptions pins the field the Create Session form
// collects and the selected-session preview shows. The adapter dropped it, so
// the text was stored but unreachable from the window.
func TestSessionRowsCarryDescriptions(t *testing.T) {
	row := sessionRow(&votingapp.SessionResponse{
		ID: "s1", Title: "Budget", Description: "Approve 2027",
		Status: "active", Candidates: []string{"c1"}, StartTime: 10, EndTime: 20,
	})
	require.Equal(t, "s1", row.ID)
	require.Equal(t, "Budget", row.Title)
	require.Equal(t, "Approve 2027", row.Description)
	require.Equal(t, "active", row.Status)
	require.Equal(t, []string{"c1"}, row.Candidates)
	require.Equal(t, int64(10), row.StartTime)
	require.Equal(t, int64(20), row.EndTime)
}

// TestLedgerBlocksCarryBlockHeights pins the heights the desktop ledger shows.
// The adapter used each payload's index in the filtered chain, so the first
// recorded block displayed height 0 and every row was one too low.
func TestLedgerBlocksCarryBlockHeights(t *testing.T) {
	chain := blockchain.New()
	_, err := chain.AddBlock(`{"module":"lottery"}`)
	require.NoError(t, err)
	_, err = chain.AddBlock(`{"module":"voting"}`)
	require.NoError(t, err)

	services := &nativeServices{chain: chain}
	rows, err := services.LedgerBlocks(0)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, int64(2), rows[0].Height, "the newest block is at height 2")
	require.Equal(t, "voting", rows[0].Module)
	require.Equal(t, int64(1), rows[1].Height)
	require.Equal(t, "lottery", rows[1].Module)

	latest, err := services.LedgerBlocks(1)
	require.NoError(t, err)
	require.Len(t, latest, 1)
	require.Equal(t, int64(2), latest[0].Height)
}
