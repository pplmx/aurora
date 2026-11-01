package handler

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

// TestEveryClassifiedErrorCanBeReturned keeps the classification table honest.
// An entry whose sentinel no code ever returns is a mapping nobody can reach: the
// reference lists the status and code as supported, and no request produces them.
//
// voting.ErrInvalidPrivateKey was exactly that until the voting service started
// returning it, which is why this check exists. The two codes below are
// documented as reserved, so they are the only permitted exceptions.
//
// The match is on package plus name, not on message text: token, nft and voting
// each declare an ErrInvalidPrivateKey with the identical message, so text alone
// cannot say which one a file returns.
func TestEveryClassifiedErrorCanBeReturned(t *testing.T) {
	// Defined and classified, but no path produces them: the token and nft
	// services report a bad signature as a false verification result rather than
	// an error, and the two write guards below are not wired to a caller.
	reserved := map[string]bool{
		"NOT_TOKEN_OWNER":    true,
		"DUPLICATE_TRANSFER": true,
	}
	// INVALID_SIGNATURE is classified for both token and nft.
	reservedSignatures := map[string]bool{"token.ErrInvalidSignature": true, "nft.ErrInvalidSignature": true}

	repo, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)

	entries := classifiedSentinels(t)
	require.Greater(t, len(entries), 60, "the table must be parsed")

	unreachable := []string{}
	for _, e := range entries {
		if reserved[e.code] || reservedSignatures[e.pkg+"."+e.name] {
			continue
		}
		if !returnsSentinel(repo, e.pkg, e.name) {
			unreachable = append(unreachable, e.pkg+"."+e.name+" ("+e.code+")")
		}
	}
	assert.Emptyf(t, unreachable,
		"these codes are classified but no code returns their sentinel: %v", unreachable)
}

type classified struct {
	pkg, name, code string
}

// classifiedSentinels reads the table's entries from source. The values are
// errors, so the package and identifier only exist in the text.
func classifiedSentinels(t *testing.T) []classified {
	t.Helper()
	body, err := os.ReadFile("helpers.go")
	require.NoError(t, err)
	start := strings.Index(string(body), "var errorClassification")
	require.NotEqual(t, -1, start, "the table must be where this test expects")
	end := strings.Index(string(body)[start:], "\n}\n")
	require.NotEqual(t, -1, end, "the table must be closed")
	block := string(body)[start : start+end]

	out := []classified{}
	for _, m := range regexp.MustCompile(`\{(\w+)\.(Err\w+),\s*http\.Status\w+,\s*"([A-Z_]+)"\}`).
		FindAllStringSubmatch(block, -1) {
		out = append(out, classified{pkg: m[1], name: m[2], code: m[3]})
	}
	return out
}

// returnsSentinel reports whether any production file can produce the sentinel.
// A file in the declaring package may name it bare; elsewhere the package
// qualifier is what identifies it, so the bare form is not enough there.
func returnsSentinel(repo, pkg, name string) bool {
	qualified := regexp.MustCompile(`\b` + pkg + `\.` + name + `\b`)
	bare := regexp.MustCompile(`\b` + name + `\b`)
	declDir := ""

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
			// Two files are tables, not producers: the HTTP classification this
			// test reads, and the CLI's matching table. A declaration is also not
			// a use.
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
			if declDir == "" {
				declDir = filepath.Join(repo, "internal", "domain", pkg)
			}
			if strings.HasPrefix(path, declDir+string(filepath.Separator)) && bare.MatchString(text) {
				found = true
			}
			return nil
		})
	}
	return found
}
