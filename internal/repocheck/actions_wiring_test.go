package repocheck

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEveryActionFieldIsWired covers the seam between the window's view layer
// and the command that builds it. Actions is a struct of callbacks: adding a
// field is how the view asks for something new, and a field the command never
// fills is a nil callback the view calls on the frame that needs it. The view
// tests build their own Actions, so they would not notice.
func TestEveryActionFieldIsWired(t *testing.T) {
	actions := readRepoFile(t, "desktop", "internal", "nativeui", "actions.go")
	main := readRepoFile(t, "desktop", "cmd", "aurora-desktop", "main.go")

	fields := regexp.MustCompile(`(?m)^\t([A-Z]\w*)\s+(?:func|string|bool|int)\b`).
		FindAllStringSubmatch(actions, -1)
	require.Greater(t, len(fields), 20, "the Actions struct must be parsed")

	// The construction site, so a field named in a comment does not pass.
	start := strings.Index(main, "nativeui.Actions{")
	require.NotEqual(t, -1, start, "the command must build the Actions value")
	end := strings.Index(main[start:], "\n\t\t})")
	require.NotEqual(t, -1, end, "the Actions literal must be closed")
	literal := main[start : start+end]

	missing := []string{}
	for _, f := range fields {
		if !strings.Contains(literal, f[1]+":") {
			missing = append(missing, f[1])
		}
	}
	assert.Emptyf(t, missing, "Actions fields the command never sets: %v", missing)
}
