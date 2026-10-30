package repocheck

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestJustRecipesShowOneLineDescriptions keeps `just --list` readable. just
// prints only the last comment line above a recipe as its description, so a
// long explanation with no closing summary turned the list into fragments like
// "the two recipes disagree about formatting map literals."
func TestJustRecipesShowOneLineDescriptions(t *testing.T) {
	lines := strings.Split(readRepoFile(t, "justfile"), "\n")
	count := 0
	for i, line := range lines {
		name, ok := justRecipeOn(line)
		if !ok {
			continue
		}
		count++
		desc := ""
		if i > 0 {
			above := strings.TrimSpace(lines[i-1])
			if strings.HasPrefix(above, "#") {
				desc = strings.TrimSpace(strings.TrimPrefix(above, "#"))
			}
		}
		require.NotEmptyf(t, desc,
			"recipe %q has no description; just --list would show a bare name", name)
		require.LessOrEqualf(t, len([]rune(desc)), 72,
			"recipe %q description is longer than one line: %q", name, desc)
		first := []rune(desc)[0]
		require.Truef(t, first >= 'A' && first <= 'Z',
			"recipe %q description must start with a capital letter, not a sentence fragment: %q",
			name, desc)
	}
	require.GreaterOrEqual(t, count, 20, "the scan must see the justfile's recipes")
}

// justRecipeOn reports whether line is a recipe header and returns its name.
// Assignments, attributes, settings and comment lines are not recipes, and
// recipe bodies are indented.
func justRecipeOn(line string) (string, bool) {
	if line == "" || line[0] == ' ' || line[0] == '\t' || line[0] == '#' {
		return "", false
	}
	if strings.Contains(line, ":=") || strings.HasPrefix(line, "[") ||
		strings.HasPrefix(line, "set ") || strings.HasPrefix(line, "alias ") ||
		strings.HasPrefix(line, "export ") || strings.HasPrefix(line, "@") {
		return "", false
	}
	colon := strings.Index(line, ":")
	if colon <= 0 {
		return "", false
	}
	fields := strings.Fields(strings.TrimSpace(line[:colon]))
	if len(fields) == 0 || !bareWord.MatchString(fields[0]) {
		return "", false
	}
	return fields[0], true
}
