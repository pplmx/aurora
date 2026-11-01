package repocheck

import (
	"fmt"
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

// TestOracleQueryLimitDocMatchesTheCode covers the one collection read whose
// numbers the reference states in prose rather than in a table. Every other
// list documents its page size and cap, and `/oracle/query` documents neither,
// so a caller could not tell that asking for 500 readings yields 100.
func TestOracleQueryLimitDocMatchesTheCode(t *testing.T) {
	code := readRepoFile(t, "internal", "api", "handler", "oracle.go")
	defaultLimit := readConst(t, code, "defaultQueryLimit")
	maxLimit := readConst(t, code, "maxQueryLimit")
	require.Greater(t, defaultLimit, 0, "the query endpoint needs a default page")
	require.Greater(t, maxLimit, defaultLimit, "the cap must allow more than the default")

	// The CLI states the same pair on its own help path; a reader switching
	// surfaces must not get a different page for the same --limit.
	cli := readRepoFile(t, "cmd", "aurora", "cmd", "oracle_read.go")
	require.Equal(t, maxLimit, readConst(t, cli, "maxCLIQueryLimit"),
		"the CLI cap mirrors the API's")
	assert.Contains(t, cli, "return 10",
		"the CLI default matches the API default; the two must move together")

	doc := strings.Join(strings.Fields(readRepoFile(t, "docs", "api.md")), " ")
	// Read the sentence that starts at the endpoint's name, so the cap asserted
	// here is the one for /oracle/query and not the eight other endpoints that
	// also say "`limit` 上限 100".
	start := strings.Index(doc, "`/oracle/query` 默认返回")
	require.GreaterOrEqual(t, start, 0,
		"the reference must carry a sentence about /oracle/query's page size")
	end := strings.Index(doc[start:], "`/oracle/health` 的每项")
	require.Greater(t, end, 0, "the query note must end before the health note")
	note := doc[start : start+end]

	assert.Contains(t, note, strconv.Itoa(defaultLimit)+" 条读数",
		"the /oracle/query note must state the default page the handler applies")
	assert.Contains(t, note, "`limit` 上限 "+strconv.Itoa(maxLimit),
		"the /oracle/query note must state the cap the handler applies")
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
	// Forms in use: `4 << 20`, `64 << 10`, `128`, and `10 * 1024 * 1024`.
	// Multiplication is accepted because several constants are written that
	// way; it folds left to right so a chain of factors evaluates as Go would.
	if i := strings.Index(expr, "<<"); i >= 0 {
		left, err := strconv.Atoi(strings.TrimSpace(expr[:i]))
		require.NoError(t, err)
		right, err := strconv.Atoi(strings.TrimSpace(expr[i+2:]))
		require.NoError(t, err)
		return left << right
	}
	if strings.Contains(expr, "*") {
		value := 1
		for _, factor := range strings.Split(expr, "*") {
			n, err := strconv.Atoi(strings.TrimSpace(factor))
			require.NoErrorf(t, err, "%s = %q", name, expr)
			value *= n
		}
		return value
	}
	value, err := strconv.Atoi(expr)
	require.NoErrorf(t, err, "%s = %q", name, expr)
	return value
}

// TestPaginationCeilingsMatchTheDocs binds the per-endpoint page bounds in the
// REST reference to the constants that enforce them. Seven list endpoints each
// carry a limit ceiling and an offset ceiling, and two of them deliberately
// differ from the rest (`/token/history` and `/nft/list` stop at 1000 where the
// others allow 10000). The reference spells those numbers out, so a change to
// one constant has to reach the prose — otherwise a client pages past a bound
// the docs still advertise as available.
func TestPaginationCeilingsMatchTheDocs(t *testing.T) {
	doc := strings.Join(strings.Fields(readRepoFile(t, "docs", "api.md")), " ")

	sources := map[string]string{}
	for _, name := range []string{"blockchain.go", "lottery.go", "nft.go", "oracle.go", "token.go", "voting.go"} {
		sources[name] = readRepoFile(t, "internal", "api", "handler", name)
	}

	type want struct {
		constName string
		file      string
		value     int
		mustSay   string
	}
	for _, tc := range []want{
		{"maxBlockPageLimit", "blockchain.go", 100, ""},
		{"maxBlockPageOffset", "blockchain.go", 10000, ""},
		{"maxLotteryHistoryLimit", "lottery.go", 100, ""},
		{"maxLotteryHistoryOffset", "lottery.go", 10000, ""},
		{"maxNFTListLimit", "nft.go", 100, ""},
		{"maxNFTListOffset", "nft.go", 1000, ""},
		{"maxNFTHistoryLimit", "nft.go", 100, ""},
		{"maxNFTHistoryOffset", "nft.go", 10000, ""},
		{"maxHistoryLimit", "token.go", 100, ""},
		{"maxHistoryOffset", "token.go", 1000, ""},
		{"maxTokenListLimit", "token.go", 100, ""},
		{"maxTokenListOffset", "token.go", 10000, ""},
		{"maxOracleSourceLimit", "oracle.go", 100, ""},
		{"maxOracleSourceOffset", "oracle.go", 10000, ""},
		{"maxQueryLimit", "oracle.go", 100, ""},
		{"maxVotingListLimit", "voting.go", 100, ""},
		{"maxVotingListOffset", "voting.go", 10000, ""},
	} {
		got := readConst(t, sources[tc.file], tc.constName)
		require.Equalf(t, tc.value, got, "%s must stay at %d", tc.constName, tc.value)
	}

	// The two divergent ceilings are the ones worth calling out, because a
	// reader who assumes every list pages to 10000 would over-page these.
	require.Contains(t, doc, "`offset` 上限 **1000**",
		"the reference must flag the 1000-offset endpoints")
	require.Contains(t, doc, "`offset` 上限则是 10000",
		"the reference must contrast the 10000-offset endpoints")
}

// TestTokenFieldLimitsAreDocumented binds the token name and symbol ceilings to
// the places a user reads them. Both are enforced (symbol 10 accepts ten bytes
// and rejects eleven), but neither the CLI help nor the REST reference stated
// them: the same command's -d flag already said "0-127", so the omission read
// as "no limit" next to a flag that had one. A client learned the bound only by
// tripping TOKEN_SYMBOL_TOO_LONG after the fact.
func TestTokenFieldLimitsAreDocumented(t *testing.T) {
	validator := readRepoFile(t, "internal", "domain", "token", "validator.go")
	nameLimit := readConst(t, validator, "MaxTokenNameLength")
	symbolLimit := readConst(t, validator, "MaxTokenSymbolLength")
	require.Positive(t, nameLimit, "names have a ceiling")
	require.Positive(t, symbolLimit, "symbols have a ceiling")

	// The REST reference describes the create body field by field.
	doc := strings.Join(strings.Fields(readRepoFile(t, "docs", "api.md")), " ")
	assert.Containsf(t, doc, fmt.Sprintf("`name`（≤%d 字节）", nameLimit),
		"docs/api.md must state the %d-byte name ceiling", nameLimit)
	assert.Containsf(t, doc, fmt.Sprintf("`symbol`（≤%d 字节）", symbolLimit),
		"docs/api.md must state the %d-byte symbol ceiling", symbolLimit)

	// The CLI help is where a user meets the bound while typing the flag. The
	// check reads the specific flag entry rather than the file: a "the number
	// appears somewhere in a 600-line catalog" assertion stays green after the
	// help text loses its bound, because other entries carry small integers too.
	for _, name := range []string{"messages_en.go", "messages_zh.go", "messages_zh_hant.go"} {
		catalog := readRepoFile(t, "internal", "i18n", name)
		values := map[string]string{}
		for _, m := range catalogEntry.FindAllStringSubmatch(catalog, -1) {
			values[m[1]] = m[2]
		}
		for key, limit := range map[string]int{
			"token.name":   nameLimit,
			"token.symbol": symbolLimit,
		} {
			value, ok := values[key]
			require.Truef(t, ok, "%s must keep the %q help entry", name, key)
			assert.Containsf(t, value, strconv.Itoa(limit),
				"%s: %q must state the %d-byte ceiling, got %q", name, key, limit, value)
		}
	}
}
