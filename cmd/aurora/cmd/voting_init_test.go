package cmd

import (
	"os"
	"path/filepath"
	"runtime"
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
	// Reset the singletons so the concurrent calls actually
	// re-enter the constructor path.
	resetVotingForTest()

	// Use a temporary directory for the SQLite DB so the test
	// does not stomp on real data.
	tmp := t.TempDir()
	prev := os.Getenv("AURORA_DATA_DIR")
	t.Setenv("AURORA_DATA_DIR", filepath.Join(tmp, "data"))
	t.Cleanup(func() { _ = os.Setenv("AURORA_DATA_DIR", prev) })

	// chdir so blockchain.DBPath() resolves to the temp dir.
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("chdir(%q): %v", repoRoot, err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })

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
	resetVotingForTest()
}

// resetVotingForTest wipes the package-level voting singletons so
// tests can exercise the initialisation path repeatedly.
func resetVotingForTest() {
	votingRepo = nil
	votingService = nil
}
