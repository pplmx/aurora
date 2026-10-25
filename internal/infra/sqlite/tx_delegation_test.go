package sqlite

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/pplmx/aurora/internal/domain/oracle"
)

// These are the seams every other layer reaches through: the DB handle a
// transaction manager is built from, the path an export must refuse to
// overwrite, and the tx-scoped repository that makes a domain row and its
// audit block commit together. A regression here is invisible until a write
// silently escapes its transaction.

func TestLotteryRepository_DBPathAndHandle(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "lottery_path.db")
	repo, err := NewLotteryRepository(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	assert.Equal(t, dbPath, repo.DBPath(),
		"the repository must report the file it writes: export refuses to overwrite it")
	assert.NotNil(t, repo.GetDB())
	assert.NoError(t, repo.GetDB().Ping(), "the handle must be the live pool")
}

func TestLotteryRepository_WithTxCommitsAndRollsBack(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "lottery_tx.db")
	repo, err := NewLotteryRepository(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	record := &lottery.LotteryRecord{ID: "tx-commit", Seed: "seed", Participants: []string{"A"}}

	// A committed transaction is visible afterwards.
	require.NoError(t, repo.WithTransaction(func(tx *sql.Tx) error {
		return repo.WithTx(tx).Save(record)
	}))
	stored, err := repo.GetByID("tx-commit")
	require.NoError(t, err)
	require.NotNil(t, stored, "a committed write must be visible")

	// A failing transaction leaves nothing behind, which is what lets a domain
	// row and its audit block share one commit.
	boom := errors.New("roll back")
	err = repo.WithTransaction(func(tx *sql.Tx) error {
		if err := repo.WithTx(tx).Save(&lottery.LotteryRecord{
			ID: "tx-rollback", Seed: "seed", Participants: []string{"B"},
		}); err != nil {
			return err
		}
		return boom
	})
	require.ErrorIs(t, err, boom, "the callback's error must reach the caller")

	_, err = repo.GetByID("tx-rollback")
	assert.ErrorIs(t, err, lottery.ErrNotFound,
		"a rolled-back write must not be visible")
}

func TestOracleRepository_WithTxCommitsAndRollsBack(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	source := &oracle.DataSource{ID: "src-tx", Name: "feed", URL: "https://example.com"}
	require.NoError(t, repo.SaveSource(source))

	data := &oracle.OracleData{ID: "data-tx", SourceID: "src-tx", Value: "1", Timestamp: 1}
	require.NoError(t, repo.WithTransaction(func(tx *sql.Tx) error {
		return repo.WithTx(tx).SaveData(data)
	}))

	latest, err := repo.GetLatestData("src-tx")
	require.NoError(t, err)
	require.NotNil(t, latest, "a committed observation must be visible")
	assert.Equal(t, "data-tx", latest.ID)

	boom := errors.New("roll back")
	err = repo.WithTransaction(func(tx *sql.Tx) error {
		if err := repo.WithTx(tx).SaveData(&oracle.OracleData{
			ID: "data-rollback", SourceID: "src-tx", Value: "2", Timestamp: 2,
		}); err != nil {
			return err
		}
		return boom
	})
	require.ErrorIs(t, err, boom)

	latest, err = repo.GetLatestData("src-tx")
	require.NoError(t, err)
	require.NotNil(t, latest)
	assert.Equal(t, "data-tx", latest.ID,
		"the rolled-back observation must not become the latest")
}

func TestOracleRepository_DBHandle(t *testing.T) {
	repo, cleanup := setupOracleTestDB(t)
	defer cleanup()

	assert.NotNil(t, repo.GetDB())
	assert.NoError(t, repo.GetDB().Ping())
}
