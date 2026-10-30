package oracleapp

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pplmx/aurora/internal/domain/oracle"
)

type blockingCtxFetcher struct {
	once    sync.Once
	started chan struct{}
	release chan struct{}
}

func (f *blockingCtxFetcher) FetchData(*oracle.DataSource) (*oracle.OracleData, error) {
	return &oracle.OracleData{}, nil
}

func (f *blockingCtxFetcher) FetchDataContext(_ context.Context, source *oracle.DataSource) (*oracle.OracleData, error) {
	f.once.Do(func() { close(f.started) })
	<-f.release
	return &oracle.OracleData{ID: "data-" + source.ID, SourceID: source.ID}, nil
}

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
	require.Contains(t, rt.PrometheusText(), "aurora_oracle_fetch_attempts_total")
}

func TestRuntimeRestartWaitsForPreviousScheduler(t *testing.T) {
	repo := &mockOracleRepo{sources: []*oracle.DataSource{{ID: "a", Enabled: true, Interval: 3600}}}
	fetcher := &blockingCtxFetcher{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	rt := NewRuntime(repo, fetcher, &mockChain{})

	firstStop := rt.Start(context.Background(), time.Hour)
	defer firstStop()
	<-fetcher.started

	restarted := make(chan func(), 1)
	go func() {
		restarted <- rt.Start(context.Background(), time.Hour)
	}()

	select {
	case secondStop := <-restarted:
		secondStop()
		t.Fatal("Start replaced the scheduler while its previous fetch still held the repository")
	case <-time.After(20 * time.Millisecond):
	}

	close(fetcher.release)
	select {
	case secondStop := <-restarted:
		secondStop()
	case <-time.After(time.Second):
		t.Fatal("Start did not return after the previous scheduler exited")
	}
}
