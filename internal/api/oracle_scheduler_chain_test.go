package api

import (
	"context"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// TestOracleSchedulerFetch_HasOnChainRecorder guards TASK-097 / ISS-090: the
// continuously running oracle scheduler must record fetched data on-chain just
// like the REST handler and CLI paths do. StartOracleScheduler previously
// built its FetchDataUseCase without calling SetChain, so every
// scheduler-persisted row was stored with block_height=0 (the app layer only
// calls chain.AddLotteryRecord when a chain is wired). The scheduler's fetch
// is stashed on the Server under the same mutex as the scheduler object; this
// test asserts it carries the on-chain recorder. A future change that drops
// the SetChain wiring — or the export seam that makes it observable — fails
// here before the divergence can ship.
func TestOracleSchedulerFetch_HasOnChainRecorder(t *testing.T) {
	resetForAPITest(t) // temp cwd so ./data/aurora.db and the blockchain singleton are sandboxed
	viper.Reset()
	viper.Set("api.key", "oracle-scheduler-test-key")

	srv, err := NewServer()
	require.NoError(t, err)
	require.NotNil(t, srv, "NewServer should boot against a temp DB")
	t.Cleanup(func() { _ = srv.Close() })

	// StartOracleScheduler is the only production path to the running fetch.
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	_ = srv.StartOracleScheduler(ctx, time.Second)

	srv.oracleMu.Lock()
	fetch := srv.oracleSchedulerFetch
	srv.oracleMu.Unlock()

	require.NotNil(t, fetch, "StartOracleScheduler must build a fetch use case")
	require.NotNil(t, fetch.Chain(),
		"the oracle scheduler's fetch must record on-chain (TASK-097, ISS-090); a nil chain stores every scheduler row at block_height=0")
}
