package cmd

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEveryCliTranslationCanBeReached keeps the CLI's translation table honest.
// An entry whose sentinel no code returns is a message a user can never see, and
// the table's size suggests support that is not there.
//
// The HTTP table has the same check; this one exists because the two tables list
// different sets, and nothing compared either against the code that returns the
// errors.
func TestEveryCliTranslationCanBeReached(t *testing.T) {
	// Classified but never produced: the token write guards are not wired to a
	// caller, and the signature sentinels never fire because token and nft report
	// a bad signature as a false verification result rather than an error.
	reserved := map[string]bool{
		"token.ErrNotTokenOwner":     true,
		"token.ErrDuplicateTransfer": true,
		"token.ErrInvalidSignature":  true,
		"nft.ErrInvalidSignature":    true,
		// nft reports an uninitialised store through a nil-receiver guard that
		// returns before the service can name it.
		"nft.ErrStorageNotInit": true,
	}

	entries := cliSentinelNames(t)
	require.Greater(t, len(entries), 70, "the table must be parsed")

	repo, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)

	unreachable := []string{}
	for _, e := range entries {
		if reserved[e.pkg+"."+e.name] {
			continue
		}
		if !cliSentinelReturned(repo, e.pkg, e.name) {
			unreachable = append(unreachable, e.pkg+"."+e.name)
		}
	}
	assert.Emptyf(t, unreachable,
		"these translations name a sentinel no code returns: %v", unreachable)
}

type cliSentinel struct {
	pkg, name string
}

// cliSentinelNames reads the qualification out of the table's source: the entries
// hold error values, so the package and identifier exist only as text.
func cliSentinelNames(t *testing.T) []cliSentinel {
	t.Helper()
	body, err := os.ReadFile("error.go")
	require.NoError(t, err)
	text := string(body)
	start := strings.Index(text, "var cliErrorTranslations")
	require.NotEqual(t, -1, start, "the table must be where this test expects")
	end := strings.Index(text[start:], "\n}\n")
	require.NotEqual(t, -1, end, "the table must be closed")

	out := []cliSentinel{}
	for _, m := range regexp.MustCompile(`\{(\w+)\.(Err\w+),\s*"error\.`).
		FindAllStringSubmatch(text[start:start+end], -1) {
		out = append(out, cliSentinel{pkg: m[1], name: m[2]})
	}
	return out
}

// cliSentinelReturned reports whether a production file can produce the sentinel.
// A file in the declaring package may name it bare; elsewhere the qualifier is
// what identifies it, so the bare form does not count there.
func cliSentinelReturned(repo, pkg, name string) bool {
	qualified := regexp.MustCompile(`\b` + pkg + `\.` + name + `\b`)
	bare := regexp.MustCompile(`\b` + name + `\b`)
	declDir := packageDir(repo, pkg)

	found := false
	for _, root := range []string{"internal", "cmd"} {
		_ = filepath.WalkDir(filepath.Join(repo, root), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil || found {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			base := filepath.Base(path)
			// Tables name every sentinel, and a declaration is not a use.
			if base == "helpers.go" || strings.HasPrefix(base, "errors") {
				return nil
			}
			if strings.HasSuffix(path, filepath.Join("cmd", "error.go")) {
				return nil
			}
			body, readErr := os.ReadFile(path) //nolint:gosec // path comes from the repo walk
			if readErr != nil {
				return readErr
			}
			text := string(body)
			if qualified.MatchString(text) {
				found = true
				return nil
			}
			if strings.HasPrefix(path, declDir+string(filepath.Separator)) && bare.MatchString(text) {
				found = true
			}
			return nil
		})
	}
	return found
}

// packageDir finds the directory that declares pkg. The tables name sentinels
// from the domain, the application and the ledger layers, so the declaring
// directory cannot be assumed.
func packageDir(repo, pkg string) string {
	for _, layer := range []string{"domain", "app", "infra"} {
		candidate := filepath.Join(repo, "internal", layer, pkg)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return filepath.Join(repo, "internal", "domain", pkg)
}
