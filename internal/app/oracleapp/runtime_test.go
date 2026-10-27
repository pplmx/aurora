package oracleapp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/domain/oracle"
)

func TestRuntimeStartsTheSchedulerAndExposesItsView(t *testing.T) {
	repo := &mockOracleRepo{sources: []*oracle.DataSource{{ID: "a", Enabled: true, Interval: 3600}}}
	rt := NewRuntime(repo, &mockFetcher{data: &oracle.OracleData{ID: "d1", SourceID: "a"}}, &mockChain{})

	require.Nil(t, rt.Fetch())
	require.Nil(t, rt.Stats())
	require.Empty(t, rt.PrometheusText())

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	stop := rt.Start(ctx, time.Hour)
	t.Cleanup(stop)

	require.NotNil(t, rt.Fetch())
	require.NotNil(t, rt.Fetch().Chain(), "the runtime must wire the on-chain recorder")
	require.NotNil(t, rt.Stats())
	require.Contains(t, rt.PrometheusText(), "oracle_fetch_attempts_total")
}
