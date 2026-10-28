package container

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

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
