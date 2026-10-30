package lottery

type Repository interface {
	Save(record *LotteryRecord) error
	GetByID(id string) (*LotteryRecord, error)
	// List returns one newest-first page. limit <= 0 returns the remainder
	// from offset, preserving whole-history reads for export and
	// verification.
	List(limit, offset int) ([]*LotteryRecord, error)
	// ListMatching returns one newest-first page whose id, seed, participants or
	// winners contain term, case-insensitively; an empty term behaves like List.
	// Searching in storage keeps a filtered page honest: filtering a loaded page
	// would report whichever matches happened to land on it.
	ListMatching(term string, limit, offset int) ([]*LotteryRecord, error)
	// Count returns the exact number of stored draws without loading them.
	Count() (int, error)
	GetAll() ([]*LotteryRecord, error)
	GetByBlockHeight(height int64) ([]*LotteryRecord, error)
}
