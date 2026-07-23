package blockchain

import (
	"database/sql"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestInitDB_ConcurrentNoLeak proves InitDB is safe to call from
// multiple goroutines. Without a mutex guarding the dbInstance
// package variable, two callers can both observe nil, both call
// sql.Open, and leak the first connection (it never gets closed).
//
// With a race, two concurrent goroutines can both observe dbInstance
// as nil, both call sql.Open, and the assignment "dbInstance = db"
// produces two different *sql.DB pointers. The test catches this
// by asserting all returned pointers are equal — the singleton
// contract.
func TestInitDB_ConcurrentNoLeak(t *testing.T) {
	ResetForTest()

	const goroutines = 16
	var wg sync.WaitGroup
	dbs := make([]*sql.DB, goroutines)
	errs := make([]error, goroutines)
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			db, err := InitDB()
			dbs[idx] = db
			errs[idx] = err
		}(i)
	}
	wg.Wait()

	var nonNil []*sql.DB
	for i, db := range dbs {
		if assert.NoError(t, errs[i]) {
			if db != nil {
				nonNil = append(nonNil, db)
			}
		}
	}
	if len(nonNil) == 0 {
		t.Fatal("InitDB returned no valid DBs")
	}
	first := nonNil[0]
	for _, db := range nonNil[1:] {
		assert.Same(t, first, db, "InitDB must return the same singleton (race detected)")
	}
}
