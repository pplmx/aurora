package repocheck

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCLIDocsMentionOnlyRegisteredFlags keeps the current docs from promising a
// flag no command registers: `aurora lottery reset --yes` was documented for a
// long time while only -y/--confirm existed, so the published reset command
// failed with "unknown flag". Only `aurora ...` code spans are read, so the
// desktop and API flags stay out of scope, and the global flag set is enough:
// this check catches removed or renamed flags, not one moved between commands.
func TestCLIDocsMentionOnlyRegisteredFlags(t *testing.T) {
	root := repoRoot(t)
	registered := cliLongFlags(t, filepath.Join(root, "cmd", "aurora", "cmd"))
	require.NotEmpty(t, registered, "the CLI must register flags")

	// Cobra adds these to every command without a registration in this tree.
	builtin := map[string]bool{"help": true, "version": true}
	mentions := 0
	for _, docPath := range currentCLIDocs(t, root) {
		rel, err := filepath.Rel(root, docPath)
		require.NoError(t, err)
		for _, ref := range cliFlagMentionsInDoc(t, docPath) {
			mentions++
			if builtin[ref.flag] {
				continue
			}
			require.Truef(t, registered[ref.flag],
				"%s:%d documents `--%s` on an aurora command, but no command registers it",
				filepath.ToSlash(rel), ref.line, ref.flag)
		}
	}
	require.Greater(t, mentions, 5,
		"the scan must find flags, otherwise it proves nothing")
}

// cliLongFlags collects the long name of every flag registered in dir.
func cliLongFlags(t *testing.T, dir string) map[string]bool {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	flags := make(map[string]bool)
	registration := regexp.MustCompile(`(?:Persistent)?Flags\(\)\.[A-Za-z]+\(\s*"([a-z][a-z0-9-]*)"`)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)
		for _, m := range registration.FindAllStringSubmatch(string(data), -1) {
			flags[m[1]] = true
		}
	}
	return flags
}

type flagMention struct {
	line int
	flag string
}

var docFlag = regexp.MustCompile(`(?:^|[^A-Za-z0-9_-])--([a-z][a-z0-9-]*)`)

// cliFlagMentionsInDoc finds `--flag` inside `aurora ...` code spans and
// fenced blocks, the places the docs put commands to run.
func cliFlagMentionsInDoc(t *testing.T, path string) []flagMention {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var mentions []flagMention
	fenced := false
	for i, line := range strings.Split(string(data), "\n") {
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
			command := strings.TrimSpace(text)
			command = strings.TrimPrefix(command, "$ ")
			command = strings.TrimPrefix(command, "./")
			if !strings.HasPrefix(command, "aurora ") {
				continue
			}
			for _, m := range docFlag.FindAllStringSubmatch(command, -1) {
				mentions = append(mentions, flagMention{line: i + 1, flag: m[1]})
			}
		}
	}
	return mentions
}
