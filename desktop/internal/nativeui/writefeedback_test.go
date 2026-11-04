package nativeui

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// statefulTokens is a fake that stores what it creates. The shared
// fakeServices returns a row without keeping it, which is enough for the
// write-path tests but cannot show whether the page re-reads afterwards.
type statefulTokens struct {
	*fakeServices
	rows []TokenRow
}

func newStatefulTokens() *statefulTokens {
	return &statefulTokens{fakeServices: &fakeServices{}}
}

func (s *statefulTokens) ListTokens(limit, offset int) ([]TokenRow, error) {
	s.listTokensCalls++
	if offset > len(s.rows) {
		return nil, nil
	}
	out := s.rows[offset:]
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *statefulTokens) CreateToken(name, symbol, supply, owner string, decimals int) (TokenRow, error) {
	s.createTokenCalls++
	if s.createErr != nil {
		return TokenRow{}, s.createErr
	}
	row := TokenRow{
		ID: fmt.Sprintf("id-%d", len(s.rows)+1), Name: name, Symbol: symbol,
		TotalSupply: supply, Decimals: decimals, Owner: owner,
	}
	s.rows = append(s.rows, row)
	return row, nil
}

// TestACreateReReadsTheList holds the loop a successful write closes. The status
// line says the token was made; the list beside it must show it, or the operator
// is left with a claim and no evidence. The shared fake returns a row without
// keeping it, so this uses one that stores what it is given.
func TestACreateReReadsTheList(t *testing.T) {
	svc := newStatefulTokens()
	app := New(svc)
	tester := ui.NewTester(app.View, 1280, 2400)
	app.ShowPage("Token")
	tester.Frame()
	require.Empty(t, app.token.tokens, "the store starts empty")
	before := svc.listTokensCalls

	app.token.name, app.token.symbol, app.token.supply = "Gold", "GLD", "100"
	app.token.owner = "pk"
	app.createToken()
	tester.Frame()

	require.Equal(t, "Created Gold (GLD) with supply 100. Owner: pk", app.token.status.text)
	require.Greater(t, svc.listTokensCalls, before, "the write must re-read the list")
	require.Len(t, app.token.tokens, 1, "and the new token must be in it")
	require.True(t, tester.HasText("Gold"), "the row the write created is on screen")
}

// TestACreateThatFailsDoesNotClaimTheListIsNewer: a rejected write must leave the
// list as it was. Re-reading on failure would show the same rows with no
// explanation, and a reader would take the unchanged list for a confirmation.
func TestACreateThatFailsDoesNotClaimTheListIsNewer(t *testing.T) {
	svc := newStatefulTokens()
	svc.createErr = fmt.Errorf("owner key rejected")
	app := New(svc)
	tester := ui.NewTester(app.View, 1280, 2400)
	app.ShowPage("Token")
	tester.Frame()
	before := svc.listTokensCalls

	app.token.name, app.token.symbol, app.token.supply = "Gold", "GLD", "100"
	app.token.owner = "pk"
	app.createToken()
	tester.Frame()

	require.Contains(t, app.token.status.text, "owner key rejected")
	require.Equal(t, before, svc.listTokensCalls, "a failed write must not re-read the list")
	require.Empty(t, app.token.tokens)
}
