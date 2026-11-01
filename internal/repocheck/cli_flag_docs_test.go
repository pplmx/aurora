package repocheck

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCLIDocsFlagsExist closes a gap the path check leaves open. That test stops
// at the first word that looks like a value, so it never reaches a flag: a doc
// line may name `-x` where the command registers `-s` and nothing fails.
//
// The mapping is built from the source, keyed by the command variable each
// Flags() call hangs off, and the doc lines are matched to it by their command
// path.
func TestCLIDocsFlagsExist(t *testing.T) {
	root := repoRoot(t)
	srcDir := filepath.Join(root, "cmd", "aurora", "cmd")

	// varByCommand: cobra command variable -> the path it serves.
	varByCommand := cobraCommandVariables(t, srcDir)
	require.Greater(t, len(varByCommand), 20, "the command tree must be parsed")

	// flagsByCommand: variable -> every long flag and shorthand it registers.
	flagCall := regexp.MustCompile(`(\w+)\.Flags\(\)\.(\w+?)P?\("([^"]+)"(?:,\s*"([^"]+)")?`)
	// A helper that takes the command as an argument registers its flags there,
	// so the variable appears at the call site rather than before Flags().
	helperCall := regexp.MustCompile(`addConfirmFlag\((\w+),`)
	flagsByCommand := map[string]map[string]bool{}
	for _, file := range goFiles(t, srcDir) {
		body, err := os.ReadFile(file) //nolint:gosec // path comes from the repo walk
		require.NoError(t, err)
		text := string(body)
		// addConfirmFlag registers --confirm/-y on the command it is given.
		for _, m := range helperCall.FindAllStringSubmatch(text, -1) {
			v := m[1]
			if flagsByCommand[v] == nil {
				flagsByCommand[v] = map[string]bool{}
			}
			flagsByCommand[v]["--confirm"] = true
			flagsByCommand[v]["-y"] = true
		}
		for _, m := range flagCall.FindAllStringSubmatch(text, -1) {
			v, name, short := m[1], m[3], m[4]
			if flagsByCommand[v] == nil {
				flagsByCommand[v] = map[string]bool{}
			}
			flagsByCommand[v]["--"+name] = true
			if short != "" {
				flagsByCommand[v]["-"+short] = true
			}
		}
	}
	require.Greater(t, len(flagsByCommand), 15, "flags must be found")

	// pathFlags turns the variable map into "aurora a b" -> flags. The root
	// command's own Use is "aurora", so its children already carry that word and
	// adding it again would key every entry as "aurora aurora ...".
	pathFlags := map[string]map[string]bool{}
	for v, path := range varByCommand {
		f, ok := flagsByCommand[v]
		if !ok {
			continue
		}
		if !strings.HasPrefix(path, "aurora") {
			path = "aurora " + path
		}
		pathFlags[path] = f
	}

	checked := 0
	for _, docPath := range currentCLIDocs(t, root) {
		body, err := os.ReadFile(docPath)
		require.NoError(t, err)
		rel, err := filepath.Rel(root, docPath)
		require.NoError(t, err)

		for lineNumber, line := range strings.Split(string(body), "\n") {
			trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "$ "))
			if !strings.HasPrefix(trimmed, "aurora ") || !strings.Contains(trimmed, "-") {
				continue
			}
			// Drop a trailing prose comment before reading the arguments.
			code := regexp.MustCompile(`\s{2,}#.*$`).ReplaceAllString(trimmed, "")

			path, flags := splitCommandAndFlags(code)
			if len(flags) == 0 {
				continue
			}
			want, ok := pathFlags[path]
			if !ok {
				continue // the path check covers an unknown path
			}
			for _, flag := range flags {
				checked++
				assert.Truef(t, want[flag],
					"%s:%d documents %q on %q, which does not register it",
					rel, lineNumber+1, flag, path)
			}
		}
	}
	require.Greater(t, checked, 25, "the scan must reach the documented flags")
}

// splitCommandAndFlags returns the command path and the flags a doc line uses.
// A word starting with '-' is a flag; anything else continues the path until the
// first flag is seen.
func splitCommandAndFlags(line string) (string, []string) {
	path := []string{}
	flags := []string{}
	for _, w := range strings.Fields(line) {
		if w == "aurora" {
			continue
		}
		if strings.HasPrefix(w, "--") {
			flags = append(flags, strings.SplitN(w, "=", 2)[0])
			continue
		}
		if strings.HasPrefix(w, "-") && len(w) == 2 && !looksLikeValue(w) {
			flags = append(flags, w)
			continue
		}
		// The path ends at the first flag or the first value.
		if len(flags) > 0 || looksLikeValue(w) {
			continue
		}
		path = append(path, w)
	}
	return "aurora " + strings.Join(path, " "), flags
}

// looksLikeValue reports whether the word is an example argument rather than a
// command name: quoted, bracketed, numeric, or a path.
func looksLikeValue(w string) bool {
	if w == "" {
		return true
	}
	for _, p := range []string{`"`, "'", "<", "./", "/", "~", "$", "#"} {
		if strings.HasPrefix(w, p) {
			return true
		}
	}
	return regexp.MustCompile(`^\d+$`).MatchString(w)
}

// cobraCommandVariables maps each command variable to the path it serves.
func cobraCommandVariables(t *testing.T, dir string) map[string]string {
	t.Helper()
	useRe := regexp.MustCompile(`(\w+)\s*=\s*&cobra\.Command\{`)
	useLine := regexp.MustCompile(`Use:\s*"([^"]+)"`)
	addRe := regexp.MustCompile(`(\w+)\.AddCommand\((\w+)\)`)

	useOf := map[string]string{}
	parentOf := map[string]string{}
	for _, file := range goFiles(t, dir) {
		body, err := os.ReadFile(file) //nolint:gosec // path comes from the repo walk
		require.NoError(t, err)
		text := string(body)
		for _, m := range useRe.FindAllStringSubmatchIndex(text, -1) {
			v := text[m[2]:m[3]]
			// The Use field is inside the literal that follows.
			rest := text[m[1]:]
			if u := useLine.FindStringSubmatch(rest); u != nil {
				useOf[v] = strings.Fields(u[1])[0]
			}
		}
		for _, m := range addRe.FindAllStringSubmatch(text, -1) {
			parentOf[m[2]] = m[1]
		}
	}

	out := map[string]string{}
	for v, use := range useOf {
		parts := []string{use}
		for cur := v; ; {
			p, ok := parentOf[cur]
			if !ok {
				break
			}
			parts = append([]string{useOf[p]}, parts...)
			cur = p
		}
		out[v] = strings.Join(parts, " ")
	}
	return out
}

// goFiles lists the non-test Go files below dir.
func goFiles(t *testing.T, dir string) []string {
	t.Helper()
	var files []string
	require.NoError(t, walkGoSource(dir, func(path, _ string) {
		files = append(files, path)
	}))
	return files
}
