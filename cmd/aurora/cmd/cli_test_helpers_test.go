package cmd

import (
	"bytes"
	"database/sql"
	"io"
	"os"
	"sync"
	"testing"

	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	oracleinfra "github.com/pplmx/aurora/internal/infra/sqlite"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// captureStdout redirects os.Stdout to a pipe and returns a function that,
// once called, restores the original stdout and returns everything written
// to the pipe while captured. The cobra commands in this package print via
// fmt.Println/fmt.Printf (package-level os.Stdout), not cmd.OutOrStdout,
// so tests must redirect the process stdout to observe their output.
func captureStdout(t *testing.T) func() string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	return func() string {
		t.Helper()
		_ = w.Close()
		os.Stdout = orig
		return <-done
	}
}

// resetCliForTest resets package-level state that could leak between
// tests: the shared oracle in-memory repo, the voting repo/service
// singletons, the NFT lazy-initialisation once, and the blockchain DB
// singletons. Call at the top of every CLI command test that touches DB
// or shared state.
//
// The oracle repo is a package-global (package var `repo` in oracle.go),
// so tests that add sources would otherwise pollute each other.
func resetCliForTest() {
	repo = *oracleinfra.NewInMemoryOracleRepository()

	// Voting singletons (voting.go) — reassign, not mutate in place.
	votingRepo = nil
	votingService = nil

	// NFT lazy-init (nft.go): the sync.Once can only fire once per process,
	// so a fresh Once must be installed for each test that opens the repo.
	nftRepo = nil
	nftRepoErr = nil
	nftRepoOnce = sync.Once{}

	// Blockchain DB + chain singletons; DBPath() resolves to ./data/aurora.db
	// relative to cwd, so callers must chdir into a temp dir first and invoke
	// this *after* chdir so ./data resolves inside the temp dir.
	blockchain.ResetForTest()

	resetFlags(rootCmd)
}

// resetFlags restores every flag on the command tree to its declared
// default and clears its "changed" bit. Without this, cobra's required-flag
// validation (MarkFlagRequired) leaks across tests in the same process: a
// flag set in one test's Execute() stays Changed=true, so a later test that
// omits it would not be rejected. Tests run sequentially, so resetting the
// tree between tests is safe.
func resetFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Persisting the declared default back clears values a prior test
		// set. pflag defaults: bool flags default to "false", others to
		// their registered default ("" for most string flags here).
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
	for _, c := range cmd.Commands() {
		if c.Name() == "help" || c.Name() == "completion" {
			continue
		}
		resetFlags(c)
	}
}

// withTempDir runs fn with the process cwd set to a fresh temp directory,
// restoring the previous cwd afterwards. CLI commands resolve their SQLite
// path as ./data/aurora.db relative to the process cwd, so giving each test
// its own cwd both isolates the DB and avoids touching the repo's own data/.
// Tests that use it must NOT call t.Parallel.
func withTempDir(t *testing.T, fn func(t *testing.T)) {
	t.Helper()

	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir(%q): %v", tmp, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			t.Errorf("restore cwd %q: %v", orig, err)
		}
	})

	// Reset AFTER chdir so blockchain.ResetForTest()'s os.RemoveAll("./data")
	// targets the temp dir, not repo data.
	resetCliForTest()

	fn(t)
}

// openTestAuroraDB opens ./data/aurora.db (the defaultDBPath resolved
// relative to the current test cwd) directly. Callers must be inside a
// withTempDir context so the path points at the isolated test DB.
func openTestAuroraDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", "./data/aurora.db")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
