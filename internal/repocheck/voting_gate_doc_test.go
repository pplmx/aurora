package repocheck

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gateRow matches one row of the vote-gate table in docs/api.md:
// "| 3 | 早于 `start_time` | `SESSION_NOT_STARTED` |".
var gateRow = regexp.MustCompile("\\|\\s*(\\d+)\\s*\\|[^|]*\\|\\s*`([A-Z_]+)`\\s*\\|")

// TestVoteGateDocMatchesTheCheck pins the vote-gate table to the function it
// describes. The order matters to a client: a draft session inside its window
// reports "not started" rather than "ended", and the difference decides whether
// calling start can fix it. The table was prose before, which could state the
// conditions but not prove their order.
func TestVoteGateDocMatchesTheCheck(t *testing.T) {
	code := readRepoFile(t, "internal", "domain", "voting", "entity.go")
	start := strings.Index(code, "func (s *Session) CanAcceptVotes(")
	require.GreaterOrEqual(t, start, 0, "entity.go must keep CanAcceptVotes")
	end := strings.Index(code[start:], "\n}")
	require.Greater(t, end, 0, "the function must end")
	body := code[start : start+end]

	// Each `return ErrX` in the order the function reaches it.
	returns := regexp.MustCompile(`return (Err[A-Za-z]+)`).FindAllStringSubmatch(body, -1)
	require.Len(t, returns, 5, "the gate has five outcomes")

	doc := readRepoFile(t, "docs", "api.md")
	tableStart := strings.Index(doc, "| 判定顺序 | 情形")
	require.GreaterOrEqual(t, tableStart, 0, "docs/api.md must keep the vote-gate table")
	tableEnd := strings.Index(doc[tableStart:], "\n\n")
	require.Greater(t, tableEnd, 0, "the table must end at a blank line")
	table := doc[tableStart : tableStart+tableEnd]

	rows := gateRow.FindAllStringSubmatch(table, -1)
	require.Len(t, rows, 5, "the table must list all five outcomes")

	// The table is written in the same order the function checks them, so a
	// reordering of one has to reach the other.
	for i, m := range rows {
		want := strings.TrimPrefix(returns[i][1], "Err")
		got := strings.ReplaceAll(m[2], "_", "")
		assert.Equalf(t, strings.ToLower(want), strings.ToLower(got),
			"row %s of the table says %s, but check %d returns %s", m[1], m[2], i+1, returns[i][1])
	}
}
