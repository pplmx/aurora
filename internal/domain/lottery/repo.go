package lottery

type Repository interface {
	Save(record *LotteryRecord) error
	GetByID(id string) (*LotteryRecord, error)
	// List returns one newest-first page. limit <= 0 returns the remainder
	// from offset, preserving whole-history reads for export and
	// verification.
	List(limit, offset int) ([]*LotteryRecord, error)
	GetAll() ([]*LotteryRecord, error)
	GetByBlockHeight(height int64) ([]*LotteryRecord, error)
}
