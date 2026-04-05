package lottery

type Repository interface {
	Save(record *LotteryRecord) error
	GetByID(id string) (*LotteryRecord, error)
	GetAll() ([]*LotteryRecord, error)
	GetByBlockHeight(height int64) ([]*LotteryRecord, error)
}
