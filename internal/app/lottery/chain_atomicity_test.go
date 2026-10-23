package lottery

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/pplmx/aurora/internal/domain/blockchain"
	domainlottery "github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/stretchr/testify/require"
)

type failingLotterySaveRepo struct {
	*sqlite.LotteryRepository
	db *sql.DB
}

func (r *failingLotterySaveRepo) Save(*domainlottery.LotteryRecord) error {
	return errors.New("forced lottery save failure")
}

func (r *failingLotterySaveRepo) WithTx(tx *sql.Tx) domainlottery.Repository {
	return &failingLotteryTxRepo{Repository: r.LotteryRepository.WithTx(tx)}
}

func (r *failingLotterySaveRepo) WithTransaction(fn func(tx *sql.Tx) error) error {
	return sqlite.NewTxManager(r.db).WithTransaction(fn)
}

type failingLotteryTxRepo struct {
	domainlottery.Repository
}

func (*failingLotteryTxRepo) Save(*domainlottery.LotteryRecord) error {
	return errors.New("forced lottery save failure")
}

func TestCreateLotteryRollbackLeavesNoOnChainBlock(t *testing.T) {
	repo, err := sqlite.NewLotteryRepository(filepath.Join(t.TempDir(), "aurora.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	chain := blockchain.NewBlockChainWithDB(repo.GetDB())
	failingRepo := &failingLotterySaveRepo{LotteryRepository: repo, db: repo.GetDB()}
	uc := NewCreateLotteryUseCase(failingRepo, chain)

	_, err = uc.Execute(CreateLotteryRequest{
		Participants: "Alice,Bob,Charlie",
		Seed:         "atomicity",
		WinnerCount:  1,
	})
	require.ErrorContains(t, err, "forced lottery save failure")
	require.Len(t, chain.Blocks, 1, "failed record save must not leave an in-memory block")

	var blockCount int
	require.NoError(t, repo.GetDB().QueryRow(`SELECT COUNT(*) FROM blocks WHERE height > 0`).Scan(&blockCount))
	require.Zero(t, blockCount, "failed record save must not persist a block")

	records, err := repo.GetAll()
	require.NoError(t, err)
	require.Empty(t, records)
}
