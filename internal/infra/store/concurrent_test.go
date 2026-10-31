package store

import (
	"database/sql"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestOpenShared_ConcurrentNoLeak proves OpenShared is safe to call from
// multiple goroutines. Without a mutex guarding the dbInstance
// package variable, two callers can both observe nil, both call
// sql.Open, and leak the first connection (it never gets closed).
//
// With a race, two concurrent goroutines can both observe the shared
// handle as nil, both call sql.Open, and the assignment produces two
// different *sql.DB pointers. The test catches this by asserting all
// returned pointers are equal — the singleton contract.
func TestOpenShared_ConcurrentNoLeak(t *testing.T) {
	ResetForTest()

	const goroutines = 16
	var wg sync.WaitGroup
	dbs := make([]*sql.DB, goroutines)
	errs := make([]error, goroutines)
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			db, err := OpenShared()
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
		t.Fatal("OpenShared returned no valid DBs")
	}
	first := nonNil[0]
	for _, db := range nonNil[1:] {
		assert.Same(t, first, db, "OpenShared must return the same handle (race detected)")
	}
}
