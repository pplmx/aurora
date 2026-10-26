package container

import (
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
	require.NotNil(t, c.NFTTx)
	require.NotNil(t, c.Chain)
	require.NotNil(t, c.EventStore)
	require.NotNil(t, c.OracleFetcher)

	require.NoError(t, c.Close())
	require.NoError(t, c.Close(), "Close must be idempotent for shutdown paths")
}
