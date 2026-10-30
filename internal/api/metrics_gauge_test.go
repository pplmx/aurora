package api

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pplmx/aurora/internal/metrics"
)

// TestOutboxGaugeReportsThePendingCount pins the exposed queue depth and the
// failure rule: a count error drops the line for that scrape instead of writing
// zero over a real backlog.
func TestOutboxGaugeReportsThePendingCount(t *testing.T) {
	reg := metrics.NewRegistry()
	registerRuntimeGauges(reg, func(context.Context) (int, error) { return 4, nil }, func() int { return 7 })
	out := reg.Expose()
	assert.Contains(t, out, "# TYPE aurora_outbox_pending_events gauge\n")
	assert.Contains(t, out, "\naurora_outbox_pending_events 4\n")
	assert.Contains(t, out, "\naurora_ledger_blocks 7\n")

	failing := metrics.NewRegistry()
	registerRuntimeGauges(failing, func(context.Context) (int, error) {
		return 0, errors.New("database is busy")
	}, nil)
	out = failing.Expose()
	assert.NotContains(t, out, "aurora_outbox_pending_events")
	assert.NotContains(t, out, "aurora_ledger_blocks", "no reader means no line")
}

// TestRegisterBuildInfo pins the identity metric: version and toolchain as
// labels with value 1, sorted for a byte-stable scrape, and build_time only
// when the linker injected a real one.
func TestRegisterBuildInfo(t *testing.T) {
	srv := &Server{metrics: metrics.NewRegistry()}
	srv.RegisterBuildInfo("v1.2.3", "unknown", "go1.27.3")
	out := srv.metrics.Expose()
	assert.Contains(t, out, `aurora_build_info{go="go1.27.3",version="v1.2.3"} 1`)
	assert.NotContains(t, out, "build_time", "an unknown build time is omitted")

	withTime := &Server{metrics: metrics.NewRegistry()}
	withTime.RegisterBuildInfo("v1.2.3", "2026-09-03T00:00:00Z", "go1.27.3")
	out = withTime.metrics.Expose()
	assert.Contains(t, out, `build_time="2026-09-03T00:00:00Z"`)
}
