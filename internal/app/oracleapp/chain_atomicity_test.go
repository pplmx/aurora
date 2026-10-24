package oracleapp

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/stretchr/testify/require"
)

type failingOracleSaveRepo struct {
	*sqlite.OracleRepository
	db *sql.DB
}

func (r *failingOracleSaveRepo) SaveData(*oracle.OracleData) error {
	return errors.New("forced oracle save failure")
}

func (r *failingOracleSaveRepo) WithTx(tx *sql.Tx) oracle.Repository {
	return &failingOracleTxRepo{Repository: r.OracleRepository.WithTx(tx)}
}

func (r *failingOracleSaveRepo) WithTransaction(fn func(tx *sql.Tx) error) error {
	return sqlite.NewTxManager(r.db).WithTransaction(fn)
}

type failingOracleTxRepo struct {
	oracle.Repository
}

func (*failingOracleTxRepo) SaveData(*oracle.OracleData) error {
	return errors.New("forced oracle save failure")
}

func TestFetchOracleDataRollbackLeavesNoOnChainBlock(t *testing.T) {
	repo, err := sqlite.NewOracleRepository(filepath.Join(t.TempDir(), "aurora.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	source := &oracle.DataSource{
		ID:       "source-1",
		Name:     "Source",
		Enabled:  true,
		Interval: 60,
	}
	require.NoError(t, repo.SaveSource(source))

	chain := blockchain.NewBlockChainWithDB(repo.GetDB())
	failingRepo := &failingOracleSaveRepo{OracleRepository: repo, db: repo.GetDB()}
	uc := NewFetchDataUseCaseWithDeps(failingRepo, &mockFetcher{
		data: &oracle.OracleData{
			ID:        "data-1",
			SourceID:  source.ID,
			Value:     "42",
			Timestamp: time.Now().Unix(),
		},
	})
	uc.SetChain(chain)

	_, err = uc.Execute(&FetchDataRequest{SourceID: source.ID})
	require.ErrorContains(t, err, "forced oracle save failure")
	require.Len(t, chain.Blocks, 1, "failed data save must not leave an in-memory block")

	var blockCount int
	require.NoError(t, repo.GetDB().QueryRow(`SELECT COUNT(*) FROM blocks WHERE height > 0`).Scan(&blockCount))
	require.Zero(t, blockCount, "failed data save must not persist a block")

	// The forced save failure must leave no oracle_data row behind. GetData
	// was removed as dead code, so assert the table directly.
	var storedRows int
	require.NoError(t, repo.GetDB().QueryRow(`SELECT COUNT(*) FROM oracle_data`).Scan(&storedRows))
	require.Zero(t, storedRows, "failed data save must not persist an oracle_data row")
}
