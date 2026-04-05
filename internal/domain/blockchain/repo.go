package blockchain

type Repository interface {
	SaveBlock(height int64, block *Block) error
	GetBlock(height int64) (*Block, error)
	GetAllBlocks() ([]*Block, error)
	GetLotteryRecords() ([]string, error)
	AddLotteryRecord(data string) (int64, error)
	Close() error
}
