package repocheck

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLotteryIDResolutionDocMatchesTheResolver keeps the resolution table in the
// API reference tied to resolveRecord. The order is what a client depends on: a
// seed that happens to be all digits is read as a block height before it is
// tried as a substring, so the table has to say so.
func TestLotteryIDResolutionDocMatchesTheResolver(t *testing.T) {
	code := readRepoFile(t, "internal", "app", "lotteryapp", "verify.go")
	start := strings.Index(code, "func resolveRecord(")
	require.GreaterOrEqual(t, start, 0, "verify.go must keep resolveRecord")
	end := strings.Index(code[start:], "\n}\n")
	require.Greater(t, end, 0, "the function must end")
	body := code[start : start+end]

	// The three lookups, in the order the function performs them.
	exact := strings.Index(body, "repo.GetByID(")
	height := strings.Index(body, "repo.GetByBlockHeight(")
	substr := strings.Index(body, "repo.GetAll()")
	require.GreaterOrEqual(t, exact, 0, "the resolver tries an exact id")
	require.GreaterOrEqual(t, height, 0, "the resolver tries a block height")
	require.GreaterOrEqual(t, substr, 0, "the resolver falls back to a substring")
	assert.Less(t, exact, height, "an exact id is tried before a block height")
	assert.Less(t, height, substr, "a block height is tried before a substring")

	doc := readRepoFile(t, "docs", "api.md")
	require.Contains(t, doc, "`{id}` 按下面的顺序解析", "docs/api.md must keep the resolution intro")
	tableStart := strings.Index(doc, "| 顺序 | 形式")
	require.GreaterOrEqual(t, tableStart, 0, "docs/api.md must keep the resolution table")
	tableEnd := strings.Index(doc[tableStart:], "\n\n")
	require.Greater(t, tableEnd, 0, "the table must end at a blank line")
	table := doc[tableStart : tableStart+tableEnd]

	// Each row of the table, in order.
	rows := regexp.MustCompile(`(?m)^\|\s*(\d)\s*\|[^|]*\|`).FindAllStringSubmatch(table, -1)
	require.Len(t, rows, 3, "the table must have one row per step")
	for i, m := range rows {
		assert.Equalf(t, strconv.Itoa(i+1), m[1], "the table's rows must be numbered in order")
	}
	assert.Contains(t, table, "精确抽奖 ID", "step 1 is the exact id")
	assert.Contains(t, table, "区块高度", "step 2 is the block height")
	assert.Contains(t, table, "子串", "step 3 is the substring")

	// The 200-with-false outcome is the part scripts get wrong.
	assert.Contains(t, doc, "`integrity_ok` 为 `false`", "a failed check is still a 200")
	assert.Contains(t, doc, "仍返回 200")
}
