package repocheck

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	metricNameDecl = regexp.MustCompile(`"?(aurora_[a-z_]+)"?`)
	docMetricName  = regexp.MustCompile("`(aurora_[a-z_]+)")
)

// TestEveryMetricInTheReferenceExists keeps the metrics list honest in the
// direction that misleads most: a documented metric that no longer exists sends
// an operator to build a dashboard or an alert on a series nothing emits, and
// the graph is empty rather than broken, so nothing points at the mistake.
func TestEveryMetricInTheReferenceExists(t *testing.T) {
	emitted := map[string]bool{}
	// The three places metrics are defined: the registry's own series, the
	// gauges the API server registers, and the oracle scheduler's text output.
	for _, rel := range [][]string{
		{"internal", "metrics", "metrics.go"},
		{"internal", "api", "metrics_gauge.go"},
		{"internal", "app", "oracleapp", "scheduler.go"},
	} {
		for _, m := range metricNameDecl.FindAllStringSubmatch(readRepoFile(t, rel...), -1) {
			// Histograms emit _bucket/_count/_sum as derived series; the family
			// name is what the reference documents.
			if strings.HasSuffix(m[1], "_bucket") || strings.HasSuffix(m[1], "_count") || strings.HasSuffix(m[1], "_sum") {
				continue
			}
			emitted[m[1]] = true
		}
	}
	require.NotEmpty(t, emitted, "the metric definitions must name their series")

	doc := readRepoFile(t, "docs", "api.md")
	start := strings.Index(doc, "## 指标")
	require.GreaterOrEqual(t, start, 0, "docs/api.md must keep its metrics section")
	end := strings.Index(doc[start:], "## 抽奖")
	require.Greater(t, end, 0, "the metrics section must end before the next one")
	section := doc[start : start+end]

	documented := 0
	for _, m := range docMetricName.FindAllStringSubmatch(section, -1) {
		documented++
		assert.Truef(t, emitted[m[1]],
			"docs/api.md documents %s, but no package registers it", m[1])
	}
	require.GreaterOrEqual(t, documented, 10, "the section should document the series it has")
}
