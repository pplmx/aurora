package sqlite

import (
	"github.com/pplmx/aurora/internal/domain/lottery"
	"github.com/pplmx/aurora/internal/domain/nft"
	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/pplmx/aurora/internal/domain/token"
	"github.com/pplmx/aurora/internal/domain/voting"
)

// Compile-time proof that each SQLite repository satisfies the domain
// interface it is handed out as.
//
// Without these, a method removed from an interface (or renamed on one side)
// only surfaces where the concrete type happens to be assigned to the
// interface — usually one wiring line in api.NewServer, far from the change.
// Declaring the assertion beside the implementation makes the package that
// must change fail first.
var (
	_ lottery.Repository = (*LotteryRepository)(nil)
	_ voting.Repository  = (*VotingRepository)(nil)
	_ nft.Repository     = (*NFTRepository)(nil)
	_ token.Repository   = (*TokenRepository)(nil)
	_ oracle.Repository  = (*OracleRepository)(nil)
)

// The transactional variants are what the use cases accept when a write must be
// scoped to one transaction, so they are asserted separately: an implementation
// can satisfy the read interface and still miss WithTx.
var (
	_ voting.TransactableRepository = (*VotingRepository)(nil)
	_ nft.TransactableRepository    = (*NFTRepository)(nil)
	_ token.TransactableRepository  = (*TokenRepository)(nil)
)
