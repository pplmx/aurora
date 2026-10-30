package repocheck

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDocsOnlyMentionRealJustRecipes keeps the current docs honest about the
// task entry point: a recipe rename that leaves `just <old>` in README, AGENTS
// or docs points the reader at a command just cannot run. Only code spans and
// fenced blocks are scanned, because prose uses "just" as an ordinary word.
// Historical design records under docs/superpowers are excluded, like the CLI
// reference check.
func TestDocsOnlyMentionRealJustRecipes(t *testing.T) {
	root := repoRoot(t)
	recipes := justRecipeNames(t)
	require.NotEmpty(t, recipes, "the justfile must declare recipes")

	mentions := 0
	for _, docPath := range currentCLIDocs(t, root) {
		rel, err := filepath.Rel(root, docPath)
		require.NoError(t, err)
		for name, count := range justMentionsInDoc(t, docPath) {
			mentions += count
			require.Truef(t, recipes[name],
				"%s mentions `just %s`, but the justfile has no such recipe", rel, name)
		}
	}
	require.Greater(t, mentions, 5,
		"the scan must find recipe mentions, otherwise it proves nothing")
}

// justRecipeNames lists the top-level recipes. Attribute lines, assignments and
// settings are skipped; a recipe may take parameters, so only the first word
// before the colon is the name.
func justRecipeNames(t *testing.T) map[string]bool {
	t.Helper()
	names := make(map[string]bool)
	for _, line := range strings.Split(readRepoFile(t, "justfile"), "\n") {
		if line == "" || line[0] == ' ' || line[0] == '\t' {
			continue
		}
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") ||
			strings.HasPrefix(line, "set ") || strings.HasPrefix(line, "alias ") ||
			strings.HasPrefix(line, "export ") || strings.HasPrefix(line, "@") {
			continue
		}
		colon := strings.Index(line, ":")
		if colon < 0 || strings.HasPrefix(line[colon:], ":=") {
			continue
		}
		fields := strings.Fields(strings.TrimSpace(line[:colon]))
		if len(fields) == 0 || !bareWord.MatchString(fields[0]) {
			continue
		}
		names[fields[0]] = true
	}
	return names
}

var (
	bareWord   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)
	justInvoke = regexp.MustCompile(`\bjust\s+([a-z][a-z0-9_-]*)`)
)

// justMentionsInDoc counts `just <name>` inside code spans (or inside a fenced
// block), which is where the docs put commands to run.
func justMentionsInDoc(t *testing.T, path string) map[string]int {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	mentions := make(map[string]int)
	fenced := false
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fenced = !fenced
			continue
		}
		var texts []string
		if fenced {
			texts = append(texts, line)
		} else {
			texts = inlineCodeSpans(line)
		}
		for _, text := range texts {
			for _, m := range justInvoke.FindAllStringSubmatch(text, -1) {
				mentions[m[1]]++
			}
		}
	}
	return mentions
}

// inlineCodeSpans returns the single-backtick spans of one line.
func inlineCodeSpans(line string) []string {
	var spans []string
	for {
		open := strings.IndexByte(line, '`')
		if open < 0 {
			return spans
		}
		line = line[open+1:]
		closeAt := strings.IndexByte(line, '`')
		if closeAt < 0 {
			return spans
		}
		spans = append(spans, line[:closeAt])
		line = line[closeAt+1:]
	}
}
