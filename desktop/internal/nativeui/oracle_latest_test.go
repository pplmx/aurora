package nativeui

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/stretchr/testify/require"
)

// oracleLatestApp builds the Oracle page with a latest-reading fixture, so the
// read path has something to succeed with.
func oracleLatestApp(t *testing.T, row *DataRow, err error) (*App, *fakeServices) {
	t.Helper()
	fake := &fakeServices{
		sources: []SourceRow{{ID: "src-1", Name: "btc-usd", URL: "https://example.test/btc", Enabled: true}},
	}
	fake.loadErrs.oracleLatestRow = row
	fake.loadErrs.oracleLatest = err
	app := New(fake)
	ui.NewTester(app.View, 1280, 1200).Frame()
	app.ShowPage("Oracle")
	return app, fake
}

// TestReadingTheLatestValueKeepsWhatItFetched covers the success path: the
// card renders the value and the block it was recorded at, and the action
// leaves no outcome line behind.
func TestReadingTheLatestValueKeepsWhatItFetched(t *testing.T) {
	app, fake := oracleLatestApp(t, &DataRow{ID: "d1", Value: "68421.50", BlockHeight: 7}, nil)
	app.oracle.latestID = "src-1"

	app.readOracleLatest()

	require.Equal(t, "src-1", fake.lastOracleLatestSource, "the read names the source in the field")
	require.NotNil(t, app.oracle.latest)
	require.Equal(t, "68421.50", app.oracle.latest.Value)
	require.Equal(t, int64(7), app.oracle.latest.BlockHeight)
	require.Equal(t, statusLine{}, app.oracle.status, "a successful read reports nothing")
}

// TestReadingTheLatestValueReportsAFailure is the other half: a source that
// cannot be read must leave its reason on the page instead of an empty readout
// that looks like a stored value of nothing.
func TestReadingTheLatestValueReportsAFailure(t *testing.T) {
	boom := errors.New("upstream returned 503")
	app, _ := oracleLatestApp(t, nil, boom)
	app.oracle.latestID = "src-1"

	app.readOracleLatest()

	require.True(t, app.oracle.status.isErr, "the failure is an error status")
	require.Contains(t, app.oracle.status.text, "upstream returned 503",
		"the reason is shown, not swallowed")
}

// TestReadingTheLatestValueNeedsASource covers the guard in front of both
// paths: without an id there is nothing to read, and the refusal must say which
// field is missing rather than issue an empty lookup.
func TestReadingTheLatestValueNeedsASource(t *testing.T) {
	app, fake := oracleLatestApp(t, &DataRow{Value: "1"}, nil)
	app.oracle.latestID = "   "

	app.readOracleLatest()

	require.Zero(t, fake.oracleLatestCalls, "an empty id must not reach the services")
	require.True(t, app.oracle.status.isErr)
	require.Contains(t, app.oracle.status.text, "Source ID is required for this read.")
}

// TestAProseReadingIsShownWhole covers the shape a real feed returns. The other
// latest tests use numeric values, but a source may answer with a sentence, and
// the readout has to show it as it came — a real fetch against
// api.github.com/zen returned "Avoid administrative distraction." for this
// series to read.
func TestAProseReadingIsShownWhole(t *testing.T) {
	const prose = "Avoid administrative distraction."
	app, _ := oracleLatestApp(t, &DataRow{ID: "d1", Value: prose, BlockHeight: 1}, nil)
	app.oracle.tab = 2 // Readings
	app.oracle.latestID = "src-1"
	app.readOracleLatest()

	tester := ui.NewTester(app.View, 1400, 2400)
	tester.Frame()
	require.True(t, tester.HasText(prose),
		"a feed that answers with a sentence must show it, not only numbers")
	require.True(t, tester.HasText("Block"), "and the block it was recorded at")
}
