package repocheck

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// intConstPattern matches an integer constant declaration and captures its
// expression, so readConst can evaluate the shift and multiplication forms this
// repo writes its limits with.
func intConstPattern(name string) *regexp.Regexp {
	return regexp.MustCompile(`(?m)^(?:\s*const\s+|\s*)` + name + `\s*=\s*([0-9]+(?:\s*(?:<<|\*)\s*[0-9]+)*)`)
}

// TestRequestLimitsDocMatchesTheCode keeps the limits table in the API reference
// tied to the constants that enforce it. A limit is a promise about when a
// request is refused, and a table of promises is exactly what drifts once and
// then misleads.
func TestRequestLimitsDocMatchesTheCode(t *testing.T) {
	doc := readRepoFile(t, "docs", "api.md")
	start := strings.Index(doc, "下列上限与开关作用在每个请求上")
	require.GreaterOrEqual(t, start, 0, "docs/api.md must keep the limits table")
	end := strings.Index(doc[start:], "正常的 API 调用不需要接近请求头上限")
	require.Greater(t, end, 0, "the table must end before the trailing note")
	table := doc[start : start+end]

	body := readConst(t, readRepoFile(t, "internal", "api", "middleware", "bodylimit.go"), "MaxRequestBody")
	assert.Equal(t, 4, body/(1<<20), "the body limit the code sets")
	assert.Contains(t, table, "4 MiB", "the table must state the body limit the code sets")

	header := readConst(t, readRepoFile(t, "cmd", "api", "main.go"), "maxHeaderBytes")
	assert.Equal(t, 64, header/(1<<10), "the header limit the code sets")
	assert.Contains(t, table, "64 KiB", "the table must state the header limit the code sets")

	// Both refusals name their status code, so a client can branch on it.
	assert.Contains(t, table, "`413`", "the body refusal is a 413")
	assert.Contains(t, table, "`431`", "the header refusal is a 431")
}

// TestRequestIDAndLogLimitsMatchTheDoc covers the numbers the auth section
// states in prose rather than in the table.
func TestRequestIDAndLogLimitsMatchTheDoc(t *testing.T) {
	doc := readRepoFile(t, "docs", "api.md")

	idCode := readRepoFile(t, "internal", "api", "middleware", "requestid.go")
	assert.Equal(t, 128, readConst(t, idCode, "maxRequestIDLength"))
	assert.Contains(t, doc, "128 字符", "the request-id cap the code enforces")

	logCode := readRepoFile(t, "internal", "api", "middleware", "logger.go")
	method := readConst(t, logCode, "maxLoggedMethodLength")
	path := readConst(t, logCode, "maxLoggedPathLength")
	// The prose wraps across lines, so the sentence is compared with runs of
	// whitespace collapsed.
	flat := strings.Join(strings.Fields(doc), " ")
	assert.Contains(t, flat, strconv.Itoa(method)+"/"+strconv.Itoa(path)+" 字节",
		"the log truncation widths the code enforces")
}

// TestShutdownTimeoutDocMatchesTheCode covers the one duration the reference
// states in prose. A server that stops sooner than documented loses in-flight
// writes, and a reader planning a rollout relies on the number.
func TestShutdownTimeoutDocMatchesTheCode(t *testing.T) {
	code := readRepoFile(t, "cmd", "api", "main.go")
	seconds := readConst(t, code, "shutdownTimeout")
	require.Greater(t, seconds, 0, "the server must allow some drain time")

	doc := readRepoFile(t, "docs", "api.md")
	// Compared as a bool so a failure names the missing sentence instead of
	// printing the whole document.
	assert.NotEqual(t, -1, strings.Index(doc, strconv.Itoa(seconds)+"s 内优雅停机"),
		"the reference must state the drain window the code allows")
}

// readConst evaluates one integer constant declaration in src.
func readConst(t *testing.T, src, name string) int {
	t.Helper()
	m := intConstPattern(name).FindStringSubmatch(src)
	require.NotNilf(t, m, "%s must be declared as an integer constant", name)
	expr := strings.TrimSpace(m[1])
	// Forms in use: `4 << 20`, `64 << 10`, `128`. Multiplication is accepted
	// because a future constant may be written that way.
	if i := strings.Index(expr, "<<"); i >= 0 {
		left, err := strconv.Atoi(strings.TrimSpace(expr[:i]))
		require.NoError(t, err)
		right, err := strconv.Atoi(strings.TrimSpace(expr[i+2:]))
		require.NoError(t, err)
		return left << right
	}
	if i := strings.Index(expr, "*"); i >= 0 {
		left, err := strconv.Atoi(strings.TrimSpace(expr[:i]))
		require.NoError(t, err)
		right, err := strconv.Atoi(strings.TrimSpace(expr[i+1:]))
		require.NoError(t, err)
		return left * right
	}
	value, err := strconv.Atoi(expr)
	require.NoErrorf(t, err, "%s = %q", name, expr)
	return value
}
