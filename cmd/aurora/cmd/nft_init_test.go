package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestNFTSubcommand_DoesNotPanicOnUnwritableDataDir is an end-to-end
// regression test for a critical bug:
//
// Previously, cmd/aurora/cmd/nft.go opened the SQLite DB in package
// init() and panic()'d on failure. That meant `aurora --help`,
// `aurora lottery history`, or any other unrelated subcommand would
// crash the entire binary with a stack trace when run from a directory
// where ./data cannot be created (read-only filesystem, permission
// denied, or — as we simulate here — a regular file at ./data).
//
// We can't easily reproduce this in-process (package init() runs before
// any test code can chdir), so we build the CLI as a subprocess and
// invoke it from a hostile cwd. Before the fix this panics; after the
// fix it returns a clean error.
func TestNFTSubcommand_DoesNotPanicOnUnwritableDataDir(t *testing.T) {
	// Compute repo root from this file's location: tests live in
	// cmd/aurora/cmd/, repo root is three levels up.
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))

	bin := filepath.Join(t.TempDir(), "aurora")
	build := exec.Command("go", "build", "-o", bin, "./cmd/aurora")
	build.Dir = repoRoot
	out, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}

	// Create a hostile cwd: ./data is a regular file, so any eager
	// MkdirAll("./data") inside NFT repo construction fails.
	hostile := t.TempDir()
	if err := os.WriteFile(filepath.Join(hostile, "data"), []byte("blocking"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}

	// 1) An unrelated subcommand must NOT panic — this is the original
	//    failure mode. Before the fix this panicked in init() with:
	//      "panic: failed to initialize NFT repository: ...
	//              mkdir data: not a directory"
	result := exec.Command(bin, "lottery", "history")
	result.Dir = hostile
	stdout, _ := result.CombinedOutput()
	if strings.Contains(string(stdout), "panic:") ||
		strings.Contains(string(stdout), "goroutine 1 [running]:") {
		t.Fatalf("regression: CLI panicked in init() on hostile cwd:\n%s", stdout)
	}

	// 2) An NFT subcommand must NOT panic; it must complete (either with
	//    success against an in-memory fallback DB or with a clean error).
	//    The point of this test is that it does NOT crash with a goroutine
	//    stack trace, regardless of what NFT-specific behaviour occurs.
	result2 := exec.Command(bin, "nft", "list", "--owner", "test")
	result2.Dir = hostile
	stdout2, _ := result2.CombinedOutput()
	combined := string(stdout2)
	if strings.Contains(combined, "panic:") ||
		strings.Contains(combined, "goroutine 1 [running]:") {
		t.Fatalf("regression: NFT subcommand panicked on hostile cwd:\n%s", combined)
	}
}
