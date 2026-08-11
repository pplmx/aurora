package cmd

import (
	"bytes"
	"crypto/ed25519"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"unsafe"

	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
	"github.com/pplmx/aurora/internal/infra/migrate"
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
		//
		// Slice flags (e.g. --candidates) cannot be reset via Value.Set:
		// once parsed, pflag's stringSliceValue keeps an internal "changed"
		// bit and every subsequent Set() APPENDS. We clear the backing
		// slice directly instead — see resetSliceFlag.
		if isSliceType(f.Value.Type()) {
			resetSliceFlag(f)
		} else {
			_ = f.Value.Set(f.DefValue)
		}
		f.Changed = false
	})
	for _, c := range cmd.Commands() {
		if c.Name() == "help" || c.Name() == "completion" {
			continue
		}
		resetFlags(c)
	}
}

// isSliceType reports whether a pflag value type holds a slice (stringSlice,
// intSlice, ...). Such values cannot be reset via their registered DefValue.
func isSliceType(t string) bool {
	return strings.HasSuffix(t, "Slice") || strings.HasSuffix(t, "Array")
}

// resetSliceFlag clears a pflag slice value back to empty.
//
// pflag's slice values are unexported structs with a backing pointer and an
// internal "changed" bool; once Set() is called the changed bit latches and
// every later Set() APPENDS instead of replacing. Calling Value.Set("[]") or
// Value.Set("") then leaves stale elements behind, leaking across tests in
// the same process.
//
// pflag v1.0.10 layout:
//
//	type stringSliceValue struct {
//		value   *[]string
//		changed bool
//	}
//
// Both fields are unexported, so a plain reflect.Value.Set is refused. We
// use reflect.NewAt + unsafe.Pointer to obtain addressable views of the two
// fields (a well-contained technique for reaching locked-down structs in
// test helpers; the layout is pinned by go.mod's pflag version).
func resetSliceFlag(f *pflag.Flag) {
	rv := reflect.ValueOf(f.Value)
	if rv.Kind() != reflect.Ptr {
		return
	}
	elem := rv.Elem()
	if elem.Kind() != reflect.Struct {
		return
	}

	// value *[]string -> rebuild the backing slice to empty.
	valueField := elem.FieldByName("value")
	if valueField.IsValid() && valueField.Kind() == reflect.Ptr {
		ptr := reflect.NewAt(valueField.Type(), unsafe.Pointer(valueField.UnsafeAddr()))
		p := ptr.Elem()
		if p.Elem().Kind() == reflect.Slice {
			p.Elem().Set(reflect.MakeSlice(p.Elem().Type(), 0, 0))
		}
	}

	// changed bool -> false so the next real Set() replaces rather than
	// appends.
	changedField := elem.FieldByName("changed")
	if changedField.IsValid() && changedField.Kind() == reflect.Bool {
		ptr := reflect.NewAt(changedField.Type(), unsafe.Pointer(changedField.UnsafeAddr()))
		ptr.Elem().SetBool(false)
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

// newTestPrivKey returns a fresh random Ed25519 private key (compressed
// seed+pub form) for negative-path tests that need a key that does NOT
// match the token owner.
func newTestPrivKey(t *testing.T) []byte {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return priv
}

// extractKey reads the base64 key that appears on the line AFTER the line
// containing marker, e.g. voter register prints:
//
//	📣 Public Key (share this for verification):
//	   <base64>
//
// and
//
//	🔐 Private Key (SAVE THIS SECURELY!):
//	   <base64>
func extractKey(t *testing.T, out, marker string) string {
	t.Helper()
	lines := strings.Split(out, "\n")
	for i, line := range lines {
		if strings.Contains(line, marker) {
			if i+1 < len(lines) {
				return strings.TrimSpace(lines[i+1])
			}
		}
	}
	return ""
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

// runMigrations applies the repo's real SQL migrations to ./data/aurora.db
// (relative to the current temp cwd). This validates that the checkout's
// migration files actually apply, and gives CLI tests a properly-initialised
// DB (the voting subcommands' tables come from migrations/000001).
func runMigrations(t *testing.T) {
	t.Helper()
	m, err := migrate.New("./data/aurora.db", repoMigrationsDir())
	if err != nil {
		t.Fatalf("create migrator: %v", err)
	}
	t.Cleanup(func() { _ = m.Close() })
	if _, err := m.Up(0); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
}

// repoMigrationsDir returns the absolute path to the repo's migrations/
// directory, derived from the test file's location rather than the (temp)
// process cwd.
func repoMigrationsDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "migrations")
}
