package container

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/pplmx/aurora/internal/domain/voting"
	"github.com/pplmx/aurora/internal/infra/blockchain"
)

// TestOpenWiresAndCloses pins the composition root itself: every repository and
// service field is populated, and Close can run twice without leaking or
// panicking. The API failure-path test covers partial construction; this is the
// happy path that production and the desktop both rely on.
func TestOpenWiresAndCloses(t *testing.T) {
	blockchain.ResetForTest()
	t.Cleanup(blockchain.ResetForTest)
	viper.Reset()
	t.Cleanup(viper.Reset)

	dir := t.TempDir()
	t.Chdir(dir)
	viper.Set("db.path", filepath.Join(dir, "aurora.db"))

	c, err := Open()
	require.NoError(t, err)
	require.NotNil(t, c.DB)
	require.NotNil(t, c.LotteryRepo)
	require.NotNil(t, c.VotingRepo)
	require.NotNil(t, c.NFTRepo)
	require.NotNil(t, c.TokenRepo)
	require.NotNil(t, c.OracleRepo)
	require.NotNil(t, c.TokenService)
	require.NotNil(t, c.TxManager)
	require.NotNil(t, c.VotingTx)
	require.NotNil(t, c.NFTTx)
	require.NotNil(t, c.Chain)
	require.NotNil(t, c.EventStore)
	require.NotNil(t, c.OracleFetcher)

	// The adapters see the container only through Services, so an accessor that
	// returned the wrong field would compile and hand, say, the token
	// transaction manager to voting. Each accessor must return the field it
	// names.
	var services Services = c
	require.Equal(t, lottery.Repository(c.LotteryRepo), services.LotteryRepository())
	require.Equal(t, voting.TransactableRepository(c.VotingRepo), services.VotingRepository())
	require.Equal(t, nft.TransactableRepository(c.NFTRepo), services.NFTRepository())
	require.Equal(t, oracle.Repository(c.OracleRepo), services.OracleRepository())
	require.Equal(t, c.TokenService, services.Token())
	require.Equal(t, c.TxManager, services.TokenTransactions())
	require.Equal(t, c.VotingTx, services.VotingTransactions())
	require.Equal(t, c.NFTTx, services.NFTTransactions())
	require.Equal(t, c.Chain, services.Ledger())
	require.Equal(t, c.OracleFetcher, services.Fetcher())

	backupFile, err := c.Backup(context.Background(), filepath.Join(dir, "snapshot"))
	require.NoError(t, err)
	require.DirExists(t, backupFile)

	stopOutbox := c.StartOutboxDrainer(context.Background())
	stopOutbox()
	stopOutbox()

	require.NoError(t, c.Close())
	require.NoError(t, c.Close(), "Close must be idempotent for shutdown paths")
}

// TestOpenRejectsUnusableDatabasePath keeps the composition root from falling
// back to a temporary SQLite database when its configured parent is not a
// directory. Such a fallback used to let the API and desktop accept writes that
// disappeared when the process exited.
func TestOpenRejectsUnusableDatabasePath(t *testing.T) {
	blockchain.ResetForTest()
	t.Cleanup(blockchain.ResetForTest)
	viper.Reset()
	t.Cleanup(viper.Reset)

	dir := t.TempDir()
	blocker := filepath.Join(dir, "not-a-directory")
	require.NoError(t, os.WriteFile(blocker, []byte("occupied"), 0o600))
	viper.Set("db.path", filepath.Join(blocker, "aurora.db"))

	c, err := Open()
	require.Error(t, err)
	require.Nil(t, c)
}

// TestOpenRejectsCorruptLedger keeps the composition root from serving a
// partially decoded blockchain. The old boot path logged and skipped an
// unscannable row, so the API could start with a chain that no longer matched
// the rows still present in SQLite.
func TestOpenRejectsCorruptLedger(t *testing.T) {
	blockchain.ResetForTest()
	t.Cleanup(blockchain.ResetForTest)
	viper.Reset()
	t.Cleanup(viper.Reset)

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "aurora.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	_, err = db.Exec(`
		CREATE TABLE blocks (
			height INTEGER PRIMARY KEY,
			hash TEXT NOT NULL,
			previous_hash TEXT NOT NULL,
			data TEXT NOT NULL,
			nonce INTEGER NOT NULL,
			timestamp INTEGER NOT NULL,
			created_at INTEGER NOT NULL
		);
		INSERT INTO blocks VALUES (0, 'genesis', '', 'Genesis', X'626164', 1, 1);
	`)
	require.NoError(t, err)
	require.NoError(t, db.Close())
	viper.Set("db.path", dbPath)

	c, err := Open()

	require.Error(t, err)
	require.Nil(t, c)
	require.ErrorContains(t, err, "initialize blockchain")
	require.ErrorContains(t, err, "scan block row 1")
}

func TestCloseConcurrentRunsClosersOnce(t *testing.T) {
	const callers = 32
	wantErr := errors.New("close failed")
	var closerCalls atomic.Int32
	var closeEntered sync.Once
	entered := make(chan struct{})
	release := make(chan struct{})
	c := &Container{
		closers: []func() error{func() error {
			closerCalls.Add(1)
			closeEntered.Do(func() { close(entered) })
			<-release
			return wantErr
		}},
	}

	start := make(chan struct{})
	results := make(chan error, callers)
	var ready sync.WaitGroup
	ready.Add(callers)
	for range callers {
		go func() {
			ready.Done()
			<-start
			results <- c.Close()
		}()
	}
	ready.Wait()
	close(start)

	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Error("Close never entered the closer")
	}
	close(release)

	for range callers {
		require.ErrorIs(t, <-results, wantErr)
	}
	require.Equal(t, int32(1), closerCalls.Load(), "concurrent Close calls must share one close operation")
}

type trackingOracleFetcher struct {
	closeCalls atomic.Int32
}

func (*trackingOracleFetcher) FetchData(*oracle.DataSource) (*oracle.OracleData, error) {
	return nil, errors.New("not used")
}

func (f *trackingOracleFetcher) Close() error {
	f.closeCalls.Add(1)
	return nil
}

func TestContainerOwnsOracleFetcher(t *testing.T) {
	c := &Container{}
	fetcher := &trackingOracleFetcher{}

	c.ownOracleFetcher(fetcher)
	require.Same(t, fetcher, c.OracleFetcher)
	require.Equal(t, int32(0), fetcher.closeCalls.Load())

	require.NoError(t, c.Close())
	require.NoError(t, c.Close())
	require.Equal(t, int32(1), fetcher.closeCalls.Load(),
		"the container must own the fetcher lifecycle exactly once")
}
