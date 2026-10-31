package repocheck

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEveryConfigKeyAppearsInTheSample covers the file a reader copies to start.
// A key the loader honours but the sample never mentions is a setting nobody
// knows exists; the reverse is worse, because a key the sample shows and the
// code ignores silently does nothing.
func TestEveryConfigKeyAppearsInTheSample(t *testing.T) {
	sample := readRepoFile(t, "config", "aurora.toml")

	// Collect every key the code names through viper, wherever it lives: the
	// loader in internal/config and the CLI's own defaults both count.
	setter := regexp.MustCompile(`viper\.(?:SetDefault|Get|GetString|GetBool|GetInt|GetStringSlice)\("([a-z][a-zA-Z.]*)"`)
	codeKeys := map[string]bool{}
	for _, dir := range []string{"internal", "cmd"} {
		dir = filepath.Join(repoRoot(t), dir)
		require.NoError(t, walkGoSource(dir, func(path, body string) {
			if strings.HasSuffix(path, "_test.go") {
				return
			}
			for _, m := range setter.FindAllStringSubmatch(body, -1) {
				codeKeys[m[1]] = true
			}
		}))
	}
	require.Greater(t, len(codeKeys), 15, "the scan must find the configuration keys")

	// A key counts as documented when `[section]` precedes it and the leaf
	// appears in that section, commented out or not.
	section, documented := "", map[string]bool{}
	for _, line := range strings.Split(sample, "\n") {
		if m := regexp.MustCompile(`^\[([\w.]+)\]`).FindStringSubmatch(line); m != nil {
			section = m[1]
			continue
		}
		if m := regexp.MustCompile(`^#?\s*([a-zA-Z][a-zA-Z0-9_]*)\s*=`).FindStringSubmatch(line); m != nil && section != "" {
			documented[section+"."+m[1]] = true
		}
	}

	missing := []string{}
	for key := range codeKeys {
		if !documented[key] {
			missing = append(missing, key)
		}
	}
	assert.Emptyf(t, missing, "keys the sample config never mentions: %v", missing)
}
