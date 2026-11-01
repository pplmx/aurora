package repocheck

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEveryTimestampIsUnixSeconds covers a unit that a reader can only learn
// from prose: every timestamp field in the API is an int64 of seconds, and a
// client that assumes milliseconds renders 1970 instead of failing. Only the
// voting session fields said so; timestamp, created_at and the oracle reading
// times did not.
//
// The check reads the source, because the unit is a property of how the values
// are built rather than of the type: an int64 could hold either.
func TestEveryTimestampIsUnixSeconds(t *testing.T) {
	// The builders must call Unix(), and nothing may call UnixMilli/UnixNano.
	sources := 0
	require.NoError(t, walkGoSource(filepath.Join(repoRoot(t), "internal"), func(path, body string) {
		for _, forbidden := range []string{"UnixMilli(", "UnixNano("} {
			assert.NotContainsf(t, body, forbidden,
				"%s builds a sub-second timestamp; every timestamp field is unix seconds", path)
		}
		sources += strings.Count(body, ".Unix()")
	}))
	require.Greater(t, sources, 10, "the scan must reach the timestamp builders")

	// The reference must state the unit once, where a reader planning a client
	// meets it, rather than repeating it beside every field.
	doc := readRepoFile(t, "docs", "api.md")
	require.Contains(t, doc, "时间戳一律是 unix 秒",
		"docs/api.md must state the unit every timestamp field uses")
	for _, field := range []string{"timestamp", "created_at", "start_time", "end_time"} {
		assert.Containsf(t, doc, "`"+field+"`",
			"the paragraph must name %s among the fields it covers", field)
	}
	assert.Regexp(t, regexp.MustCompile(`毫秒|纳秒`), doc,
		"the paragraph must say what the fields are not, since a reader's default guess is milliseconds")

	// A documented sample that happens to be in milliseconds would contradict
	// the paragraph. Every epoch-looking number in a JSON sample is checked
	// against a plausible second range.
	samples := regexp.MustCompile(`"(timestamp|created_at|start_time|end_time)":(\d+)`).
		FindAllStringSubmatch(doc, -1)
	require.NotEmpty(t, samples, "docs/api.md carries timestamped samples")
	for _, m := range samples {
		value, err := strconv.ParseInt(m[2], 10, 64)
		require.NoError(t, err)
		// 2001-09-09 .. 2286-11-20 in seconds. A millisecond value for any date
		// this project can produce is 13 digits and far above this.
		assert.Truef(t, value > 1_000_000_000 && value < 10_000_000_000,
			"the sample %q is %d, which is not a plausible unix-seconds value", m[1], value)
	}
}
