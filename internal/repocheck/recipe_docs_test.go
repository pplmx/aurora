package repocheck

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTheReadmeListsEveryRecipe keeps the command list and the justfile in step.
// A recipe added for a reason the reader needs — `check-verify` exists so
// `verify` does not rewrite the tree — is invisible if only the justfile knows
// about it, and the reader then runs the one that writes.
func TestTheReadmeListsEveryRecipe(t *testing.T) {
	justfile := readRepoFile(t, "justfile")
	readme := readRepoFile(t, "README.md")

	recipes := []string{}
	for _, m := range regexp.MustCompile(`(?m)^([a-z][a-z0-9-]*):`).FindAllStringSubmatch(justfile, -1) {
		// `default` prints the list; it is not a command a reader runs.
		if m[1] != "default" {
			recipes = append(recipes, m[1])
		}
	}
	require.Greater(t, len(recipes), 20, "the justfile must be parsed")

	missing := []string{}
	for _, name := range recipes {
		if !strings.Contains(readme, "just "+name) {
			missing = append(missing, name)
		}
	}
	assert.Emptyf(t, missing, "recipes the README never mentions: %v", missing)
}
