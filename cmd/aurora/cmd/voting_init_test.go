package cmd

import (
	"os"
	"sync"
	"testing"
)

// TestVotingGetRepo_ConcurrentNoRace is a regression test for the
// bare-assignment race in cmd/aurora/cmd/voting.go:
//
//	func getVotingRepo() (voting.Repository, error) {
//	    if votingRepo != nil {                  // <-- racy read
//	        return votingRepo, nil
//	    }
//	    ...
//	    votingRepo = votingrepo.NewVotingRepository(db)  // <-- racy write
//	    return votingRepo, nil
//	}
//
// Two concurrent callers both observe votingRepo == nil, both
// construct a repository (one gets leaked / GC'd), and the
// assignment is undefined behaviour per the Go memory model.
//
// Run with `go test -race`. Pre-fix: race detector fires on the
// concurrent assignment. Post-fix (Round 24): clean.
func TestVotingGetRepo_ConcurrentNoRace(t *testing.T) {
	// chdir into a temp dir so blockchain.DBPath() (./data/aurora.db) resolves
	// to a scratch database, not the repo's committed data DB. (The previous
	// version chdir'd to the repo root while intending a temp dir, so the DB
	// path was never actually isolated.)
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	// Full reset AFTER chdir: resetCliForTest() re-arms the process-wide
	// blockchain DB singleton (blockchain.ResetForTest), which a prior test's
	// t.Cleanup(blockchain.Close()) leaves closed-but-never-re-armed — a
	// closed handle whose sync.Once already fired. Without the re-arm, the
	// concurrent InitDB below returns that stale closed DB and the voting
	// repo's schema bootstrap fails with "database is closed" (TASK-197).
	t.Cleanup(resetCliForTest)
	resetCliForTest()

	const goroutines = 32
	var wg sync.WaitGroup
	wg.Add(goroutines)
	errs := make([]error, goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = getVotingRepo()
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: getVotingRepo returned error: %v", i, err)
		}
	}

	if votingRepo == nil {
		t.Fatal("votingRepo is nil after concurrent getVotingRepo calls")
	}
}

// resetVotingForTest wipes the package-level voting singletons so
// tests can exercise the initialisation path repeatedly.
func resetVotingForTest() {
	votingRepo = nil
	votingService = nil
}
