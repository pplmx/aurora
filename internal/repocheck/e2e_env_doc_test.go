package repocheck

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTheCaptureScriptsVariablesAreDocumented keeps the capture guide usable.
// The script is the only place that lists the switches, and a reader following
// docs/desktop.md meets a run that writes somewhere unexpected, or fails to
// start because it wanted a display it was never told about.
func TestTheCaptureScriptsVariablesAreDocumented(t *testing.T) {
	script := readRepoFile(t, "desktop", "scripts", "e2e-window.sh")
	doc := readRepoFile(t, "docs", "desktop.md")

	vars := map[string]bool{}
	for _, m := range regexp.MustCompile(`AURORA_E2E_[A-Z_]+`).FindAllString(script, -1) {
		vars[m] = true
	}
	require.Greater(t, len(vars), 4, "the script must declare its switches")

	missing := []string{}
	for name := range vars {
		if !strings.Contains(doc, name) {
			missing = append(missing, name)
		}
	}
	assert.Emptyf(t, missing, "capture switches docs/desktop.md never mentions: %v", missing)
}
