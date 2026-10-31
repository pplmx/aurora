package deskapp

import (
	"errors"
	"testing"

	"github.com/pplmx/aurora/desktop/internal/nativeui"
	"github.com/pplmx/aurora/internal/domain/oracle"
	"github.com/stretchr/testify/require"
)

// oracleServices is the adapter over an in-memory oracle store, with one feed
// already registered so the lifecycle paths have something to act on.
func oracleServices(t *testing.T) (*nativeServices, string) {
	t.Helper()
	services := &nativeServices{oracleRepo: oracle.NewInmemRepo()}
	row, err := services.AddOracleSource(nativeui.SourceInput{
		Name: "btc-usd", URL: "https://example.test/btc", Type: "custom",
		Method: "GET", Interval: 60,
	})
	require.NoError(t, err)
	return services, row.ID
}

// TestOracleSourceEnableDisableTogglesState covers the one adapter method with a
// branch: the Sources card's Enable and Disable buttons both land here, and a
// reversed condition would silently flip the wrong way.
func TestOracleSourceEnableDisableTogglesState(t *testing.T) {
	services, id := oracleServices(t)

	require.NoError(t, services.SetOracleSourceEnabled(id, false))
	sources, err := services.OracleSources(0, 0)
	require.NoError(t, err)
	require.Len(t, sources, 1)
	require.False(t, sources[0].Enabled, "disable must clear the flag")

	require.NoError(t, services.SetOracleSourceEnabled(id, true))
	sources, err = services.OracleSources(0, 0)
	require.NoError(t, err)
	require.True(t, sources[0].Enabled, "enable must set it again")
}

// TestOracleSourceDeleteRemovesIt covers the destructive path the Delete button
// takes, including that the feed stops being listed afterwards.
func TestOracleSourceDeleteRemovesIt(t *testing.T) {
	services, id := oracleServices(t)

	require.NoError(t, services.DeleteOracleSource(id))

	sources, err := services.OracleSources(0, 0)
	require.NoError(t, err)
	require.Empty(t, sources, "a deleted feed must not be listed")
}

// TestOracleLatestExplainsAnEmptySource covers the nil branch: a feed that has
// never been fetched has no latest value, and the card must say so rather than
// render a zero row that looks like a real reading of zero.
func TestOracleLatestExplainsAnEmptySource(t *testing.T) {
	services, id := oracleServices(t)

	_, err := services.OracleLatest(id)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no data recorded",
		"the message must say the source has no reading yet")

	rows, err := services.OracleData(id, 10)
	require.NoError(t, err)
	require.Empty(t, rows, "an unfetched feed has no stored readings")
}

// stubFetcher stands in for the HTTP client. It returns whatever the test
// configured for the source it was asked about.
type stubFetcher struct {
	value string
	err   error
	asked string
}

func (f *stubFetcher) FetchData(source *oracle.DataSource) (*oracle.OracleData, error) {
	f.asked = source.ID
	if f.err != nil {
		return nil, f.err
	}
	return &oracle.OracleData{SourceID: source.ID, Value: f.value, Timestamp: 1, BlockHeight: 1}, nil
}

// TestFetchOracleSourceStoresTheReading covers the Fetch now button: the source
// is handed to the fetcher, and what comes back is readable through the same
// two reads the Readings card uses.
func TestFetchOracleSourceStoresTheReading(t *testing.T) {
	services, id := oracleServices(t)
	fetcher := &stubFetcher{value: "68421.50"}
	services.oracleFetcher = fetcher

	require.NoError(t, services.FetchOracleSource(id))
	require.Equal(t, id, fetcher.asked, "the fetcher must receive the source the button names")

	latest, err := services.OracleLatest(id)
	require.NoError(t, err)
	require.Equal(t, "68421.50", latest.Value)

	rows, err := services.OracleData(id, 10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "68421.50", rows[0].Value)
}

// TestFetchOracleSourceReportsTheFailure keeps the upstream error visible: a
// fetch that fails must not leave the card looking freshly updated.
func TestFetchOracleSourceReportsTheFailure(t *testing.T) {
	services, id := oracleServices(t)
	boom := errors.New("upstream returned 503")
	services.oracleFetcher = &stubFetcher{err: boom}

	err := services.FetchOracleSource(id)
	require.Error(t, err)

	_, latestErr := services.OracleLatest(id)
	require.Error(t, latestErr, "a failed fetch stores nothing, so there is still no latest value")
}
