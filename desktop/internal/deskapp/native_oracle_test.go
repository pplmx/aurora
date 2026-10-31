package deskapp

import (
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
